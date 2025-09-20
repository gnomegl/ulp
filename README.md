# ulp

A high-performance credentials file processing tool with proper separation of concerns and comprehensive functionality.

## Features

- **Multithreaded Processing**: Concurrent processing with configurable worker threads for 2-3x performance improvement
- **Credential Processing**: Clean and normalize domain formats from credential files
- **Deduplication**: Remove duplicate entries with optional duplicate output file
- **Multiple Output Formats**: TXT (default), NDJSON/JSONL, and CSV output formats
- **NDJSON/JSONL Output**: Create structured JSON files for Meilisearch indexing
- **Freshness Scoring**: Calculate quality scores based on duplicate percentage, file size, and age
- **Telegram Integration**: Process Telegram channel metadata when available
- **Multiple Input Formats**: Handle URL:user:pass, domain:user:pass, pipe-separated formats
- **Directory Processing**: Recursively process entire directory structures with parallel file processing
- **File Splitting**: Optional splitting of large NDJSON files at 100MB for optimal indexing

## Installation

```bash
# Build from source
go build -o ulp .

# Or install directly
go install github.com/gnomegl/ulp@latest
```

## Usage

### Basic Commands

```bash
# Clean and deduplicate (default behavior)
./ulp input.txt output.txt

# Clean only (no deduplication)
./ulp clean input.txt output.txt

# Deduplicate with duplicate output
./ulp dedupe input.txt output.txt --dupes-file duplicates.txt

# Convert to text format (default)
./ulp txt input.txt

# Convert to NDJSON/JSONL format
./ulp jsonl input.txt

# Convert to CSV format
./ulp csv input.txt

# Full processing (clean, dedupe, and convert - default: txt)
./ulp full input.txt

# Full processing with specific format
./ulp full input.txt --format jsonl
./ulp full input.txt --format csv

# Process directory recursively
./ulp full /path/to/directory/
```

### Advanced Options

```bash
# Specify number of worker threads (default: auto-detect CPU cores)
./ulp clean input.txt output.txt -w 8
./ulp full large_file.txt --workers 4

# With Telegram metadata
./ulp full input.txt --json-file channel_export.json --channel-name "example" --channel-at "@example"

# Disable freshness scoring
./ulp jsonl input.txt --no-freshness

# Save duplicates to file
./ulp input.txt output.txt --dupes-file duplicates.txt

# Specify output directory for JSONL files
./ulp jsonl input.txt -o /path/to/output/
./ulp full input.txt --output-dir /custom/output/

# Process directory with custom output location and parallel processing
./ulp jsonl /path/to/input/dir/ -o /path/to/output/dir/ -w 8

# Enable file splitting at 100MB (default is single file)
./ulp jsonl large_input.txt --split
./ulp full large_input.txt -s
```

### Multithreading

ULP supports concurrent processing for improved performance:

```bash
# Auto-detect optimal worker count (default)
./ulp clean input.txt output.txt

# Use specific number of workers
./ulp clean input.txt output.txt --workers 4

# Process directory with 8 parallel workers
./ulp full /path/to/directory/ -w 8
```

See [MULTITHREADING.md](MULTITHREADING.md) for detailed performance benchmarks and usage.

## Architecture

The Go version is structured with proper separation of concerns:

### Package Structure

```
pkg/
├── credential/     # Core credential processing logic
│   ├── types.go                # Data structures and interfaces
│   ├── normalizer.go           # URL normalization logic
│   ├── processor.go            # Default processing implementation
│   └── concurrent_processor.go # Multithreaded processor
├── freshness/      # Freshness scoring algorithm
│   ├── types.go           # Scoring configuration and structures
│   └── calculator.go      # Score calculation implementation
├── output/         # NDJSON/JSONL output generation
│   ├── types.go           # Output data structures
│   ├── text.go            # Text writer implementation
│   ├── csv.go             # CSV writer implementation
│   └── ndjson.go          # NDJSON writer implementation
├── telegram/       # Telegram metadata processing
│   ├── types.go           # Telegram data structures
│   └── extractor.go       # Metadata extraction logic
└── fileutil/       # File processing utilities
    └── utils.go           # Common file operations
```

