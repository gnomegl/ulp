package output

import (
	"bufio"
	"fmt"
	"os"

	"github.com/gnomegl/ulp/pkg/credential"
)

type TextWriter struct {
	writer *bufio.Writer
	file   *os.File
}

func NewTextWriter(filename string) (*TextWriter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create text file: %w", err)
	}

	writer := bufio.NewWriter(file)

	return &TextWriter{
		writer: writer,
		file:   file,
	}, nil
}

func (w *TextWriter) WriteCredentials(credentials []credential.Credential, stats credential.ProcessingStats, opts WriterOptions) error {
	for _, cred := range credentials {
		line := fmt.Sprintf("%s:%s:%s\n", cred.URL, cred.Username, cred.Password)
		if _, err := w.writer.WriteString(line); err != nil {
			return fmt.Errorf("failed to write text record: %w", err)
		}
	}

	return w.writer.Flush()
}

func (w *TextWriter) Close() error {
	if err := w.writer.Flush(); err != nil {
		return err
	}
	return w.file.Close()
}
