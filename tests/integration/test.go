package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCredentialLine represents a single credential line for testing
type TestCredentialLine struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Metadata struct {
		OriginalFilename string `json:"original_filename"`
		Freshness        struct {
			FreshnessScore      float64 `json:"freshness_score"`
			FreshnessCategory   string  `json:"freshness_category"`
			DuplicatePercentage float64 `json:"duplicate_percentage"`
		} `json:"freshness"`
	} `json:"metadata"`
}

// Test data for various scenarios
var testCases = []struct {
	name        string
	input       []string
	expected    int // expected number of valid credentials
	description string
}{
	{
		name: "basic_credentials",
		input: []string{
			"example.com:user1:pass1",
			"test.com:user2:pass2",
			"demo.com:user3:pass3",
		},
		expected:    3,
		description: "Basic credential parsing",
	},
	{
		name: "duplicate_handling",
		input: []string{
			"example.com:user1:pass1",
			"example.com:user1:pass1",
			"example.com:user2:pass2",
		},
		expected:    2,
		description: "Duplicate detection and removal",
	},
	{
		name: "url_normalization",
		input: []string{
			"https://example.com:user1:pass1",
			"www.example.com:user1:pass1",
			"example.com:user1:pass1",
		},
		expected:    1,
		description: "URL normalization (should detect as duplicates)",
	},
	{
		name: "pipe_separator",
		input: []string{
			"example.com|user1|pass1",
			"test.com|user2|pass2",
		},
		expected:    2,
		description: "Pipe separator support",
	},
	{
		name: "complex_passwords",
		input: []string{
			"example.com:user1:pass:with:colons",
			"test.com:user2:p@$$w0rd!",
			"demo.com:user3:pass|with|pipes",
		},
		expected:    3,
		description: "Complex password handling",
	},
	{
		name: "invalid_lines",
		input: []string{
			"",
			"invalid",
			"example.com:user",
			"example.com::pass",
			"example.com:user:",
			"valid.com:user:pass",
		},
		expected:    1,
		description: "Invalid line filtering",
	},
	{
		name: "mixed_formats",
		input: []string{
			"https://secure.example.com/login:admin:secret123",
			"ftp://files.test.com:ftpuser:ftppass",
			"mail.demo.org:email@user.com:emailpass",
			"192.168.1.1:root:admin",
		},
		expected:    4,
		description: "Mixed URL formats and protocols",
	},
	{
		name: "freshness_scoring",
		input: []string{
			"site1.com:user1:pass1",
			"site2.com:user2:pass2",
			"site3.com:user3:pass3",
			"site1.com:user1:pass1", // duplicate
			"site2.com:user2:pass2", // duplicate
		},
		expected:    3,
		description: "Freshness scoring with duplicates",
	},
}

func TestMain(m *testing.M) {
	// Build the Go binary before running tests
	cmd := exec.Command("go", "build", "-o", "ulp-go", ".")
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build ulp-go: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	exitCode := m.Run()

	// Cleanup
	os.Remove("ulp-go")
	os.Exit(exitCode)
}

func TestBasicFunctionality(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temporary input file
			inputFile := createTempFile(t, tc.input)
			defer os.Remove(inputFile)

			// Create temporary output file
			outputFile := filepath.Join(t.TempDir(), "output.txt")

			// Run the command
			cmd := exec.Command("../../ulp-go", "clean", inputFile, outputFile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Command failed: %v\nOutput: %s", err, output)
			}

			// Verify output file exists
			if _, err := os.Stat(outputFile); err != nil {
				t.Fatalf("Output file not created: %v", err)
			}

			// Read and verify output
			data, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			actualCount := 0
			for _, line := range lines {
				if line != "" {
					actualCount++
				}
			}

			if actualCount != tc.expected {
				t.Errorf("Expected %d valid credentials, got %d", tc.expected, actualCount)
			}
		})
	}
}

