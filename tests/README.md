# ULP Tests

This directory contains all test files and test data for the ULP (URL Line Processor) tool.

## Directory Structure

```
tests/
├── data/                        # Test data files
│   ├── binary_test.dat         # Binary file for testing binary detection
│   ├── credentials_basic.txt   # Basic credential test file
│   ├── credentials_pipe_separated.txt  # Pipe-separated credentials
│   ├── credentials_url_formats.txt     # Various URL format tests
│   ├── credentials_with_duplicates.txt # Duplicate detection tests
│   └── test_creds.txt          # Original test credentials
├── integration/                 # Integration tests
│   └── test.go                 # Main integration test suite
├── run_tests.sh                # Test runner script
└── README.md                   # This file
```

## Running Tests

### Quick Start
```bash
# Run all tests
./run_tests.sh

# Run only integration tests
cd integration && go test -v

# Run specific test
cd integration && go test -v -run TestBinaryFileDetection

# Run benchmarks
cd integration && go test -bench=.
```

### From Project Root
```bash
# Build the tool first
go build -o ulp-go main.go

# Run integration tests
go test -v ./tests/integration/...

# Run unit tests
go test -v ./pkg/...
```

## Test Categories

### Integration Tests (`integration/test.go`)
- **Basic Functionality**: Credential parsing, deduplication, normalization
- **JSONL Generation**: Output format, metadata, freshness scoring
- **Directory Processing**: Recursive processing, output structure
- **Binary File Detection**: Skipping binary files, mixed file handling
- **Error Handling**: Invalid inputs, malformed data
- **Performance Tests**: Large file processing benchmarks

### Test Data Files

#### `credentials_basic.txt`
Simple credential file with clean format for basic parsing tests.

#### `credentials_with_duplicates.txt`
Contains duplicate entries to test deduplication functionality.

#### `credentials_url_formats.txt`
Various URL formats (https://, www., IP addresses) to test normalization.

#### `credentials_pipe_separated.txt`
Tests pipe character (|) separator support and complex passwords.

#### `binary_test.dat`
Binary file containing null bytes to test binary file detection.

#### `test_creds.txt`
Original test file with mixed scenarios.

## Adding New Tests

1. **Integration Tests**: Add new test functions to `integration/test.go`
2. **Test Data**: Add new data files to the `data/` directory
3. **Unit Tests**: Create test files in the appropriate `pkg/*/` directories

## Test Coverage

Run with coverage:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```