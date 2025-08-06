package cmd

import (
	"fmt"
	"os"

	"github.com/gnomegl/ulp/pkg/credential"
	"github.com/gnomegl/ulp/pkg/fileutil"
	"github.com/spf13/cobra"
)

var (
	dupesFile string
)

var dedupeCmd = &cobra.Command{
	Use:   "dedupe [input-file] [output-file]",
	Short: "Deduplicate credential files with optional duplicate output",
	Long: `Deduplicate credential files with optional duplicate output.
Processes files or directories recursively and removes duplicate entries.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runDedupe,
}

func init() {
	dedupeCmd.Flags().StringVarP(&dupesFile, "dupes-file", "d", "", "Output duplicate lines to this file")
	rootCmd.AddCommand(dedupeCmd)
}

func runDedupe(cmd *cobra.Command, args []string) error {
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
		EnableDeduplication: true,
		SaveDuplicates:      dupesFile != "",
		DuplicatesFile:      dupesFile,
	}

	if fileutil.IsDirectory(inputPath) {
		if dupesFile != "" {
			fmt.Fprintf(os.Stderr, "Warning: --dupes-file option ignored when processing directories (individual dupes files created per input file)\n")
		}
		return processDirectoryDedupe(processor, inputPath, outputPath, opts)
	} else {
		return processFileDedupe(processor, inputPath, outputPath, opts)
	}
}

func processFileDedupe(processor credential.CredentialProcessor, inputPath, outputPath string, opts credential.ProcessingOptions) error {
	fmt.Fprintf(os.Stderr, "Deduplicating: %s -> %s\n", inputPath, outputPath)

	result, err := processor.ProcessFile(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process file: %w", err)
	}

	var lines []string
	for _, cred := range result.Credentials {
		domain := cred.URL
		if len(domain) >= 8 && domain[:8] == "https://" {
			domain = domain[8:]
		} else if len(domain) >= 7 && domain[:7] == "http://" {
			domain = domain[7:]
		}
		line := fmt.Sprintf("%s:%s:%s", domain, cred.Username, cred.Password)
		lines = append(lines, line)
	}

	if err := fileutil.WriteLinesToFile(outputPath, lines); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Deduplicated file: %s\n", outputPath)
	if opts.SaveDuplicates && opts.DuplicatesFile != "" {
		fmt.Printf("Duplicate lines saved to: %s\n", opts.DuplicatesFile)
		fmt.Printf("Total duplicates removed: %d\n", len(result.Duplicates))
	} else {
		fmt.Printf("Duplicates removed (use --dupes-file to save duplicates to a file)\n")
	}
	fmt.Printf("Lines not matching format were ignored\n")

	return nil
}

func processDirectoryDedupe(processor credential.CredentialProcessor, inputPath, outputPath string, opts credential.ProcessingOptions) error {
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
		outputFilePath := fileutil.GetDefaultOutputPath(outputPath+"/"+relPath, "_deduped")

		var lines []string
		for _, cred := range result.Credentials {
			domain := cred.URL
			if len(domain) >= 8 && domain[:8] == "https://" {
				domain = domain[8:]
			} else if len(domain) >= 7 && domain[:7] == "http://" {
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
