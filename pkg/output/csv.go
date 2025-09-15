package output

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/gnomegl/ulp/pkg/credential"
)

// generateCSVDocID creates a hash from the cleaned username, url, and password
func generateCSVDocID(username, url, password string) string {
	data := fmt.Sprintf("%s:%s:%s", username, url, password)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

type CSVWriter struct {
	writer *csv.Writer
	file   *os.File
}

func NewCSVWriter(filename string) (*CSVWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV file: %w", err)
	}

	writer := csv.NewWriter(file)

	// Write header with doc_id
	header := []string{"doc_id", "channel", "username", "password", "url", "date"}
	if err := writer.Write(header); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	return &CSVWriter{
		writer: writer,
		file:   file,
	}, nil
}

func (w *CSVWriter) WriteCredentials(credentials []credential.Credential, stats credential.ProcessingStats, opts WriterOptions) error {
	for _, cred := range credentials {
		record := w.createRecord(cred, opts)
		if err := w.writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	w.writer.Flush()
	return w.writer.Error()
}

func (w *CSVWriter) createRecord(cred credential.Credential, opts WriterOptions) []string {
	// Generate doc_id from cleaned credentials
	docID := generateCSVDocID(cred.Username, cred.URL, cred.Password)

	// Initialize record with doc_id and empty values
	record := []string{docID, "", cred.Username, cred.Password, cred.URL, ""}

	// Add telegram metadata if available
	if opts.TelegramMetadata != nil {
		record[1] = opts.TelegramMetadata.ChannelName
		if opts.TelegramMetadata.DatePosted != nil {
			record[5] = opts.TelegramMetadata.DatePosted.Format(time.RFC3339)
		}
	}

	return record
}

func (w *CSVWriter) Close() error {
	w.writer.Flush()
	if err := w.writer.Error(); err != nil {
		return err
	}
	return w.file.Close()
}
