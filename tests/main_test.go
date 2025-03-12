// tests/main_test.go

package tests

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestMain sets up the environment for all tests
func TestMain(m *testing.M) {
	// Enable test mode for progress reporting
	SetTestMode(true)

	// Run tests and exit with returned code
	os.Exit(m.Run())
}

// TestCompressDecompressFile tests the compression and decompression of a single file
func TestCompressDecompressFile(t *testing.T) {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "agcp-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a test file with random content
	testFile := filepath.Join(testDir, "testfile.dat")
	content := make([]byte, 1024*1024) // 1MB test file
	_, err = rand.Read(content)
	if err != nil {
		t.Fatalf("Failed to generate random content: %v", err)
	}

	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compress the test file
	compressedFile := filepath.Join(testDir, "testfile.agcp")
	err = Compress(testFile, compressedFile)
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	// Decompress the file
	decompressedDir := filepath.Join(testDir, "decompressed")
	err = os.MkdirAll(decompressedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create decompressed directory: %v", err)
	}

	// Change working directory to decompressed directory to test relative path decompression
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	err = os.Chdir(decompressedDir)
	if err != nil {
		t.Fatalf("Failed to change working directory: %v", err)
	}
	defer os.Chdir(origWd)

	err = Decompress(compressedFile, "")
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	// Verify file contents
	decompressedFile := filepath.Join(decompressedDir, "testfile.dat")
	decompressedContent, err := os.ReadFile(decompressedFile)
	if err != nil {
		t.Fatalf("Failed to read decompressed file: %v", err)
	}

	if !bytes.Equal(content, decompressedContent) {
		t.Fatalf("Decompressed content does not match original content")
	}
}

// TestCompressDecompressDirectory tests the compression and decompression of a directory
func TestCompressDecompressDirectory(t *testing.T) {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "agcp-test-dir")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a nested directory structure with files
	subDir1 := filepath.Join(testDir, "subdir1")
	subDir2 := filepath.Join(testDir, "subdir2")
	subSubDir := filepath.Join(subDir1, "subsubdir")

	for _, dir := range []string{subDir1, subDir2, subSubDir} {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create test files
	files := map[string]int{
		filepath.Join(testDir, "root.txt"):    256,
		filepath.Join(subDir1, "file1.txt"):   512,
		filepath.Join(subDir2, "file2.txt"):   1024,
		filepath.Join(subSubDir, "file3.txt"): 2048,
		filepath.Join(subSubDir, "empty.txt"): 0,
	}

	for file, size := range files {
		content := make([]byte, size)
		for i := 0; i < size; i++ {
			content[i] = byte(i % 256)
		}
		err = os.WriteFile(file, content, 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", file, err)
		}
	}

	// Compress the directory
	compressedFile := filepath.Join(testDir, "test-dir.agcp")
	err = Compress(testDir, compressedFile)
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	// Decompress to a new location
	decompressedDir := filepath.Join(testDir, "decompressed")
	err = Decompress(compressedFile, decompressedDir)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	// Verify directory structure and file contents
	for filePath, size := range files {
		// Convert original path to expected decompressed path
		relPath, err := filepath.Rel(testDir, filePath)
		if err != nil {
			t.Fatalf("Failed to get relative path: %v", err)
		}

		decompressedPath := filepath.Join(decompressedDir, relPath)

		// Check if file exists
		if _, err := os.Stat(decompressedPath); os.IsNotExist(err) {
			t.Fatalf("Decompressed file %s does not exist", decompressedPath)
		}

		// Check file content
		original := make([]byte, size)
		for i := 0; i < size; i++ {
			original[i] = byte(i % 256)
		}

		decompressedContent, err := os.ReadFile(decompressedPath)
		if err != nil {
			t.Fatalf("Failed to read decompressed file %s: %v", decompressedPath, err)
		}

		if !bytes.Equal(original, decompressedContent) {
			t.Fatalf("Decompressed content does not match original for %s", decompressedPath)
		}
	}
}

// TestArchiveMetadata tests that the archive metadata is correct
func TestArchiveMetadata(t *testing.T) {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "agcp-metadata-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a test file
	testFile := filepath.Join(testDir, "metadata-test.txt")
	content := []byte("This is a test file for metadata validation.")
	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compress the file
	compressedFile := filepath.Join(testDir, "metadata-test.agcp")
	err = Compress(testFile, compressedFile)
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	// Manually parse the archive to validate metadata
	f, err := os.Open(compressedFile)
	if err != nil {
		t.Fatalf("Failed to open compressed file: %v", err)
	}
	defer f.Close()

	// Read magic number
	var magicBytes [4]byte
	_, err = io.ReadFull(f, magicBytes[:])
	if err != nil {
		t.Fatalf("Failed to read magic number: %v", err)
	}
	if string(magicBytes[:]) != Magic {
		t.Fatalf("Invalid magic number: expected %q, got %q", Magic, string(magicBytes[:]))
	}

	// Read version
	var versionByte uint8
	err = binary.Read(f, binary.BigEndian, &versionByte)
	if err != nil {
		t.Fatalf("Failed to read version: %v", err)
	}
	if int(versionByte) != Version {
		t.Fatalf("Invalid version: expected %d, got %d", Version, versionByte)
	}

	// Read archive type
	var archiveType byte
	err = binary.Read(f, binary.BigEndian, &archiveType)
	if err != nil {
		t.Fatalf("Failed to read archive type: %v", err)
	}
	if archiveType != byte(ArchiveFile) {
		t.Fatalf("Invalid archive type: expected %d, got %d", ArchiveFile, archiveType)
	}

	// Read root name length
	var rootNameLen uint16
	err = binary.Read(f, binary.BigEndian, &rootNameLen)
	if err != nil {
		t.Fatalf("Failed to read root name length: %v", err)
	}

	// Read root name
	rootNameBytes := make([]byte, rootNameLen)
	_, err = io.ReadFull(f, rootNameBytes)
	if err != nil {
		t.Fatalf("Failed to read root name: %v", err)
	}
	rootName := string(rootNameBytes)
	expectedName := "metadata-test.txt"
	if rootName != expectedName {
		t.Fatalf("Invalid root name: expected %q, got %q", expectedName, rootName)
	}

	// Read number of entries
	var numEntries uint32
	err = binary.Read(f, binary.BigEndian, &numEntries)
	if err != nil {
		t.Fatalf("Failed to read number of entries: %v", err)
	}
	if numEntries != 1 {
		t.Fatalf("Invalid number of entries: expected 1, got %d", numEntries)
	}
}

// TestErrorCases tests various error conditions
func TestErrorCases(t *testing.T) {
	testCases := []struct {
		name        string
		inputPath   string
		outputPath  string
		operation   string
		expectedErr bool
	}{
		{
			name:        "Non-existent input file",
			inputPath:   "non-existent-file.txt",
			outputPath:  "output.agcp",
			operation:   "compress",
			expectedErr: true,
		},
		{
			name:        "Non-existent input archive",
			inputPath:   "non-existent-archive.agcp",
			outputPath:  "",
			operation:   "decompress",
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.operation == "compress" {
				err = Compress(tc.inputPath, tc.outputPath)
			} else {
				err = Decompress(tc.inputPath, tc.outputPath)
			}

			if tc.expectedErr && err == nil {
				t.Errorf("Expected error but got none")
			} else if !tc.expectedErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
