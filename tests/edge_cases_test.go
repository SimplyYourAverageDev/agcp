// tests/edge_cases_test.go

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestEmptyFile tests handling of empty files
func TestEmptyFile(t *testing.T) {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "agcp-empty-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create an empty file
	emptyFile := filepath.Join(testDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	// Compress the empty file
	compressedFile := filepath.Join(testDir, "empty.agcp")
	err = Compress(emptyFile, compressedFile)
	if err != nil {
		t.Fatalf("Failed to compress empty file: %v", err)
	}

	// Verify the compressed file exists and has some content (headers)
	info, err := os.Stat(compressedFile)
	if err != nil {
		t.Fatalf("Compressed file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("Compressed file is empty, expected header data")
	}

	// Decompress the file
	decompressedDir := filepath.Join(testDir, "decompressed")
	err = Decompress(compressedFile, decompressedDir)
	if err != nil {
		t.Fatalf("Failed to decompress empty file: %v", err)
	}

	// Verify the decompressed file exists and is empty
	decompressedFile := filepath.Join(decompressedDir, "empty.txt")
	info, err = os.Stat(decompressedFile)
	if err != nil {
		t.Fatalf("Decompressed file does not exist: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("Decompressed file is not empty, size: %d", info.Size())
	}
}

// TestLargeNumberOfFiles tests handling of a directory with many small files
func TestLargeNumberOfFiles(t *testing.T) {
	// Skip in short mode as this creates many files
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "agcp-many-files-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a large number of small files
	numFiles := 100 // Adjust as needed for testing
	for i := 0; i < numFiles; i++ {
		filename := filepath.Join(testDir, filepath.FromSlash(filepath.Join("files", filepath.Join("subfolder", filepath.FromSlash(filepath.Join("depth3", filepath.FromSlash(filepath.Join("depth4", fmt.Sprintf("file%d.txt", i)))))))))

		// Create all parent directories
		err = os.MkdirAll(filepath.Dir(filename), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory structure: %v", err)
		}

		// Create a small file with content
		content := []byte(fmt.Sprintf("This is file %d with some content.", i))
		err = os.WriteFile(filename, content, 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}

	// Compress the directory
	compressedFile := filepath.Join(testDir, "many-files.agcp")
	err = Compress(filepath.Join(testDir, "files"), compressedFile)
	if err != nil {
		t.Fatalf("Failed to compress directory with many files: %v", err)
	}

	// Decompress the archive
	decompressedDir := filepath.Join(testDir, "decompressed")
	err = Decompress(compressedFile, decompressedDir)
	if err != nil {
		t.Fatalf("Failed to decompress directory with many files: %v", err)
	}

	// Verify that all files were decompressed correctly
	expectedDir := filepath.Join(testDir, "files")
	var checkError error
	err = filepath.Walk(expectedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Calculate relative path from expectedDir
			relPath, err := filepath.Rel(expectedDir, path)
			if err != nil {
				return err
			}

			// Check if the corresponding file exists in decompressed directory
			decompressedPath := filepath.Join(decompressedDir, relPath)
			_, err = os.Stat(decompressedPath)
			if err != nil {
				checkError = err
				return filepath.SkipDir
			}

			// Compare file contents
			origContent, err := os.ReadFile(path)
			if err != nil {
				checkError = err
				return filepath.SkipDir
			}

			decompContent, err := os.ReadFile(decompressedPath)
			if err != nil {
				checkError = err
				return filepath.SkipDir
			}

			if string(origContent) != string(decompContent) {
				checkError = err
				return filepath.SkipDir
			}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Error walking the directory: %v", err)
	}

	if checkError != nil {
		t.Fatalf("Error verifying decompressed files: %v", checkError)
	}
}

// TestConcurrentProcessing tests concurrent compression and decompression
func TestConcurrentProcessing(t *testing.T) {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "agcp-concurrent-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create multiple test files
	numFiles := 5
	filePaths := make([]string, numFiles)
	compressedPaths := make([]string, numFiles)
	decompressedDirs := make([]string, numFiles)

	for i := 0; i < numFiles; i++ {
		// Create a test file with unique content
		fileName := fmt.Sprintf("test-file-%d.dat", i)
		filePath := filepath.Join(testDir, fileName)
		filePaths[i] = filePath

		// Create content of varying sizes
		size := 100 * 1024 * (i + 1) // 100KB * (i+1)
		content := make([]byte, size)
		for j := 0; j < size; j++ {
			content[j] = byte((j + i) % 256)
		}

		err = os.WriteFile(filePath, content, 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filePath, err)
		}

		// Set up paths for compressed and decompressed files
		compressedPaths[i] = filepath.Join(testDir, fmt.Sprintf("compressed-%d.agcp", i))
		decompressedDirs[i] = filepath.Join(testDir, fmt.Sprintf("decompressed-%d", i))
	}

	// Concurrently compress all files
	var wg sync.WaitGroup
	errors := make(chan error, numFiles*2)

	// Compress concurrently
	for i := 0; i < numFiles; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := Compress(filePaths[idx], compressedPaths[idx])
			if err != nil {
				errors <- err
			}
		}(i)
	}
	wg.Wait()

	// Decompress concurrently
	for i := 0; i < numFiles; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := Decompress(compressedPaths[idx], decompressedDirs[idx])
			if err != nil {
				errors <- err
			}
		}(i)
	}
	wg.Wait()

	// Check for errors
	select {
	case err := <-errors:
		t.Fatalf("Error during concurrent processing: %v", err)
	default:
		// No errors
	}

	// Verify all files were processed correctly
	for i := 0; i < numFiles; i++ {
		// Original file content
		originalContent, err := os.ReadFile(filePaths[i])
		if err != nil {
			t.Fatalf("Failed to read original file %s: %v", filePaths[i], err)
		}

		// Decompressed file content
		decompressedFile := filepath.Join(decompressedDirs[i], filepath.Base(filePaths[i]))
		decompressedContent, err := os.ReadFile(decompressedFile)
		if err != nil {
			t.Fatalf("Failed to read decompressed file %s: %v", decompressedFile, err)
		}

		// Compare contents
		if len(originalContent) != len(decompressedContent) {
			t.Fatalf("File %d: content length mismatch, original: %d, decompressed: %d",
				i, len(originalContent), len(decompressedContent))
		}

		for j := 0; j < len(originalContent); j++ {
			if originalContent[j] != decompressedContent[j] {
				t.Fatalf("File %d: content mismatch at byte %d", i, j)
			}
		}
	}
}

