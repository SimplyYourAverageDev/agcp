package core

// Constants for archive format
const (
	Magic   = "AGCP" // Magic number to identify the archive
	Version = 1      // Archive format version
)

// ArchiveType distinguishes between file and directory archives
type ArchiveType byte

const (
	ArchiveFile ArchiveType = 0 // Single file archive
	ArchiveDir  ArchiveType = 1 // Directory archive
)

// Entry holds file information for compression
type Entry struct {
	RelPath  string // Relative path within the archive
	FilePath string // Full file path on disk
}

// DecompressTask defines a decompression job
type DecompressTask struct {
	RelPath        string // Relative path within the archive
	OriginalSize   uint64 // Original uncompressed file size
	CompressedSize uint64 // Compressed size in the archive
	DestPath       string // Destination path for extraction
}