### Key Components

1. **Credential Processor**: Handles line-by-line processing, normalization, and deduplication
2. **Freshness Calculator**: Implements the 1-5 scoring algorithm based on duplicate percentage, file size, and age
3. **NDJSON Writer**: Creates structured JSON output with automatic file splitting
4. **Telegram Extractor**: Processes Telegram channel exports for metadata enrichment
5. **CLI Commands**: Cobra-based command structure with comprehensive flag support

## Input Format Support

- `domain.com:username:password`
- `https://domain.com:username:password`
- `www.domain.com/path:username:password`
- `domain.com|username|password` (pipe characters converted to colons)

## Output Formats

### Text Output (Default)
Simple, clean format for credentials:
```
url:email:password
https://example.com:user1:pass123
https://site.com:admin:secretpass
```

### CSV Output
Structured CSV with metadata:
```csv
channel,username,password,url,date
example_channel,user1,pass123,https://example.com,2024-01-01T12:00:00Z
```

### NDJSON Output Structure

```json
{
  "url": "https://domain.com/path",
  "username": "user",
  "password": "pass",
  "metadata": {
    "original_filename": "input.txt",
    "telegram_channel_id": "123456",
    "telegram_channel_name": "channel",
    "telegram_channel_at": "@channel",
    "date_posted": "2024-01-01T12:00:00Z",
    "message_content": "...",
    "message_id": "789",
    "freshness": {
      "freshness_score": 4.0,
      "freshness_category": "good",
      "duplicate_percentage": 0.140,
      "total_lines_processed": 1000,
      "valid_credentials": 860,
      "duplicates_removed": 140,
      "scoring_algorithm_version": "1.0"
    }
  }
}
```

## Freshness Scoring

The freshness scoring system evaluates credential file quality on a **1-5 scale**:

### Score Categories
- **excellent** (4.5-5.0): High-quality, minimal duplicates
- **good** (3.5-4.4): Good quality with acceptable duplicate levels
- **fair** (2.5-3.4): Average quality, moderate duplicates
- **poor** (1.5-2.4): Low quality, high duplicate percentage
- **stale** (1.0-1.4): Very poor quality, mostly duplicates

### Scoring Factors
- **Base Score**: Determined by duplicate percentage
- **Size Bonus**: +0.5 for large files (>1000 lines) with <10% duplicates
- **Age Penalty**: Up to -1.0 for files older than 30 days

## Testing

### Running Tests

```bash
# Build and run comprehensive test suite
go test -v test.go

# Run all package tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run specific package tests
go test ./pkg/credential -v
go test ./pkg/freshness -v

# Run benchmarks
go test -bench=. test.go

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Coverage

The comprehensive test suite (`test.go`) includes:

- **Basic Functionality**: Credential parsing, deduplication, URL normalization
- **JSONL Generation**: Structured output with metadata and freshness scoring
- **Directory Processing**: Recursive file handling with proper output structure
- **Freshness Scoring**: Accuracy testing for all score categories
- **Telegram Integration**: Metadata extraction and enrichment
- **Performance Testing**: Large file processing and benchmarks
- **Error Handling**: Invalid inputs, missing files, malformed data
- **Edge Cases**: Complex passwords, various separators, invalid lines

### Example Test Run

```bash
# Run the comprehensive test suite
go test -v test.go

# Output:
# === RUN   TestBasicFunctionality
# === RUN   TestBasicFunctionality/basic_credentials
# === RUN   TestBasicFunctionality/duplicate_handling
# === RUN   TestBasicFunctionality/url_normalization
# ...
# === RUN   TestPerformance
# --- PASS: TestPerformance (0.45s)
#     test.go:463: Processed 10,000 lines in 234ms
# PASS
```