func TestJSONLOutput(t *testing.T) {
	// Test JSONL generation
	input := []string{
		"example.com:user1:pass1",
		"test.com:user2:pass2",
		"example.com:user1:pass1", // duplicate
	}

	inputFile := createTempFile(t, input)
	defer os.Remove(inputFile)

	// Run jsonl command
	cmd := exec.Command("../../ulp-go", "jsonl", inputFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("JSONL command failed: %v\nOutput: %s", err, output)
	}

	// Find the generated JSONL file
	baseDir := filepath.Dir(inputFile)
	baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	jsonlPattern := filepath.Join(baseDir, baseName+"_ms_*.jsonl")
	
	matches, err := filepath.Glob(jsonlPattern)
	if err != nil || len(matches) == 0 {
		t.Fatalf("No JSONL file found matching pattern: %s", jsonlPattern)
	}

	// Read and verify JSONL content
	jsonlData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("Failed to read JSONL file: %v", err)
	}
	defer os.Remove(matches[0])

	lines := strings.Split(strings.TrimSpace(string(jsonlData)), "\n")
	if len(lines) != 2 { // Should have 2 unique credentials
		t.Errorf("Expected 2 JSONL lines, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var cred TestCredentialLine
		if err := json.Unmarshal([]byte(line), &cred); err != nil {
			t.Errorf("Invalid JSON on line %d: %v", i+1, err)
		}

		// Verify freshness score exists
		if cred.Metadata.Freshness.FreshnessScore == 0 {
			t.Errorf("Missing freshness score on line %d", i+1)
		}
	}
}

func TestDirectoryProcessing(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	
	// Create subdirectories and files
	subDir1 := filepath.Join(tempDir, "subdir1")
	subDir2 := filepath.Join(tempDir, "subdir2")
	os.MkdirAll(subDir1, 0755)
	os.MkdirAll(subDir2, 0755)

	// Create test files
	createFileWithContent(t, filepath.Join(subDir1, "creds1.txt"), []string{
		"site1.com:user1:pass1",
		"site2.com:user2:pass2",
	})
	createFileWithContent(t, filepath.Join(subDir2, "creds2.txt"), []string{
		"site3.com:user3:pass3",
		"site4.com:user4:pass4",
	})

	// Create output directory
	outputDir := filepath.Join(t.TempDir(), "output")

	// Run directory processing
	cmd := exec.Command("../../ulp-go", "full", tempDir, "--output-dir", outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Directory processing failed: %v\nOutput: %s", err, output)
	}

	// Verify output structure
	expectedFiles := []string{
		filepath.Join(outputDir, "subdir1", "creds1_cleaned.txt"),
		filepath.Join(outputDir, "subdir2", "creds2_cleaned.txt"),
	}

	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); err != nil {
			t.Errorf("Expected output file not found: %s", expectedFile)
		}
	}
}

func TestFreshnessScoring(t *testing.T) {
	// Test freshness scoring accuracy
	testData := []struct {
		name             string
		duplicatePercent float64
		expectedScore    float64
		expectedCategory string
	}{
		{"excellent", 0.02, 5.0, "excellent"},
		{"good", 0.10, 4.0, "good"},
		{"fair", 0.25, 3.0, "fair"},
		{"poor", 0.45, 2.0, "poor"},
		{"stale", 0.70, 1.0, "stale"},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			// Create test data with specific duplicate percentage
			totalLines := 100
			duplicates := int(float64(totalLines) * td.duplicatePercent)
			
			var input []string
			for i := 0; i < totalLines-duplicates; i++ {
				input = append(input, fmt.Sprintf("site%d.com:user%d:pass%d", i, i, i))
			}
			// Add duplicates
			for i := 0; i < duplicates; i++ {
				input = append(input, "duplicate.com:user:pass")
			}

			inputFile := createTempFile(t, input)
			defer os.Remove(inputFile)

			// Run jsonl command to get freshness scoring
			cmd := exec.Command("../../ulp-go", "jsonl", inputFile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Command failed: %v\nOutput: %s", err, output)
			}

			// Find and read JSONL file
			baseDir := filepath.Dir(inputFile)
			baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
			jsonlPattern := filepath.Join(baseDir, baseName+"_ms_*.jsonl")
			
			matches, err := filepath.Glob(jsonlPattern)
			if err != nil || len(matches) == 0 {
				t.Fatalf("No JSONL file found")
			}

			jsonlData, err := os.ReadFile(matches[0])
			if err != nil {
				t.Fatalf("Failed to read JSONL: %v", err)
			}
			defer os.Remove(matches[0])

			// Check first line for freshness score
			lines := strings.Split(string(jsonlData), "\n")
			if len(lines) > 0 && lines[0] != "" {
				var cred TestCredentialLine
				if err := json.Unmarshal([]byte(lines[0]), &cred); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}

				// Allow some tolerance in score comparison
				scoreDiff := cred.Metadata.Freshness.FreshnessScore - td.expectedScore
				if scoreDiff < -0.5 || scoreDiff > 0.5 {
					t.Errorf("Expected score ~%.1f, got %.1f", td.expectedScore, cred.Metadata.Freshness.FreshnessScore)
				}

				if cred.Metadata.Freshness.FreshnessCategory != td.expectedCategory {
					t.Errorf("Expected category %s, got %s", td.expectedCategory, cred.Metadata.Freshness.FreshnessCategory)
				}
			}
		})
	}
}

