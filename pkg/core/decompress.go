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

	"github.com/pierrec/lz4/v4"
)

// Decompress handles the decompression process
func Decompress(input, decompressedName string) error {
	f, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer f.Close()

	// Read and validate archive header
	tasks, startOffset, outputDir, archiveType, err := readArchiveHeader(f, decompressedName)
	if err != nil {
		return err
	}

	// Calculate total size for progress tracking
	var totalSize uint64
	for _, task := range tasks {
		totalSize += task.OriginalSize
	}
	if totalSize == 0 {
		totalSize = 1
	}
	progress.Init(totalSize)
	defer progress.Stop()

	return decompressFiles(input, startOffset, tasks, archiveType, outputDir)
}

// readArchiveHeader reads and validates the archive header
func readArchiveHeader(f *os.File, decompressedName string) ([]DecompressTask, int64, string, ArchiveType, error) {
	br := bufio.NewReader(f)

	// Read magic number
	var magicBytes [4]byte
	if _, err := io.ReadFull(br, magicBytes[:]); err != nil {
		return nil, 0, "", ArchiveDir, fmt.Errorf("read magic: %w", err)
	}
	if string(magicBytes[:]) != Magic {
		return nil, 0, "", ArchiveDir, fmt.Errorf("invalid magic number: %q", string(magicBytes[:]))
	}

	// Read version
	var versionByte uint8
	if err := binary.Read(br, binary.BigEndian, &versionByte); err != nil {
		return nil, 0, "", ArchiveDir, fmt.Errorf("read version: %w", err)
	}
	if versionByte != Version {
		return nil, 0, "", ArchiveDir, fmt.Errorf("unsupported version: %d", versionByte)
	}

	// Read archive type
	var archiveType ArchiveType
	if err := binary.Read(br, binary.BigEndian, &archiveType); err != nil {
		return nil, 0, "", ArchiveDir, fmt.Errorf("read archive type: %w", err)
	}

	// Read root name
	var rootNameLen uint16
	if err := binary.Read(br, binary.BigEndian, &rootNameLen); err != nil {
		return nil, 0, "", ArchiveDir, fmt.Errorf("read root name length: %w", err)
	}
	rootNameBytes := make([]byte, rootNameLen)
	if _, err := io.ReadFull(br, rootNameBytes); err != nil {
		return nil, 0, "", ArchiveDir, fmt.Errorf("read root name: %w", err)
	}
	rootName := string(rootNameBytes)

	// Decide the top-level output path.
	// Directory archives: default to the original root folder name.
	// Single-file archives: default to current directory; a provided name is treated as the full output file path.
	var outputDir string
	if decompressedName != "" {
		outputDir = decompressedName
	} else if archiveType == ArchiveDir {
		outputDir = rootName
	} else {
		outputDir = "."
	}

	// Read number of entries
	var numEntries uint32
	if err := binary.Read(br, binary.BigEndian, &numEntries); err != nil {
		return nil, 0, "", ArchiveDir, fmt.Errorf("read num entries: %w", err)
	}

	// Read metadata for each entry
	tasks := make([]DecompressTask, numEntries)
	for i := 0; i < int(numEntries); i++ {
		var relPathLen uint16
		if err := binary.Read(br, binary.BigEndian, &relPathLen); err != nil {
			return nil, 0, "", ArchiveDir, fmt.Errorf("read relPathLen %d: %w", i, err)
		}
		relPathBytes := make([]byte, relPathLen)
		if _, err := io.ReadFull(br, relPathBytes); err != nil {
			return nil, 0, "", ArchiveDir, fmt.Errorf("read relPath %d: %w", i, err)
		}
		relPath := string(relPathBytes)

		var originalSize, compressedSize uint64
		if err := binary.Read(br, binary.BigEndian, &originalSize); err != nil {
			return nil, 0, "", ArchiveDir, fmt.Errorf("read originalSize %d: %w", i, err)
		}
		if err := binary.Read(br, binary.BigEndian, &compressedSize); err != nil {
			return nil, 0, "", ArchiveDir, fmt.Errorf("read compressedSize %d: %w", i, err)
		}

		// Determine destination path
		destPath := determineDestPath(archiveType, outputDir, relPath, rootName, f.Name(), decompressedName)

		tasks[i] = DecompressTask{
			RelPath:        relPath,
			OriginalSize:   originalSize,
			CompressedSize: compressedSize,
			DestPath:       destPath,
		}
	}

	// Calculate start offset for compressed data
	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, 0, "", ArchiveDir, fmt.Errorf("seek current: %w", err)
	}
	buffered := br.Buffered()
	startOffset := offset - int64(buffered)

	return tasks, startOffset, outputDir, archiveType, nil
}

