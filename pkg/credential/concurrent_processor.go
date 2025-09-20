package credential

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gnomegl/ulp/pkg/fileutil"
)

type ConcurrentProcessor struct {
	normalizer URLNormalizer
	workers    int
}

func NewConcurrentProcessor(workers int) *ConcurrentProcessor {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &ConcurrentProcessor{
		normalizer: NewDefaultURLNormalizer(),
		workers:    workers,
	}
}

type lineResult struct {
	lineNum    int
	credential *Credential
	original   string
	err        error
}

type fileJob struct {
	path string
	info os.FileInfo
}

func (p *ConcurrentProcessor) ProcessLine(line string) (*Credential, error) {
	if line == "" {
		return nil, fmt.Errorf("empty line")
	}

	if !strings.Contains(line, ":") && !strings.Contains(line, "|") {
		return nil, fmt.Errorf("line doesn't match credential format")
	}

	normalized := p.normalizer.Normalize(line)
	if normalized == "" {
		return nil, fmt.Errorf("normalization resulted in empty string")
	}

	var urlPart, username, password string

	if strings.HasPrefix(normalized, "android://") {
		if idx := strings.Index(normalized, "/:"); idx != -1 {
			urlPart = normalized[:idx+1]
			remaining := normalized[idx+2:]

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
		parts := strings.Split(normalized, ":")
		if len(parts) < 3 {
			return nil, fmt.Errorf("insufficient parts after splitting (need at least 3)")
		}

		urlPart = parts[0]
		username = parts[1]
		password = strings.Join(parts[2:], ":")
	}

	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("username or password is empty")
	}

	fullURL := urlPart
	if !strings.Contains(fullURL, "://") {
		fullURL = "https://" + fullURL
	}

	return &Credential{
		URL:      fullURL,
		Username: username,
		Password: password,
	}, nil
}

func (p *ConcurrentProcessor) ProcessFile(filename string, opts ProcessingOptions) (*ProcessingResult, error) {
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

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", filename, err)
	}

	if fileInfo.Size() < 1*1024*1024 && p.workers <= 1 {
		return p.processFileSequential(file, filename, opts)
	}

	return p.processFileConcurrent(file, filename, opts)
}

func (p *ConcurrentProcessor) processFileSequential(file *os.File, filename string, opts ProcessingOptions) (*ProcessingResult, error) {
	var credentials []Credential
	var duplicates []string
	stats := ProcessingStats{}
	seenHashes := make(map[string]bool)

	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		stats.TotalLines++
		lineCount++

		if lineCount%1000 == 0 {
			fmt.Fprintf(os.Stderr, ".")
		}

		cred, err := p.ProcessLine(line)
		if err != nil {
			stats.LinesIgnored++
			continue
		}

		if opts.EnableDeduplication {
			credKey := fmt.Sprintf("%s:%s:%s", cred.URL, cred.Username, cred.Password)
			if seenHashes[credKey] {
				stats.DuplicatesFound++
				if opts.SaveDuplicates {
					duplicates = append(duplicates, line)
				}
				continue
			}
			seenHashes[credKey] = true
		}

		credentials = append(credentials, *cred)
		stats.ValidCredentials++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filename, err)
	}

	if opts.SaveDuplicates && opts.DuplicatesFile != "" && len(duplicates) > 0 {
		if err := saveDuplicatesToFile(opts.DuplicatesFile, duplicates); err != nil {
			return nil, fmt.Errorf("failed to save duplicates: %w", err)
		}
	}

	return &ProcessingResult{
		Credentials: credentials,
		Stats:       stats,
		Duplicates:  duplicates,
	}, nil
}

