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

// ANSI color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

// Size units for formatting
const (
	sizeUnit     = 1024
	tickInterval = 250 * time.Millisecond
)

// Config holds configuration for the progress tracker
type Config struct {
	TestMode      bool
	OperationName string
	Output        io.Writer
}

// Tracker manages progress tracking for an operation
type Tracker struct {
	totalSize       uint64
	processedBytes  atomic.Uint64
	done            chan struct{}
	running         bool
	mu              sync.Mutex
	config          Config
	startTime       time.Time
	prevBytes       uint64
	prevPercentage  float64
	lastOutputTime  time.Time
}

// Global tracker for backward compatibility
var (
	globalTracker *Tracker
	globalMu      sync.Mutex
)

// NewTracker creates a new progress tracker
func NewTracker(totalSize uint64, cfg Config) *Tracker {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	if totalSize == 0 {
		totalSize = 1
	}
	return &Tracker{
		totalSize: totalSize,
		config:    cfg,
	}
}

// Start begins progress tracking in a background goroutine
func (t *Tracker) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return
	}

	t.done = make(chan struct{})
	t.running = true
	t.startTime = time.Now()
	t.lastOutputTime = time.Now()

	go t.run()
}

// Stop halts the progress tracker
func (t *Tracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		close(t.done)
		t.running = false
	}
}

// Add increments the processed byte count
func (t *Tracker) Add(n uint64) {
	if n > 0 {
		t.processedBytes.Add(n)
	}
}

// run is the main tracking loop
func (t *Tracker) run() {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	op := t.operationName()
	t.printStart(op)

	for {
		select {
		case <-ticker.C:
			t.tick(op)
		case <-t.done:
			t.printComplete(op)
			return
		}
	}
}

// operationName returns the configured or default operation name
func (t *Tracker) operationName() string {
	if t.config.OperationName != "" {
		return t.config.OperationName
	}
	return "Processing"
}

// tick handles a single progress update
func (t *Tracker) tick(op string) {
	currentBytes := t.processedBytes.Load()
	rate := (currentBytes - t.prevBytes) * 4 // bytes/sec from 250ms interval
	t.prevBytes = currentBytes

	currentPct := float64(currentBytes) / float64(t.totalSize) * 100
	remaining := t.totalSize - currentBytes

	shouldUpdate := t.shouldUpdate(currentPct)
	if shouldUpdate {
		t.lastOutputTime = time.Now()
		t.printProgress(op, currentBytes, currentPct, rate, remaining)
	}
	t.prevPercentage = currentPct
}

// shouldUpdate determines if progress should be printed
func (t *Tracker) shouldUpdate(currentPct float64) bool {
	elapsed := time.Since(t.lastOutputTime)
	pctDiff := currentPct - t.prevPercentage

	return elapsed >= time.Second ||
		pctDiff >= 10 ||
		(currentPct >= 100 && t.prevPercentage < 100)
}

// printStart outputs the initial progress message
func (t *Tracker) printStart(op string) {
	if t.config.TestMode {
		fmt.Fprintf(t.config.Output, "%s%s▶ Starting %s...%s\n",
			ColorBold, ColorBlue, op, ColorReset)
	} else {
		fmt.Fprintf(t.config.Output, "Starting %s...\n", op)
	}
}

// printProgress outputs current progress
func (t *Tracker) printProgress(op string, currentBytes uint64, pct float64, rate, remaining uint64) {
	if t.config.TestMode {
		t.printTestProgress(op, pct)
	} else {
		t.printNormalProgress(op, currentBytes, pct, rate, remaining)
	}
	if f, ok := t.config.Output.(*os.File); ok {
		f.Sync()
	}
}

// printTestProgress outputs simplified progress for tests
func (t *Tracker) printTestProgress(op string, pct float64) {
	if pct >= 100 && t.prevPercentage < 100 {
		bar := ProgressBar(100, 20)
		fmt.Fprintf(t.config.Output, "%s%s✓ %s complete! %s 100%%%s\n",
			ColorBold, ColorGreen, op, bar, ColorReset)
	} else if pct-t.prevPercentage >= 25 || pct >= 100 {
		bar := ProgressBar(pct, 20)
		fmt.Fprintf(t.config.Output, "%s%s• %s progress: %s %.0f%%%s\n",
			ColorBold, ColorBlue, op, bar, pct, ColorReset)
	}
}

