package credential

import (
	"testing"
)

func TestProcessLine(t *testing.T) {
	processor := NewDefaultProcessor()

	tests := []struct {
		name         string
		input        string
		expectError  bool
		expectedURL  string
		expectedUser string
		expectedPass string
	}{
		{
			name:         "Basic domain format",
			input:        "example.com:user:pass",
			expectError:  false,
			expectedURL:  "https://example.com",
			expectedUser: "user",
			expectedPass: "pass",
		},
		{
			name:         "HTTPS URL format",
			input:        "https://example.com:user:pass",
			expectError:  false,
			expectedURL:  "https://example.com",
			expectedUser: "user",
			expectedPass: "pass",
		},
		{
			name:         "WWW prefix",
			input:        "www.example.com:user:pass",
			expectError:  false,
			expectedURL:  "https://example.com",
			expectedUser: "user",
			expectedPass: "pass",
		},
		{
			name:         "Pipe separator",
			input:        "example.com|user|pass",
			expectError:  false,
			expectedURL:  "https://example.com",
			expectedUser: "user",
			expectedPass: "pass",
		},
		{
			name:         "Password with colons",
			input:        "example.com:user:pass:with:colons",
			expectError:  false,
			expectedURL:  "https://example.com",
			expectedUser: "user",
			expectedPass: "pass:with:colons",
		},
		{
			name:        "Empty line",
			input:       "",
			expectError: true,
		},
		{
			name:        "No separators",
			input:       "invalid line",
			expectError: true,
		},
		{
			name:        "Insufficient parts",
			input:       "example.com:user",
			expectError: true,
		},
		{
			name:        "Empty username",
			input:       "example.com::pass",
			expectError: true,
		},
		{
			name:        "Empty password",
			input:       "example.com:user:",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred, err := processor.ProcessLine(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cred.URL != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, cred.URL)
			}

			if cred.Username != tt.expectedUser {
				t.Errorf("Expected username %s, got %s", tt.expectedUser, cred.Username)
			}

			if cred.Password != tt.expectedPass {
				t.Errorf("Expected password %s, got %s", tt.expectedPass, cred.Password)
			}
		})
	}
}
