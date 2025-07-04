// Package lib provides compression and decompression functions for the AGCP format.
// This package re-exports the functionality from the core package for backward compatibility.
package lib

import (
	"agcp/pkg/core"
	"agcp/pkg/progress"
)

// Constants for archive format re-exported from core
const (
	Magic   = core.Magic   // Magic number to identify the archive
	Version = core.Version // Archive format version
)

// ArchiveType re-exported from core
type ArchiveType = core.ArchiveType

// Re-export archive types
const (
	ArchiveFile = core.ArchiveFile
	ArchiveDir  = core.ArchiveDir
)

// Entry re-exported from core
type Entry = core.Entry

// DecompressTask re-exported from core
type DecompressTask = core.DecompressTask

// InitProgress initializes the progress tracking system
func InitProgress() {
	progress.Init(0)
}

// StopProgress stops the progress tracking system
func StopProgress() {
	progress.Stop()
}

// Compress is a wrapper around core.Compress
func Compress(input, output string) error {
	return core.Compress(input, output)
}

// Decompress is a wrapper around core.Decompress
func Decompress(input, decompressedName string) error {
	return core.Decompress(input, decompressedName)
}