// printNormalProgress outputs detailed progress
func (t *Tracker) printNormalProgress(op string, currentBytes uint64, pct float64, rate, remaining uint64) {
	sizeInfo := FormatSize(currentBytes)
	rateInfo := FormatRate(rate)

	if t.totalSize > 1 {
		totalInfo := FormatSize(t.totalSize)
		eta := CalculateETA(remaining, rate)
		bar := ProgressBar(pct, 20)
		fmt.Fprintf(t.config.Output, "%s %s of %s %s %.1f%% | Rate: %s | ETA: %s\n",
			op, sizeInfo, totalInfo, bar, pct, rateInfo, eta)
	} else {
		fmt.Fprintf(t.config.Output, "%s %s | Rate: %s\n", op, sizeInfo, rateInfo)
	}
}

// printComplete outputs the final completion message
func (t *Tracker) printComplete(op string) {
	processed := t.processedBytes.Load()
	elapsed := time.Since(t.startTime).Seconds()
	sizeInfo := FormatSize(processed)

	if t.config.TestMode {
		fmt.Fprintf(t.config.Output, "%s%s✓ %s completed: %s in %.1f seconds%s\n",
			ColorBold, ColorGreen, op, sizeInfo, elapsed, ColorReset)
	} else {
		avgRate := FormatRate(uint64(float64(processed) / elapsed))
		fmt.Fprintf(t.config.Output, "%s completed: %s in %.1f seconds (avg rate: %s)\n",
			op, sizeInfo, elapsed, avgRate)
	}
}

// FormatSize returns a human-readable size string
func FormatSize(bytes uint64) string {
	if bytes < sizeUnit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(sizeUnit), 0
	for n := bytes / sizeUnit; n >= sizeUnit; n /= sizeUnit {
		div *= sizeUnit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatRate returns a human-readable rate string
func FormatRate(bytesPerSec uint64) string {
	if bytesPerSec < sizeUnit {
		return fmt.Sprintf("%d B/s", bytesPerSec)
	}
	div, exp := uint64(sizeUnit), 0
	for n := bytesPerSec / sizeUnit; n >= sizeUnit; n /= sizeUnit {
		div *= sizeUnit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB/s", float64(bytesPerSec)/float64(div), "KMGTPE"[exp])
}

// ProgressBar returns a visual progress bar string
func ProgressBar(percentage float64, width int) string {
	completed := int(percentage * float64(width) / 100)
	if completed > width {
		completed = width
	}

	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(strings.Repeat("█", completed))
	sb.WriteString(strings.Repeat("░", width-completed))
	sb.WriteString("]")
	return sb.String()
}

// CalculateETA estimates time remaining
func CalculateETA(bytesRemaining, rate uint64) string {
	if rate == 0 {
		return "calculating..."
	}

	seconds := float64(bytesRemaining) / float64(rate)

	switch {
	case seconds < 60:
		return fmt.Sprintf("%.0f seconds", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%.1f minutes", seconds/60)
	default:
		return fmt.Sprintf("%.1f hours", seconds/3600)
	}
}

// Writer wraps an io.Writer to track bytes written
type Writer struct {
	W io.Writer
}

// Write implements io.Writer and tracks progress
func (pw *Writer) Write(p []byte) (n int, err error) {
	n, err = pw.W.Write(p)
	if err == nil && n > 0 {
		AddBytes(uint64(n))
	}
	return
}

// --- Global API for backward compatibility ---

// Init initializes the global progress tracker
func Init(size uint64) {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalTracker != nil && globalTracker.running {
		return
	}

	globalTracker = NewTracker(size, Config{
		TestMode:      isTestMode,
		OperationName: operationName,
	})
	globalTracker.Start()
}

// Stop stops the global progress tracker
func Stop() {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalTracker != nil {
		globalTracker.Stop()
	}
}

// AddBytes adds bytes to the global tracker
func AddBytes(n uint64) {
	globalMu.Lock()
	t := globalTracker
	globalMu.Unlock()

	if t != nil {
		t.Add(n)
	}
}

// Global configuration (for backward compatibility)
var (
	isTestMode    bool
	operationName string
	configMu      sync.Mutex
)

// SetTestMode enables or disables test mode for the global tracker
func SetTestMode(enabled bool) {
	configMu.Lock()
	defer configMu.Unlock()
	isTestMode = enabled
}

// SetOperationName sets the operation name for the global tracker
func SetOperationName(name string) {
	configMu.Lock()
	defer configMu.Unlock()
	operationName = name
}
