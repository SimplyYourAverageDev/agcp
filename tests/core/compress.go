package core

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"agcp/pkg/progress"

	"github.com/pierrec/lz4/v4"
)

// Compress handles the compression process for files or directories
func Compress(input, output string) error {
	info, err := os.Stat(input)
	if err != nil {
		return fmt.Errorf("stat input: %w", err)
	}

	var archiveType ArchiveType
	var rootName string
	var entries []Entry
	if info.IsDir() {
		archiveType = ArchiveDir
		rootName = filepath.Base(input)
		entries, err = collectDirEntries(input)
		if err != nil {
			return fmt.Errorf("collect entries: %w", err)
		}
	} else {
		archiveType = ArchiveFile
		rootName = filepath.Base(input)
		entries = []Entry{{RelPath: "", FilePath: input}}
	}

	// Calculate total size for progress
	totalSize := calculateTotalSize(entries)
	progress.Init(totalSize)
	defer progress.Stop()

	return compressFiles(entries, output, archiveType, rootName)
}

// calculateTotalSize calculates the total size of all files to be compressed
func calculateTotalSize(entries []Entry) uint64 {
	var totalSize uint64
	for _, entry := range entries {
		info, err := os.Stat(entry.FilePath)
		if err != nil {
			continue
		}
		totalSize += uint64(info.Size())
	}
	if totalSize == 0 {
		totalSize = 1 // Avoid division by zero
	}
	return totalSize
}

// collectDirEntries gathers all files in a directory with relative paths
func collectDirEntries(root string) ([]Entry, error) {
	var entries []Entry
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return fmt.Errorf("relative path for %s: %w", path, err)
			}
			entries = append(entries, Entry{RelPath: relPath, FilePath: path})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory %s: %w", root, err)
	}
	return entries, nil
}

// compressFiles compresses files using LZ4 streaming and writes to the archive
func compressFiles(entries []Entry, output string, archiveType ArchiveType, rootName string) error {
	// Clean up existing output file
	if _, err := os.Stat(output); err == nil {
		if err := os.Remove(output); err != nil {
			return fmt.Errorf("remove existing output: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check output existence: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer f.Close()

	// Write header
	if err := writeArchiveHeader(f, archiveType, rootName, entries); err != nil {
		return err
	}

	// Write metadata placeholders
	entryOffsets := make([]int64, len(entries))
	for i, entry := range entries {
		entryOffsets[i], err = f.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("seek for entry %d: %w", i, err)
		}
		placeholderSize := 2 + len(entry.RelPath) + 8 + 8 // relPathLen + relPath + sizes
		if _, err = f.Write(make([]byte, placeholderSize)); err != nil {
			return fmt.Errorf("write placeholder %d: %w", i, err)
		}
	}

	// Compress and update metadata
	for i, entry := range entries {
		startPos, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("seek start for %s: %w", entry.FilePath, err)
		}
		originalSize, err := compressFileStreaming(entry.FilePath, f)
		if err != nil {
			return fmt.Errorf("compress %s: %w", entry.FilePath, err)
		}
		endPos, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			return fmt.Errorf("seek end for %s: %w", entry.FilePath, err)
		}
		compressedSize := uint64(endPos - startPos)

		// Update metadata
		if err := updateEntryMetadata(f, entryOffsets[i], entry.RelPath, originalSize, compressedSize); err != nil {
			return err
		}

		if _, err = f.Seek(endPos, io.SeekStart); err != nil {
			return fmt.Errorf("seek back %d: %w", i, err)
		}
	}
	return nil
}

// writeArchiveHeader writes the archive header to the output file
func writeArchiveHeader(f *os.File, archiveType ArchiveType, rootName string, entries []Entry) error {
	if _, err := f.Write([]byte(Magic)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}
	if err := binary.Write(f, binary.BigEndian, uint8(Version)); err != nil {
		return fmt.Errorf("write version: %w", err)
	}
	if err := binary.Write(f, binary.BigEndian, archiveType); err != nil {
		return fmt.Errorf("write archive type: %w", err)
	}

	rootNameBytes := []byte(rootName)
	if err := binary.Write(f, binary.BigEndian, uint16(len(rootNameBytes))); err != nil {
		return fmt.Errorf("write root name length: %w", err)
	}
	if _, err := f.Write(rootNameBytes); err != nil {
		return fmt.Errorf("write root name: %w", err)
	}
	if err := binary.Write(f, binary.BigEndian, uint32(len(entries))); err != nil {
		return fmt.Errorf("write number of entries: %w", err)
	}

	return nil
}

// updateEntryMetadata updates the metadata for an entry in the archive
func updateEntryMetadata(f *os.File, offset int64, relPath string, originalSize, compressedSize uint64) error {
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return fmt.Errorf("seek metadata: %w", err)
	}

	relPathBytes := []byte(relPath)
	if err := binary.Write(f, binary.BigEndian, uint16(len(relPathBytes))); err != nil {
		return fmt.Errorf("write relPathLen: %w", err)
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

// compressFileStreaming compresses a file in chunks
func compressFileStreaming(filePath string, w io.Writer) (uint64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer f.Close()

	zw := lz4.NewWriter(w)
	defer zw.Close()

	info, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat %s: %w", filePath, err)
	}

	if info.Size() == 0 {
		return 0, nil // Empty file, no data written
	}

	buf := make([]byte, 32*1024)
	var totalBytes uint64
	for {
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return 0, fmt.Errorf("read %s: %w", filePath, err)
		}
		if n == 0 {
			break
		}
		if _, err = zw.Write(buf[:n]); err != nil {
			return 0, fmt.Errorf("write compressed %s: %w", filePath, err)
		}
		totalBytes += uint64(n)
		progress.AddBytes(uint64(n))
	}
	if err := zw.Close(); err != nil {
		return 0, fmt.Errorf("close LZ4 writer %s: %w", filePath, err)
	}
	return totalBytes, nil
}
