package telegram

import "time"

type ChannelExport struct {
	ID       int64     `json:"id"`
	Messages []Message `json:"messages"`
}

type Message struct {
	ID   int64   `json:"id"`
	Date int64   `json:"date"`
	File string  `json:"file,omitempty"`
	Raw  RawData `json:"raw,omitempty"`
}

type RawData struct {
	Message string `json:"Message,omitempty"`
}

type ChannelMetadata struct {
	ID             string
	Name           string
	At             string
	MessageID      string
	MessageContent string
	DatePosted     *time.Time
}

type MetadataExtractor interface {
	ExtractFromFile(jsonFile string, filename string) (*ChannelMetadata, error)
	ExtractFromExport(export *ChannelExport, filename string) (*ChannelMetadata, error)
}
