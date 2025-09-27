package output

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gnomegl/ulp/pkg/credential"
)

func generateNDJSONDocID(username, url, password string) string {
	data := fmt.Sprintf("%s:%s:%s", username, url, password)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

type NDJSONWriter struct {
	fileManager   *NDJSONFileManager
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
	return &NDJSONWriter{}
}

func (w *NDJSONWriter) WriteCredentials(credentials []credential.Credential, stats credential.ProcessingStats, opts WriterOptions) error {
	w.fileManager = &NDJSONFileManager{
		baseName:    opts.OutputBaseName,
		fileCounter: 1,
		maxSize:     opts.MaxFileSize,
		noSplit:     opts.NoSplit,
	}

	if err := w.fileManager.CreateNewFile(); err != nil {
		return fmt.Errorf("failed to create initial file: %w", err)
	}

	w.currentFile = w.fileManager.currentFile
	w.currentWriter = bufio.NewWriter(w.currentFile)

	for _, cred := range credentials {
		docID := generateNDJSONDocID(cred.Username, cred.URL, cred.Password)

		doc := w.createDocument(cred, opts)

		output := map[string]interface{}{
			"doc_id":   docID,
			"url":      doc.URL,
			"username": doc.Username,
			"password": doc.Password,
		}

		if doc.Channel != "" {
			output["channel"] = doc.Channel
		}

		metadata := Metadata{
			OriginalFilename: opts.OutputBaseName,
		}

		if opts.TelegramMetadata != nil && opts.TelegramMetadata.DatePosted != nil {
			metadata.DatePosted = opts.TelegramMetadata.DatePosted.Format(time.RFC3339)
		}

		output["metadata"] = metadata

		jsonBytes, err := json.Marshal(output)
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

func (w *NDJSONWriter) createDocument(cred credential.Credential, opts WriterOptions) Document {
	doc := Document{
		Username: cred.Username,
		Password: cred.Password,
		URL:      cred.URL,
	}

	if opts.TelegramMetadata != nil {
		doc.Channel = opts.TelegramMetadata.ChannelName
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