func (p *ConcurrentProcessor) processFileConcurrent(file *os.File, filename string, opts ProcessingOptions) (*ProcessingResult, error) {
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filename, err)
	}

	totalLines := len(lines)
	fmt.Fprintf(os.Stderr, "Processing %d lines with %d workers...\n", totalLines, p.workers)

	lineChan := make(chan struct {
		lineNum int
		line    string
	}, 100)
	resultChan := make(chan lineResult, 100)

	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range lineChan {
				cred, err := p.ProcessLine(work.line)
				resultChan <- lineResult{
					lineNum:    work.lineNum,
					credential: cred,
					original:   work.line,
					err:        err,
				}
			}
		}()
	}

	results := make([]lineResult, totalLines)
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		processedCount := 0
		for result := range resultChan {
			results[result.lineNum] = result
			processedCount++
			if processedCount%1000 == 0 {
				fmt.Fprintf(os.Stderr, ".")
			}
		}
	}()

	for i, line := range lines {
		lineChan <- struct {
			lineNum int
			line    string
		}{lineNum: i, line: line}
	}
	close(lineChan)

	wg.Wait()
	close(resultChan)
	resultWg.Wait()

	var credentials []Credential
	var duplicates []string
	stats := ProcessingStats{TotalLines: totalLines}
	seenHashes := make(map[string]bool)

	for _, result := range results {
		if result.err != nil {
			stats.LinesIgnored++
			continue
		}

		if result.credential == nil {
			continue
		}

		if opts.EnableDeduplication {
			credKey := fmt.Sprintf("%s:%s:%s",
				result.credential.URL,
				result.credential.Username,
				result.credential.Password)
			if seenHashes[credKey] {
				stats.DuplicatesFound++
				if opts.SaveDuplicates {
					duplicates = append(duplicates, result.original)
				}
				continue
			}
			seenHashes[credKey] = true
		}

		credentials = append(credentials, *result.credential)
		stats.ValidCredentials++
	}

	fmt.Fprintf(os.Stderr, "\n")

	if opts.SaveDuplicates && opts.DuplicatesFile != "" && len(duplicates) > 0 {
		if err := saveDuplicatesToFile(opts.DuplicatesFile, duplicates); err != nil {
			return nil, fmt.Errorf("failed to save duplicates: %w", err)
		}
	}

	return &ProcessingResult{
		Credentials: credentials,
		Stats:       stats,
		Duplicates:  duplicates,
	}, nil
}

func (p *ConcurrentProcessor) ProcessDirectory(dirname string, opts ProcessingOptions) (map[string]*ProcessingResult, error) {
	var files []fileJob
	err := filepath.Walk(dirname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, fileJob{path: path, info: info})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dirname, err)
	}

	totalFiles := len(files)
	fmt.Fprintf(os.Stderr, "Found %d files to process in %s\n", totalFiles, dirname)

	jobChan := make(chan fileJob, p.workers)
	resultChan := make(chan struct {
		path   string
		result *ProcessingResult
		err    error
	}, p.workers)

	var processedFiles int32
	var skippedFiles int32

	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobChan {
				isBinary, err := fileutil.IsBinaryFile(job.path)
				if err != nil {
					atomic.AddInt32(&skippedFiles, 1)
					current := atomic.AddInt32(&processedFiles, 1)
					fmt.Fprintf(os.Stderr, "[%d/%d] Worker %d: Warning: failed to check if file is binary %s: %v\n",
						current, totalFiles, workerID, job.path, err)
					resultChan <- struct {
						path   string
						result *ProcessingResult
						err    error
					}{path: job.path, result: nil, err: err}
					continue
				}
				if isBinary {
					atomic.AddInt32(&skippedFiles, 1)
					current := atomic.AddInt32(&processedFiles, 1)
					fmt.Fprintf(os.Stderr, "[%d/%d] Worker %d: Skipping binary file: %s\n",
						current, totalFiles, workerID, filepath.Base(job.path))
					continue
				}

				current := atomic.LoadInt32(&processedFiles)
				fmt.Fprintf(os.Stderr, "[%d/%d] Worker %d: Processing: %s",
					current+1, totalFiles, workerID, filepath.Base(job.path))

				result, err := p.ProcessFile(job.path, opts)
				if err != nil {
					atomic.AddInt32(&skippedFiles, 1)
					atomic.AddInt32(&processedFiles, 1)
					fmt.Fprintf(os.Stderr, " - Error: %v\n", err)
					resultChan <- struct {
						path   string
						result *ProcessingResult
						err    error
					}{path: job.path, result: nil, err: err}
					continue
				}

				atomic.AddInt32(&processedFiles, 1)
				fmt.Fprintf(os.Stderr, " - Done (%d credentials found)\n", len(result.Credentials))
				resultChan <- struct {
					path   string
					result *ProcessingResult
					err    error
				}{path: job.path, result: result, err: nil}
			}
		}(i)
	}

	results := make(map[string]*ProcessingResult)
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		for i := 0; i < totalFiles; i++ {
			res := <-resultChan
			if res.err == nil && res.result != nil {
				results[res.path] = res.result
			}
		}
	}()

	for _, job := range files {
		jobChan <- job
	}
	close(jobChan)

	wg.Wait()
	close(resultChan)
	resultWg.Wait()

	fmt.Fprintf(os.Stderr, "\nDirectory processing complete: %d files processed, %d skipped\n",
		int(processedFiles)-int(skippedFiles), int(skippedFiles))

	return results, nil
}

func saveDuplicatesToFile(filename string, duplicates []string) error {
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
