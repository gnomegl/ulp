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
	fullStdout   bool
)

var fullCmd = &cobra.Command{
	Use:   "full [input-file]",
	Short: "Full processing - clean, dedupe, and convert to TXT/JSONL/CSV in one pass",
	Long: `Full processing - clean, dedupe, and convert to TXT, JSONL, or CSV in one pass.
This is the recommended command for complete processing of credential files.
Supports TXT (default), JSONL, and CSV output formats.`,
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
	fullCmd.Flags().StringVarP(&outputFormat, "format", "f", "txt", "Output format: txt, jsonl, or csv (default: txt)")
	fullCmd.Flags().BoolVar(&fullStdout, "stdout", false, "Output to stdout instead of file")
	rootCmd.AddCommand(fullCmd)
}

func runFull(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	if err := ValidateInputFile(inputPath); err != nil {
		return err
	}

	if fullStdout {
		return processToStdout(inputPath, outputFormat)
	}

	if jsonFile == "" && fileutil.IsDirectory(inputPath) {
		extractor := telegram.NewDefaultExtractor()
		if autoJSON, err := extractor.AutoDetectJSONFile(inputPath); err == nil {
			jsonFile = autoJSON
			PrintQuiet("Auto-detected JSON file: %s\n", autoJSON)
		}
	}

	processor := credential.NewConcurrentProcessor(workers)
	opts := CreateProcessingOptions(true, false, "")

	if fileutil.IsDirectory(inputPath) {
		return processDirectoryFull(processor, inputPath, opts)
	} else {
		return processFileFull(processor, inputPath, opts)
	}
}

func processFileFull(processor credential.CredentialProcessor, inputPath string, opts credential.ProcessingOptions) error {
	PrintQuiet("Processing file: %s\n", inputPath)

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
	switch outputFormat {
	case "csv":
		outputFiles, err = writeCSVOutput(result, effectiveOutputDir, writerOpts)
	case "jsonl":
		outputFiles, err = writeNDJSONOutput(result, effectiveOutputDir, writerOpts)
	default: // txt is default
		outputFiles, err = writeTextOutput(result, effectiveOutputDir, writerOpts)
	}

	if err != nil {
		return fmt.Errorf("failed to write %s output: %w", outputFormat, err)
	}

	printStatistics(result, outputFiles, outputFormat)
	return nil
}

func processDirectoryFull(processor credential.CredentialProcessor, inputPath string, opts credential.ProcessingOptions) error {
	PrintQuiet("Processing directory: %s\n", inputPath)

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
		switch outputFormat {
		case "csv":
			outputFiles, err = writeCSVOutput(result, fileOutputDir, writerOpts)
		case "jsonl":
			outputFiles, err = writeNDJSONOutput(result, fileOutputDir, writerOpts)
		default:
			outputFiles, err = writeTextOutput(result, fileOutputDir, writerOpts)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write %s output for %s: %v\n", outputFormat, filePath, err)
			continue
		}

		totalFiles++
		totalCredentials += len(result.Credentials)
		totalDuplicates += len(result.Duplicates)

		PrintQuiet("Processed %s -> %s\n", filePath, outputFiles[0])
	}

	PrintQuiet("\nDirectory processing completed:\n")
	PrintQuiet("  Files processed: %d\n", totalFiles)
	PrintQuiet("  Total credentials: %d\n", totalCredentials)
	PrintQuiet("  Total duplicates removed: %d\n", totalDuplicates)
	PrintQuiet("  Output format: %s\n", outputFormat)
	PrintQuiet("  Output directory: %s\n", effectiveOutputDir)

	return nil
}

func writeTextOutput(result *credential.ProcessingResult, outputDir string, writerOpts output.WriterOptions) ([]string, error) {
	outputFile := filepath.Join(outputDir, writerOpts.OutputBaseName+".txt")
	writer, err := output.NewTextWriter(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create text writer: %w", err)
	}

	if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
		return nil, fmt.Errorf("failed to write credentials: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close text writer: %w", err)
	}

	return []string{outputFile}, nil
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
	PrintQuiet("\nProcessing completed:\n")
	PrintQuiet("  Total credentials: %d\n", len(result.Credentials))
	PrintQuiet("  Duplicates removed: %d\n", len(result.Duplicates))
	PrintQuiet("  Output format: %s\n", format)

	if len(outputFiles) == 1 {
		PrintQuiet("  Output file: %s\n", outputFiles[0])
	} else {
		PrintQuiet("  Output files: %d files created\n", len(outputFiles))
		for i, file := range outputFiles {
			PrintQuiet("    [%d] %s\n", i+1, file)
		}
	}

	if noFreshness {
		PrintQuiet("  Freshness scoring: disabled\n")
	} else {
		PrintQuiet("  Freshness scoring: enabled\n")
	}
}
