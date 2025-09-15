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
	"github.com/gnomegl/ulp/pkg/freshness"
)

type StdoutWriter struct {
	format string
	writer *bufio.Writer
}

func NewStdoutWriter(format string) *StdoutWriter {
	return &StdoutWriter{
		format: format,
		writer: bufio.NewWriter(os.Stdout),
	}
}

// generateDocID creates a hash from the cleaned username, url, and password
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

	// Write header with doc_id
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

func (w *StdoutWriter) writeJSONL(credentials []credential.Credential, stats credential.ProcessingStats, opts WriterOptions) error {
	encoder := json.NewEncoder(w.writer)

	var freshnessScore *freshness.Score
	if opts.EnableFreshness {
		calculator := freshness.NewDefaultCalculator()
		score := calculator.Calculate(
			stats.TotalLines,
			stats.ValidCredentials,
			stats.DuplicatesFound,
			nil,
			0,
		)
		freshnessScore = score
	}

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
			Freshness:        freshnessScore,
		}

		if opts.TelegramMetadata != nil {
			metadata.TelegramChannelID = opts.TelegramMetadata.ChannelID
			metadata.TelegramChannelName = opts.TelegramMetadata.ChannelName
			metadata.TelegramChannelAt = opts.TelegramMetadata.ChannelAt
			metadata.MessageContent = opts.TelegramMetadata.MessageContent
			metadata.MessageID = opts.TelegramMetadata.MessageID
			if opts.TelegramMetadata.DatePosted != nil {
				metadata.DatePosted = opts.TelegramMetadata.DatePosted.Format(time.RFC3339)
				doc.Date = metadata.DatePosted
			}
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
		if doc.Date != "" {
			output["date"] = doc.Date
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
