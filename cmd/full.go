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
	outputFormat string
)

var fullCmd = &cobra.Command{
	Use:   "full [input-file]",
	Short: "Full processing - clean, dedupe, and convert to JSONL/CSV in one pass",
	Long: `Full processing - clean, dedupe, and convert to JSONL or CSV in one pass.
This is the recommended command for complete processing of credential files.
Supports both JSONL (default) and CSV output formats.`,
	Args: cobra.ExactArgs(1),
	RunE: runFull,
}

func init() {
	fullCmd.Flags().StringVarP(&jsonFile, "json-file", "j", "", "JSON metadata file (optional)")
	fullCmd.Flags().StringVarP(&channelName, "channel-name", "c", "", "Telegram channel name (optional)")
	fullCmd.Flags().StringVarP(&channelAt, "channel-at", "a", "", "Telegram channel @ handle (optional)")
	fullCmd.Flags().BoolVar(&noFreshness, "no-freshness", false, "Disable freshness scoring")
	fullCmd.Flags().BoolVarP(&split, "split", "s", false, "Enable file splitting at 100MB (default: single file)")
	fullCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory for files (defaults to input file's directory)")
	fullCmd.Flags().StringVarP(&outputFormat, "format", "f", "jsonl", "Output format: jsonl or csv (default: jsonl)")
	rootCmd.AddCommand(fullCmd)
}

func runFull(cmd *cobra.Command, args []string) error {
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
		EnableDeduplication: true, // Full processing includes deduplication
		SaveDuplicates:      false,
	}

	if fileutil.IsDirectory(inputPath) {
		return processDirectoryFull(processor, inputPath, opts)
	} else {
		return processFileFull(processor, inputPath, opts)
	}
}

func processFileFull(processor credential.CredentialProcessor, inputPath string, opts credential.ProcessingOptions) error {
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
				ChannelName:    getChannelNameFull(meta.Name),
				ChannelAt:      getChannelAtFull(meta.At),
				DatePosted:     meta.DatePosted,
				MessageContent: meta.MessageContent,
				MessageID:      meta.MessageID,
			}
		}
	}

	// Handle different output formats
	if outputFormat == "csv" {
		// Determine output filename with directory
		outputBaseName := fileutil.GetNDJSONBaseName(inputPath)
		if outputDir != "" {
			// Ensure output directory exists
			if err := fileutil.EnsureDirectoryExists(outputDir); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
			outputBaseName = filepath.Join(outputDir, filepath.Base(outputBaseName))
		}
		csvFilename := outputBaseName + ".csv"

		// Create CSV writer
		writer, err := output.NewCSVWriter(csvFilename)
		if err != nil {
			return fmt.Errorf("failed to create CSV writer: %w", err)
		}
		defer writer.Close()

		// Write credentials to CSV
		writerOpts := output.WriterOptions{
			OutputBaseName:   outputBaseName,
			TelegramMetadata: telegramMeta,
			EnableFreshness:  false, // CSV doesn't include freshness
			NoSplit:          true,  // CSV is always single file
		}

		if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
			return fmt.Errorf("failed to write CSV: %w", err)
		}

		fmt.Printf("CSV file created: %s\n", csvFilename)
	} else {
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
	}

	// Print processing statistics
	fmt.Fprintf(os.Stderr, "Processed %d total lines\n", result.Stats.TotalLines)
	fmt.Fprintf(os.Stderr, "Valid credentials: %d\n", result.Stats.ValidCredentials)
	fmt.Fprintf(os.Stderr, "Duplicates removed: %d\n", result.Stats.DuplicatesFound)

	return nil
}

func processDirectoryFull(processor credential.CredentialProcessor, inputPath string, opts credential.ProcessingOptions) error {
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
					ChannelName:    getChannelNameFull(meta.Name),
					ChannelAt:      getChannelAtFull(meta.At),
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

		// Handle different output formats
		if outputFormat == "csv" {
			csvFilename := outputBaseName + ".csv"

			// Create CSV writer
			writer, err := output.NewCSVWriter(csvFilename)
			if err != nil {
				return fmt.Errorf("failed to create CSV writer for %s: %w", filePath, err)
			}

			writerOpts := output.WriterOptions{
				OutputBaseName:   outputBaseName,
				TelegramMetadata: telegramMeta,
				EnableFreshness:  false,
				NoSplit:          true,
			}

			if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
				writer.Close()
				return fmt.Errorf("failed to write CSV for %s: %w", filePath, err)
			}

			writer.Close()
		} else {
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

		// Print processing statistics for this file
		fmt.Fprintf(os.Stderr, "Processed %d total lines, %d valid credentials, %d duplicates removed\n",
			result.Stats.TotalLines, result.Stats.ValidCredentials, result.Stats.DuplicatesFound)
	}

	fmt.Printf("Directory processing completed: %s\n", inputPath)
	if outputFormat == "csv" {
		fmt.Printf("CSV files created with .csv suffix for each processed file\n")
	} else {
		if !split {
			fmt.Printf("NDJSON files created with _ms.jsonl suffix for each processed file\n")
		} else {
			fmt.Printf("NDJSON files created with _ms_*.jsonl suffix for each processed file\n")
		}
	}

	return nil
}

func getChannelNameFull(metaName string) string {
	if channelName != "" {
		return channelName
	}
	return metaName
}

func getChannelAtFull(metaAt string) string {
	if channelAt != "" {
		return channelAt
	}
	return metaAt
}
