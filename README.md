# AGCP - Andrew's Go Compression Program

AGCP is a file and directory compression utility written in Go, using LZ4 compression for fast and efficient archiving.

## Features

- Compresses both single files and directories
- Multi-threaded decompression for improved performance
- Progress reporting during operation
- Simple command-line interface
- Preserves directory structure
- Auto-generated output filenames

## Usage

### Compression

```
./agcp compress input [output.agcp]
```

- If `output.agcp` is not specified, a default name will be generated based on the input file or directory name.

### Decompression

```
./agcp decompress input.agcp [decompressed_name]
```

- If `decompressed_name` is not specified, the archive will be extracted with its original name.

## Examples

Compress a single file:
```
./agcp compress document.pdf
```

Compress a directory:
```
./agcp compress my_project my_project_backup.agcp
```

Decompress an archive:
```
./agcp decompress archive.agcp
```

Decompress to a specific location:
```
./agcp decompress archive.agcp extracted_data
```

## Testing

The project includes comprehensive test coverage for various scenarios. Tests are located in the `/tests` directory.

### Running the tests

Change to the tests directory and run the tests:
```
cd tests
go test -v
```

Run tests that should currently pass:
```
cd tests
go test -v -run "TestCompressDecompressFile|TestCompressDecompressDirectory|TestArchiveMetadata|TestErrorCases"
```

Run tests with short flag (skips long-running tests):
```
cd tests
go test -v -short
```

Run specific test:
```
cd tests
go test -v -run TestCompressDecompressFile
```

Run benchmarks:
```
cd tests
go test -v -bench=.
```

### Test Coverage

The test suite includes:

1. Basic functionality tests (currently implemented and passing):
   - Single file compression/decompression
   - Directory compression/decompression
   - Archive metadata validation
   - Basic error handling

2. Edge case tests (for future enhancements):
   - Empty files
   - Large number of files
   - Unicode paths and content

3. Performance benchmarks (for future optimization):
   - Compression benchmarks with various file sizes
   - Decompression benchmarks with various file sizes

4. Concurrent operation tests (for future enhancements):
   - Multiple concurrent compression/decompression tasks

Note: Some tests are designed for future enhancements and may not pass with the current implementation. These tests serve as a roadmap for future development.

## License

This project is open source software. 