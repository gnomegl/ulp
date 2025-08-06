package cmd

import (
	"fmt"
	"os"

	"github.com/gnomegl/ulp/pkg/credential"
	"github.com/gnomegl/ulp/pkg/fileutil"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean [input-file] [output-file]",
	Short: "Clean and normalize credential files by standardizing domain formats",
	Long: `Clean and normalize credential files by standardizing domain formats.
Processes files or directories recursively and normalizes URL formats.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runClean,
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	var outputPath string
	if len(args) > 1 {
		outputPath = args[1]
	} else {
		outputPath = fileutil.GetDefaultOutputPath(inputPath, "_processed")
	}

	if !fileutil.FileExists(inputPath) {
		return fmt.Errorf("input file or directory '%s' not found", inputPath)
	}

	processor := credential.NewDefaultProcessor()
	opts := credential.ProcessingOptions{
		EnableDeduplication: false, // Clean only, no deduplication
		SaveDuplicates:      false,
	}

	if fileutil.IsDirectory(inputPath) {
		return processDirectory(processor, inputPath, outputPath, opts)
	} else {
		return processFile(processor, inputPath, outputPath, opts)
	}
}

func processFile(processor credential.CredentialProcessor, inputPath, outputPath string, opts credential.ProcessingOptions) error {
	fmt.Fprintf(os.Stderr, "Cleaning: %s -> %s\n", inputPath, outputPath)

	result, err := processor.ProcessFile(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
	}

	var lines []string
	for _, cred := range result.Credentials {
		// Convert back to domain:user:pass format
		domain := cred.URL
		if domain[:8] == "https://" {
			domain = domain[8:]
		} else if domain[:7] == "http://" {
			domain = domain[7:]
		}
		line := fmt.Sprintf("%s:%s:%s", domain, cred.Username, cred.Password)
		lines = append(lines, line)
	}

	if err := fileutil.WriteLinesToFile(outputPath, lines); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Cleaned file: %s\n", outputPath)
	fmt.Printf("Lines not matching format were ignored\n")

	return nil
}

func processDirectory(processor credential.CredentialProcessor, inputPath, outputPath string, opts credential.ProcessingOptions) error {
	fmt.Fprintf(os.Stderr, "Processing directory recursively: %s -> %s\n", inputPath, outputPath)

	if err := fileutil.EnsureDirectoryExists(outputPath); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	results, err := processor.ProcessDirectory(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process directory: %w", err)
	}

	for filePath, result := range results {
		relPath := fileutil.GetRelativePath(inputPath, filePath)
		outputFilePath := fileutil.GetDefaultOutputPath(outputPath+"/"+relPath, "_cleaned")

		if err := fileutil.EnsureDirectoryExists(fileutil.GetDefaultOutputPath(outputPath, "")); err != nil {
			return fmt.Errorf("failed to create output subdirectory: %w", err)
		}

		var lines []string
		for _, cred := range result.Credentials {
			domain := cred.URL
			if domain[:8] == "https://" {
				domain = domain[8:]
			} else if domain[:7] == "http://" {
				domain = domain[7:]
			}
			line := fmt.Sprintf("%s:%s:%s", domain, cred.Username, cred.Password)
			lines = append(lines, line)
		}

		if err := fileutil.WriteLinesToFile(outputFilePath, lines); err != nil {
			return fmt.Errorf("failed to write output file %s: %w", outputFilePath, err)
		}
	}

	fmt.Printf("Directory processing completed: %s -> %s\n", inputPath, outputPath)
	fmt.Printf("Lines not matching format were ignored\n")

	return nil
}
