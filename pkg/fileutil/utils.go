package fileutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func GetDefaultOutputPath(inputPath string, suffix string) string {
	if IsDirectory(inputPath) {
		return inputPath + "_processed"
	}

	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	return filepath.Join(dir, nameWithoutExt+suffix+ext)
}

func GetNDJSONBaseName(inputPath string) string {
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	return nameWithoutExt + "_ms"
}

func WriteLinesToFile(filename string, lines []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write line: %w", err)
		}
	}

	return nil
}

func EnsureDirectoryExists(path string) error {
	return os.MkdirAll(path, 0755)
}

func GetRelativePath(basePath, fullPath string) string {
	if !strings.HasPrefix(fullPath, basePath) {
		return fullPath
	}

	relPath := strings.TrimPrefix(fullPath, basePath)
	relPath = strings.TrimPrefix(relPath, "/")

	return relPath
}

// IsBinaryFile checks if a file is likely to be binary by examining the first 512 bytes
func IsBinaryFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Read first 512 bytes (or less if file is smaller)
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, err
	}

	// Skip UTF-8 BOM if present (EF BB BF)
	start := 0
	if n >= 3 && buffer[0] == 0xEF && buffer[1] == 0xBB && buffer[2] == 0xBF {
		start = 3
	}

	// Check for null bytes (common indicator of binary files)
	for i := start; i < n; i++ {
		if buffer[i] == 0 {
			return true, nil
		}
	}

	// Count non-printable characters (excluding common whitespace)
	nonPrintable := 0
	totalChecked := 0
	for i := start; i < n; i++ {
		b := buffer[i]
		totalChecked++
		
		// Allow common whitespace characters
		if b < 32 && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		}
		// Characters above 127 are often part of valid UTF-8
		// Only count them if they're not part of a valid UTF-8 sequence
		if b > 127 {
			// Simple check: if it's a UTF-8 continuation byte (10xxxxxx), skip it
			if (b & 0xC0) != 0x80 {
				// This could be the start of a UTF-8 sequence, check if it's valid
				if (b & 0xE0) == 0xC0 || // 110xxxxx (2-byte sequence)
				   (b & 0xF0) == 0xE0 || // 1110xxxx (3-byte sequence)
				   (b & 0xF8) == 0xF0 {  // 11110xxx (4-byte sequence)
					// Likely a valid UTF-8 start byte, don't count as non-printable
				} else {
					nonPrintable++
				}
			}
		}
	}

	// If more than 30% of characters are non-printable, consider it binary
	if totalChecked > 0 && float64(nonPrintable)/float64(totalChecked) > 0.3 {
		return true, nil
	}

	return false, nil
}

// IsTextFile checks if a file is a text file (inverse of IsBinaryFile)
func IsTextFile(path string) (bool, error) {
	isBinary, err := IsBinaryFile(path)
	if err != nil {
		return false, err
	}
	return !isBinary, nil
}
