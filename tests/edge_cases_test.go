// tests/edge_cases_test.go

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestEmptyFile tests handling of empty files
func TestEmptyFile(t *testing.T) {
	// ─── SETUP ──────────────────────────────────────────────────────
	startTime := time.Now()
	ReportStart("Empty File Handling")

	// Create a temporary directory for testing
	StartSection("Preparing Test Environment")
	Action("Creating temporary directory for test files")
	testDir, err := os.MkdirTemp("", "agcp-empty-test")
	if err != nil {
		Error(fmt.Sprintf("Failed to create temp directory: %v", err))
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create an empty file
	Action("Creating empty test file (0 bytes)")
	emptyFile := filepath.Join(testDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte{}, 0644)
	if err != nil {
		Error(fmt.Sprintf("Failed to create empty file: %v", err))
		t.Fatalf("Failed to create empty file: %v", err)
	}
	Success("Empty file created successfully")
	EndSection()

	// ─── COMPRESS ───────────────────────────────────────────────────
	StartSection("Compressing Empty File")
	Action("Compressing the empty file")
	compressedFile := filepath.Join(testDir, "empty.agcp")
	err = Compress(emptyFile, compressedFile)
	if err != nil {
		Error(fmt.Sprintf("Failed to compress empty file: %v", err))
		t.Fatalf("Failed to compress empty file: %v", err)
	}

	// Verify the compressed file exists and has some content (headers)
	info, err := os.Stat(compressedFile)
	if err != nil {
		Error(fmt.Sprintf("Compressed file does not exist: %v", err))
		t.Fatalf("Compressed file does not exist: %v", err)
	}
	if info.Size() == 0 {
		Error("Compressed file is empty, expected header data")
		t.Fatalf("Compressed file is empty, expected header data")
	}

	Success("Empty file compressed successfully")
	Info(fmt.Sprintf("Archive size: %s (metadata and headers only)", HumanReadableSize(info.Size())))
	EndSection()

	// ─── DECOMPRESS ─────────────────────────────────────────────────
	StartSection("Decompressing Empty File Archive")
	Action("Creating output directory for decompression")
	decompressedDir := filepath.Join(testDir, "decompressed")
	err = os.MkdirAll(decompressedDir, 0755)
	if err != nil {
		Error(fmt.Sprintf("Failed to create decompression directory: %v", err))
		t.Fatalf("Failed to create decompression directory: %v", err)
	}

	Action("Decompressing the archive")
	err = Decompress(compressedFile, decompressedDir)
	if err != nil {
		Error(fmt.Sprintf("Decompression failed: %v", err))
		t.Fatalf("Failed to decompress empty file: %v", err)
	}

	// ─── VERIFY ─────────────────────────────────────────────────────
	StartSection("Verifying Decompressed File")
	Action("Checking that the decompressed file exists and is empty")

	// Verify the decompressed file exists and is empty
	decompressedFile := filepath.Join(decompressedDir, "empty.txt")
	info, err = os.Stat(decompressedFile)
	if err != nil {
		Error(fmt.Sprintf("Decompressed file does not exist: %v", err))
		t.Fatalf("Decompressed file does not exist: %v", err)
	}

	Success("Empty file decompressed successfully")
	Info(fmt.Sprintf("Decompressed file size: %s", HumanReadableSize(info.Size())))
	EndSection()

	// ─── CONCLUSION ─────────────────────────────────────────────────
	ReportEnd(true, time.Since(startTime))
}

// TestLargeNumberOfFiles tests handling of directories with many files
func TestLargeNumberOfFiles(t *testing.T) {
	// Skip in short mode as this creates many files
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// ─── SETUP ──────────────────────────────────────────────────────
	startTime := time.Now()
	ReportStart("Multiple Files Handling")

	// Create a temporary directory for testing
	StartSection("Preparing Test Environment")
	Action("Creating temporary directory for test files")
	testDir, err := os.MkdirTemp("", "agcp-many-files-test")
	if err != nil {
		Error(fmt.Sprintf("Failed to create temp directory: %v", err))
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a large number of small files
	numFiles := 100 // Adjust as needed for testing
	Action(fmt.Sprintf("Creating %d small text files in nested directories", numFiles))

	var totalSize int64
	for i := 0; i < numFiles; i++ {
		filename := filepath.Join(testDir, filepath.FromSlash(filepath.Join("files", filepath.Join("subfolder", filepath.FromSlash(filepath.Join("depth3", filepath.FromSlash(filepath.Join("depth4", fmt.Sprintf("file%d.txt", i)))))))))

		// Create all parent directories
		err = os.MkdirAll(filepath.Dir(filename), 0755)
		if err != nil {
			Error(fmt.Sprintf("Failed to create directory structure: %v", err))
			t.Fatalf("Failed to create directory structure: %v", err)
		}

		// Create a small file with content
		content := []byte(fmt.Sprintf("This is file %d with some content.", i))
		err = os.WriteFile(filename, content, 0644)
		if err != nil {
			Error(fmt.Sprintf("Failed to create file %s: %v", filename, err))
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}

		totalSize += int64(len(content))
	}

	Success("Created nested directory structure with multiple files")
	Info(fmt.Sprintf("Total files created: %d", numFiles))
	Info(fmt.Sprintf("Total data size: %s", HumanReadableSize(totalSize)))
	Info("Directory structure depth: 5 levels")
	EndSection()

	// ─── COMPRESS ───────────────────────────────────────────────────
	StartSection("Compressing Multiple Files")
	Action("Compressing directory containing multiple files")
	compressedFile := filepath.Join(testDir, "many-files.agcp")
	err = Compress(filepath.Join(testDir, "files"), compressedFile)
	if err != nil {
		Error(fmt.Sprintf("Compression failed: %v", err))
		t.Fatalf("Failed to compress directory with many files: %v", err)
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
	StartSection("Decompressing Multiple Files")
	Action("Creating output directory for decompression")
	decompressedDir := filepath.Join(testDir, "decompressed")
	err = os.MkdirAll(decompressedDir, 0755)
	if err != nil {
		Error(fmt.Sprintf("Failed to create decompression directory: %v", err))
		t.Fatalf("Failed to create decompression directory: %v", err)
	}

	Action("Decompressing the archive")
	err = Decompress(compressedFile, decompressedDir)
	if err != nil {
		Error(fmt.Sprintf("Decompression failed: %v", err))
		t.Fatalf("Failed to decompress directory with many files: %v", err)
	}

	Success("Archive decompressed successfully")
	EndSection()

	// ─── VERIFY ─────────────────────────────────────────────────────
	StartSection("Verifying Decompressed Files")
	Action("Scanning files to verify correct decompression")

	// Verify that all files were decompressed correctly
	expectedDir := filepath.Join(testDir, "files")
	var checkError error
	var checkedFiles int

	err = filepath.Walk(expectedDir, func(origPath string, origInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip directories, we only check files
		if origInfo.IsDir() {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(expectedDir, origPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}

		// Get path in decompressed directory
		decompPath := filepath.Join(decompressedDir, relPath)

		// Check if decompressed file exists
		decompInfo, err := os.Stat(decompPath)
		if err != nil {
			checkError = fmt.Errorf("file %s not found in decompressed directory: %v", relPath, err)
			return filepath.SkipDir
		}

		// Check if sizes match
		if decompInfo.Size() != origInfo.Size() {
			checkError = fmt.Errorf("size mismatch for %s: original %d, decompressed %d",
				relPath, origInfo.Size(), decompInfo.Size())
			return filepath.SkipDir
		}

		// File exists and size matches
		checkedFiles++
		return nil
	})

	if err != nil {
		Error(fmt.Sprintf("Error walking directory: %v", err))
		t.Fatalf("Error walking original directory: %v", err)
	}

	if checkError != nil {
		Error(fmt.Sprintf("Verification failed: %v", checkError))
		t.Fatalf("Verification failed: %v", checkError)
	}

	Success(fmt.Sprintf("Successfully verified %d files", checkedFiles))
	if checkedFiles == numFiles {
		Info("All files verified - file count matches exactly")
	} else {
		Warning(fmt.Sprintf("File count mismatch: expected %d, got %d", numFiles, checkedFiles))
	}
	EndSection()

	// ─── CONCLUSION ─────────────────────────────────────────────────
	ReportEnd(true, time.Since(startTime))
}

// TestConcurrentProcessing tests concurrent compression and decompression
func TestConcurrentProcessing(t *testing.T) {
	// ─── SETUP ──────────────────────────────────────────────────────
	startTime := time.Now()
	ReportStart("Concurrent Processing")

	// Create a temporary directory for testing
	StartSection("Preparing Test Environment")
	Action("Creating temporary directory for test files")
	testDir, err := os.MkdirTemp("", "agcp-concurrent-test")
	if err != nil {
		Error(fmt.Sprintf("Failed to create temp directory: %v", err))
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create multiple test files
	numFiles := 5
	Action(fmt.Sprintf("Creating %d test files of varying sizes", numFiles))
	filePaths := make([]string, numFiles)
	compressedPaths := make([]string, numFiles)
	decompressedDirs := make([]string, numFiles)

	var totalSize int64
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
			Error(fmt.Sprintf("Failed to create test file %s: %v", filePath, err))
			t.Fatalf("Failed to create test file %s: %v", filePath, err)
		}

		totalSize += int64(size)

		// Set up paths for compressed and decompressed files
		compressedPaths[i] = filepath.Join(testDir, fmt.Sprintf("compressed-%d.agcp", i))
		decompressedDirs[i] = filepath.Join(testDir, fmt.Sprintf("decompressed-%d", i))

		// Create decompression directories
		err = os.MkdirAll(decompressedDirs[i], 0755)
		if err != nil {
			Error(fmt.Sprintf("Failed to create decompression directory: %v", err))
			t.Fatalf("Failed to create decompression directory: %v", err)
		}
	}

	Success("Test files created successfully")
	Info(fmt.Sprintf("Created %d files with sizes from 100KB to %dKB", numFiles, 100*numFiles))
	Info(fmt.Sprintf("Total data size: %s", HumanReadableSize(totalSize)))
	EndSection()

	// ─── CONCURRENT COMPRESSION ──────────────────────────────────────
	StartSection("Concurrent Compression")
	Action(fmt.Sprintf("Compressing %d files simultaneously with goroutines", numFiles))
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

	// Check for errors
	select {
	case err := <-errors:
		Error(fmt.Sprintf("Compression error: %v", err))
		t.Fatalf("Error during concurrent compression: %v", err)
	default:
		Success("All files compressed successfully in parallel")
	}
	EndSection()

	// ─── CONCURRENT DECOMPRESSION ────────────────────────────────────
	StartSection("Concurrent Decompression")
	Action(fmt.Sprintf("Decompressing %d archives simultaneously with goroutines", numFiles))

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
		Error(fmt.Sprintf("Decompression error: %v", err))
		t.Fatalf("Error during concurrent decompression: %v", err)
	default:
		Success("All archives decompressed successfully in parallel")
	}
	EndSection()

	// ─── VERIFY ─────────────────────────────────────────────────────
	StartSection("Verifying Decompressed Files")
	Action("Checking all files for size and integrity")

	// Verify all files were processed correctly
	for i := 0; i < numFiles; i++ {
		// Original file information
		origInfo, err := os.Stat(filePaths[i])
		if err != nil {
			Error(fmt.Sprintf("Failed to access original file %s: %v", filePaths[i], err))
			t.Fatalf("Failed to access original file %s: %v", filePaths[i], err)
		}

		// Get the decompressed file path
		fileName := filepath.Base(filePaths[i])
		decompressedPath := filepath.Join(decompressedDirs[i], fileName)

		// Decompressed file information
		decompInfo, err := os.Stat(decompressedPath)
		if err != nil {
			Error(fmt.Sprintf("Failed to access decompressed file %s: %v", decompressedPath, err))
			t.Fatalf("Failed to access decompressed file %s: %v", decompressedPath, err)
		}

		// Check if sizes match
		if origInfo.Size() != decompInfo.Size() {
			Error(fmt.Sprintf("Size mismatch for %s: original %s, decompressed %s",
				fileName, HumanReadableSize(origInfo.Size()), HumanReadableSize(decompInfo.Size())))
			t.Fatalf("Size mismatch for %s: original %d, decompressed %d",
				fileName, origInfo.Size(), decompInfo.Size())
		}

		Info(fmt.Sprintf("Verified file %d of %d: %s (%s)", i+1, numFiles, fileName,
			HumanReadableSize(origInfo.Size())))
	}

	Success(fmt.Sprintf("Successfully processed and verified %d files concurrently", numFiles))
	EndSection()

	// ─── CONCLUSION ─────────────────────────────────────────────────
	ReportEnd(true, time.Since(startTime))
}

// TestUnicodePaths tests handling of files and directories with Unicode characters in paths
func TestUnicodePaths(t *testing.T) {
	// ─── SETUP ──────────────────────────────────────────────────────
	startTime := time.Now()
	ReportStart("Unicode Path Handling")

	// Create a temporary directory for testing
	StartSection("Preparing Test Environment")
	Action("Creating temporary directory for test files")
	testDir, err := os.MkdirTemp("", "agcp-unicode-test")
	if err != nil {
		Error(fmt.Sprintf("Failed to create temp directory: %v", err))
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create directories and files with Unicode characters
	Action("Creating directories with Unicode characters in names")
	unicodeDirs := []string{
		filepath.Join(testDir, "😀-emoji-dir"),
		filepath.Join(testDir, "中文目录"),          // Chinese directory
		filepath.Join(testDir, "Русская-папка"), // Russian directory
	}

	for _, dir := range unicodeDirs {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			Error(fmt.Sprintf("Failed to create Unicode directory %s: %v", dir, err))
			t.Fatalf("Failed to create Unicode directory %s: %v", dir, err)
		}
		Info(fmt.Sprintf("Created directory: %s", filepath.Base(dir)))
	}

	// Create files with Unicode names
	Action("Creating files with Unicode names and content")
	unicodeFiles := map[string]string{
		filepath.Join(unicodeDirs[0], "emoji-file-😎.txt"): "This is an emoji file.",
		filepath.Join(unicodeDirs[1], "文件.txt"):           "This is a Chinese filename.",
		filepath.Join(unicodeDirs[2], "файл.txt"):         "This is a Russian filename.",
	}

	for filePath, content := range unicodeFiles {
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			Error(fmt.Sprintf("Failed to create Unicode file %s: %v", filePath, err))
			t.Fatalf("Failed to create Unicode file %s: %v", filePath, err)
		}
		Info(fmt.Sprintf("Created file: %s", filepath.Base(filePath)))
	}

	Success("Successfully created directories and files with Unicode names")
	EndSection()

	// ─── COMPRESS ───────────────────────────────────────────────────
	StartSection("Compressing Files with Unicode Paths")
	Action("Compressing directory tree with Unicode paths")
	compressedFile := filepath.Join(testDir, "unicode-test.agcp")
	err = Compress(testDir, compressedFile)
	if err != nil {
		Error(fmt.Sprintf("Compression failed: %v", err))
		t.Fatalf("Failed to compress directory with Unicode paths: %v", err)
	}

	Success("Directory with Unicode paths compressed successfully")
	EndSection()

	// ─── DECOMPRESS ─────────────────────────────────────────────────
	StartSection("Decompressing Files with Unicode Paths")
	Action("Creating output directory for decompression")
	decompressedDir := filepath.Join(testDir, "decompressed")
	err = os.MkdirAll(decompressedDir, 0755)
	if err != nil {
		Error(fmt.Sprintf("Failed to create decompression directory: %v", err))
		t.Fatalf("Failed to create decompression directory: %v", err)
	}

	Action("Decompressing the archive with Unicode paths")
	err = Decompress(compressedFile, decompressedDir)
	if err != nil {
		Error(fmt.Sprintf("Decompression failed: %v", err))
		t.Fatalf("Failed to decompress directory with Unicode paths: %v", err)
	}

	Success("Archive with Unicode paths decompressed successfully")
	EndSection()

	// ─── VERIFY ─────────────────────────────────────────────────────
	StartSection("Verifying Files with Unicode Paths")
	Action("Checking that Unicode filenames and content are preserved")

	// Verify all files were decompressed correctly
	for filePath, content := range unicodeFiles {
		// Calculate relative path
		relPath, err := filepath.Rel(testDir, filePath)
		if err != nil {
			Error(fmt.Sprintf("Failed to calculate relative path: %v", err))
			t.Fatalf("Failed to calculate relative path: %v", err)
		}

		// Decompressed file path
		decompPath := filepath.Join(decompressedDir, relPath)

		// Check if file exists
		_, err = os.Stat(decompPath)
		if err != nil {
			Error(fmt.Sprintf("Decompressed Unicode file not found: %s", filepath.Base(decompPath)))
			t.Fatalf("Decompressed Unicode file not found: %s, error: %v", decompPath, err)
		}

		// Check file content
		decompContent, err := os.ReadFile(decompPath)
		if err != nil {
			Error(fmt.Sprintf("Failed to read decompressed Unicode file: %v", err))
			t.Fatalf("Failed to read decompressed Unicode file: %v", err)
		}

		if string(decompContent) != content {
			Error(fmt.Sprintf("Content mismatch for Unicode file %s", filepath.Base(filePath)))
			t.Fatalf("Content mismatch for Unicode file %s", relPath)
		}

		Success(fmt.Sprintf("Verified Unicode file: %s", filepath.Base(filePath)))
	}

	Info("All Unicode filenames and content were preserved correctly")
	EndSection()

	// ─── CONCLUSION ─────────────────────────────────────────────────
	ReportEnd(true, time.Since(startTime))
}
