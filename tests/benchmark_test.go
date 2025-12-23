// tests/benchmark_test.go

package tests

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"agcp/pkg/core"
	"agcp/pkg/progress"
)

// BenchmarkCompressSmallFile benchmarks compression of a small file (1KB)
func BenchmarkCompressSmallFile(b *testing.B) {
	benchmarkCompressFile(b, 1024, "small_1kb")
}

// BenchmarkCompressMediumFile benchmarks compression of a medium file (1MB)
func BenchmarkCompressMediumFile(b *testing.B) {
	benchmarkCompressFile(b, 1024*1024, "medium_1mb")
}

// BenchmarkCompressLargeFile benchmarks compression of a large file (10MB)
func BenchmarkCompressLargeFile(b *testing.B) {
	benchmarkCompressFile(b, 10*1024*1024, "large_10mb")
}

// BenchmarkCompressVeryLargeFile benchmarks compression of a very large file (100MB)
func BenchmarkCompressVeryLargeFile(b *testing.B) {
	benchmarkCompressFile(b, 100*1024*1024, "very_large_100mb")
}

// benchmarkCompressFile is the generic benchmark helper for file compression
func benchmarkCompressFile(b *testing.B, size int, name string) {
	// Disable progress output during benchmarks
	progress.SetTestMode(true)
	defer progress.SetTestMode(false)

	// Setup
	testDir, err := os.MkdirTemp("", "agcp-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, name+".dat")
	outputFile := filepath.Join(testDir, name+".agcp")

	// Create test file with random data
	content := make([]byte, size)
	if _, err := rand.Read(content); err != nil {
		b.Fatalf("Failed to generate random content: %v", err)
	}
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		b.Fatalf("Failed to write test file: %v", err)
	}

	b.SetBytes(int64(size))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := core.Compress(testFile, outputFile); err != nil {
			b.Fatalf("Compression failed: %v", err)
		}
		os.Remove(outputFile)
	}
}

// BenchmarkDecompressSmallFile benchmarks decompression of a small file (1KB)
func BenchmarkDecompressSmallFile(b *testing.B) {
	benchmarkDecompressFile(b, 1024, "small_1kb")
}

// BenchmarkDecompressMediumFile benchmarks decompression of a medium file (1MB)
func BenchmarkDecompressMediumFile(b *testing.B) {
	benchmarkDecompressFile(b, 1024*1024, "medium_1mb")
}

// BenchmarkDecompressLargeFile benchmarks decompression of a large file (10MB)
func BenchmarkDecompressLargeFile(b *testing.B) {
	benchmarkDecompressFile(b, 10*1024*1024, "large_10mb")
}

// BenchmarkDecompressVeryLargeFile benchmarks decompression of a very large file (100MB)
func BenchmarkDecompressVeryLargeFile(b *testing.B) {
	benchmarkDecompressFile(b, 100*1024*1024, "very_large_100mb")
}

// benchmarkDecompressFile is the generic benchmark helper for file decompression
func benchmarkDecompressFile(b *testing.B, size int, name string) {
	// Disable progress output during benchmarks
	progress.SetTestMode(true)
	defer progress.SetTestMode(false)

	// Setup
	testDir, err := os.MkdirTemp("", "agcp-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, name+".dat")
	archiveFile := filepath.Join(testDir, name+".agcp")
	outputDir := filepath.Join(testDir, "output")

	// Create test file with random data
	content := make([]byte, size)
	if _, err := rand.Read(content); err != nil {
		b.Fatalf("Failed to generate random content: %v", err)
	}
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		b.Fatalf("Failed to write test file: %v", err)
	}

	// Create the archive
	if err := core.Compress(testFile, archiveFile); err != nil {
		b.Fatalf("Failed to create archive: %v", err)
	}

	b.SetBytes(int64(size))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := core.Decompress(archiveFile, outputDir); err != nil {
			b.Fatalf("Decompression failed: %v", err)
		}
		os.RemoveAll(outputDir)
	}
}

