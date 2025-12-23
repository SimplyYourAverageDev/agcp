package core

import (
	"sync"
)

// Archive format constants
const (
	// Magic is the 4-byte identifier at the start of every AGCP archive
	Magic = "AGCP"

	// Version is the current archive format version
	Version uint8 = 1

	// DefaultBufferSize is the default chunk size for streaming operations (32 KB)
	DefaultBufferSize = 32 * 1024

	// MaxPathLength is the maximum allowed path length in the archive
	MaxPathLength = 4096
)

// ArchiveType distinguishes between file and directory archives
type ArchiveType byte

const (
	// ArchiveFile represents a single file archive
	ArchiveFile ArchiveType = 0

	// ArchiveDir represents a directory archive
	ArchiveDir ArchiveType = 1
)

// String returns a human-readable representation of the archive type
func (t ArchiveType) String() string {
	switch t {
	case ArchiveFile:
		return "file"
	case ArchiveDir:
		return "directory"
	default:
		return "unknown"
	}
}

// Entry represents a file to be compressed with its paths
type Entry struct {
	// RelPath is the relative path within the archive
	RelPath string

	// FilePath is the absolute path on disk
	FilePath string

	// Size is the file size in bytes (cached for performance)
	Size int64
}

// DecompressTask represents a file extraction job
type DecompressTask struct {
	// RelPath is the relative path within the archive
	RelPath string

	// OriginalSize is the uncompressed file size in bytes
	OriginalSize uint64

	// CompressedSize is the compressed size in the archive
	CompressedSize uint64

	// DestPath is the full destination path for extraction
	DestPath string

	// Offset is the byte offset within the archive where compressed data starts
	Offset int64
}

// BufferPool provides reusable byte buffers to reduce allocations
type BufferPool struct {
	pool sync.Pool
	size int
}

// NewBufferPool creates a new buffer pool with the specified buffer size
func NewBufferPool(size int) *BufferPool {
	if size <= 0 {
		size = DefaultBufferSize
	}
	return &BufferPool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, size)
				return &buf
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get() *[]byte {
	return bp.pool.Get().(*[]byte)
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf *[]byte) {
	if buf != nil && len(*buf) == bp.size {
		bp.pool.Put(buf)
	}
}

// Size returns the buffer size used by this pool
func (bp *BufferPool) Size() int {
	return bp.size
}

// Global buffer pool for default operations
var defaultBufferPool = NewBufferPool(DefaultBufferSize)

// GetBuffer retrieves a buffer from the default pool
func GetBuffer() *[]byte {
	return defaultBufferPool.Get()
}

// PutBuffer returns a buffer to the default pool
func PutBuffer(buf *[]byte) {
	defaultBufferPool.Put(buf)
}

// ArchiveHeader contains metadata read from an archive
type ArchiveHeader struct {
	// Magic is the 4-byte archive identifier
	Magic [4]byte

	// Version is the format version number
	Version uint8

	// Type indicates whether this is a file or directory archive
	Type ArchiveType

	// RootName is the original name of the compressed file or directory
	RootName string

	// EntryCount is the number of files in the archive
	EntryCount uint32
}

// CompressionStats holds statistics about a compression operation
type CompressionStats struct {
	// OriginalSize is the total uncompressed size
	OriginalSize uint64

	// CompressedSize is the total compressed size
	CompressedSize uint64

	// FileCount is the number of files processed
	FileCount int

	// Ratio returns the compression ratio as a percentage
	Ratio float64
}

// CalculateRatio computes the compression ratio
func (s *CompressionStats) CalculateRatio() float64 {
	if s.OriginalSize == 0 {
		return 0
	}
	s.Ratio = float64(s.CompressedSize) / float64(s.OriginalSize) * 100
	return s.Ratio
}
