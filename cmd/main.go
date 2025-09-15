package cmd

import (
	"fmt"
	"os"

	"github.com/gnomegl/ulp/pkg/credential"
	"github.com/gnomegl/ulp/pkg/fileutil"
	"github.com/spf13/cobra"
)

var (
	noDedupe bool
)

var mainCmd = &cobra.Command{
	Use:   "ulp [input-file] [output-file]",
	Short: "Default command - clean and deduplicate credential files",
	Long: `Default command - clean and deduplicate credential files.
This is the main processing command that cleans and deduplicates by default.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runMain,
}

func init() {
	mainCmd.Flags().BoolVar(&noDedupe, "no-dedupe", false, "Disable deduplication (only clean)")
	mainCmd.Flags().StringVarP(&dupesFile, "dupes-file", "d", "", "Output duplicate lines to this file (implies deduplication)")
	mainCmd.Flags().StringVarP(&jsonFile, "json-file", "j", "", "JSON metadata file (optional)")
	mainCmd.Flags().StringVarP(&channelName, "channel-name", "c", "", "Telegram channel name (optional)")
	mainCmd.Flags().StringVarP(&channelAt, "channel-at", "a", "", "Telegram channel @ handle (optional)")

	rootCmd.RunE = runMain
	rootCmd.Flags().BoolVar(&noDedupe, "no-dedupe", false, "Disable deduplication (only clean)")
	rootCmd.Flags().StringVarP(&dupesFile, "dupes-file", "d", "", "Output duplicate lines to this file (implies deduplication)")
	rootCmd.Flags().StringVarP(&jsonFile, "json-file", "j", "", "JSON metadata file (optional)")
	rootCmd.Flags().StringVarP(&channelName, "channel-name", "c", "", "Telegram channel name (optional)")
	rootCmd.Flags().StringVarP(&channelAt, "channel-at", "a", "", "Telegram channel @ handle (optional)")
}

func runMain(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

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

	enableDedupe := !noDedupe || dupesFile != ""

	opts := credential.ProcessingOptions{
		EnableDeduplication: enableDedupe,
		SaveDuplicates:      dupesFile != "",
		DuplicatesFile:      dupesFile,
	}

	if fileutil.IsDirectory(inputPath) {
		if dupesFile != "" {
			fmt.Fprintf(os.Stderr, "Warning: --dupes-file option ignored when processing directories (individual dupes files created per input file)\n")
		}
		return processDirectoryMain(processor, inputPath, outputPath, opts)
	} else {
		return processFileMain(processor, inputPath, outputPath, opts)
	}
}

func processFileMain(processor credential.CredentialProcessor, inputPath, outputPath string, opts credential.ProcessingOptions) error {
	if opts.EnableDeduplication {
		fmt.Fprintf(os.Stderr, "Cleaning and deduplicating: %s -> %s\n", inputPath, outputPath)
	} else {
		fmt.Fprintf(os.Stderr, "Cleaning: %s -> %s\n", inputPath, outputPath)
	}

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

	fmt.Printf("Processed file: %s\n", outputPath)
	if opts.SaveDuplicates && opts.DuplicatesFile != "" {
		fmt.Printf("Duplicate lines saved to: %s\n", opts.DuplicatesFile)
		fmt.Printf("Total duplicates removed: %d\n", len(result.Duplicates))
	} else if opts.EnableDeduplication {
		fmt.Printf("Duplicates removed (use --dupes-file to save duplicates to a file)\n")
	}
	fmt.Printf("Lines not matching format were ignored\n")

	return nil
}

func processDirectoryMain(processor credential.CredentialProcessor, inputPath, outputPath string, opts credential.ProcessingOptions) error {
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

		var lines []string
		for _, cred := range result.Credentials {
			domain := credential.ExtractNormalizedDomain(cred.URL)
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
