// tests/main_test.go

package tests

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMain sets up the environment for all tests
func TestMain(m *testing.M) {
	// Setup and teardown for all tests
	fmt.Println("Preparing to run AGCP test suite...")
	fmt.Println("────────────────────────────────────")

	// Run the tests
	result := m.Run()

	fmt.Println("────────────────────────────────────")
	if result == 0 {
		fmt.Println("All tests passed successfully!")
	} else {
		fmt.Println("Some tests failed. Please check the output above for details.")
	}

	os.Exit(result)
}

// TestSingleFileCompression tests compressing and decompressing a single file
func TestSingleFileCompression(t *testing.T) {
	// ─── SETUP ──────────────────────────────────────────────────────
	startTime := time.Now()
	ReportStart("Single File Compression")

	// Create a temporary directory for testing
	StartSection("Preparing Test Environment")
	Action("Creating temporary directory for test files")
	testDir, err := os.MkdirTemp("", "agcp-test")
	if err != nil {
		Error(fmt.Sprintf("Failed to create temp directory: %v", err))
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a test file with random content
	Action("Creating test file with 1MB of random data")
	testFile := filepath.Join(testDir, "testfile.dat")
	content := make([]byte, 1024*1024) // 1MB test file
	_, err = rand.Read(content)
	if err != nil {
		Error(fmt.Sprintf("Failed to generate random content: %v", err))
		t.Fatalf("Failed to generate random content: %v", err)
	}

	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		Error(fmt.Sprintf("Failed to write test file: %v", err))
		t.Fatalf("Failed to write test file: %v", err)
	}
	Success("Test file created successfully")
	Info(fmt.Sprintf("Test file size: %s", HumanReadableSize(int64(len(content)))))
	EndSection()

	// ─── COMPRESS ───────────────────────────────────────────────────
	StartSection("Compressing Single File")
	Action("Compressing test file")
	compressedFile := filepath.Join(testDir, "testfile.agcp")
	err = Compress(testFile, compressedFile)
	if err != nil {
		Error(fmt.Sprintf("Compression failed: %v", err))
		t.Fatalf("Compression failed: %v", err)
	}

	// Get compression stats
	compressedInfo, err := os.Stat(compressedFile)
	if err != nil {
		Warning(fmt.Sprintf("Could not get compressed file info: %v", err))
	} else {
		originalSize := int64(len(content))
		compressedSize := compressedInfo.Size()
		compressionRatio := float64(compressedSize) / float64(originalSize) * 100

		Success("File compressed successfully")
		Info(fmt.Sprintf("Original size: %s", HumanReadableSize(originalSize)))
		Info(fmt.Sprintf("Compressed size: %s", HumanReadableSize(compressedSize)))
		Info(fmt.Sprintf("Compression ratio: %.1f%%", compressionRatio))
	}
	EndSection()

	// ─── DECOMPRESS ─────────────────────────────────────────────────
	StartSection("Decompressing File")
	Action("Creating output directory for decompressed file")
	decompressedDir := filepath.Join(testDir, "decompressed")
	err = os.MkdirAll(decompressedDir, 0755)
	if err != nil {
		Error(fmt.Sprintf("Failed to create decompressed directory: %v", err))
		t.Fatalf("Failed to create decompressed directory: %v", err)
	}

	Action("Changing to decompressed directory to test relative path handling")
	origWd, err := os.Getwd()
	if err != nil {
		Error(fmt.Sprintf("Failed to get working directory: %v", err))
		t.Fatalf("Failed to get working directory: %v", err)
	}
	err = os.Chdir(decompressedDir)
	if err != nil {
		Error(fmt.Sprintf("Failed to change working directory: %v", err))
		t.Fatalf("Failed to change working directory: %v", err)
	}
	defer os.Chdir(origWd)

	Action("Decompressing the archive")
	err = Decompress(compressedFile, "")
	if err != nil {
		Error(fmt.Sprintf("Decompression failed: %v", err))
		t.Fatalf("Decompression failed: %v", err)
	}
	Success("File decompressed successfully")
	EndSection()

	// ─── VERIFY ─────────────────────────────────────────────────────
	StartSection("Verifying Decompressed File")
	Action("Comparing original and decompressed file contents")
	decompressedFile := filepath.Join(decompressedDir, "testfile.dat")

	// Check if file exists
	_, err = os.Stat(decompressedFile)
	if err != nil {
		Error(fmt.Sprintf("Decompressed file not found: %v", err))
		t.Fatalf("Decompressed file not found: %v", err)
	}

	// Verify content
	decompressedContent, err := os.ReadFile(decompressedFile)
	if err != nil {
		Error(fmt.Sprintf("Failed to read decompressed file: %v", err))
		t.Fatalf("Failed to read decompressed file: %v", err)
	}

	if !bytes.Equal(content, decompressedContent) {
		Error("Content verification failed - files do not match")
		t.Fatalf("Decompressed content does not match original content")
	}

	Success("File integrity verified - decompressed file matches original exactly")
	EndSection()

	// ─── CONCLUSION ─────────────────────────────────────────────────
	ReportEnd(true, time.Since(startTime))
}

