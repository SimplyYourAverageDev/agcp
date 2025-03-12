// tests/tests.go

// Package tests contains tests for the agcp package
package tests

import (
	"agcp/lib"
)

var (
	// Export functions from lib package
	Compress   = lib.Compress
	Decompress = lib.Decompress

	// Export constants
	Magic   = lib.Magic
	Version = lib.Version

	// Export types
	ArchiveFile = lib.ArchiveFile
	ArchiveDir  = lib.ArchiveDir
)