// BenchmarkCompressDirectory benchmarks directory compression
func BenchmarkCompressDirectory(b *testing.B) {
	// Disable progress output during benchmarks
	progress.SetTestMode(true)
	defer progress.SetTestMode(false)

	// Setup
	testDir, err := os.MkdirTemp("", "agcp-bench-dir-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	sourceDir := filepath.Join(testDir, "source")
	archiveFile := filepath.Join(testDir, "archive.agcp")

	// Create directory structure with files
	totalSize := createTestDirectory(b, sourceDir, 10, 100*1024) // 10 files, 100KB each

	b.SetBytes(int64(totalSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := core.Compress(sourceDir, archiveFile); err != nil {
			b.Fatalf("Compression failed: %v", err)
		}
		os.Remove(archiveFile)
	}
}

// BenchmarkDecompressDirectory benchmarks directory decompression
func BenchmarkDecompressDirectory(b *testing.B) {
	// Disable progress output during benchmarks
	progress.SetTestMode(true)
	defer progress.SetTestMode(false)

	// Setup
	testDir, err := os.MkdirTemp("", "agcp-bench-dir-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	sourceDir := filepath.Join(testDir, "source")
	archiveFile := filepath.Join(testDir, "archive.agcp")
	outputDir := filepath.Join(testDir, "output")

	// Create directory structure and archive
	totalSize := createTestDirectory(b, sourceDir, 10, 100*1024)
	if err := core.Compress(sourceDir, archiveFile); err != nil {
		b.Fatalf("Failed to create archive: %v", err)
	}

	b.SetBytes(int64(totalSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := core.Decompress(archiveFile, outputDir); err != nil {
			b.Fatalf("Decompression failed: %v", err)
		}
		os.RemoveAll(outputDir)
	}
}

// BenchmarkBufferPool benchmarks buffer pool operations
func BenchmarkBufferPool(b *testing.B) {
	pool := core.NewBufferPool(32 * 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		pool.Put(buf)
	}
}

// BenchmarkBufferPoolParallel benchmarks parallel buffer pool operations
func BenchmarkBufferPoolParallel(b *testing.B) {
	pool := core.NewBufferPool(32 * 1024)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get()
			pool.Put(buf)
		}
	})
}

// BenchmarkCompressCompressibleData benchmarks compression of highly compressible data
func BenchmarkCompressCompressibleData(b *testing.B) {
	// Disable progress output during benchmarks
	progress.SetTestMode(true)
	defer progress.SetTestMode(false)

	// Setup
	testDir, err := os.MkdirTemp("", "agcp-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	size := 10 * 1024 * 1024 // 10MB
	testFile := filepath.Join(testDir, "compressible.dat")
	outputFile := filepath.Join(testDir, "compressible.agcp")

	// Create highly compressible content (repeating pattern)
	content := make([]byte, size)
	pattern := []byte("This is a repeating pattern for compression testing. ")
	for i := 0; i < size; i++ {
		content[i] = pattern[i%len(pattern)]
	}
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		b.Fatalf("Failed to write test file: %v", err)
	}

	b.SetBytes(int64(size))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := core.Compress(testFile, outputFile); err != nil {
			b.Fatalf("Compression failed: %v", err)
		}
		os.Remove(outputFile)
	}
}

// BenchmarkConcurrentDecompression benchmarks concurrent file decompression
func BenchmarkConcurrentDecompression(b *testing.B) {
	// Disable progress output during benchmarks
	progress.SetTestMode(true)
	defer progress.SetTestMode(false)

	// Setup
	testDir, err := os.MkdirTemp("", "agcp-bench-concurrent-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	sourceDir := filepath.Join(testDir, "source")
	archiveFile := filepath.Join(testDir, "archive.agcp")
	outputDir := filepath.Join(testDir, "output")

	// Create directory with many files to test concurrency
	totalSize := createTestDirectory(b, sourceDir, 50, 50*1024) // 50 files, 50KB each
	if err := core.Compress(sourceDir, archiveFile); err != nil {
		b.Fatalf("Failed to create archive: %v", err)
	}

	b.SetBytes(int64(totalSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := core.Decompress(archiveFile, outputDir); err != nil {
			b.Fatalf("Decompression failed: %v", err)
		}
		os.RemoveAll(outputDir)
	}
}

// createTestDirectory creates a test directory with specified number of files
func createTestDirectory(b *testing.B, dir string, fileCount, fileSize int) int {
	b.Helper()

	if err := os.MkdirAll(dir, 0755); err != nil {
		b.Fatalf("Failed to create directory: %v", err)
	}

	// Create subdirectories
	subDirs := []string{
		filepath.Join(dir, "subdir1"),
		filepath.Join(dir, "subdir2"),
		filepath.Join(dir, "subdir1", "nested"),
	}
	for _, subDir := range subDirs {
		if err := os.MkdirAll(subDir, 0755); err != nil {
			b.Fatalf("Failed to create subdirectory: %v", err)
		}
	}

	totalSize := 0
	for i := 0; i < fileCount; i++ {
		// Distribute files across directories
		var filePath string
		switch i % 4 {
		case 0:
			filePath = filepath.Join(dir, "file"+string(rune('a'+i))+".dat")
		case 1:
			filePath = filepath.Join(subDirs[0], "file"+string(rune('a'+i))+".dat")
		case 2:
			filePath = filepath.Join(subDirs[1], "file"+string(rune('a'+i))+".dat")
		case 3:
			filePath = filepath.Join(subDirs[2], "file"+string(rune('a'+i))+".dat")
		}

		content := make([]byte, fileSize)
		if _, err := rand.Read(content); err != nil {
			b.Fatalf("Failed to generate random content: %v", err)
		}
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			b.Fatalf("Failed to write file: %v", err)
		}
		totalSize += fileSize
	}

	return totalSize
}