// determineDestPath decides where an extracted entry should be written.
//
//	archiveType      – whether the archive represents a directory or a single file
//	baseOutputDir    – resolved top-level output directory (root folder for directory archives or "." for file archives)
//	relPath          – relative path stored in the archive entry metadata
//	rootName         – name of the root directory or file recorded in the header
//	inputPath        – path of the .agcp archive on disk (used only for fallback names)
//	userOutputName   – raw value provided by the user on the CLI (may be empty)
func determineDestPath(archiveType ArchiveType, baseOutputDir, relPath, rootName, inputPath, userOutputName string) string {
	switch archiveType {
	case ArchiveDir:
		// Always preserve structure inside the chosen base directory.
		return filepath.Join(baseOutputDir, relPath)

	case ArchiveFile:
		if userOutputName != "" {
			// If destination exists and is a directory, place the file inside it.
			if info, err := os.Stat(userOutputName); err == nil && info.IsDir() {
				// Use provided directory but preserve original/root filename.
				if relPath == "" {
					return filepath.Join(userOutputName, rootName)
				}
				return filepath.Join(userOutputName, relPath)
			}

			// If relPath is non-empty treat userOutputName as base dir to preserve structure.
			if relPath != "" {
				return filepath.Join(userOutputName, relPath)
			}

			// Otherwise treat it as the exact file path the user wants.
			return userOutputName
		}

		if relPath != "" {
			return filepath.Join(baseOutputDir, relPath)
		}

		// Generate filename from header or archive filename.
		fileName := rootName
		if fileName == "" {
			fileName = filepath.Base(inputPath)
			if ext := filepath.Ext(fileName); ext == ".agcp" {
				fileName = fileName[:len(fileName)-len(ext)]
			}
		}
		return filepath.Join(baseOutputDir, fileName)
	}
	return ""
}

// decompressFiles decompresses files concurrently
func decompressFiles(archivePath string, startOffset int64, tasks []DecompressTask, archiveType ArchiveType, baseOutput string) error {
	// Calculate offsets for each compressed file in the archive
	offsets := make([]int64, len(tasks))
	currentOffset := startOffset
	for i, task := range tasks {
		offsets[i] = currentOffset
		currentOffset += int64(task.CompressedSize)
	}

	// For directory archives ensure the top-level directory exists.
	if archiveType == ArchiveDir {
		if err := os.MkdirAll(baseOutput, 0755); err != nil {
			return fmt.Errorf("create root dir %s: %w", baseOutput, err)
		}
	}

	// Pre-create directories for all files
	for _, task := range tasks {
		if err := os.MkdirAll(filepath.Dir(task.DestPath), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", task.DestPath, err)
		}
	}

	// Use a semaphore to limit concurrent goroutines
	sem := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup
	errCh := make(chan error, len(tasks))

	// Decompress files concurrently
	for i, task := range tasks {
		wg.Add(1)
		go func(task DecompressTask, offset int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			f, err := os.Open(archivePath)
			if err != nil {
				errCh <- fmt.Errorf("open archive for %s: %w", task.DestPath, err)
				return
			}
			defer f.Close()

			sr := io.NewSectionReader(f, offset, int64(task.CompressedSize))
			if err := decompressFileStreaming(sr, task); err != nil {
				errCh <- err
				return
			}
		}(task, offsets[i])
	}
	wg.Wait()
	close(errCh)

	// Return first error if any
	if len(errCh) > 0 {
		return <-errCh
	}
	return nil
}

// decompressFileStreaming decompresses a file in chunks
func decompressFileStreaming(r io.Reader, task DecompressTask) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(task.DestPath), 0755); err != nil {
		return fmt.Errorf("create parent dir for %s: %w", task.DestPath, err)
	}

	// Handle empty files
	if task.OriginalSize == 0 {
		f, err := os.Create(task.DestPath)
		if err != nil {
			return fmt.Errorf("create empty %s: %w", task.DestPath, err)
		}
		return f.Close()
	}

	// Create output file
	f, err := os.Create(task.DestPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", task.DestPath, err)
	}
	defer f.Close()

	// Decompress
	zr := lz4.NewReader(r)
	pw := &progress.Writer{W: f}
	n, err := io.CopyN(pw, zr, int64(task.OriginalSize))
	if err != nil && err != io.EOF {
		return fmt.Errorf("copy %s: %w", task.DestPath, err)
	}
	if uint64(n) != task.OriginalSize {
		return fmt.Errorf("copy %s: expected %d bytes, got %d", task.DestPath, task.OriginalSize, n)
	}
	return nil
}
