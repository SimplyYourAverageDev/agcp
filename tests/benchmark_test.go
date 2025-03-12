// tests/benchmark_test.go

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// BenchmarkCompression benchmarks the compression performance with different file sizes
func BenchmarkCompression(b *testing.B) {
	sizes := []int{
		1024 * 1024,      // 1MB
		10 * 1024 * 1024, // 10MB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size-%dMB", size/(1024*1024)), func(b *testing.B) {
			// Create a temporary directory for testing
			testDir, err := os.MkdirTemp("", "agcp-bench")
			if err != nil {
				b.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(testDir)

			// Create a test file with deterministic content
			testFile := filepath.Join(testDir, "testfile.dat")
			content := make([]byte, size)
			for i := 0; i < size; i++ {
				content[i] = byte(i % 256)
			}

			err = os.WriteFile(testFile, content, 0644)
			if err != nil {
				b.Fatalf("Failed to write test file: %v", err)
			}

			compressedFile := filepath.Join(testDir, "compressed.agcp")

			// Reset timer before running the benchmark
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Clean up from previous iterations
				os.Remove(compressedFile)

				// Silence output during benchmark
				originalStdout := os.Stdout
				os.Stdout, _ = os.Open(os.DevNull)

				err = Compress(testFile, compressedFile)

				// Restore stdout
				os.Stdout = originalStdout

				if err != nil {
					b.Fatalf("Compression failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkDecompression benchmarks the decompression performance
func BenchmarkDecompression(b *testing.B) {
	sizes := []int{
		1024 * 1024,      // 1MB
		10 * 1024 * 1024, // 10MB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size-%dMB", size/(1024*1024)), func(b *testing.B) {
			// Create a temporary directory for testing
			testDir, err := os.MkdirTemp("", "agcp-bench")
			if err != nil {
				b.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(testDir)

			// Create and compress a test file
			testFile := filepath.Join(testDir, "testfile.dat")
			content := make([]byte, size)
			for i := 0; i < size; i++ {
				content[i] = byte(i % 256)
			}

			err = os.WriteFile(testFile, content, 0644)
			if err != nil {
				b.Fatalf("Failed to write test file: %v", err)
			}

			compressedFile := filepath.Join(testDir, "compressed.agcp")

			// Silence output during preparation
			originalStdout := os.Stdout
			os.Stdout, _ = os.Open(os.DevNull)

			err = Compress(testFile, compressedFile)

			// Restore stdout
			os.Stdout = originalStdout

			if err != nil {
				b.Fatalf("Compression failed during setup: %v", err)
			}

			// Reset timer before running the benchmark
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a fresh decompression directory
				decompressedDir := filepath.Join(testDir, fmt.Sprintf("decompressed_%d", i))

				// Silence output during benchmark
				os.Stdout, _ = os.Open(os.DevNull)

				err = Decompress(compressedFile, decompressedDir)

				// Restore stdout
				os.Stdout = originalStdout

				if err != nil {
					b.Fatalf("Decompression failed: %v", err)
				}

				// Clean up after this iteration
				os.RemoveAll(decompressedDir)
			}
		})
	}
}

// TestProgressLogger tests that the progress logger functions correctly
func TestProgressLogger(t *testing.T) {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "agcp-progress-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a test file
	testFile := filepath.Join(testDir, "progress-test.dat")
	size := 2 * 1024 * 1024 // 2MB to ensure progress is visible
	content := make([]byte, size)
	for i := 0; i < size; i++ {
		content[i] = byte(i % 256)
	}

	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Temporarily disable test mode to see actual progress output
	SetTestMode(false)

	// Redirect stdout to capture progress output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Start compression in a goroutine
	compressedFile := filepath.Join(testDir, "progress-test.agcp")
	done := make(chan error)
	go func() {
		done <- Compress(testFile, compressedFile)
	}()

	// Wait a bit for progress logger to run
	time.Sleep(1500 * time.Millisecond)

	// Close the write end of the pipe to capture output
	w.Close()
	os.Stdout = oldStdout

	// Restore test mode
	SetTestMode(true)

	// Read stdout and verify progress was reported
	outBytes := make([]byte, 1024)
	n, _ := r.Read(outBytes)
	output := string(outBytes[:n])

	// Check for error from compression
	err = <-done
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	// Very basic check - just make sure progress was reported
	if len(output) == 0 {
		t.Fatalf("No progress output detected")
	}

	// Verify that human-readable progress was reported
	if !strings.Contains(output, "Starting processing") {
		t.Fatalf("Progress output doesn't contain expected text 'Starting processing': %s", output)
	}

	// Also test the test mode format
	r2, w2, _ := os.Pipe()
	os.Stdout = w2

	// Compress again with test mode enabled
	go func() {
		done <- Compress(testFile, compressedFile+".2")
	}()

	// Wait a bit for test mode progress logger to run
	time.Sleep(1500 * time.Millisecond)

	// Capture and restore
	w2.Close()
	os.Stdout = oldStdout

	outBytes2 := make([]byte, 1024)
	n2, _ := r2.Read(outBytes2)
	testOutput := string(outBytes2[:n2])

	// Check the error
	err = <-done
	if err != nil {
		t.Fatalf("Test mode compression failed: %v", err)
	}

	// Check test mode output
	if !strings.Contains(testOutput, "[TEST]") {
		t.Fatalf("Test mode output doesn't contain expected '[TEST]' marker: %s", testOutput)
	}
}
