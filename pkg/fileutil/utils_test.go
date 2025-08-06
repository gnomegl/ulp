package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "text_file",
			content:  []byte("example.com:user:pass\ntest.com:user2:pass2"),
			expected: false,
		},
		{
			name:     "binary_with_nulls",
			content:  []byte("some text\x00\x00\x00binary data"),
			expected: true,
		},
		{
			name:     "high_non_printable",
			content:  []byte("\x01\x02\x03\x04\x05\x06\x07\x08\x09"),
			expected: true,
		},
		{
			name:     "utf8_text",
			content:  []byte("Hello, 世界! This is UTF-8 text."),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpFile := filepath.Join(t.TempDir(), "test_file")
			err := os.WriteFile(tmpFile, tt.content, 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Test IsBinaryFile
			result, err := IsBinaryFile(tmpFile)
			if err != nil {
				t.Fatalf("IsBinaryFile failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("IsBinaryFile() = %v, want %v", result, tt.expected)
			}

			// Test IsTextFile (should be opposite)
			textResult, err := IsTextFile(tmpFile)
			if err != nil {
				t.Fatalf("IsTextFile failed: %v", err)
			}

			if textResult != !tt.expected {
				t.Errorf("IsTextFile() = %v, want %v", textResult, !tt.expected)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	// Test existing file
	tmpFile := filepath.Join(t.TempDir(), "exists.txt")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !FileExists(tmpFile) {
		t.Error("FileExists() returned false for existing file")
	}

	// Test non-existing file
	if FileExists(filepath.Join(t.TempDir(), "not_exists.txt")) {
		t.Error("FileExists() returned true for non-existing file")
	}
}

func TestIsDirectory(t *testing.T) {
	// Test directory
	tmpDir := t.TempDir()
	if !IsDirectory(tmpDir) {
		t.Error("IsDirectory() returned false for directory")
	}

	// Test file
	tmpFile := filepath.Join(tmpDir, "file.txt")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if IsDirectory(tmpFile) {
		t.Error("IsDirectory() returned true for file")
	}
}