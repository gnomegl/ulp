package output

import (
	"bufio"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gnomegl/ulp/pkg/credential"
)

type StdoutWriter struct {
	format           string
	writer           *bufio.Writer
	telegramMetadata *TelegramMetadata
}

func NewStdoutWriter(format string) *StdoutWriter {
	return &StdoutWriter{
		format: format,
		writer: bufio.NewWriter(os.Stdout),
	}
}

func NewStdoutWriterWithMetadata(format string, telegramMeta *TelegramMetadata) *StdoutWriter {
	return &StdoutWriter{
		format:           format,
		writer:           bufio.NewWriter(os.Stdout),
		telegramMetadata: telegramMeta,
	}
}

func generateDocID(username, url, password string) string {
	data := fmt.Sprintf("%s:%s:%s", username, url, password)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (w *StdoutWriter) WriteCredentials(credentials []credential.Credential, stats credential.ProcessingStats, opts WriterOptions) error {
	switch w.format {
	case "csv":
		return w.writeCSV(credentials, opts)
	case "jsonl":
		return w.writeJSONL(credentials, stats, opts)
	default: // txt
		return w.writeText(credentials)
	}
}

func (w *StdoutWriter) writeText(credentials []credential.Credential) error {
	for _, cred := range credentials {
		if _, err := fmt.Fprintf(w.writer, "%s:%s:%s\n", cred.URL, cred.Username, cred.Password); err != nil {
			return err
		}
	}
	return w.writer.Flush()
}

func (w *StdoutWriter) writeCSV(credentials []credential.Credential, opts WriterOptions) error {
	csvWriter := csv.NewWriter(w.writer)

	header := []string{"doc_id", "channel", "username", "password", "url", "date"}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	for _, cred := range credentials {
		docID := generateDocID(cred.Username, cred.URL, cred.Password)
		record := []string{docID, "", cred.Username, cred.Password, cred.URL, ""}

		if opts.TelegramMetadata != nil {
			record[1] = opts.TelegramMetadata.ChannelName
			if opts.TelegramMetadata.DatePosted != nil {
				record[5] = opts.TelegramMetadata.DatePosted.Format(time.RFC3339)
			}
		}

		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

func (w *StdoutWriter) writeCSVBatch(credentials []credential.Credential) error {
	csvWriter := csv.NewWriter(w.writer)

	for _, cred := range credentials {
		docID := generateDocID(cred.Username, cred.URL, cred.Password)
		record := []string{docID, "", cred.Username, cred.Password, cred.URL, ""}

		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}

func (w *StdoutWriter) writeJSONL(credentials []credential.Credential, stats credential.ProcessingStats, opts WriterOptions) error {
	encoder := json.NewEncoder(w.writer)

	for _, cred := range credentials {
		docID := generateDocID(cred.Username, cred.URL, cred.Password)

		doc := Document{
			URL:      cred.URL,
			Username: cred.Username,
			Password: cred.Password,
		}

		if opts.TelegramMetadata != nil && opts.TelegramMetadata.ChannelName != "" {
			doc.Channel = opts.TelegramMetadata.ChannelName
		}

		metadata := Metadata{
			OriginalFilename: opts.OutputBaseName,
		}

		if opts.TelegramMetadata != nil && opts.TelegramMetadata.DatePosted != nil {
			metadata.DatePosted = opts.TelegramMetadata.DatePosted.Format(time.RFC3339)
		}

		output := map[string]interface{}{
			"doc_id":   docID,
			"url":      doc.URL,
			"username": doc.Username,
			"password": doc.Password,
			"metadata": metadata,
		}

		if doc.Channel != "" {
			output["channel"] = doc.Channel
		}

		if err := encoder.Encode(output); err != nil {
			return err
		}
	}

	return w.writer.Flush()
}

func (w *StdoutWriter) Flush() error {
	return w.writer.Flush()
}

func (w *StdoutWriter) Close() error {
	return w.writer.Flush()
}

type StdoutBatchWriter struct {
	writer *StdoutWriter
}

func NewStdoutBatchWriter(format string) *StdoutBatchWriter {
	return &StdoutBatchWriter{
		writer: NewStdoutWriter(format),
	}
}

func NewStdoutBatchWriterWithMetadata(format string, telegramMeta *TelegramMetadata) *StdoutBatchWriter {
	writer := NewStdoutWriter(format)
	writer.telegramMetadata = telegramMeta
	return &StdoutBatchWriter{
		writer: writer,
	}
}

func (b *StdoutBatchWriter) WriteBatch(credentials []credential.Credential) error {
	if b.writer.format == "csv" {
		return b.writer.writeCSVBatch(credentials)
	}
	stats := credential.ProcessingStats{}
	opts := WriterOptions{}
	if b.writer.telegramMetadata != nil {
		opts.TelegramMetadata = b.writer.telegramMetadata
	}
	return b.writer.WriteCredentials(credentials, stats, opts)
}

func (b *StdoutBatchWriter) Flush() error {
	return b.writer.Flush()
}

func (b *StdoutBatchWriter) Close() error {
	return b.writer.Close()
}