// TestDirectoryCompression tests compressing and decompressing a directory structure
func TestDirectoryCompression(t *testing.T) {
	// ─── SETUP ──────────────────────────────────────────────────────
	startTime := time.Now()
	ReportStart("Directory Structure Compression")

	// Create a temporary directory for testing
	StartSection("Preparing Test Environment")
	Action("Creating temporary directory for test files")
	testDir, err := os.MkdirTemp("", "agcp-test-dir")
	if err != nil {
		Error(fmt.Sprintf("Failed to create temp directory: %v", err))
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a nested directory structure
	Action("Creating nested directory structure")
	subDir1 := filepath.Join(testDir, "subdir1")
	subDir2 := filepath.Join(testDir, "subdir2")
	subSubDir := filepath.Join(subDir1, "subsubdir")

	for _, dir := range []string{subDir1, subDir2, subSubDir} {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			Error(fmt.Sprintf("Failed to create directory %s: %v", dir, err))
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create test files with different sizes
	Action("Creating test files with different sizes")
	files := map[string]int{
		filepath.Join(testDir, "root.txt"):    256,
		filepath.Join(subDir1, "file1.txt"):   512,
		filepath.Join(subDir2, "file2.txt"):   1024,
		filepath.Join(subSubDir, "file3.txt"): 2048,
		filepath.Join(subSubDir, "empty.txt"): 0,
	}

	var totalSize int64
	for file, size := range files {
		content := make([]byte, size)
		for i := 0; i < size; i++ {
			content[i] = byte(i % 256)
		}
		err = os.WriteFile(file, content, 0644)
		if err != nil {
			Error(fmt.Sprintf("Failed to write file %s: %v", file, err))
			t.Fatalf("Failed to write file %s: %v", file, err)
		}
		totalSize += int64(size)
	}

	Success("Created directory structure with test files")
	Info(fmt.Sprintf("Created %d files totaling %s", len(files), HumanReadableSize(totalSize)))
	Info("Directory structure:")
	Info("  root/")
	Info("  ├── root.txt (256 bytes)")
	Info("  ├── subdir1/")
	Info("  │   ├── file1.txt (512 bytes)")
	Info("  │   └── subsubdir/")
	Info("  │       ├── file3.txt (2048 bytes)")
	Info("  │       └── empty.txt (0 bytes)")
	Info("  └── subdir2/")
	Info("      └── file2.txt (1024 bytes)")
	EndSection()

	// ─── COMPRESS ───────────────────────────────────────────────────
	StartSection("Compressing Directory Structure")
	Action("Compressing the entire directory")
	compressedFile := filepath.Join(testDir, "test-dir.agcp")
	err = Compress(testDir, compressedFile)
	if err != nil {
		Error(fmt.Sprintf("Compression failed: %v", err))
		t.Fatalf("Compression failed: %v", err)
	}

	// Get compression stats
	compressedInfo, err := os.Stat(compressedFile)
	if err != nil {
		Warning(fmt.Sprintf("Could not get compressed file info: %v", err))
	} else {
		compressedSize := compressedInfo.Size()
		compressionRatio := float64(compressedSize) / float64(totalSize) * 100

		Success("Directory compressed successfully")
		Info(fmt.Sprintf("Original size: %s", HumanReadableSize(totalSize)))
		Info(fmt.Sprintf("Compressed size: %s", HumanReadableSize(compressedSize)))
		Info(fmt.Sprintf("Compression ratio: %.1f%%", compressionRatio))
	}
	EndSection()

	// ─── DECOMPRESS ─────────────────────────────────────────────────
	StartSection("Decompressing Directory Structure")
	Action("Creating output directory for decompressed files")
	decompressedDir := filepath.Join(testDir, "decompressed")

	Action("Decompressing the archive")
	err = Decompress(compressedFile, decompressedDir)
	if err != nil {
		Error(fmt.Sprintf("Decompression failed: %v", err))
		t.Fatalf("Decompression failed: %v", err)
	}
	Success("Directory decompressed successfully")
	EndSection()

	// ─── VERIFY ─────────────────────────────────────────────────────
	StartSection("Verifying Decompressed Files")
	Action("Comparing original and decompressed files")

	// Check each file
	var verifiedFiles int
	for filePath, size := range files {
		// Convert original path to expected decompressed path
		relPath, err := filepath.Rel(testDir, filePath)
		if err != nil {
			Error(fmt.Sprintf("Failed to get relative path: %v", err))
			t.Fatalf("Failed to get relative path: %v", err)
		}

		decompressedPath := filepath.Join(decompressedDir, relPath)

		// Check if file exists
		_, err = os.Stat(decompressedPath)
		if err != nil {
			Error(fmt.Sprintf("Decompressed file %s not found: %v", relPath, err))
			t.Fatalf("Decompressed file %s does not exist", decompressedPath)
		}

		// Check file content
		original := make([]byte, size)
		for i := 0; i < size; i++ {
			original[i] = byte(i % 256)
		}

		decompressedContent, err := os.ReadFile(decompressedPath)
		if err != nil {
			Error(fmt.Sprintf("Failed to read decompressed file %s: %v", relPath, err))
			t.Fatalf("Failed to read decompressed file %s: %v", decompressedPath, err)
		}

		if !bytes.Equal(original, decompressedContent) {
			Error(fmt.Sprintf("Content mismatch for file %s", relPath))
			t.Fatalf("Decompressed content does not match original for %s", decompressedPath)
		}

		verifiedFiles++
	}

	Success(fmt.Sprintf("All %d files verified with matching content", verifiedFiles))
	EndSection()

	// ─── CONCLUSION ─────────────────────────────────────────────────
	ReportEnd(true, time.Since(startTime))
}

// TestArchiveMetadata tests the archive format and metadata
func TestArchiveMetadata(t *testing.T) {
	// ─── SETUP ──────────────────────────────────────────────────────
	startTime := time.Now()
	ReportStart("Archive Metadata Verification")

	// Create a temporary directory for testing
	StartSection("Preparing Test Environment")
	Action("Creating temporary directory for test files")
	testDir, err := os.MkdirTemp("", "agcp-metadata-test")
	if err != nil {
		Error(fmt.Sprintf("Failed to create temp directory: %v", err))
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a test file
	Action("Creating test file with sample content")
	testFile := filepath.Join(testDir, "metadata-test.txt")
	content := []byte("This is a test file for metadata validation.")
	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		Error(fmt.Sprintf("Failed to write test file: %v", err))
		t.Fatalf("Failed to write test file: %v", err)
	}

	Success("Test file created successfully")
	EndSection()

	// ─── COMPRESS ───────────────────────────────────────────────────
	StartSection("Creating Test Archive")
	Action("Compressing test file to create archive")
	compressedFile := filepath.Join(testDir, "metadata-test.agcp")
	err = Compress(testFile, compressedFile)
	if err != nil {
		Error(fmt.Sprintf("Compression failed: %v", err))
		t.Fatalf("Compression failed: %v", err)
	}

	Success("Archive created successfully")
	EndSection()

	// ─── INSPECT ────────────────────────────────────────────────────
	StartSection("Inspecting Archive Metadata")
	Action("Reading archive header and metadata")

	f, err := os.Open(compressedFile)
	if err != nil {
		Error(fmt.Sprintf("Failed to open compressed file: %v", err))
		t.Fatalf("Failed to open compressed file: %v", err)
	}
	defer f.Close()

	// Read magic number
	var magicBytes [4]byte
	_, err = io.ReadFull(f, magicBytes[:])
	if err != nil {
		Error(fmt.Sprintf("Failed to read magic number: %v", err))
		t.Fatalf("Failed to read magic number: %v", err)
	}
	if string(magicBytes[:]) != Magic {
		Error(fmt.Sprintf("Invalid magic number: expected %q, got %q", Magic, string(magicBytes[:])))
		t.Fatalf("Invalid magic number: expected %q, got %q", Magic, string(magicBytes[:]))
	}
	Success(fmt.Sprintf("Magic number verified: %q", Magic))

	// Read version
	var versionByte uint8
	err = binary.Read(f, binary.BigEndian, &versionByte)
	if err != nil {
		Error(fmt.Sprintf("Failed to read version: %v", err))
		t.Fatalf("Failed to read version: %v", err)
	}
	if int(versionByte) != Version {
		Error(fmt.Sprintf("Invalid version: expected %d, got %d", Version, versionByte))
		t.Fatalf("Invalid version: expected %d, got %d", Version, versionByte)
	}
	Success(fmt.Sprintf("Archive format version: %d", versionByte))

	// Read archive type
	var archiveType byte
	err = binary.Read(f, binary.BigEndian, &archiveType)
	if err != nil {
		Error(fmt.Sprintf("Failed to read archive type: %v", err))
		t.Fatalf("Failed to read archive type: %v", err)
	}
	if archiveType != byte(ArchiveFile) {
		Error(fmt.Sprintf("Invalid archive type: expected %d (File), got %d", ArchiveFile, archiveType))
		t.Fatalf("Invalid archive type: expected %d, got %d", ArchiveFile, archiveType)
	}
	Success(fmt.Sprintf("Archive type: %d (Single File)", archiveType))

	// Read root name length
	var rootNameLen uint16
	err = binary.Read(f, binary.BigEndian, &rootNameLen)
	if err != nil {
		Error(fmt.Sprintf("Failed to read root name length: %v", err))
		t.Fatalf("Failed to read root name length: %v", err)
	}

	// Read root name
	rootNameBytes := make([]byte, rootNameLen)
	_, err = io.ReadFull(f, rootNameBytes)
	if err != nil {
		Error(fmt.Sprintf("Failed to read root name: %v", err))
		t.Fatalf("Failed to read root name: %v", err)
	}
	rootName := string(rootNameBytes)
	expectedName := "metadata-test.txt"
	if rootName != expectedName {
		Error(fmt.Sprintf("Invalid root name: expected %q, got %q", expectedName, rootName))
		t.Fatalf("Invalid root name: expected %q, got %q", expectedName, rootName)
	}
	Success(fmt.Sprintf("Archive root name: %q", rootName))

	// Read number of entries
	var numEntries uint32
	err = binary.Read(f, binary.BigEndian, &numEntries)
	if err != nil {
		Error(fmt.Sprintf("Failed to read number of entries: %v", err))
		t.Fatalf("Failed to read number of entries: %v", err)
	}
	if numEntries != 1 {
		Error(fmt.Sprintf("Invalid number of entries: expected 1, got %d", numEntries))
		t.Fatalf("Invalid number of entries: expected 1, got %d", numEntries)
	}
	Success(fmt.Sprintf("Number of entries in archive: %d", numEntries))

	Info("Summary of archive metadata:")
	Info(fmt.Sprintf("  Magic number: %q", Magic))
	Info(fmt.Sprintf("  Format version: %d", versionByte))
	Info("  Archive type: Single File")
	Info(fmt.Sprintf("  Root name: %q", rootName))
	Info(fmt.Sprintf("  Number of entries: %d", numEntries))

	EndSection()

	// ─── CONCLUSION ─────────────────────────────────────────────────
	ReportEnd(true, time.Since(startTime))
}

// TestErrorHandling checks how the program responds to error conditions
func TestErrorHandling(t *testing.T) {
	// ─── SETUP ──────────────────────────────────────────────────────
	startTime := time.Now()
	ReportStart("Error Handling Verification")

	// ─── TEST CASE 1 ────────────────────────────────────────────────
	StartSection("Test Case: Non-existent Input File")
	Action("Attempting to compress a non-existent file")

	// Try to compress a non-existent file
	err := Compress("non-existent-file.txt", "output.agcp")

	if err == nil {
		Error("Expected an error but none was returned")
		t.Errorf("Expected error but got none")
		ReportEnd(false, time.Since(startTime))
		return
	}

	Success("Error detected correctly")
	Info(fmt.Sprintf("Error message: %v", err))
	EndSection()

	// ─── TEST CASE 2 ────────────────────────────────────────────────
	StartSection("Test Case: Non-existent Archive")
	Action("Attempting to decompress a non-existent archive")

	// Try to decompress a non-existent archive
	err = Decompress("non-existent-archive.agcp", "")

	if err == nil {
		Error("Expected an error but none was returned")
		t.Errorf("Expected error but got none")
		ReportEnd(false, time.Since(startTime))
		return
	}

	Success("Error detected correctly")
	Info(fmt.Sprintf("Error message: %v", err))
	EndSection()

	// ─── CONCLUSION ─────────────────────────────────────────────────
	ReportEnd(true, time.Since(startTime))
}
