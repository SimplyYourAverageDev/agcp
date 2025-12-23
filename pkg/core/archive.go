package core

import (
	"bufio"
	"io"
	"sync"

	"github.com/pierrec/lz4/v4"
)

// Archive format constants
const (
	// Magic is the 4-byte identifier at the start of every AGCP archive
	Magic = "AGCP"

	// Version is the current archive format version
	Version uint8 = 1

	// DefaultBufferSize is the default chunk size for streaming operations (64 KB)
	DefaultBufferSize = 64 * 1024

	// LargeBufferSize is used for large file operations (256 KB)
	LargeBufferSize = 256 * 1024

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
	RelPath  string // Relative path within the archive
	FilePath string // Absolute path on disk
	Size     int64  // File size in bytes (cached for performance)
}

// DecompressTask represents a file extraction job
type DecompressTask struct {
	RelPath        string // Relative path within the archive
	OriginalSize   uint64 // Uncompressed file size in bytes
	CompressedSize uint64 // Compressed size in the archive
	DestPath       string // Full destination path for extraction
	Offset         int64  // Byte offset within the archive
}

// ArchiveHeader contains metadata read from an archive
type ArchiveHeader struct {
	Magic      [4]byte     // 4-byte archive identifier
	Version    uint8       // Format version number
	Type       ArchiveType // File or directory archive
	RootName   string      // Original name of the compressed file/directory
	EntryCount uint32      // Number of files in the archive
}

// CompressionStats holds statistics about a compression operation
type CompressionStats struct {
	OriginalSize   uint64  // Total uncompressed size
	CompressedSize uint64  // Total compressed size
	FileCount      int     // Number of files processed
	Ratio          float64 // Compression ratio as percentage
}

// CalculateRatio computes the compression ratio
func (s *CompressionStats) CalculateRatio() float64 {
	if s.OriginalSize == 0 {
		return 0
	}
	s.Ratio = float64(s.CompressedSize) / float64(s.OriginalSize) * 100
	return s.Ratio
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

// Global pools for default operations
var (
	defaultBufferPool = NewBufferPool(DefaultBufferSize)
	largeBufferPool   = NewBufferPool(LargeBufferSize)

	// Buffered writer pool (8KB buffer for filesystem efficiency)
	bufWriterPool = sync.Pool{
		New: func() interface{} {
			return bufio.NewWriterSize(nil, 8192)
		},
	}

	// LZ4 writer pool - reuse LZ4 writers to avoid allocation
	lz4WriterPool = sync.Pool{
		New: func() interface{} {
			return lz4.NewWriter(nil)
		},
	}

	// LZ4 reader pool - reuse LZ4 readers to avoid allocation
	lz4ReaderPool = sync.Pool{
		New: func() interface{} {
			return lz4.NewReader(nil)
		},
	}
)

// GetBuffer retrieves a buffer from the default pool
func GetBuffer() *[]byte {
	return defaultBufferPool.Get()
}

// PutBuffer returns a buffer to the default pool
func PutBuffer(buf *[]byte) {
	defaultBufferPool.Put(buf)
}

// GetLargeBuffer retrieves a buffer from the large buffer pool
func GetLargeBuffer() *[]byte {
	return largeBufferPool.Get()
}

// PutLargeBuffer returns a buffer to the large buffer pool
func PutLargeBuffer(buf *[]byte) {
	largeBufferPool.Put(buf)
}

// getBufWriter gets a buffered writer from the pool
func getBufWriter(w io.Writer) *bufio.Writer {
	bw := bufWriterPool.Get().(*bufio.Writer)
	bw.Reset(w)
	return bw
}

// putBufWriter returns a buffered writer to the pool after flushing
func putBufWriter(bw *bufio.Writer) {
	bw.Reset(nil)
	bufWriterPool.Put(bw)
}

// getLZ4Writer gets an LZ4 writer from the pool
func getLZ4Writer(w io.Writer) *lz4.Writer {
	zw := lz4WriterPool.Get().(*lz4.Writer)
	zw.Reset(w)
	return zw
}

// putLZ4Writer returns an LZ4 writer to the pool
func putLZ4Writer(zw *lz4.Writer) {
	zw.Reset(nil)
	lz4WriterPool.Put(zw)
}

// getLZ4Reader gets an LZ4 reader from the pool
func getLZ4Reader(r io.Reader) *lz4.Reader {
	zr := lz4ReaderPool.Get().(*lz4.Reader)
	zr.Reset(r)
	return zr
}

// putLZ4Reader returns an LZ4 reader to the pool
func putLZ4Reader(zr *lz4.Reader) {
	zr.Reset(nil)
	lz4ReaderPool.Put(zr)
}

// copyBufferN copies up to n bytes from src to dst using the provided buffer
// Returns the number of bytes copied and any error
func copyBufferN(dst io.Writer, src io.Reader, n int64, buf []byte) (int64, error) {
	return io.CopyBuffer(dst, io.LimitReader(src, n), buf)
}
