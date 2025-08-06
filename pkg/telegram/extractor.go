package telegram

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DefaultExtractor struct{}

func NewDefaultExtractor() *DefaultExtractor {
	return &DefaultExtractor{}
}

func (e *DefaultExtractor) ExtractFromFile(jsonFile string, filename string) (*ChannelMetadata, error) {
	// Read and parse JSON file
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	// Try to fix common JSON issues (like Infinity values)
	dataStr := string(data)
	dataStr = strings.ReplaceAll(dataStr, "Infinity", "null")

	var export ChannelExport
	if err := json.Unmarshal([]byte(dataStr), &export); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return e.ExtractFromExport(&export, filename)
}

func (e *DefaultExtractor) ExtractFromExport(export *ChannelExport, filename string) (*ChannelMetadata, error) {
	metadata := &ChannelMetadata{
		ID: strconv.FormatInt(export.ID, 10),
	}

	// Extract channel name and @ handle from filename if available
	baseName := filepath.Base(filename)
	if match := regexp.MustCompile(`@([^-]+)`).FindStringSubmatch(baseName); len(match) > 1 {
		metadata.At = "@" + match[1]
		metadata.Name = match[1]
	}

	// Try to find matching message by message ID in filename first (more reliable)
	if match := regexp.MustCompile(`^[0-9]+_([0-9]+)_`).FindStringSubmatch(baseName); len(match) > 1 {
		fileMessageID := match[1]

		for _, message := range export.Messages {
			if strconv.FormatInt(message.ID, 10) == fileMessageID {
				metadata.MessageID = strconv.FormatInt(message.ID, 10)
				metadata.MessageContent = message.Raw.Message
				if message.Date > 0 {
					dateTime := time.Unix(message.Date, 0)
					metadata.DatePosted = &dateTime
				}
				return metadata, nil
			}
		}
	}

	// Fallback to exact filename match
	for _, message := range export.Messages {
		if message.File == baseName {
			metadata.MessageID = strconv.FormatInt(message.ID, 10)
			metadata.MessageContent = message.Raw.Message
			if message.Date > 0 {
				dateTime := time.Unix(message.Date, 0)
				metadata.DatePosted = &dateTime
			}
			return metadata, nil
		}
	}

	// Return metadata even if no message match found
	return metadata, nil
}

func (e *DefaultExtractor) AutoDetectJSONFile(inputPath string) (string, error) {
	if strings.HasSuffix(inputPath, "/") {
		inputPath = strings.TrimSuffix(inputPath, "/")
	}

	// For directory input, look for matching JSON file
	dirName := filepath.Base(inputPath)
	parentDir := filepath.Dir(inputPath)
	jsonFile := filepath.Join(parentDir, dirName+".json")

	if _, err := os.ReadFile(jsonFile); err == nil {
		return jsonFile, nil
	}

	return "", fmt.Errorf("no matching JSON file found for %s", inputPath)
}
