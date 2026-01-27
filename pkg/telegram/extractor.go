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
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

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

	baseName := filepath.Base(filename)
	if match := regexp.MustCompile(`@([^-]+)`).FindStringSubmatch(baseName); len(match) > 1 {
		metadata.At = "@" + match[1]
		metadata.Name = match[1]
	}

	// Handle tdl naming convention: channelid_messageid_actualfilename
	// Try to strip the prefix and match against the actual filename
	cleanedBaseName := baseName
	if match := regexp.MustCompile(`^(\d+)_(\d+)_(.+)$`).FindStringSubmatch(baseName); len(match) > 3 {
		fileChannelID := match[1]
		fileMessageID := match[2]
		cleanedBaseName = match[3] // The actual filename without the tdl prefix

		// First try to match by message ID if channel ID matches
		if strconv.FormatInt(export.ID, 10) == fileChannelID {
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

		// If no match by message ID, try matching by the cleaned filename
		for _, message := range export.Messages {
			if message.File == cleanedBaseName {
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

	// Try exact match with original basename
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

	// Try partial match
	for _, message := range export.Messages {
		if message.File != "" && strings.Contains(baseName, strings.TrimSuffix(message.File, filepath.Ext(message.File))) {
			metadata.MessageID = strconv.FormatInt(message.ID, 10)
			metadata.MessageContent = message.Raw.Message
			if message.Date > 0 {
				dateTime := time.Unix(message.Date, 0)
				metadata.DatePosted = &dateTime
			}
			return metadata, nil
		}
	}

	return metadata, nil
}

func (e *DefaultExtractor) AutoDetectJSONFile(inputPath string) (string, error) {
	if strings.HasSuffix(inputPath, "/") {
		inputPath = strings.TrimSuffix(inputPath, "/")
	}

	dirName := filepath.Base(inputPath)
	parentDir := filepath.Dir(inputPath)
	jsonFile := filepath.Join(parentDir, dirName+".json")

	if _, err := os.ReadFile(jsonFile); err == nil {
		return jsonFile, nil
	}

	return "", fmt.Errorf("no matching JSON file found for %s", inputPath)
}
