package cmd

import (
	"github.com/gnomegl/ulp/internal/command"
	"github.com/gnomegl/ulp/pkg/credential"
	"github.com/gnomegl/ulp/pkg/fileutil"
	"github.com/spf13/cobra"
)

var (
	cleanBaseCmd command.BaseCommand
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
	inputPath, outputPath := ParseArguments(args, "_processed")

	if err := cleanBaseCmd.ValidateInput(inputPath); err != nil {
		return err
	}

	processor := credential.NewDefaultProcessor()
	opts := CreateProcessingOptions(false, false, "")

	if fileutil.IsDirectory(inputPath) {
		PrintProcessingStatus(inputPath, outputPath)
		err := ProcessDirectory(processor, inputPath, outputPath, opts, true)
		if err == nil {
			PrintCompletionStatus(outputPath)
			PrintIgnoredLinesWarning()
		}
		return err
	} else {
		PrintProcessingStatus(inputPath, outputPath)
		err := ProcessSingleFile(processor, inputPath, outputPath, opts, true)
		if err == nil {
			PrintCompletionStatus(outputPath)
			PrintIgnoredLinesWarning()
		}
		return err
	}
}
