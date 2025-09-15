package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gnomegl/ulp/pkg/credential"
	"github.com/gnomegl/ulp/pkg/fileutil"
	"github.com/gnomegl/ulp/pkg/output"
	"github.com/gnomegl/ulp/pkg/telegram"
)

type CommonProcessor struct {
	InputPath  string
	OutputPath string
	Options    credential.ProcessingOptions
}

type ProcessingResult struct {
	Credentials []credential.Credential
	Duplicates  []credential.Credential
	Stats       ProcessingStats
}

type ProcessingStats struct {
	TotalLines     int
	ProcessedLines int
	DuplicateLines int
	InvalidLines   int
	FilesProcessed int
}

func getChannelNameWithDefault(channelNameFlag, metaName string) string {
	if channelNameFlag != "" {
		return channelNameFlag
	}
	return metaName
}

func getChannelAtWithDefault(channelAtFlag, metaAt string) string {
	if channelAtFlag != "" {
		return channelAtFlag
	}
	return metaAt
}

func ExtractCredentialLines(credentials []credential.Credential, normalize bool) []string {
	var lines []string
	for _, cred := range credentials {
		domain := cred.URL
		if normalize {
			domain = credential.ExtractNormalizedDomain(cred.URL)
		} else {
			domain = stripHTTPPrefix(domain)
		}
		line := fmt.Sprintf("%s:%s:%s", domain, cred.Username, cred.Password)
		lines = append(lines, line)
	}
	return lines
}

func stripHTTPPrefix(domain string) string {
	if len(domain) >= 8 && domain[:8] == "https://" {
		return domain[8:]
	}
	if len(domain) >= 7 && domain[:7] == "http://" {
		return domain[7:]
	}
	return domain
}

func ExtractTelegramMetadata(jsonFile, inputPath, channelNameFlag, channelAtFlag string) *output.TelegramMetadata {
	if jsonFile == "" {
		return nil
	}

	extractor := telegram.NewDefaultExtractor()
	meta, err := extractor.ExtractFromFile(jsonFile, inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to extract Telegram metadata: %v\n", err)
		return nil
	}

	return &output.TelegramMetadata{
		ChannelID:      meta.ID,
		ChannelName:    getChannelNameWithDefault(channelNameFlag, meta.Name),
		ChannelAt:      getChannelAtWithDefault(channelAtFlag, meta.At),
		DatePosted:     meta.DatePosted,
		MessageContent: meta.MessageContent,
		MessageID:      meta.MessageID,
	}
}

func ValidateInputFile(inputPath string) error {
	if !fileutil.FileExists(inputPath) {
		return fmt.Errorf("input file or directory '%s' not found", inputPath)
	}
	return nil
}

func EnsureOutputDirectory(outputPath string) error {
	if err := fileutil.EnsureDirectoryExists(outputPath); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}

func CreateWriterOptions(baseName string, telegramMeta *output.TelegramMetadata, enableFreshness, noSplit bool) output.WriterOptions {
	return output.WriterOptions{
		MaxFileSize:      100 * 1024 * 1024,
		OutputBaseName:   baseName,
		TelegramMetadata: telegramMeta,
		EnableFreshness:  enableFreshness,
		NoSplit:          noSplit,
	}
}

func PrintDirectoryWarning() {
	fmt.Fprintf(os.Stderr, "Warning: --dupes-file option ignored when processing directories (individual dupes files created per input file)\n")
}

func PrintProcessingStatus(inputPath, outputPath string) {
	fmt.Fprintf(os.Stderr, "Processing: %s -> %s\n", inputPath, outputPath)
}

func PrintCompletionStatus(outputPath string) {
	fmt.Fprintf(os.Stderr, "Completed: %s\n", outputPath)
}

func PrintIgnoredLinesWarning() {
	fmt.Fprintf(os.Stderr, "Lines not matching the expected format were ignored\n")
}

func GetOutputBaseName(inputPath string) string {
	baseName := filepath.Base(inputPath)
	if ext := filepath.Ext(baseName); ext != "" {
		baseName = baseName[:len(baseName)-len(ext)]
	}
	return baseName
}

