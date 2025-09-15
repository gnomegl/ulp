package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gnomegl/ulp/pkg/credential"
	"github.com/gnomegl/ulp/pkg/fileutil"
	"github.com/gnomegl/ulp/pkg/output"
	"github.com/gnomegl/ulp/pkg/telegram"
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

	if err := ValidateInputFile(inputPath); err != nil {
		return err
	}

	if jsonFile == "" && fileutil.IsDirectory(inputPath) {
		extractor := telegram.NewDefaultExtractor()
		if autoJSON, err := extractor.AutoDetectJSONFile(inputPath); err == nil {
			jsonFile = autoJSON
			fmt.Fprintf(os.Stderr, "Auto-detected JSON file: %s\n", autoJSON)
		}
	}

	processor := credential.NewDefaultProcessor()
	opts := CreateProcessingOptions(true, false, "")

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

	telegramMeta := ExtractTelegramMetadata(jsonFile, inputPath, channelName, channelAt)

	outputBaseName := GetOutputBaseName(inputPath)
	effectiveOutputDir := outputDir
	if effectiveOutputDir == "" {
		effectiveOutputDir = filepath.Dir(inputPath)
	}

	if err := EnsureOutputDirectory(effectiveOutputDir); err != nil {
		return err
	}

	writerOpts := CreateWriterOptions(outputBaseName, telegramMeta, !noFreshness, !split)

	var outputFiles []string
	if outputFormat == "csv" {
		outputFiles, err = writeCSVOutput(result, effectiveOutputDir, writerOpts)
	} else {
		outputFiles, err = writeNDJSONOutput(result, effectiveOutputDir, writerOpts)
	}

	if err != nil {
		return fmt.Errorf("failed to write %s output: %w", outputFormat, err)
	}

	printStatistics(result, outputFiles, outputFormat)
	return nil
}

func processDirectoryFull(processor credential.CredentialProcessor, inputPath string, opts credential.ProcessingOptions) error {
	fmt.Fprintf(os.Stderr, "Processing directory: %s\n", inputPath)

	results, err := processor.ProcessDirectory(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process directory: %w", err)
	}

	effectiveOutputDir := outputDir
	if effectiveOutputDir == "" {
		effectiveOutputDir = inputPath + "_output"
	}

	if err := EnsureOutputDirectory(effectiveOutputDir); err != nil {
		return err
	}

	totalFiles := 0
	totalCredentials := 0
	totalDuplicates := 0

	for filePath, result := range results {
		telegramMeta := ExtractTelegramMetadata(jsonFile, filePath, channelName, channelAt)

		relPath := fileutil.GetRelativePath(inputPath, filePath)
		fileOutputDir := filepath.Join(effectiveOutputDir, filepath.Dir(relPath))

		if err := EnsureOutputDirectory(fileOutputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create directory %s: %v\n", fileOutputDir, err)
			continue
		}

		outputBaseName := GetOutputBaseName(filePath)

		writerOpts := CreateWriterOptions(outputBaseName, telegramMeta, !noFreshness, !split)

		var outputFiles []string
		if outputFormat == "csv" {
			outputFiles, err = writeCSVOutput(result, fileOutputDir, writerOpts)
		} else {
			outputFiles, err = writeNDJSONOutput(result, fileOutputDir, writerOpts)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write %s output for %s: %v\n", outputFormat, filePath, err)
			continue
		}

		totalFiles++
		totalCredentials += len(result.Credentials)
		totalDuplicates += len(result.Duplicates)

		fmt.Printf("Processed %s -> %s\n", filePath, outputFiles[0])
	}

	fmt.Printf("\nDirectory processing completed:\n")
	fmt.Printf("  Files processed: %d\n", totalFiles)
	fmt.Printf("  Total credentials: %d\n", totalCredentials)
	fmt.Printf("  Total duplicates removed: %d\n", totalDuplicates)
	fmt.Printf("  Output format: %s\n", outputFormat)
	fmt.Printf("  Output directory: %s\n", effectiveOutputDir)

	return nil
}

func writeCSVOutput(result *credential.ProcessingResult, outputDir string, writerOpts output.WriterOptions) ([]string, error) {
	outputFile := filepath.Join(outputDir, writerOpts.OutputBaseName+"_ms.csv")
	writer, err := output.NewCSVWriter(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV writer: %w", err)
	}

	if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
		return nil, fmt.Errorf("failed to write credentials: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close CSV writer: %w", err)
	}

	return []string{outputFile}, nil
}

func writeNDJSONOutput(result *credential.ProcessingResult, outputDir string, writerOpts output.WriterOptions) ([]string, error) {
	writerOpts.OutputBaseName = filepath.Join(outputDir, writerOpts.OutputBaseName)

	writer := output.NewNDJSONWriter(writerOpts.MaxFileSize)

	if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
		return nil, fmt.Errorf("failed to write credentials: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close NDJSON writer: %w", err)
	}

	var outputFiles []string
	if writerOpts.NoSplit {
		outputFiles = append(outputFiles, writerOpts.OutputBaseName+"_ms.jsonl")
	} else {
		outputFiles = append(outputFiles, writerOpts.OutputBaseName+"_ms_001.jsonl")
	}

	return outputFiles, nil
}

func printStatistics(result *credential.ProcessingResult, outputFiles []string, format string) {
	fmt.Printf("\nProcessing completed:\n")
	fmt.Printf("  Total credentials: %d\n", len(result.Credentials))
	fmt.Printf("  Duplicates removed: %d\n", len(result.Duplicates))
	fmt.Printf("  Output format: %s\n", format)

	if len(outputFiles) == 1 {
		fmt.Printf("  Output file: %s\n", outputFiles[0])
	} else {
		fmt.Printf("  Output files: %d files created\n", len(outputFiles))
		for i, file := range outputFiles {
			fmt.Printf("    [%d] %s\n", i+1, file)
		}
	}

	if noFreshness {
		fmt.Printf("  Freshness scoring: disabled\n")
	} else {
		fmt.Printf("  Freshness scoring: enabled\n")
	}
}
