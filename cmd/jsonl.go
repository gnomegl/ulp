package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gnomegl/ulp/internal/command"
	"github.com/gnomegl/ulp/internal/flags"
	"github.com/gnomegl/ulp/pkg/credential"
	"github.com/gnomegl/ulp/pkg/fileutil"
	"github.com/gnomegl/ulp/pkg/output"
	"github.com/spf13/cobra"
)

var (
	jsonlCmdFlags flags.CommonFlags
	jsonlBaseCmd  command.BaseCommand
	jsonlStdout   bool
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
	flags.AddTelegramFlags(jsonlCmd, &jsonlCmdFlags)
	flags.AddOutputFlags(jsonlCmd, &jsonlCmdFlags)
	jsonlCmd.Flags().BoolVar(&jsonlStdout, "stdout", false, "Output to stdout instead of file")
	rootCmd.AddCommand(jsonlCmd)
}

func runJSONL(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	if err := ValidateInputFile(inputPath); err != nil {
		return err
	}

	if jsonlStdout {
		return processToStdout(inputPath, "jsonl")
	}

	if jsonlCmdFlags.JsonFile == "" && !fileutil.IsDirectory(inputPath) {
		dir := filepath.Dir(inputPath)
		base := filepath.Base(inputPath)
		possibleJSON := filepath.Join(dir, strings.TrimSuffix(base, filepath.Ext(base))+".json")
		if fileutil.FileExists(possibleJSON) {
			jsonlCmdFlags.JsonFile = possibleJSON
			fmt.Fprintf(os.Stderr, "Auto-detected JSON file: %s\n", possibleJSON)
		}
	}

	processor := credential.NewConcurrentProcessor(workers)
	opts := CreateProcessingOptions(true, false, "")

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

	telegramMeta := ExtractTelegramMetadata(
		jsonlCmdFlags.JsonFile,
		inputPath,
		jsonlCmdFlags.ChannelName,
		jsonlCmdFlags.ChannelAt,
	)

	writer := output.NewNDJSONWriter(100 * 1024 * 1024)
	defer writer.Close()

	outputBaseName := GetOutputBaseName(inputPath)
	outputBaseName = outputBaseName + "_ms"

	if jsonlCmdFlags.OutputDir != "" {
		if err := EnsureOutputDirectory(jsonlCmdFlags.OutputDir); err != nil {
			return err
		}
		outputBaseName = filepath.Join(jsonlCmdFlags.OutputDir, filepath.Base(outputBaseName))
	}

	writerOpts := CreateWriterOptions(
		outputBaseName,
		telegramMeta,
		!jsonlCmdFlags.NoFreshness,
		!jsonlCmdFlags.Split,
	)

	if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
		return fmt.Errorf("failed to write NDJSON: %w", err)
	}

	if !jsonlCmdFlags.Split {
		fmt.Fprintf(os.Stderr, "NDJSON file created: %s.jsonl\n", outputBaseName)
	} else {
		fmt.Fprintf(os.Stderr, "NDJSON files created with base name: %s_*.jsonl\n", outputBaseName)
	}

	return nil
}

func processDirectoryJSONL(processor credential.CredentialProcessor, inputPath string, opts credential.ProcessingOptions) error {
	fmt.Fprintf(os.Stderr, "\n=== Processing directory: %s ===\n", inputPath)

	results, err := processor.ProcessDirectory(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process directory: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\n=== Writing JSONL files ===\n")
	fileCount := 0
	totalCount := len(results)

	for filePath, result := range results {
		fileCount++
		fmt.Fprintf(os.Stderr, "[%d/%d] Writing JSONL for: %s", fileCount, totalCount, filepath.Base(filePath))

		telegramMeta := ExtractTelegramMetadata(
			jsonlCmdFlags.JsonFile,
			filePath,
			jsonlCmdFlags.ChannelName,
			jsonlCmdFlags.ChannelAt,
		)

		outputBaseName := GetOutputBaseName(filePath)
		outputBaseName = outputBaseName + "_ms"

		if jsonlCmdFlags.OutputDir != "" {
			relPath := fileutil.GetRelativePath(inputPath, filePath)
			outputPath := filepath.Join(jsonlCmdFlags.OutputDir, filepath.Dir(relPath))

			if err := EnsureOutputDirectory(outputPath); err != nil {
				return err
			}

			outputBaseName = filepath.Join(outputPath, filepath.Base(outputBaseName))
		}

		writer := output.NewNDJSONWriter(100 * 1024 * 1024)

		writerOpts := CreateWriterOptions(
			outputBaseName,
			telegramMeta,
			!jsonlCmdFlags.NoFreshness,
			!jsonlCmdFlags.Split,
		)

		if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
			writer.Close()
			return fmt.Errorf("failed to write NDJSON for %s: %w", filePath, err)
		}

		writer.Close()
		fmt.Fprintf(os.Stderr, " - Done\n")
	}

	fmt.Fprintf(os.Stderr, "\n=== Processing completed ===\n")
	fmt.Fprintf(os.Stderr, "Successfully processed %d files from: %s\n", totalCount, inputPath)
	if !jsonlCmdFlags.Split {
		fmt.Fprintf(os.Stderr, "NDJSON files created with _ms.jsonl suffix for each processed file\n")
	} else {
		fmt.Fprintf(os.Stderr, "NDJSON files created with _ms_*.jsonl suffix for each processed file\n")
	}

	return nil
}
