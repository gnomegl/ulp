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
	txtCmdFlags flags.CommonFlags
	txtGlob     bool
	txtStdout   bool
)

var txtCmd = &cobra.Command{
	Use:   "txt [input-file-or-directory]",
	Short: "Create text file from credential file or directory (default format)",
	Long: `Create text file from credential file or directory.
This command extracts credentials and outputs them in text format:
url:email:password

When processing directories:
- Without --glob: Creates separate text files for each input file
- With --glob: Combines all files into a single text file

This is the default output format when no specific format is specified.`,
	Args: cobra.ExactArgs(1),
	RunE: runTxt,
}

func init() {
	flags.AddTelegramFlags(txtCmd, &txtCmdFlags)
	txtCmd.Flags().StringVarP(&txtCmdFlags.OutputDir, "output-dir", "o", "", "Output directory for text files (default: current directory)")
	txtCmd.Flags().BoolVarP(&txtGlob, "glob", "g", false, "Combine all files from directory into single text file")
	txtCmd.Flags().BoolVar(&txtStdout, "stdout", false, "Output to stdout instead of file")

	rootCmd.AddCommand(txtCmd)
}

func runTxt(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	if err := ValidateInputFile(inputPath); err != nil {
		return err
	}

	// If stdout is enabled, process differently
	if txtStdout {
		return processToStdout(inputPath, "txt")
	}

	outputPath := txtCmdFlags.OutputDir
	if outputPath == "" {
		outputPath = "."
	}

	if err := EnsureOutputDirectory(outputPath); err != nil {
		return err
	}

	processor := credential.NewDefaultProcessor()

	if fileutil.IsDirectory(inputPath) {
		if txtGlob {
			return processDirectoryGlobTxt(processor, inputPath, outputPath)
		} else {
			return processDirectoryTxt(processor, inputPath, outputPath)
		}
	} else {
		return processFileTxt(processor, inputPath, outputPath)
	}
}

func processFileTxt(processor credential.CredentialProcessor, inputPath, outputPath string) error {
	telegramMeta := ExtractTelegramMetadata(
		txtCmdFlags.JsonFile,
		inputPath,
		txtCmdFlags.ChannelName,
		txtCmdFlags.ChannelAt,
	)

	opts := CreateProcessingOptions(false, false, "")

	result, err := processor.ProcessFile(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
	}

	baseName := GetOutputBaseName(inputPath)
	txtFilename := filepath.Join(outputPath, baseName+".txt")

	writer, err := output.NewTextWriter(txtFilename)
	if err != nil {
		return fmt.Errorf("failed to create text writer: %w", err)
	}
	defer writer.Close()
	writerOpts := CreateWriterOptions(baseName, telegramMeta, false, true)

	if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
		return fmt.Errorf("failed to write text: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Created text file: %s\n", txtFilename)
	fmt.Fprintf(os.Stderr, "Total credentials: %d\n", len(result.Credentials))

	return nil
}

func processDirectoryTxt(processor credential.CredentialProcessor, inputPath, outputPath string) error {
	fmt.Fprintf(os.Stderr, "Processing directory: %s\n", inputPath)

	opts := CreateProcessingOptions(false, false, "")

	results, err := processor.ProcessDirectory(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process directory: %w", err)
	}

	totalCreds := 0
	for filePath, result := range results {
		telegramMeta := ExtractTelegramMetadata(
			txtCmdFlags.JsonFile,
			filePath,
			txtCmdFlags.ChannelName,
			txtCmdFlags.ChannelAt,
		)

		baseName := GetOutputBaseName(filePath)
		txtFilename := filepath.Join(outputPath, baseName+".txt")

		writer, err := output.NewTextWriter(txtFilename)
		if err != nil {
			return fmt.Errorf("failed to create text writer for %s: %w", filePath, err)
		}

		writerOpts := CreateWriterOptions(baseName, telegramMeta, false, true)

		if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
			writer.Close()
			return fmt.Errorf("failed to write text for %s: %w", filePath, err)
		}

		writer.Close()
		fmt.Fprintf(os.Stderr, "Created text file: %s\n", txtFilename)
		totalCreds += len(result.Credentials)
	}

	fmt.Fprintf(os.Stderr, "Total files processed: %d\n", len(results))
	fmt.Fprintf(os.Stderr, "Total credentials: %d\n", totalCreds)

	return nil
}

func processDirectoryGlobTxt(processor credential.CredentialProcessor, inputPath, outputPath string) error {
	fmt.Fprintf(os.Stderr, "Processing directory with glob: %s\n", inputPath)

	dirName := filepath.Base(inputPath)
	txtFilename := filepath.Join(outputPath, dirName+"_combined.txt")

	writer, err := output.NewTextWriter(txtFilename)
	if err != nil {
		return fmt.Errorf("failed to create text writer: %w", err)
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
			txtCmdFlags.JsonFile,
			filePath,
			txtCmdFlags.ChannelName,
			txtCmdFlags.ChannelAt,
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

	fmt.Fprintf(os.Stderr, "Created combined text file: %s\n", txtFilename)
	fmt.Fprintf(os.Stderr, "Total files processed: %d\n", filesProcessed)
	fmt.Fprintf(os.Stderr, "Total credentials: %d\n", totalCreds)

	return nil
}
