package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"agcp/pkg/core"
	"agcp/pkg/progress"
)

func main() {
	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	fmt.Printf("Available CPU cores: %d\n", runtime.NumCPU())

	operation := os.Args[1]
	switch operation {
	case "compress":
		if err := handleCompress(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	case "decompress":
		if err := handleDecompress(); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Invalid operation:", operation)
		printUsage()
		os.Exit(1)
	}
}

// printUsage prints the command-line usage information
func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  ./agcp compress input [output.agcp]")
	fmt.Println("  ./agcp decompress input.agcp [decompressed_name]")
}

// handleCompress handles the compression operation
func handleCompress() error {
	if len(os.Args) != 3 && len(os.Args) != 4 {
		fmt.Println("Usage: ./agcp compress input [output.agcp]")
		os.Exit(1)
	}

	input := os.Args[2]
	output := determineOutputPath(input)

	// Initialize progress tracking
	progress.Init(0) // Size will be calculated in Compress
	defer progress.Stop()

	return core.Compress(input, output)
}

// determineOutputPath determines the output path for compression
func determineOutputPath(input string) string {
	// If output is provided as an argument, use it
	if len(os.Args) == 4 {
		return os.Args[3]
	}

	// Otherwise, use input name + .agcp extension
	autoName := filepath.Base(input) + ".agcp"
	if _, err := os.Stat(autoName); os.IsNotExist(err) {
		return autoName
	}

	// Default fallback
	return "output.agcp"
}

// handleDecompress handles the decompression operation
func handleDecompress() error {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Println("Usage: ./agcp decompress input.agcp [decompressed_name]")
		os.Exit(1)
	}

	input := os.Args[2]
	decompressedName := ""
	if len(os.Args) == 4 {
		decompressedName = os.Args[3]
	}

	// Initialize progress tracking
	progress.Init(0) // Size will be calculated in Decompress
	defer progress.Stop()

	return core.Decompress(input, decompressedName)
}