// TestUnicodePaths tests handling of paths with Unicode characters
func TestUnicodePaths(t *testing.T) {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "agcp-unicode-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a test file with Unicode in both path and content
	unicodePath := filepath.Join(testDir, "用户文档.txt")
	unicodeContent := []byte("Unicode content: こんにちは世界 - مرحبا بالعالم - Привет, мир!")

	err = os.WriteFile(unicodePath, unicodeContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create Unicode file: %v", err)
	}

	// Compress the file
	compressedFile := filepath.Join(testDir, "unicode.agcp")
	err = Compress(unicodePath, compressedFile)
	if err != nil {
		t.Fatalf("Failed to compress Unicode file: %v", err)
	}

	// Decompress the file
	decompressedDir := filepath.Join(testDir, "decompressed")
	err = Decompress(compressedFile, decompressedDir)
	if err != nil {
		t.Fatalf("Failed to decompress Unicode file: %v", err)
	}

	// Verify the file exists and content matches
	decompressedFile := filepath.Join(decompressedDir, "用户文档.txt")
	decompressedContent, err := os.ReadFile(decompressedFile)
	if err != nil {
		t.Fatalf("Failed to read decompressed Unicode file: %v", err)
	}

	if string(unicodeContent) != string(decompressedContent) {
		t.Fatalf("Unicode content mismatch")
	}
}
