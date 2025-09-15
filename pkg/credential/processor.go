package credential

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gnomegl/ulp/pkg/fileutil"
)

type DefaultProcessor struct {
	normalizer URLNormalizer
	seenHashes map[string]bool
}

func NewDefaultProcessor() *DefaultProcessor {
	return &DefaultProcessor{
		normalizer: NewDefaultURLNormalizer(),
		seenHashes: make(map[string]bool),
	}
}

func (p *DefaultProcessor) ProcessLine(line string) (*Credential, error) {
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	// Skip lines that don't contain : or |
	if !strings.Contains(line, ":") && !strings.Contains(line, "|") {
		return nil, fmt.Errorf("line doesn't match credential format")
	}

	// Normalize the line
	normalized := p.normalizer.Normalize(line)
	if normalized == "" {
		return nil, fmt.Errorf("normalization resulted in empty string")
	}

	// Split into parts - handle Android URLs specially
	var urlPart, username, password string

	if strings.HasPrefix(normalized, "android://") {
		// For Android URLs, find the /: separator
		if idx := strings.Index(normalized, "/:"); idx != -1 {
			urlPart = normalized[:idx+1]    // Include the trailing /
			remaining := normalized[idx+2:] // Skip the /:

			// Split the remaining part by first colon
			colonIdx := strings.Index(remaining, ":")
			if colonIdx == -1 {
				return nil, fmt.Errorf("invalid Android URL format: missing password")
			}
			username = remaining[:colonIdx]
			password = remaining[colonIdx+1:]
		} else {
			return nil, fmt.Errorf("invalid Android URL format: missing /: separator")
		}
	} else {
		// Standard splitting for other URLs
		parts := strings.Split(normalized, ":")
		if len(parts) < 3 {
			return nil, fmt.Errorf("insufficient parts after splitting (need at least 3)")
		}

		urlPart = parts[0]
		username = parts[1]
		password = strings.Join(parts[2:], ":") // Rejoin in case password contains colons
	}

	// Validate that username and password are not empty
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("username or password is empty")
	}

	// Ensure URL has protocol (but preserve special schemes like android://)
	fullURL := urlPart
	if !strings.Contains(fullURL, "://") {
		// Only add https:// if no protocol is present
		fullURL = "https://" + fullURL
	}

	return &Credential{
		URL:      fullURL,
		Username: username,
		Password: password,
	}, nil
}

func (p *DefaultProcessor) ProcessFile(filename string, opts ProcessingOptions) (*ProcessingResult, error) {
	// Check if the file is binary before processing
	isBinary, err := fileutil.IsBinaryFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to check if file is binary %s: %w", filename, err)
	}
	if isBinary {
		return nil, fmt.Errorf("file %s appears to be a binary file, skipping", filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	var credentials []Credential
	var duplicates []string
	stats := ProcessingStats{}

	// Reset seen hashes for this file if deduplication is enabled
	if opts.EnableDeduplication {
		p.seenHashes = make(map[string]bool)
	}

	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		stats.TotalLines++
		lineCount++

		// Show progress every 1000 lines for large files
		if lineCount%1000 == 0 {
			fmt.Fprintf(os.Stderr, ".")
		}

		cred, err := p.ProcessLine(line)
		if err != nil {
			stats.LinesIgnored++
			continue
		}

		// Check for duplicates if deduplication is enabled
		if opts.EnableDeduplication {
			credKey := fmt.Sprintf("%s:%s:%s", cred.URL, cred.Username, cred.Password)
			if p.seenHashes[credKey] {
				stats.DuplicatesFound++
				if opts.SaveDuplicates {
					duplicates = append(duplicates, line)
				}
				continue
			}
			p.seenHashes[credKey] = true
		}

		credentials = append(credentials, *cred)
		stats.ValidCredentials++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filename, err)
	}

	// Save duplicates to file if requested
	if opts.SaveDuplicates && opts.DuplicatesFile != "" && len(duplicates) > 0 {
		if err := p.saveDuplicatesToFile(opts.DuplicatesFile, duplicates); err != nil {
			return nil, fmt.Errorf("failed to save duplicates: %w", err)
		}
	}

	return &ProcessingResult{
		Credentials: credentials,
		Stats:       stats,
		Duplicates:  duplicates,
	}, nil
}

func (p *DefaultProcessor) ProcessDirectory(dirname string, opts ProcessingOptions) (map[string]*ProcessingResult, error) {
	results := make(map[string]*ProcessingResult)

	// First, count total files
	var totalFiles, processedFiles, skippedFiles int
	err := filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalFiles++
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count files in directory %s: %w", dirname, err)
	}

	fmt.Fprintf(os.Stderr, "Found %d files to process in %s\n", totalFiles, dirname)

	err = filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file is binary before processing
		isBinary, err := fileutil.IsBinaryFile(path)
		if err != nil {
			// Log warning but continue processing other files
			skippedFiles++
			fmt.Fprintf(os.Stderr, "[%d/%d] Warning: failed to check if file is binary %s: %v\n",
				processedFiles+skippedFiles, totalFiles, path, err)
			return nil
		}
		if isBinary {
			// Skip binary files with progress
			skippedFiles++
			fmt.Fprintf(os.Stderr, "[%d/%d] Skipping binary file: %s\n",
				processedFiles+skippedFiles, totalFiles, filepath.Base(path))
			return nil
		}

		// Process each file
		fmt.Fprintf(os.Stderr, "[%d/%d] Processing: %s",
			processedFiles+skippedFiles+1, totalFiles, filepath.Base(path))

		result, err := p.ProcessFile(path, opts)
		if err != nil {
			// Log error but continue processing other files
			skippedFiles++
			fmt.Fprintf(os.Stderr, " - Error: %v\n", err)
			return nil
		}

		processedFiles++
		fmt.Fprintf(os.Stderr, " - Done (%d credentials found)\n", len(result.Credentials))
		results[path] = result
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to process directory %s: %w", dirname, err)
	}

	fmt.Fprintf(os.Stderr, "\nDirectory processing complete: %d files processed, %d skipped\n",
		processedFiles, skippedFiles)

	return results, nil
}

func (p *DefaultProcessor) saveDuplicatesToFile(filename string, duplicates []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, dup := range duplicates {
		if _, err := writer.WriteString(dup + "\n"); err != nil {
			return err
		}
	}

	return nil
}
