package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gnomegl/ulp/internal/flags"
	"github.com/gnomegl/ulp/pkg/credential"
	"github.com/gnomegl/ulp/pkg/fileutil"
	"github.com/gnomegl/ulp/pkg/output"
	"github.com/spf13/cobra"
)

var (
	csvCmdFlags flags.CommonFlags
	glob        bool
	csvStdout   bool
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
	flags.AddTelegramFlags(csvCmd, &csvCmdFlags)
	csvCmd.Flags().StringVarP(&csvCmdFlags.OutputDir, "output-dir", "o", "", "Output directory for CSV files (default: current directory)")
	csvCmd.Flags().BoolVarP(&glob, "glob", "g", false, "Combine all files from directory into single CSV file")
	csvCmd.Flags().BoolVar(&csvStdout, "stdout", false, "Output to stdout instead of file")

	rootCmd.AddCommand(csvCmd)
}

func runCSV(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	if err := ValidateInputFile(inputPath); err != nil {
		return err
	}

	// If stdout is enabled, process differently
	if csvStdout {
		return processToStdout(inputPath, "csv")
	}

	outputPath := csvCmdFlags.OutputDir
	if outputPath == "" {
		outputPath = "."
	}

	if err := EnsureOutputDirectory(outputPath); err != nil {
		return err
	}

	processor := credential.NewDefaultProcessor()

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
	telegramMeta := ExtractTelegramMetadata(
		csvCmdFlags.JsonFile,
		inputPath,
		csvCmdFlags.ChannelName,
		csvCmdFlags.ChannelAt,
	)

	opts := CreateProcessingOptions(false, false, "")

	result, err := processor.ProcessFile(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
	}

	baseName := GetOutputBaseName(inputPath)
	csvFilename := filepath.Join(outputPath, baseName+".csv")

	writer, err := output.NewCSVWriter(csvFilename)
	if err != nil {
		return fmt.Errorf("failed to create CSV writer: %w", err)
	}
	defer writer.Close()
	writerOpts := CreateWriterOptions(baseName, telegramMeta, false, true)

	if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Created CSV file: %s\n", csvFilename)
	fmt.Fprintf(os.Stderr, "Total credentials: %d\n", len(result.Credentials))

	return nil
}

func processDirectoryCSV(processor credential.CredentialProcessor, inputPath, outputPath string) error {
	fmt.Fprintf(os.Stderr, "Processing directory: %s\n", inputPath)

	opts := CreateProcessingOptions(false, false, "")

	results, err := processor.ProcessDirectory(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process directory: %w", err)
	}

	totalCreds := 0
	for filePath, result := range results {
		telegramMeta := ExtractTelegramMetadata(
			csvCmdFlags.JsonFile,
			filePath,
			csvCmdFlags.ChannelName,
			csvCmdFlags.ChannelAt,
		)

		baseName := GetOutputBaseName(filePath)
		csvFilename := filepath.Join(outputPath, baseName+".csv")

		writer, err := output.NewCSVWriter(csvFilename)
		if err != nil {
			return fmt.Errorf("failed to create CSV writer for %s: %w", filePath, err)
		}

		writerOpts := CreateWriterOptions(baseName, telegramMeta, false, true)

		if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
			writer.Close()
			return fmt.Errorf("failed to write CSV for %s: %w", filePath, err)
		}

		writer.Close()
		fmt.Fprintf(os.Stderr, "Created CSV file: %s\n", csvFilename)
		totalCreds += len(result.Credentials)
	}

	fmt.Fprintf(os.Stderr, "Total files processed: %d\n", len(results))
	fmt.Fprintf(os.Stderr, "Total credentials: %d\n", totalCreds)

	return nil
}

func processDirectoryGlobCSV(processor credential.CredentialProcessor, inputPath, outputPath string) error {
	fmt.Fprintf(os.Stderr, "Processing directory with glob: %s\n", inputPath)

	dirName := filepath.Base(inputPath)
	csvFilename := filepath.Join(outputPath, dirName+"_combined.csv")

	writer, err := output.NewCSVWriter(csvFilename)
	if err != nil {
		return fmt.Errorf("failed to create CSV writer: %w", err)
	}
	defer writer.Close()

	opts := CreateProcessingOptions(false, false, "")

	results, err := processor.ProcessDirectory(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process directory: %w", err)
	}

	totalCreds := 0
	filesProcessed := 0

	for filePath, result := range results {
		telegramMeta := ExtractTelegramMetadata(
			csvCmdFlags.JsonFile,
			filePath,
			csvCmdFlags.ChannelName,
			csvCmdFlags.ChannelAt,
		)

		writerOpts := CreateWriterOptions(
			filepath.Base(filePath),
			telegramMeta,
			false,
			true,
		)

		if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
			return fmt.Errorf("failed to write credentials from %s: %w", filePath, err)
		}

		totalCreds += len(result.Credentials)
		filesProcessed++
		fmt.Fprintf(os.Stderr, "Processed: %s (%d credentials)\n", filePath, len(result.Credentials))
	}

	fmt.Fprintf(os.Stderr, "Created combined CSV file: %s\n", csvFilename)
	fmt.Fprintf(os.Stderr, "Total files processed: %d\n", filesProcessed)
	fmt.Fprintf(os.Stderr, "Total credentials: %d\n", totalCreds)

	return nil
}