func TestTelegramMetadata(t *testing.T) {
	// Create test Telegram JSON
	telegramData := `{
		"name": "TestChannel",
		"id": 123456,
		"messages": [
			{
				"id": 1001,
				"date": "2024-01-01T12:00:00",
				"text": "Check out these credentials:\nexample.com:user1:pass1\ntest.com:user2:pass2"
			}
		]
	}`

	jsonFile := filepath.Join(t.TempDir(), "telegram.json")
	if err := os.WriteFile(jsonFile, []byte(telegramData), 0644); err != nil {
		t.Fatalf("Failed to create Telegram JSON: %v", err)
	}

	// Create credential file
	input := []string{
		"example.com:user1:pass1",
		"test.com:user2:pass2",
	}
	inputFile := createTempFile(t, input)
	defer os.Remove(inputFile)

	// Run full command with Telegram metadata
	outputDir := t.TempDir()
	cmd := exec.Command("../../ulp-go", "full", inputFile, 
		"--json-file", jsonFile,
		"--channel-name", "TestChannel",
		"--channel-at", "@testchannel",
		"--output-dir", outputDir)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command with Telegram metadata failed: %v\nOutput: %s", err, output)
	}

	// Find and verify JSONL output
	jsonlPattern := filepath.Join(outputDir, "*_ms_*.jsonl")
	matches, err := filepath.Glob(jsonlPattern)
	if err != nil || len(matches) == 0 {
		t.Fatalf("No JSONL file found")
	}

	jsonlData, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("Failed to read JSONL: %v", err)
	}

	// Verify Telegram metadata is included
	lines := strings.Split(strings.TrimSpace(string(jsonlData)), "\n")
	for _, line := range lines {
		var cred map[string]interface{}
		if err := json.Unmarshal([]byte(line), &cred); err != nil {
			continue
		}

		metadata, ok := cred["metadata"].(map[string]interface{})
		if !ok {
			t.Error("Missing metadata in JSONL")
			continue
		}

		if metadata["telegram_channel_name"] != "TestChannel" {
			t.Error("Telegram channel name not found in metadata")
		}
		if metadata["telegram_channel_at"] != "@testchannel" {
			t.Error("Telegram channel @ not found in metadata")
		}
	}
}

func TestPerformance(t *testing.T) {
	// Test with large file
	var input []string
	for i := 0; i < 10000; i++ {
		input = append(input, fmt.Sprintf("site%d.com:user%d:pass%d", i, i, i))
	}

	inputFile := createTempFile(t, input)
	defer os.Remove(inputFile)

	start := time.Now()
	cmd := exec.Command("../../ulp-go", "clean", inputFile, filepath.Join(t.TempDir(), "output.txt"))
	if err := cmd.Run(); err != nil {
		t.Fatalf("Performance test failed: %v", err)
	}
	duration := time.Since(start)

	// Should complete within reasonable time (adjust as needed)
	if duration > 5*time.Second {
		t.Errorf("Processing took too long: %v", duration)
	}

	t.Logf("Processed 10,000 lines in %v", duration)
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing_input_file",
			args:        []string{"clean", "/nonexistent/file.txt", "output.txt"},
			expectError: true,
			errorMsg:    "input file not found",
		},
		{
			name:        "invalid_command",
			args:        []string{"invalid"},
			expectError: true,
			errorMsg:    "unknown command",
		},
		{
			name:        "missing_arguments",
			args:        []string{"clean"},
			expectError: true,
			errorMsg:    "missing required arguments",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("../../ulp-go", tc.args...)
			output, err := cmd.CombinedOutput()

			if tc.expectError && err == nil {
				t.Errorf("Expected error but command succeeded")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v\nOutput: %s", err, output)
			}
		})
	}
}

