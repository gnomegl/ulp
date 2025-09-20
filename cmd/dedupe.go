package cmd

import (
	"fmt"
	"os"

	"github.com/gnomegl/ulp/internal/command"
	"github.com/gnomegl/ulp/internal/flags"
	"github.com/gnomegl/ulp/pkg/credential"
	"github.com/gnomegl/ulp/pkg/fileutil"
	"github.com/spf13/cobra"
)

var (
	dedupeCmdFlags flags.CommonFlags
	dedupeBaseCmd  command.BaseCommand
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
	dedupeCmd.Flags().StringVarP(&dedupeCmdFlags.DupesFile, "dupes-file", "d", "", "Output duplicate lines to this file")
	rootCmd.AddCommand(dedupeCmd)
}

func runDedupe(cmd *cobra.Command, args []string) error {
	inputPath, outputPath := ParseArguments(args, "_processed")

	dedupeBaseCmd.Flags = dedupeCmdFlags
	if err := dedupeBaseCmd.ValidateInput(inputPath); err != nil {
		return err
	}

	processor := credential.NewConcurrentProcessor(workers)
	opts := CreateProcessingOptions(
		true,
		dedupeCmdFlags.DupesFile != "",
		dedupeCmdFlags.DupesFile,
	)

	if fileutil.IsDirectory(inputPath) {
		if dedupeCmdFlags.DupesFile != "" {
			PrintDirectoryWarning()
		}
		PrintProcessingStatus(inputPath, outputPath)
		err := ProcessDirectory(processor, inputPath, outputPath, opts, false)
		if err == nil {
			PrintCompletionStatus(outputPath)
			PrintIgnoredLinesWarning()
		}
		return err
	} else {
		PrintProcessingStatus(inputPath, outputPath)
		err := ProcessSingleFile(processor, inputPath, outputPath, opts, false)
		if err == nil {
			PrintCompletionStatus(outputPath)
			if opts.SaveDuplicates && opts.DuplicatesFile != "" {
				result, _ := processor.ProcessFile(inputPath, opts)
				fmt.Fprintf(os.Stderr, "Duplicate lines saved to: %s\n", opts.DuplicatesFile)
				fmt.Fprintf(os.Stderr, "Total duplicates removed: %d\n", len(result.Duplicates))
			} else {
				fmt.Fprintf(os.Stderr, "Duplicates removed (use --dupes-file to save duplicates to a file)\n")
			}
			PrintIgnoredLinesWarning()
		}
		return err
	}
}
