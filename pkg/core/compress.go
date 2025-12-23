package core

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"agcp/pkg/progress"
)

// Compressor handles file and directory compression
type Compressor struct {
	bufferPool *BufferPool
}

// NewCompressor creates a new Compressor with default settings
func NewCompressor() *Compressor {
	return &Compressor{
		bufferPool: largeBufferPool, // Use larger buffers for better throughput
	}
}

// Compress compresses a file or directory to the specified output path
func Compress(input, output string) error {
	return NewCompressor().Compress(input, output)
}

// Compress performs the compression operation
func (c *Compressor) Compress(input, output string) error {
	info, err := os.Stat(input)
	if err != nil {
		return fmt.Errorf("stat input: %w", err)
	}

	archiveType, rootName, entries, err := c.collectEntries(input, info)
	if err != nil {
		return err
	}

	totalSize := c.calculateTotalSize(entries)
	progress.Init(totalSize)
	defer progress.Stop()

	return c.writeArchive(entries, output, archiveType, rootName)
}

// collectEntries gathers all files to be compressed
func (c *Compressor) collectEntries(input string, info os.FileInfo) (ArchiveType, string, []Entry, error) {
	rootName := filepath.Base(input)

	if info.IsDir() {
		entries, err := c.walkDirectory(input)
		if err != nil {
			return ArchiveDir, "", nil, fmt.Errorf("collect entries: %w", err)
		}
		return ArchiveDir, rootName, entries, nil
	}

	// Single file
	entry := Entry{
		RelPath:  "",
		FilePath: input,
		Size:     info.Size(),
	}
	return ArchiveFile, rootName, []Entry{entry}, nil
}

// walkDirectory recursively collects all files in a directory
func (c *Compressor) walkDirectory(root string) ([]Entry, error) {
	// Pre-allocate with estimated capacity
	entries := make([]Entry, 0, 64)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("relative path for %s: %w", path, err)
		}

		entries = append(entries, Entry{
			RelPath:  relPath,
			FilePath: path,
			Size:     info.Size(),
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk directory %s: %w", root, err)
	}

	return entries, nil
}

// calculateTotalSize computes the total size of all entries
func (c *Compressor) calculateTotalSize(entries []Entry) uint64 {
	var total uint64
	for i := range entries {
		if entries[i].Size > 0 {
			total += uint64(entries[i].Size)
		}
	}
	if total == 0 {
		total = 1
	}
	return total
}

// writeArchive creates the compressed archive file
func (c *Compressor) writeArchive(entries []Entry, output string, archiveType ArchiveType, rootName string) error {
	if err := c.prepareOutputPath(output); err != nil {
		return err
	}

	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer f.Close()

	// Use buffered writer for better I/O performance
	bw := getBufWriter(f)
	defer func() {
		bw.Flush()
		putBufWriter(bw)
	}()

	if err := c.writeHeader(bw, archiveType, rootName, len(entries)); err != nil {
		return err
	}

	// Flush buffered writer before seeking
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("flush header: %w", err)
	}

	entryOffsets, err := c.writePlaceholders(f, entries)
	if err != nil {
		return err
	}

	return c.compressEntries(f, entries, entryOffsets)
}

// prepareOutputPath ensures the output path is ready for writing
func (c *Compressor) prepareOutputPath(output string) error {
	if _, err := os.Stat(output); err == nil {
		if err := os.Remove(output); err != nil {
			return fmt.Errorf("remove existing output: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check output existence: %w", err)
	}

	dir := filepath.Dir(output)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}

	return nil
}

// writeHeader writes the archive header
func (c *Compressor) writeHeader(w io.Writer, archiveType ArchiveType, rootName string, entryCount int) error {
	// Magic number
	if _, err := w.Write([]byte(Magic)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}

	// Version
	if err := binary.Write(w, binary.BigEndian, Version); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	// Archive type
	if err := binary.Write(w, binary.BigEndian, archiveType); err != nil {
		return fmt.Errorf("write archive type: %w", err)
	}

	// Root name
	rootNameBytes := []byte(rootName)
	if err := binary.Write(w, binary.BigEndian, uint16(len(rootNameBytes))); err != nil {
		return fmt.Errorf("write root name length: %w", err)
	}
	if _, err := w.Write(rootNameBytes); err != nil {
		return fmt.Errorf("write root name: %w", err)
	}

	// Entry count
	if err := binary.Write(w, binary.BigEndian, uint32(entryCount)); err != nil {
		return fmt.Errorf("write entry count: %w", err)
	}

	return nil
}

