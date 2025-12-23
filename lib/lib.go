// Package lib provides compression and decompression functions for the AGCP format.
// This package re-exports the functionality from the core package for public use.
package lib

import (
	"agcp/pkg/core"
	"agcp/pkg/progress"
)

// Archive format constants
const (
	// Magic is the 4-byte identifier at the start of every AGCP archive
	Magic = core.Magic

	// Version is the current archive format version
	Version = core.Version

	// DefaultBufferSize is the default chunk size for streaming operations
	DefaultBufferSize = core.DefaultBufferSize

	// MaxPathLength is the maximum allowed path length in the archive
	MaxPathLength = core.MaxPathLength
)

// ArchiveType re-exported from core
type ArchiveType = core.ArchiveType

// Archive type constants
const (
	ArchiveFile = core.ArchiveFile
	ArchiveDir  = core.ArchiveDir
)

// Entry represents a file to be compressed
type Entry = core.Entry

// DecompressTask represents a file extraction job
type DecompressTask = core.DecompressTask

// ArchiveHeader contains metadata from an archive
type ArchiveHeader = core.ArchiveHeader

// CompressionStats holds statistics about a compression operation
type CompressionStats = core.CompressionStats

// BufferPool provides reusable byte buffers
type BufferPool = core.BufferPool

// Compressor handles file and directory compression
type Compressor = core.Compressor

// Decompressor handles archive extraction
type Decompressor = core.Decompressor

// NewCompressor creates a new Compressor with default settings
func NewCompressor() *Compressor {
	return core.NewCompressor()
}

// NewDecompressor creates a new Decompressor with default settings
func NewDecompressor() *Decompressor {
	return core.NewDecompressor()
}

// NewBufferPool creates a new buffer pool with the specified buffer size
func NewBufferPool(size int) *BufferPool {
	return core.NewBufferPool(size)
}

// Compress compresses a file or directory to the specified output path
func Compress(input, output string) error {
	return core.Compress(input, output)
}

// Decompress extracts an archive to the specified destination
func Decompress(input, decompressedName string) error {
	return core.Decompress(input, decompressedName)
}

// InitProgress initializes the progress tracking system
func InitProgress(size uint64) {
	progress.Init(size)
}

// StopProgress stops the progress tracking system
func StopProgress() {
	progress.Stop()
}

// SetProgressTestMode enables or disables test mode for progress output
func SetProgressTestMode(enabled bool) {
	progress.SetTestMode(enabled)
}

// SetProgressOperationName sets the operation name for progress output
func SetProgressOperationName(name string) {
	progress.SetOperationName(name)
}
