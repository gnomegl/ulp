package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gnome/ulp/pkg/credential"
	"github.com/gnome/ulp/pkg/fileutil"
	"github.com/gnome/ulp/pkg/output"
	"github.com/gnome/ulp/pkg/telegram"
	"github.com/spf13/cobra"
)

var (
	jsonFile    string
	channelName string
	channelAt   string
	noFreshness bool
	split       bool
	outputDir   string
)

var jsonlCmd = &cobra.Command{
	Use:   "jsonl [input-file]",
	Short: "Convert cleaned credential files to NDJSON/JSONL format for Meilisearch indexing with freshness scoring",
	Long: `Convert cleaned credential files to NDJSON/JSONL format for Meilisearch indexing with freshness scoring.
Processes files or directories recursively and creates NDJSON files with metadata.`,
	Args: cobra.ExactArgs(1),
	RunE: runJSONL,
}

func init() {
	jsonlCmd.Flags().StringVarP(&jsonFile, "json-file", "j", "", "JSON metadata file (optional)")
	jsonlCmd.Flags().StringVarP(&channelName, "channel-name", "c", "", "Telegram channel name (optional)")
	jsonlCmd.Flags().StringVarP(&channelAt, "channel-at", "a", "", "Telegram channel @ handle (optional)")
	jsonlCmd.Flags().BoolVar(&noFreshness, "no-freshness", false, "Disable freshness scoring")
	jsonlCmd.Flags().BoolVarP(&split, "split", "s", false, "Enable file splitting at 100MB (default: single file)")
	jsonlCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory for JSONL files (defaults to input file's directory)")
	rootCmd.AddCommand(jsonlCmd)
}

func runJSONL(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	if !fileutil.FileExists(inputPath) {
		return fmt.Errorf("input file or directory '%s' not found", inputPath)
	}

	// Auto-detect JSON file if not provided and input is a directory
	if jsonFile == "" && fileutil.IsDirectory(inputPath) {
		extractor := telegram.NewDefaultExtractor()
		if autoJSON, err := extractor.AutoDetectJSONFile(inputPath); err == nil {
			jsonFile = autoJSON
			fmt.Fprintf(os.Stderr, "Auto-detected JSON file: %s\n", autoJSON)
		}
	}

	processor := credential.NewDefaultProcessor()
	opts := credential.ProcessingOptions{
		EnableDeduplication: true, // Enable deduplication for JSONL output
		SaveDuplicates:      false,
	}

	if fileutil.IsDirectory(inputPath) {
		return processDirectoryJSONL(processor, inputPath, opts)
	} else {
		return processFileJSONL(processor, inputPath, opts)
	}
}

func processFileJSONL(processor credential.CredentialProcessor, inputPath string, opts credential.ProcessingOptions) error {
	fmt.Fprintf(os.Stderr, "Processing file: %s\n", inputPath)

	result, err := processor.ProcessFile(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
	}

	// Extract Telegram metadata if JSON file is provided
	var telegramMeta *output.TelegramMetadata
	if jsonFile != "" {
		extractor := telegram.NewDefaultExtractor()
		meta, err := extractor.ExtractFromFile(jsonFile, inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to extract Telegram metadata: %v\n", err)
		} else {
			telegramMeta = &output.TelegramMetadata{
				ChannelID:      meta.ID,
				ChannelName:    getChannelName(meta.Name),
				ChannelAt:      getChannelAt(meta.At),
				DatePosted:     meta.DatePosted,
				MessageContent: meta.MessageContent,
				MessageID:      meta.MessageID,
			}
		}
	}

	// Create NDJSON writer
	writer := output.NewNDJSONWriter(100 * 1024 * 1024) // 100MB max file size
	defer writer.Close()

	// Determine output base name with directory
	outputBaseName := fileutil.GetNDJSONBaseName(inputPath)
	if outputDir != "" {
		// Ensure output directory exists
		if err := fileutil.EnsureDirectoryExists(outputDir); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		outputBaseName = filepath.Join(outputDir, filepath.Base(outputBaseName))
	}

	// Write credentials to NDJSON
	writerOpts := output.WriterOptions{
		MaxFileSize:      100 * 1024 * 1024,
		OutputBaseName:   outputBaseName,
		TelegramMetadata: telegramMeta,
		EnableFreshness:  !noFreshness,
		NoSplit:          !split,
	}

	if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
		return fmt.Errorf("failed to write NDJSON: %w", err)
	}

	if !split {
		fmt.Printf("NDJSON file created: %s.jsonl\n", outputBaseName)
	} else {
		fmt.Printf("NDJSON files created with base name: %s_*.jsonl\n", outputBaseName)
	}

	return nil
}

func processDirectoryJSONL(processor credential.CredentialProcessor, inputPath string, opts credential.ProcessingOptions) error {
	fmt.Fprintf(os.Stderr, "Processing directory recursively: %s\n", inputPath)

	results, err := processor.ProcessDirectory(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process directory: %w", err)
	}

	for filePath, result := range results {
		fmt.Fprintf(os.Stderr, "Processing file: %s\n", filePath)

		// Extract Telegram metadata for this specific file
		var telegramMeta *output.TelegramMetadata
		if jsonFile != "" {
			extractor := telegram.NewDefaultExtractor()
			meta, err := extractor.ExtractFromFile(jsonFile, filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to extract Telegram metadata for %s: %v\n", filePath, err)
			} else {
				telegramMeta = &output.TelegramMetadata{
					ChannelID:      meta.ID,
					ChannelName:    getChannelName(meta.Name),
					ChannelAt:      getChannelAt(meta.At),
					DatePosted:     meta.DatePosted,
					MessageContent: meta.MessageContent,
					MessageID:      meta.MessageID,
				}
			}
		}

		// Determine output base name with directory
		outputBaseName := fileutil.GetNDJSONBaseName(filePath)
		if outputDir != "" {
			// Preserve directory structure in output
			relPath := fileutil.GetRelativePath(inputPath, filePath)
			outputPath := filepath.Join(outputDir, filepath.Dir(relPath))
			
			// Ensure output directory exists
			if err := fileutil.EnsureDirectoryExists(outputPath); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
			
			outputBaseName = filepath.Join(outputPath, filepath.Base(outputBaseName))
		}

		// Create NDJSON writer for this file
		writer := output.NewNDJSONWriter(100 * 1024 * 1024)

		writerOpts := output.WriterOptions{
			MaxFileSize:      100 * 1024 * 1024,
			OutputBaseName:   outputBaseName,
			TelegramMetadata: telegramMeta,
			EnableFreshness:  !noFreshness,
			NoSplit:          !split,
		}

		if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
			writer.Close()
			return fmt.Errorf("failed to write NDJSON for %s: %w", filePath, err)
		}

		writer.Close()
	}

	fmt.Printf("Directory processing completed: %s\n", inputPath)
	if !split {
		fmt.Printf("NDJSON files created with _ms.jsonl suffix for each processed file\n")
	} else {
		fmt.Printf("NDJSON files created with _ms_*.jsonl suffix for each processed file\n")
	}

	return nil
}

func getChannelName(metaName string) string {
	if channelName != "" {
		return channelName
	}
	return metaName
}

func getChannelAt(metaAt string) string {
	if channelAt != "" {
		return channelAt
	}
	return metaAt
}