// Helper functions

func createTempFile(t *testing.T, lines []string) string {
	file, err := os.CreateTemp(t.TempDir(), "test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer file.Close()

	for _, line := range lines {
		fmt.Fprintln(file, line)
	}

	return file.Name()
}

func createFileWithContent(t *testing.T, path string, lines []string) {
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create file %s: %v", path, err)
	}
	defer file.Close()

	for _, line := range lines {
		fmt.Fprintln(file, line)
	}
}

// Test binary file detection
func TestBinaryFileDetection(t *testing.T) {
	// Create a binary file with null bytes
	binaryFile := filepath.Join(t.TempDir(), "binary_test.txt")
	file, err := os.Create(binaryFile)
	if err != nil {
		t.Fatalf("Failed to create binary test file: %v", err)
	}
	
	// Write some text followed by null bytes
	file.Write([]byte("test.com:user:pass\x00\x00\x00binary data"))
	file.Close()

	// Try to process the binary file
	outputFile := filepath.Join(t.TempDir(), "output.txt")
	cmd := exec.Command("../../ulp-go", "clean", binaryFile, outputFile)
	output, err := cmd.CombinedOutput()
	
	// Should fail with binary file error
	if err == nil {
		t.Errorf("Expected error when processing binary file, but got none")
	}
	
	if !strings.Contains(string(output), "binary file") {
		t.Errorf("Expected binary file error message, got: %s", string(output))
	}
}

// Test directory processing with mixed binary and text files
func TestDirectoryWithBinaryFiles(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a text file
	textFile := filepath.Join(tempDir, "text.txt")
	createFileWithContent(t, textFile, []string{
		"site1.com:user1:pass1",
		"site2.com:user2:pass2",
	})
	
	// Create a binary file
	binaryFile := filepath.Join(tempDir, "binary.dat")
	file, err := os.Create(binaryFile)
	if err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}
	file.Write([]byte("\x00\x01\x02\x03\x04\x05"))
	file.Close()
	
	// Process directory
	cmd := exec.Command("../../ulp-go", "jsonl", tempDir)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Fatalf("Failed to process directory: %v\nOutput: %s", err, string(output))
	}
	
	// Check that binary file was skipped
	if !strings.Contains(string(output), "Skipping binary file") {
		t.Errorf("Expected binary file skip message, got: %s", string(output))
	}
	
	// Check that text file was processed
	jsonlFile := filepath.Join(tempDir, "text_ms.jsonl")
	if _, err := os.Stat(jsonlFile); os.IsNotExist(err) {
		t.Errorf("Expected JSONL file for text.txt to be created")
	}
}

// Benchmark tests

func BenchmarkProcessing(b *testing.B) {
	// Create test data
	var input []string
	for i := 0; i < 1000; i++ {
		input = append(input, fmt.Sprintf("site%d.com:user%d:pass%d", i, i, i))
	}

	inputFile := createTempFile(&testing.T{}, input)
	defer os.Remove(inputFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputFile := filepath.Join(b.TempDir(), fmt.Sprintf("output_%d.txt", i))
		cmd := exec.Command("../../ulp-go", "clean", inputFile, outputFile)
		if err := cmd.Run(); err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkJSONLGeneration(b *testing.B) {
	// Create test data
	var input []string
	for i := 0; i < 1000; i++ {
		input = append(input, fmt.Sprintf("site%d.com:user%d:pass%d", i, i, i))
	}

	inputFile := createTempFile(&testing.T{}, input)
	defer os.Remove(inputFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("../../ulp-go", "jsonl", inputFile)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}