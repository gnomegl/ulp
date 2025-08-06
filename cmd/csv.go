package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gnome/ulp/pkg/credential"
	"github.com/gnome/ulp/pkg/fileutil"
	"github.com/gnome/ulp/pkg/output"
	"github.com/gnome/ulp/pkg/telegram"
	"github.com/spf13/cobra"
)

var (
	glob bool
)

var csvCmd = &cobra.Command{
	Use:   "csv [input-file-or-directory]",
	Short: "Create CSV file from credential file or directory",
	Long: `Create CSV file from credential file or directory.
This command extracts credentials and outputs them in CSV format with columns:
channel, username, password, domain, date

When processing directories:
- Without --glob: Creates separate CSV files for each input file
- With --glob: Combines all files into a single CSV file`,
	Args: cobra.ExactArgs(1),
	RunE: runCSV,
}

func init() {
	csvCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory for CSV files (default: current directory)")
	csvCmd.Flags().StringVarP(&jsonFile, "json-file", "j", "", "JSON metadata file (optional)")
	csvCmd.Flags().StringVarP(&channelName, "channel-name", "c", "", "Telegram channel name (optional)")
	csvCmd.Flags().StringVarP(&channelAt, "channel-at", "a", "", "Telegram channel @ handle (optional)")
	csvCmd.Flags().BoolVarP(&glob, "glob", "g", false, "Combine all files from directory into single CSV file")
	
	rootCmd.AddCommand(csvCmd)
}

func runCSV(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	if !fileutil.FileExists(inputPath) {
		return fmt.Errorf("input file or directory '%s' not found", inputPath)
	}

	// Determine output directory
	outputPath := outputDir
	if outputPath == "" {
		outputPath = "."
	}

	// Ensure output directory exists
	if err := fileutil.EnsureDirectoryExists(outputPath); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	processor := credential.NewDefaultProcessor()

	// Check if input is a directory
	if fileutil.IsDirectory(inputPath) {
		if glob {
			return processDirectoryGlobCSV(processor, inputPath, outputPath)
		} else {
			return processDirectoryCSV(processor, inputPath, outputPath)
		}
	} else {
		return processFileCSV(processor, inputPath, outputPath)
	}
}

func processFileCSV(processor credential.CredentialProcessor, inputPath, outputPath string) error {
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
				ChannelName:    meta.Name,
				ChannelAt:      meta.At,
				DatePosted:     meta.DatePosted,
				MessageContent: meta.MessageContent,
				MessageID:      meta.MessageID,
			}
			
			// Override with command line options if provided
			if channelName != "" {
				telegramMeta.ChannelName = channelName
			}
			if channelAt != "" {
				telegramMeta.ChannelAt = channelAt
			}
		}
	}

	// Process without deduplication by default (for CSV we want all entries)
	opts := credential.ProcessingOptions{
		EnableDeduplication: false,
		SaveDuplicates:      false,
	}

	result, err := processor.ProcessFile(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
	}

	// Generate output filename
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	csvFilename := filepath.Join(outputPath, baseName + ".csv")

	// Create CSV writer
	writer, err := output.NewCSVWriter(csvFilename)
	if err != nil {
		return fmt.Errorf("failed to create CSV writer: %w", err)
	}
	defer writer.Close()

	// Write credentials to CSV
	writerOpts := output.WriterOptions{
		OutputBaseName:   baseName,
		TelegramMetadata: telegramMeta,
		EnableFreshness:  false,
		NoSplit:          true,
	}

	if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Created CSV file: %s\n", csvFilename)
	fmt.Printf("Total credentials: %d\n", len(result.Credentials))

	return nil
}

