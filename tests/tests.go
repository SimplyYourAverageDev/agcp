// tests/tests.go

// Package tests contains tests for the agcp package
package tests

import (
	"agcp/lib"
	"agcp/pkg/progress"
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

// SetTestMode enables or disables test mode for progress output
func SetTestMode(enabled bool) {
	progress.SetTestMode(enabled)
}
