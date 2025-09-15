package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gnomegl/ulp/internal/flags"
	"github.com/gnomegl/ulp/pkg/credential"
	"github.com/gnomegl/ulp/pkg/fileutil"
	"github.com/gnomegl/ulp/pkg/telegram"
)

type BaseCommand struct {
	Flags flags.CommonFlags
}

func (b *BaseCommand) ValidateInput(inputPath string) error {
	if !fileutil.FileExists(inputPath) {
		return fmt.Errorf("input file or directory '%s' not found", inputPath)
	}
	return nil
}

func (b *BaseCommand) ExtractTelegramMetadata(inputPath string) (*telegram.ChannelMetadata, error) {
	jsonPath := b.Flags.JsonFile

	if jsonPath == "" && !fileutil.IsDirectory(inputPath) {
		dir := filepath.Dir(inputPath)
		base := filepath.Base(inputPath)
		possibleJSON := filepath.Join(dir, strings.TrimSuffix(base, filepath.Ext(base))+".json")
		if fileutil.FileExists(possibleJSON) {
			jsonPath = possibleJSON
		}
	}

	if jsonPath == "" {
		return nil, nil
	}

	extractor := telegram.NewDefaultExtractor()
	meta, err := extractor.ExtractFromFile(jsonPath, inputPath)
	if err != nil {
		return nil, fmt.Errorf("error processing Telegram JSON: %w", err)
	}

	if b.Flags.ChannelName != "" {
		meta.Name = b.Flags.ChannelName
	}
	if b.Flags.ChannelAt != "" {
		meta.At = b.Flags.ChannelAt
	}

	return meta, nil
}

func (b *BaseCommand) GetChannelName(meta *telegram.ChannelMetadata) string {
	if meta == nil {
		return ""
	}
	return meta.Name
}

func (b *BaseCommand) GetChannelAt(meta *telegram.ChannelMetadata) string {
	if meta == nil {
		return ""
	}
	return meta.At
}

func (b *BaseCommand) ReportStats(stats credential.ProcessingStats) {
	fmt.Fprintf(os.Stderr, "Processed %d total lines\n", stats.TotalLines)
	fmt.Fprintf(os.Stderr, "Valid credentials: %d\n", stats.ValidCredentials)
	if stats.DuplicatesFound > 0 {
		fmt.Fprintf(os.Stderr, "Duplicates removed: %d\n", stats.DuplicatesFound)
		if stats.ValidCredentials > 0 {
			duplicatePercentage := float64(stats.DuplicatesFound) / float64(stats.ValidCredentials+stats.DuplicatesFound) * 100
			fmt.Fprintf(os.Stderr, "Duplicate percentage: %.1f%%\n", duplicatePercentage)
		}
	}
}

func (b *BaseCommand) GenerateOutputPath(inputPath, outputPath, suffix string) string {
	if outputPath != "" {
		return outputPath
	}

	dir := filepath.Dir(inputPath)
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))

	if b.Flags.OutputDir != "" {
		dir = b.Flags.OutputDir
	}

	return filepath.Join(dir, base+suffix)
}

func (b *BaseCommand) GetRelativeOutputPath(inputPath, relPath, suffix string) string {
	base := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	outputRelPath := filepath.Join(filepath.Dir(relPath), base+suffix)

	if b.Flags.OutputDir != "" {
		return filepath.Join(b.Flags.OutputDir, outputRelPath)
	}

	return filepath.Join(filepath.Dir(inputPath), outputRelPath)
}