// writePlaceholders reserves space for entry metadata
func (c *Compressor) writePlaceholders(f *os.File, entries []Entry) ([]int64, error) {
	offsets := make([]int64, len(entries))

	// Reuse a single zero buffer for placeholders
	maxPlaceholderSize := 2 + MaxPathLength + 16
	zeroBuf := make([]byte, maxPlaceholderSize)

	for i, entry := range entries {
		offset, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, fmt.Errorf("seek for entry %d: %w", i, err)
		}
		offsets[i] = offset

		// Placeholder: relPathLen(2) + relPath + originalSize(8) + compressedSize(8)
		placeholderSize := 2 + len(entry.RelPath) + 16
		if _, err := f.Write(zeroBuf[:placeholderSize]); err != nil {
			return nil, fmt.Errorf("write placeholder %d: %w", i, err)
		}
	}

	return offsets, nil
}

// compressEntries compresses each entry and updates metadata
func (c *Compressor) compressEntries(f *os.File, entries []Entry, offsets []int64) error {
	for i, entry := range entries {
		startPos, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("seek start for %s: %w", entry.FilePath, err)
		}

		originalSize, err := c.compressFile(entry.FilePath, f)
		if err != nil {
			return fmt.Errorf("compress %s: %w", entry.FilePath, err)
		}

		endPos, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("seek end for %s: %w", entry.FilePath, err)
		}

		compressedSize := uint64(endPos - startPos)

		if err := c.updateMetadata(f, offsets[i], entry.RelPath, originalSize, compressedSize); err != nil {
			return err
		}

		if _, err := f.Seek(endPos, io.SeekStart); err != nil {
			return fmt.Errorf("seek after metadata update: %w", err)
		}
	}

	return nil
}

// updateMetadata writes the actual metadata for an entry
func (c *Compressor) updateMetadata(f *os.File, offset int64, relPath string, originalSize, compressedSize uint64) error {
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return fmt.Errorf("seek to metadata: %w", err)
	}

	relPathBytes := []byte(relPath)
	if err := binary.Write(f, binary.BigEndian, uint16(len(relPathBytes))); err != nil {
		return fmt.Errorf("write relPath length: %w", err)
	}
	if _, err := f.Write(relPathBytes); err != nil {
		return fmt.Errorf("write relPath: %w", err)
	}
	if err := binary.Write(f, binary.BigEndian, originalSize); err != nil {
		return fmt.Errorf("write originalSize: %w", err)
	}
	if err := binary.Write(f, binary.BigEndian, compressedSize); err != nil {
		return fmt.Errorf("write compressedSize: %w", err)
	}

	return nil
}

// compressFile compresses a single file using pooled LZ4 writer
func (c *Compressor) compressFile(filePath string, w io.Writer) (uint64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat file: %w", err)
	}

	if info.Size() == 0 {
		return 0, nil
	}

	// Use buffered reader for input file
	br := bufio.NewReaderSize(f, 32*1024)

	// Get pooled LZ4 writer
	zw := getLZ4Writer(w)

	// Get buffer from pool
	buf := c.bufferPool.Get()

	var totalBytes uint64
	for {
		n, err := br.Read(*buf)
		if n > 0 {
			if _, werr := zw.Write((*buf)[:n]); werr != nil {
				c.bufferPool.Put(buf)
				putLZ4Writer(zw)
				return 0, fmt.Errorf("write compressed data: %w", werr)
			}
			totalBytes += uint64(n)
			progress.AddBytes(uint64(n))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			c.bufferPool.Put(buf)
			putLZ4Writer(zw)
			return 0, fmt.Errorf("read file: %w", err)
		}
	}

	// Return buffer to pool
	c.bufferPool.Put(buf)

	// Close and return LZ4 writer to pool
	if err := zw.Close(); err != nil {
		putLZ4Writer(zw)
		return 0, fmt.Errorf("close LZ4 writer: %w", err)
	}
	putLZ4Writer(zw)

	return totalBytes, nil
}
