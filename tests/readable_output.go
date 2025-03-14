// tests/readable_output.go

package tests

import (
	"fmt"
	"strings"
	"time"
)

// Colors and formatting for terminal output
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
)

// Global variables for indentation management
var (
	currentIndent = 0
	indentSize    = 2
)

// ReportStart prints a header for the test run
func ReportStart(title string) {
	clearScreen()
	divider := strings.Repeat("═", 60)
	fmt.Println()
	fmt.Printf("%s%s%s\n", colorBold, divider, colorReset)
	fmt.Printf("%s%s AGCP TEST: %s %s\n", colorBold, colorCyan, title, colorReset)
	fmt.Printf("%s%s%s\n", colorBold, divider, colorReset)
	fmt.Println()
}

// ReportEnd prints a footer for the test run
func ReportEnd(success bool, duration time.Duration) {
	divider := strings.Repeat("═", 60)
	fmt.Println()
	fmt.Printf("%s%s%s\n", colorBold, divider, colorReset)
	if success {
		fmt.Printf("%s%s✓ Test completed successfully in %.1f seconds%s\n",
			colorBold, colorGreen, duration.Seconds(), colorReset)
	} else {
		fmt.Printf("%s%s✗ Test failed after %.1f seconds%s\n",
			colorBold, colorRed, duration.Seconds(), colorReset)
	}
	fmt.Printf("%s%s%s\n", colorBold, divider, colorReset)
	fmt.Println()
}

// StartSection begins a new test section with a header
func StartSection(name string) {
	fmt.Println()
	fmt.Printf("%s%s▶ %s%s\n", colorBold, colorBlue, name, colorReset)
	currentIndent++
}

// EndSection completes a test section
func EndSection() {
	currentIndent--
	fmt.Println()
}

// Action describes a test action being performed
func Action(msg string) {
	indent := strings.Repeat(" ", currentIndent*indentSize)
	fmt.Printf("%s%s→ %s%s\n", indent, colorCyan, msg, colorReset)
}

// Info provides information about the test state
func Info(msg string) {
	indent := strings.Repeat(" ", currentIndent*indentSize)
	fmt.Printf("%s%s• %s%s\n", indent, colorBlue, msg, colorReset)
}

// Success reports a successful test assertion
func Success(msg string) {
	indent := strings.Repeat(" ", currentIndent*indentSize)
	fmt.Printf("%s%s✓ %s%s\n", indent, colorGreen, msg, colorReset)
}

// Warning reports a caution-worthy situation
func Warning(msg string) {
	indent := strings.Repeat(" ", currentIndent*indentSize)
	fmt.Printf("%s%s! %s%s\n", indent, colorYellow, msg, colorReset)
}

// Error reports a test error
func Error(msg string) {
	indent := strings.Repeat(" ", currentIndent*indentSize)
	fmt.Printf("%s%s✗ %s%s\n", indent, colorRed, msg, colorReset)
}

// HumanReadableSize returns a human-readable file size
func HumanReadableSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d bytes", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ProgressBar returns a simple progress bar
func ProgressBar(percentage float64, width int) string {
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

// clearScreen clears the terminal screen
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}
