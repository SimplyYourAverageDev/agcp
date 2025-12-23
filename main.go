package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"agcp/pkg/core"
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
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	case "decompress":
		if err := handleDecompress(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "Invalid operation:", operation)
		printUsage()
		os.Exit(1)
	}
}

// printUsage prints the command-line usage information
func printUsage() {
	fmt.Println("AGCP - Andrew's Go Compression Program")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  agcp compress <input> [output.agcp]")
	fmt.Println("  agcp decompress <input.agcp> [output_name]")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  agcp compress myfile.txt")
	fmt.Println("  agcp compress mydir/ archive.agcp")
	fmt.Println("  agcp decompress archive.agcp")
	fmt.Println("  agcp decompress archive.agcp extracted/")
}

// handleCompress handles the compression operation
func handleCompress() error {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Println("Usage: agcp compress <input> [output.agcp]")
		os.Exit(1)
	}

	input := os.Args[2]
	output := resolveOutputPath(input)

	return core.Compress(input, output)
}

// resolveOutputPath determines the output path for compression
func resolveOutputPath(input string) string {
	if len(os.Args) == 4 {
		return os.Args[3]
	}

	// Use input name with .agcp extension
	autoName := filepath.Base(input) + ".agcp"
	if _, err := os.Stat(autoName); os.IsNotExist(err) {
		return autoName
	}

	return "output.agcp"
}

// handleDecompress handles the decompression operation
func handleDecompress() error {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Println("Usage: agcp decompress <input.agcp> [output_name]")
		os.Exit(1)
	}

	input := os.Args[2]
	destName := ""
	if len(os.Args) == 4 {
		destName = os.Args[3]
	}

	return core.Decompress(input, destName)
}
