package core

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"agcp/pkg/progress"
)

// Decompressor handles archive extraction
type Decompressor struct {
	bufferPool  *BufferPool
	concurrency int
}

// NewDecompressor creates a new Decompressor with default settings
func NewDecompressor() *Decompressor {
	return &Decompressor{
		bufferPool:  largeBufferPool, // Use larger buffers for better throughput
		concurrency: runtime.NumCPU(),
	}
}

// Decompress extracts an archive to the specified destination
func Decompress(input, decompressedName string) error {
	return NewDecompressor().Decompress(input, decompressedName)
}

// Decompress performs the extraction operation
func (d *Decompressor) Decompress(input, destName string) error {
	f, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer f.Close()

	header, tasks, dataOffset, err := d.parseArchive(f, destName)
	if err != nil {
		return err
	}

	totalSize := d.calculateTotalSize(tasks)
	progress.Init(totalSize)
	defer progress.Stop()

	outputDir := d.resolveOutputDir(header, destName)

	return d.extractFiles(input, dataOffset, tasks, header.Type, outputDir)
}

// parseArchive reads the archive header and builds extraction tasks
func (d *Decompressor) parseArchive(f *os.File, destName string) (*ArchiveHeader, []DecompressTask, int64, error) {
	br := bufio.NewReaderSize(f, 8192)

	header, err := d.readHeader(br)
	if err != nil {
		return nil, nil, 0, err
	}

	outputDir := d.resolveOutputDir(header, destName)
	tasks, err := d.readEntries(br, header, outputDir, f.Name(), destName)
	if err != nil {
		return nil, nil, 0, err
	}

	// Calculate data start offset
	filePos, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("seek current: %w", err)
	}
	dataOffset := filePos - int64(br.Buffered())

	return header, tasks, dataOffset, nil
}

// readHeader reads and validates the archive header
func (d *Decompressor) readHeader(br *bufio.Reader) (*ArchiveHeader, error) {
	header := &ArchiveHeader{}

	// Magic number
	if _, err := io.ReadFull(br, header.Magic[:]); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if string(header.Magic[:]) != Magic {
		return nil, fmt.Errorf("invalid magic number: %q", string(header.Magic[:]))
	}

	// Version
	if err := binary.Read(br, binary.BigEndian, &header.Version); err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	if header.Version != Version {
		return nil, fmt.Errorf("unsupported version: %d", header.Version)
	}

	// Archive type
	if err := binary.Read(br, binary.BigEndian, &header.Type); err != nil {
		return nil, fmt.Errorf("read archive type: %w", err)
	}

	// Root name
	var rootNameLen uint16
	if err := binary.Read(br, binary.BigEndian, &rootNameLen); err != nil {
		return nil, fmt.Errorf("read root name length: %w", err)
	}
	rootNameBytes := make([]byte, rootNameLen)
	if _, err := io.ReadFull(br, rootNameBytes); err != nil {
		return nil, fmt.Errorf("read root name: %w", err)
	}
	header.RootName = string(rootNameBytes)

	// Entry count
	if err := binary.Read(br, binary.BigEndian, &header.EntryCount); err != nil {
		return nil, fmt.Errorf("read entry count: %w", err)
	}

	return header, nil
}

// readEntries reads metadata for all entries in the archive
func (d *Decompressor) readEntries(br *bufio.Reader, header *ArchiveHeader, outputDir, inputPath, destName string) ([]DecompressTask, error) {
	tasks := make([]DecompressTask, header.EntryCount)

	for i := uint32(0); i < header.EntryCount; i++ {
		var relPathLen uint16
		if err := binary.Read(br, binary.BigEndian, &relPathLen); err != nil {
			return nil, fmt.Errorf("read relPath length %d: %w", i, err)
		}

		relPathBytes := make([]byte, relPathLen)
		if _, err := io.ReadFull(br, relPathBytes); err != nil {
			return nil, fmt.Errorf("read relPath %d: %w", i, err)
		}
		relPath := string(relPathBytes)

		var originalSize, compressedSize uint64
		if err := binary.Read(br, binary.BigEndian, &originalSize); err != nil {
			return nil, fmt.Errorf("read originalSize %d: %w", i, err)
		}
		if err := binary.Read(br, binary.BigEndian, &compressedSize); err != nil {
			return nil, fmt.Errorf("read compressedSize %d: %w", i, err)
		}

		destPath := d.resolveDestPath(header.Type, outputDir, relPath, header.RootName, inputPath, destName)

		tasks[i] = DecompressTask{
			RelPath:        relPath,
			OriginalSize:   originalSize,
			CompressedSize: compressedSize,
			DestPath:       destPath,
		}
	}

	return tasks, nil
}

// resolveOutputDir determines the base output directory
func (d *Decompressor) resolveOutputDir(header *ArchiveHeader, destName string) string {
	if destName != "" {
		return destName
	}
	if header.Type == ArchiveDir {
		return header.RootName
	}
	return "."
}

// resolveDestPath determines the destination path for a single entry
func (d *Decompressor) resolveDestPath(archiveType ArchiveType, outputDir, relPath, rootName, inputPath, userOutput string) string {
	switch archiveType {
	case ArchiveDir:
		return filepath.Join(outputDir, relPath)

	case ArchiveFile:
		return d.resolveFileDestPath(outputDir, relPath, rootName, inputPath, userOutput)

	default:
		return ""
	}
}

