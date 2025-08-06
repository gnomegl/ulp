package output

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gnome/ulp/pkg/credential"
	"github.com/gnome/ulp/pkg/freshness"
)

type NDJSONWriter struct {
	fileManager   *NDJSONFileManager
	freshnessCalc freshness.Calculator
	currentWriter *bufio.Writer
	currentFile   *os.File
}

type NDJSONFileManager struct {
	baseName    string
	fileCounter int
	currentSize int64
	maxSize     int64
	currentFile *os.File
	noSplit     bool
}

func NewNDJSONWriter(maxFileSize int64) *NDJSONWriter {
	return &NDJSONWriter{
		freshnessCalc: freshness.NewDefaultCalculator(),
	}
}

func (w *NDJSONWriter) WriteCredentials(credentials []credential.Credential, stats credential.ProcessingStats, opts WriterOptions) error {
	// Initialize file manager
	w.fileManager = &NDJSONFileManager{
		baseName:    opts.OutputBaseName,
		fileCounter: 1,
		maxSize:     opts.MaxFileSize,
		noSplit:     opts.NoSplit,
	}

	// Create first file
	if err := w.fileManager.CreateNewFile(); err != nil {
		return fmt.Errorf("failed to create initial file: %w", err)
	}

	// Initialize writer
	w.currentFile = w.fileManager.currentFile
	w.currentWriter = bufio.NewWriter(w.currentFile)

	// Calculate freshness score if enabled
	var freshnessScore *freshness.Score
	if opts.EnableFreshness {
		var fileDate *time.Time
		if opts.TelegramMetadata != nil && opts.TelegramMetadata.DatePosted != nil {
			fileDate = opts.TelegramMetadata.DatePosted
		}

		freshnessScore = w.freshnessCalc.Calculate(
			stats.TotalLines,
			stats.ValidCredentials,
			stats.DuplicatesFound,
			fileDate,
			0, // File size bytes - could be calculated if needed
		)
	}

	// Write each credential as NDJSON
	for _, cred := range credentials {
		doc := w.createDocument(cred, opts, freshnessScore)

		jsonBytes, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("failed to marshal document: %w", err)
		}

		jsonLine := string(jsonBytes) + "\n"
		lineSize := int64(len(jsonLine))

		// Check if we need a new file (only if splitting is enabled)
		if !opts.NoSplit && w.fileManager.currentSize+lineSize > w.fileManager.maxSize && w.fileManager.currentSize > 0 {
			// Flush current writer
			if w.currentWriter != nil {
				w.currentWriter.Flush()
			}

			if err := w.fileManager.CreateNewFile(); err != nil {
				return fmt.Errorf("failed to create new file: %w", err)
			}

			// Reinitialize writer
			w.currentFile = w.fileManager.currentFile
			w.currentWriter = bufio.NewWriter(w.currentFile)
		}

		// Write the line
		if _, err := w.currentWriter.WriteString(jsonLine); err != nil {
			return fmt.Errorf("failed to write line: %w", err)
		}

		w.fileManager.currentSize += lineSize
	}

	// Flush the writer
	if w.currentWriter != nil {
		if err := w.currentWriter.Flush(); err != nil {
			return fmt.Errorf("failed to flush writer: %w", err)
		}
	}

	return nil
}

func (w *NDJSONWriter) createDocument(cred credential.Credential, opts WriterOptions, freshnessScore *freshness.Score) Document {
	doc := Document{
		Username: cred.Username,
		Password: cred.Password,
		URL:      cred.URL,
	}

	if opts.TelegramMetadata != nil {
		doc.Channel = opts.TelegramMetadata.ChannelName
		if opts.TelegramMetadata.DatePosted != nil {
			doc.Date = opts.TelegramMetadata.DatePosted.Format(time.RFC3339)
		}
	}

	return doc
}

func (w *NDJSONWriter) Close() error {
	if w.currentWriter != nil {
		if err := w.currentWriter.Flush(); err != nil {
			return err
		}
	}
	if w.fileManager != nil {
		return w.fileManager.Close()
	}
	return nil
}

func (fm *NDJSONFileManager) CreateNewFile() error {
	// Close current file if open
	if fm.currentFile != nil {
		fm.currentFile.Close()
	}

	// Create new filename
	var filename string
	if fm.noSplit {
		// When not splitting, use simple filename without counter
		filename = fmt.Sprintf("%s.jsonl", fm.baseName)
	} else {
		// When splitting, use numbered filenames
		filename = fmt.Sprintf("%s_%03d.jsonl", fm.baseName, fm.fileCounter)
	}

	// Create new file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}

	fm.currentFile = file
	fm.currentSize = 0
	fm.fileCounter++

	fmt.Fprintf(os.Stderr, "Created NDJSON file: %s\n", filename)
	return nil
}

func (fm *NDJSONFileManager) GetCurrentFile() string {
	if fm.currentFile != nil {
		return fm.currentFile.Name()
	}
	return ""
}

func (fm *NDJSONFileManager) GetCurrentSize() int64 {
	return fm.currentSize
}

func (fm *NDJSONFileManager) AddToCurrentSize(size int64) {
	fm.currentSize += size
}

func (fm *NDJSONFileManager) Close() error {
	if fm.currentFile != nil {
		return fm.currentFile.Close()
	}
	return nil
}