func processDirectoryCSV(processor credential.CredentialProcessor, inputPath, outputPath string) error {
	fmt.Fprintf(os.Stderr, "Processing directory: %s\n", inputPath)

	// Process without deduplication by default
	opts := credential.ProcessingOptions{
		EnableDeduplication: false,
		SaveDuplicates:      false,
	}

	results, err := processor.ProcessDirectory(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process directory: %w", err)
	}

	totalCreds := 0
	for filePath, result := range results {
		// Extract Telegram metadata for this specific file if available
		var telegramMeta *output.TelegramMetadata
		if jsonFile != "" {
			extractor := telegram.NewDefaultExtractor()
			meta, err := extractor.ExtractFromFile(jsonFile, filePath)
			if err == nil {
				telegramMeta = &output.TelegramMetadata{
					ChannelID:      meta.ID,
					ChannelName:    meta.Name,
					ChannelAt:      meta.At,
					DatePosted:     meta.DatePosted,
					MessageContent: meta.MessageContent,
					MessageID:      meta.MessageID,
				}
				
				// Override with command line options if provided
				if channelName != "" {
					telegramMeta.ChannelName = channelName
				}
				if channelAt != "" {
					telegramMeta.ChannelAt = channelAt
				}
			}
		}

		// Generate output filename for this file
		baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		csvFilename := filepath.Join(outputPath, baseName + ".csv")

		// Create CSV writer
		writer, err := output.NewCSVWriter(csvFilename)
		if err != nil {
			return fmt.Errorf("failed to create CSV writer for %s: %w", filePath, err)
		}

		// Write credentials to CSV
		writerOpts := output.WriterOptions{
			OutputBaseName:   baseName,
			TelegramMetadata: telegramMeta,
			EnableFreshness:  false,
			NoSplit:          true,
		}

		if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
			writer.Close()
			return fmt.Errorf("failed to write CSV for %s: %w", filePath, err)
		}

		writer.Close()
		fmt.Fprintf(os.Stderr, "Created CSV file: %s\n", csvFilename)
		totalCreds += len(result.Credentials)
	}

	fmt.Printf("Total files processed: %d\n", len(results))
	fmt.Printf("Total credentials: %d\n", totalCreds)

	return nil
}

func processDirectoryGlobCSV(processor credential.CredentialProcessor, inputPath, outputPath string) error {
	fmt.Fprintf(os.Stderr, "Processing directory with glob: %s\n", inputPath)

	// Generate output filename for combined CSV
	dirName := filepath.Base(inputPath)
	csvFilename := filepath.Join(outputPath, dirName + "_combined.csv")

	// Create CSV writer
	writer, err := output.NewCSVWriter(csvFilename)
	if err != nil {
		return fmt.Errorf("failed to create CSV writer: %w", err)
	}
	defer writer.Close()

	// Process without deduplication by default
	opts := credential.ProcessingOptions{
		EnableDeduplication: false,
		SaveDuplicates:      false,
	}

	results, err := processor.ProcessDirectory(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process directory: %w", err)
	}

	totalCreds := 0
	filesProcessed := 0

	// Process each file and add to the combined CSV
	for filePath, result := range results {
		// Extract Telegram metadata for this specific file if available
		var telegramMeta *output.TelegramMetadata
		if jsonFile != "" {
			extractor := telegram.NewDefaultExtractor()
			meta, err := extractor.ExtractFromFile(jsonFile, filePath)
			if err == nil {
				telegramMeta = &output.TelegramMetadata{
					ChannelID:      meta.ID,
					ChannelName:    meta.Name,
					ChannelAt:      meta.At,
					DatePosted:     meta.DatePosted,
					MessageContent: meta.MessageContent,
					MessageID:      meta.MessageID,
				}
				
				// Override with command line options if provided
				if channelName != "" {
					telegramMeta.ChannelName = channelName
				}
				if channelAt != "" {
					telegramMeta.ChannelAt = channelAt
				}
			}
		}

		// Write credentials to the combined CSV
		writerOpts := output.WriterOptions{
			OutputBaseName:   filepath.Base(filePath),
			TelegramMetadata: telegramMeta,
			EnableFreshness:  false,
			NoSplit:          true,
		}

		if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
			return fmt.Errorf("failed to write credentials from %s: %w", filePath, err)
		}

		totalCreds += len(result.Credentials)
		filesProcessed++
		fmt.Fprintf(os.Stderr, "Processed: %s (%d credentials)\n", filePath, len(result.Credentials))
	}

	fmt.Fprintf(os.Stderr, "Created combined CSV file: %s\n", csvFilename)
	fmt.Printf("Total files processed: %d\n", filesProcessed)
	fmt.Printf("Total credentials: %d\n", totalCreds)

	return nil
}