func ProcessSingleFile(processor credential.CredentialProcessor, inputPath, outputPath string, opts credential.ProcessingOptions, normalize bool) error {
	result, err := processor.ProcessFile(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process file %s: %w", inputPath, err)
	}

	lines := ExtractCredentialLines(result.Credentials, normalize)

	if err := fileutil.WriteLinesToFile(outputPath, lines); err != nil {
		return fmt.Errorf("failed to write output file %s: %w", outputPath, err)
	}

	if opts.SaveDuplicates && opts.DuplicatesFile != "" && len(result.Duplicates) > 0 {
		if err := fileutil.WriteLinesToFile(opts.DuplicatesFile, result.Duplicates); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write duplicates file %s: %v\n", opts.DuplicatesFile, err)
		}
	}

	return nil
}

func ProcessDirectory(processor credential.CredentialProcessor, inputPath, outputPath string, opts credential.ProcessingOptions, normalize bool) error {
	if err := EnsureOutputDirectory(outputPath); err != nil {
		return err
	}

	results, err := processor.ProcessDirectory(inputPath, opts)
	if err != nil {
		return fmt.Errorf("failed to process directory %s: %w", inputPath, err)
	}

	for filePath, result := range results {
		relPath := fileutil.GetRelativePath(inputPath, filePath)
		outputFilePath := filepath.Join(outputPath, relPath)

		outputDir := filepath.Dir(outputFilePath)
		if err := EnsureOutputDirectory(outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create directory %s: %v\n", outputDir, err)
			continue
		}

		lines := ExtractCredentialLines(result.Credentials, normalize)

		if err := fileutil.WriteLinesToFile(outputFilePath, lines); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write output file %s: %v\n", outputFilePath, err)
			continue
		}

		if opts.SaveDuplicates && len(result.Duplicates) > 0 {
			dupFilePath := strings.TrimSuffix(outputFilePath, filepath.Ext(outputFilePath)) + "_dupes.txt"
			if err := fileutil.WriteLinesToFile(dupFilePath, result.Duplicates); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to write duplicates file %s: %v\n", dupFilePath, err)
			}
		}
	}

	return nil
}

func ParseArguments(args []string, defaultSuffix string) (inputPath, outputPath string) {
	inputPath = args[0]

	if len(args) > 1 {
		outputPath = args[1]
	} else {
		outputPath = fileutil.GetDefaultOutputPath(inputPath, defaultSuffix)
	}

	return inputPath, outputPath
}

func CreateProcessingOptions(enableDedup, saveDupes bool, dupesFile string) credential.ProcessingOptions {
	return credential.ProcessingOptions{
		EnableDeduplication: enableDedup,
		SaveDuplicates:      saveDupes,
		DuplicatesFile:      dupesFile,
	}
}

func processToStdout(inputPath, format string) error {
	processor := credential.NewDefaultProcessor()
	opts := CreateProcessingOptions(true, false, "")

	// Create stdout writer once
	writer := output.NewStdoutWriter(format)

	if fileutil.IsDirectory(inputPath) {
		results, err := processor.ProcessDirectory(inputPath, opts)
		if err != nil {
			return fmt.Errorf("failed to process directory: %w", err)
		}

		// Process and output each file result immediately
		for filePath, result := range results {
			writerOpts := output.WriterOptions{
				OutputBaseName:  GetOutputBaseName(filePath),
				EnableFreshness: false,
			}

			if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
				return fmt.Errorf("failed to write to stdout: %w", err)
			}

			// Flush after each file to output immediately
			if err := writer.Flush(); err != nil {
				return fmt.Errorf("failed to flush stdout: %w", err)
			}
		}
	} else {
		result, err := processor.ProcessFile(inputPath, opts)
		if err != nil {
			return fmt.Errorf("failed to process file: %w", err)
		}

		writerOpts := output.WriterOptions{
			OutputBaseName:  GetOutputBaseName(inputPath),
			EnableFreshness: false,
		}

		if err := writer.WriteCredentials(result.Credentials, result.Stats, writerOpts); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
	}

	return writer.Close()
}
