package output

import (
	"time"

	"github.com/gnomegl/ulp/pkg/credential"
)

type Document struct {
	Channel  string `json:"channel,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
}

type Metadata struct {
	OriginalFilename string `json:"original_filename"`
	DatePosted       string `json:"date_posted,omitempty"`
}

type TelegramMetadata struct {
	ChannelID      string
	ChannelName    string
	ChannelAt      string
	DatePosted     *time.Time
	MessageContent string
	MessageID      string
}

type WriterOptions struct {
	MaxFileSize      int64
	OutputBaseName   string
	TelegramMetadata *TelegramMetadata
	EnableFreshness  bool
	NoSplit          bool
}

type Writer interface {
	WriteCredentials(credentials []credential.Credential, stats credential.ProcessingStats, opts WriterOptions) error
	Close() error
}

type FileManager interface {
	CreateNewFile() error
	GetCurrentFile() string
	GetCurrentSize() int64
	AddToCurrentSize(size int64)
	Close() error
}
