package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Colors for terminal output
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

// Global variables for progress tracking
var (
	totalBytesProcessed atomic.Uint64
	totalSize           uint64
	done                chan struct{}
	progressRunning     bool
	progressMutex       sync.Mutex
	isTestMode          bool   // Flag to indicate test mode
	operationName       string // Operation name for output
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
// In test mode, progress output is formatted for better readability in tests
func SetTestMode(enabled bool) {
	progressMutex.Lock()
	defer progressMutex.Unlock()
	isTestMode = enabled
}

// SetOperationName sets the current operation name for output
func SetOperationName(name string) {
	progressMutex.Lock()
	defer progressMutex.Unlock()
	operationName = name
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

// progressBar returns a visual progress bar
func progressBar(percentage float64, width int) string {
	completed := int(percentage * float64(width) / 100)
	if completed > width {
		completed = width
	}

	bar := "["
	bar += strings.Repeat("█", completed)
	bar += strings.Repeat("░", width-completed)
	bar += "]"

	return bar
}

// calculateETA calculates the estimated time remaining
func calculateETA(bytesRemaining uint64, rate uint64) string {
	if rate == 0 {
		return "calculating..."
	}

	secondsRemaining := float64(bytesRemaining) / float64(rate)

	if secondsRemaining < 60 {
		return fmt.Sprintf("%.0f seconds", secondsRemaining)
	} else if secondsRemaining < 3600 {
		return fmt.Sprintf("%.1f minutes", secondsRemaining/60)
	} else {
		return fmt.Sprintf("%.1f hours", secondsRemaining/3600)
	}
}

// logger logs processing progress periodically
func logger() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	var prevBytes uint64
	var prevPercentage float64
	startTime := time.Now()
	lastOutputTime := time.Now()

	// Operation description
	op := "Processing"
	if operationName != "" {
		op = operationName
	}

	// Initial output
	if isTestMode {
		fmt.Printf("%s%s▶ Starting %s...%s\n", colorBold, colorBlue, op, colorReset)
	} else {
		fmt.Printf("Starting %s...\n", op)
	}

	for {
		select {
		case <-ticker.C:
			currentBytes := totalBytesProcessed.Load()
			rate := (currentBytes - prevBytes) * 4 // Bytes per second (250ms interval)
			prevBytes = currentBytes

			bytesRemaining := totalSize - currentBytes
			currentPercentage := float64(currentBytes) / float64(totalSize) * 100

			// Only show update if there's significant change or enough time has passed
			timeSinceLastOutput := time.Since(lastOutputTime)
			percentageDiff := currentPercentage - prevPercentage
			shouldUpdate := timeSinceLastOutput >= time.Second ||
				percentageDiff >= 10 ||
				(currentPercentage >= 100 && prevPercentage < 100)

			if shouldUpdate {
				lastOutputTime = time.Now()

				// Show different output for test mode vs normal mode
				if isTestMode {
					// Only show progress at key percentages for tests
					if currentPercentage >= 100 && prevPercentage < 100 {
						pb := progressBar(100, 20)
						fmt.Printf("%s%s✓ %s complete! %s 100%%%s\n",
							colorBold, colorGreen, op, pb, colorReset)
					} else if percentageDiff >= 25 || currentPercentage >= 100 {
						pb := progressBar(currentPercentage, 20)
						fmt.Printf("%s%s• %s progress: %s %.0f%%%s\n",
							colorBold, colorBlue, op, pb, currentPercentage, colorReset)
					}
				} else {
					// Normal mode - more detailed output
					sizeInfo := formatSize(currentBytes)
					rateInfo := formatRate(rate)

					if totalSize > 1 {
						totalSizeInfo := formatSize(totalSize)
						etaInfo := calculateETA(bytesRemaining, rate)
						pb := progressBar(currentPercentage, 20)

						fmt.Printf("%s %s of %s %s %.1f%% | Rate: %s | ETA: %s\n",
							op, sizeInfo, totalSizeInfo, pb, currentPercentage, rateInfo, etaInfo)
					} else {
						fmt.Printf("%s %s | Rate: %s\n", op, sizeInfo, rateInfo)
					}
				}
			}

			prevPercentage = currentPercentage
			os.Stdout.Sync()

		case <-done:
			// Final output on completion
			processedBytes := totalBytesProcessed.Load()
			totalTime := time.Since(startTime).Seconds()
			sizeInfo := formatSize(processedBytes)

			if isTestMode {
				fmt.Printf("%s%s✓ %s completed: %s in %.1f seconds%s\n",
					colorBold, colorGreen, op, sizeInfo, totalTime, colorReset)
			} else {
				avgRate := formatRate(uint64(float64(processedBytes) / totalTime))
				fmt.Printf("%s completed: %s in %.1f seconds (avg rate: %s)\n",
					op, sizeInfo, totalTime, avgRate)
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
