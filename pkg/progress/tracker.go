package progress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Global variables for progress tracking
var (
	totalBytesProcessed atomic.Uint64
	totalSize           uint64
	done                chan struct{}
	progressRunning     bool
	progressMutex       sync.Mutex
	isTestMode          bool // New flag to indicate test mode
)

// Init initializes the progress tracking system
func Init(size uint64) {
	progressMutex.Lock()
	defer progressMutex.Unlock()

	if progressRunning {
		return
	}

	totalBytesProcessed.Store(0)
	totalSize = size
	if totalSize == 0 {
		totalSize = 1 // Avoid division by zero
	}

	done = make(chan struct{})
	progressRunning = true
	go logger()
}

// SetTestMode enables or disables test mode
// In test mode, progress output is minimal to avoid cluttering test output
func SetTestMode(enabled bool) {
	progressMutex.Lock()
	defer progressMutex.Unlock()
	isTestMode = enabled
}

// Stop stops the progress tracking
func Stop() {
	progressMutex.Lock()
	defer progressMutex.Unlock()

	if progressRunning {
		close(done)
		progressRunning = false
	}
}

// AddBytes adds processed bytes to the counter
func AddBytes(n uint64) {
	if n > 0 {
		totalBytesProcessed.Add(n)
	}
}

// formatSize returns a human-readable size string
func formatSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatRate returns a human-readable rate string
func formatRate(bytesPerSec uint64) string {
	const unit = 1024
	if bytesPerSec < unit {
		return fmt.Sprintf("%d B/s", bytesPerSec)
	}
	div, exp := uint64(unit), 0
	for n := bytesPerSec / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB/s", float64(bytesPerSec)/float64(div), "KMGTPE"[exp])
}

// logger logs processing progress periodically
func logger() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	var prevBytes uint64
	var prevPercentage float64
	startTime := time.Now()
	lastOutputTime := time.Now()

	// Initial output for test verification
	if isTestMode {
		fmt.Printf("[TEST] Progress tracking initialized\n")
	} else {
		fmt.Printf("Starting processing...\n")
	}

	for {
		select {
		case <-ticker.C:
			currentBytes := totalBytesProcessed.Load()
			rate := (currentBytes - prevBytes) * 4 // Bytes per second (250ms interval)
			prevBytes = currentBytes

			// Calculate elapsed time
			elapsed := time.Since(startTime).Seconds()
			if elapsed < 0.001 {
				elapsed = 0.001 // Avoid division by zero
			}

			currentPercentage := float64(currentBytes) / float64(totalSize) * 100

			// In test mode, output minimal information and only at key percentages
			if isTestMode {
				// Only output for significant changes (25%, 50%, 75%, 100%)
				if currentPercentage >= 100 && prevPercentage < 100 {
					fmt.Printf("[TEST] Processing complete (100%%)\n")
				} else if currentPercentage >= 75 && prevPercentage < 75 {
					fmt.Printf("[TEST] Processing at 75%%\n")
				} else if currentPercentage >= 50 && prevPercentage < 50 {
					fmt.Printf("[TEST] Processing at 50%%\n")
				} else if currentPercentage >= 25 && prevPercentage < 25 {
					fmt.Printf("[TEST] Processing at 25%%\n")
				}
			} else {
				// For normal mode, show human-readable output
				// Only show updates every second or for significant percentage changes
				timeSinceLastOutput := time.Since(lastOutputTime)
				percentageDiff := currentPercentage - prevPercentage

				if timeSinceLastOutput >= time.Second || percentageDiff >= 10 ||
					(currentPercentage >= 100 && prevPercentage < 100) {

					lastOutputTime = time.Now()
					humanReadableSize := formatSize(currentBytes)
					humanReadableRate := formatRate(rate)

					if totalSize > 1 { // If we have a meaningful total size
						timeRemaining := "calculating..."
						if rate > 0 {
							secondsRemaining := float64(totalSize-currentBytes) / float64(rate)
							if secondsRemaining < 60 {
								timeRemaining = fmt.Sprintf("%.0f seconds", secondsRemaining)
							} else if secondsRemaining < 3600 {
								timeRemaining = fmt.Sprintf("%.1f minutes", secondsRemaining/60)
							} else {
								timeRemaining = fmt.Sprintf("%.1f hours", secondsRemaining/3600)
							}
						}

						fmt.Printf("Processed %s of %s (%.1f%%) | Rate: %s | ETA: %s\n",
							humanReadableSize, formatSize(totalSize),
							currentPercentage, humanReadableRate, timeRemaining)
					} else {
						fmt.Printf("Processed %s | Rate: %s\n",
							humanReadableSize, humanReadableRate)
					}
				}
			}

			prevPercentage = currentPercentage
			// Flush stdout for testing purposes
			os.Stdout.Sync()
		case <-done:
			// Final output
			if !isTestMode {
				totalTime := time.Since(startTime).Seconds()
				humanReadableSize := formatSize(totalBytesProcessed.Load())
				avgRate := formatRate(uint64(float64(totalBytesProcessed.Load()) / totalTime))
				fmt.Printf("Completed processing %s in %.1f seconds (avg rate: %s)\n",
					humanReadableSize, totalTime, avgRate)
			}
			return
		}
	}
}

// Writer is a writer that tracks bytes written for progress reporting
type Writer struct {
	W io.Writer
}

// Write implements io.Writer and tracks bytes written
func (pw *Writer) Write(p []byte) (n int, err error) {
	n, err = pw.W.Write(p)
	if err == nil && n > 0 {
		AddBytes(uint64(n))
	}
	return
}
