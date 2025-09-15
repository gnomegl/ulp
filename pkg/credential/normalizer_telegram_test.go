package credential

import (
	"testing"
)

func TestCleanTelegramGarbage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Clean normal URL",
			input:    "example.com:user:pass",
			expected: "example.com:user:pass",
		},
		{
			name:     "Remove corrupted UTF-8 patterns",
			input:    "ÂÂÃ¢ÂÂÃ¢ÂÂÃ¢ÂÂ¢Ã¢ÂÂÃ¢ÂÂÃ¢ÂÂÃ¢ÂÂ¢ example.com:user:pass",
			expected: "example.com:user:pass",
		},
		{
			name:     "Remove emoji corruption from Telegram",
			input:    "Ã°ÂÂÂ | +70.000.000 UHQ rows",
			expected: "| +70.000.000 UHQ rows",
		},
		{
			name:     "Clean mixed corruption",
			input:    "Ã°ÂÂÂ | Source: 100% Private Fresh Logs",
			expected: "| Source: 100% Private Fresh Logs",
		},
		{
			name:     "Remove section sign corruption",
			input:    "Ã°ÂÂ§Â¹ | test data",
			expected: "| test data",
		},
		{
			name:     "Preserve valid credentials with special chars",
			input:    "site.com:admin@123:P@$$w0rd!",
			expected: "site.com:admin@123:P@$$w0rd!",
		},
		{
			name:     "Handle Android URLs",
			input:    "android://token@com.app/:user:pass",
			expected: "android://token@com.app/:user:pass",
		},
		{
			name:     "Remove emoji but keep valid text",
			input:    "🔥 example.com:user:pass 💯",
			expected: "example.com:user:pass",
		},
		{
			name:     "Complex corruption pattern",
			input:    "¢ÂÂÃ¢ÂÂ test.com:user:pass ÂÂÃ¢ÂÂ",
			expected: "test.com:user:pass",
		},
		{
			name:     "Multiple spaces cleanup",
			input:    "example.com:user:pass     extra    spaces",
			expected: "example.com:user:pass extra spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanTelegramGarbage(tt.input)
			if result != tt.expected {
				t.Errorf("cleanTelegramGarbage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeWithTelegramGarbage(t *testing.T) {
	normalizer := NewDefaultURLNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normalize with corruption removal",
			input:    "ÂÂÃ¢ÂÂ https://www.example.com:user:pass",
			expected: "example.com:user:pass",
		},
		{
			name:     "Handle corruption in middle",
			input:    "https://site.com ÂÂÃ¢ÂÂ :user:pass",
			expected: "site.com :user:pass",
		},
		{
			name:     "Telegram metadata line",
			input:    "Ã°ÂÂÂ | Source: Private",
			expected: "| Source: Private",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.Normalize(tt.input)
			if result != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