// resolveFileDestPath handles the complex logic for single-file archive destinations
func (d *Decompressor) resolveFileDestPath(outputDir, relPath, rootName, inputPath, userOutput string) string {
	if userOutput != "" {
		// Check if destination is an existing directory
		if info, err := os.Stat(userOutput); err == nil && info.IsDir() {
			if relPath == "" {
				return filepath.Join(userOutput, rootName)
			}
			return filepath.Join(userOutput, relPath)
		}

		// Non-empty relPath: treat as base directory
		if relPath != "" {
			return filepath.Join(userOutput, relPath)
		}

		// Use as exact file path
		return userOutput
	}

	if relPath != "" {
		return filepath.Join(outputDir, relPath)
	}

	// Generate filename from header or archive name
	fileName := rootName
	if fileName == "" {
		fileName = filepath.Base(inputPath)
		if ext := filepath.Ext(fileName); ext == ".agcp" {
			fileName = fileName[:len(fileName)-len(ext)]
		}
	}
	return filepath.Join(outputDir, fileName)
}

// calculateTotalSize computes the total uncompressed size
func (d *Decompressor) calculateTotalSize(tasks []DecompressTask) uint64 {
	var total uint64
	for i := range tasks {
		total += tasks[i].OriginalSize
	}
	if total == 0 {
		total = 1
	}
	return total
}

// extractFiles decompresses all files concurrently
func (d *Decompressor) extractFiles(archivePath string, dataOffset int64, tasks []DecompressTask, archiveType ArchiveType, outputDir string) error {
	// Compute offsets for each entry
	d.computeOffsets(tasks, dataOffset)

	// Create all required directories
	if err := d.createDirectories(tasks, archiveType, outputDir); err != nil {
		return err
	}

	return d.extractConcurrently(archivePath, tasks)
}

// computeOffsets calculates the file offset for each entry
func (d *Decompressor) computeOffsets(tasks []DecompressTask, startOffset int64) {
	offset := startOffset
	for i := range tasks {
		tasks[i].Offset = offset
		offset += int64(tasks[i].CompressedSize)
	}
}

// createDirectories creates all necessary output directories
func (d *Decompressor) createDirectories(tasks []DecompressTask, archiveType ArchiveType, outputDir string) error {
	// For directory archives, create the root directory
	if archiveType == ArchiveDir {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("create root dir %s: %w", outputDir, err)
		}
	}

	// Collect unique directories to avoid redundant calls
	dirs := make(map[string]struct{}, len(tasks))
	for i := range tasks {
		dir := filepath.Dir(tasks[i].DestPath)
		if dir != "" && dir != "." {
			dirs[dir] = struct{}{}
		}
	}

	// Create all directories
	for dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return nil
}

// extractConcurrently extracts files using a worker pool
func (d *Decompressor) extractConcurrently(archivePath string, tasks []DecompressTask) error {
	sem := make(chan struct{}, d.concurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, len(tasks))

	for i := range tasks {
		wg.Add(1)
		go func(task *DecompressTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := d.extractFile(archivePath, task); err != nil {
				errCh <- err
			}
		}(&tasks[i])
	}

	wg.Wait()
	close(errCh)

	// Return first error if any
	for err := range errCh {
		return err
	}

	return nil
}

// extractFile extracts a single file from the archive
func (d *Decompressor) extractFile(archivePath string, task *DecompressTask) error {
	// Handle empty files
	if task.OriginalSize == 0 {
		f, err := os.Create(task.DestPath)
		if err != nil {
			return fmt.Errorf("create empty file %s: %w", task.DestPath, err)
		}
		return f.Close()
	}

	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer archive.Close()

	// Create section reader for this entry's data
	sr := io.NewSectionReader(archive, task.Offset, int64(task.CompressedSize))

	return d.decompressToFile(sr, task)
}

// decompressToFile writes decompressed data to the destination file
func (d *Decompressor) decompressToFile(r io.Reader, task *DecompressTask) error {
	f, err := os.Create(task.DestPath)
	if err != nil {
		return fmt.Errorf("create file %s: %w", task.DestPath, err)
	}
	defer f.Close()

	// Use buffered writer for better I/O performance
	bw := getBufWriter(f)

	// Get pooled LZ4 reader
	zr := getLZ4Reader(r)

	// Get buffer from pool for copying
	buf := d.bufferPool.Get()

	// Create progress tracking writer
	pw := &progress.Writer{W: bw}

	// Copy with size limit using pooled buffer
	written, err := copyBufferN(pw, zr, int64(task.OriginalSize), *buf)

	// Return buffer to pool
	d.bufferPool.Put(buf)

	// Return LZ4 reader to pool
	putLZ4Reader(zr)

	// Flush and return buffered writer
	if flushErr := bw.Flush(); flushErr != nil && err == nil {
		err = flushErr
	}
	putBufWriter(bw)

	if err != nil && err != io.EOF {
		return fmt.Errorf("decompress %s: %w", task.DestPath, err)
	}

	if uint64(written) != task.OriginalSize {
		return fmt.Errorf("size mismatch for %s: expected %d, got %d", task.DestPath, task.OriginalSize, written)
	}

	return nil
}
