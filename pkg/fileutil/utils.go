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

func IsBinaryFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, err
	}

	start := 0
	if n >= 3 && buffer[0] == 0xEF && buffer[1] == 0xBB && buffer[2] == 0xBF {
		start = 3
	}

	for i := start; i < n; i++ {
		if buffer[i] == 0 {
			return true, nil
		}
	}

	nonPrintable := 0
	totalChecked := 0
	for i := start; i < n; i++ {
		b := buffer[i]
		totalChecked++

		if b < 32 && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		}
		if b > 127 {
			if (b & 0xC0) != 0x80 {
				if (b&0xE0) == 0xC0 ||
					(b&0xF0) == 0xE0 ||
					(b&0xF8) == 0xF0 {
				} else {
					nonPrintable++
				}
			}
		}
	}

	if totalChecked > 0 && float64(nonPrintable)/float64(totalChecked) > 0.3 {
		return true, nil
	}

	return false, nil
}

func IsTextFile(path string) (bool, error) {
	isBinary, err := IsBinaryFile(path)
	if err != nil {
		return false, err
	}
	return !isBinary, nil
}
