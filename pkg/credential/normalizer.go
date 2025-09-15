package credential

import (
	"regexp"
	"strings"
)

type DefaultURLNormalizer struct{}

func NewDefaultURLNormalizer() *DefaultURLNormalizer {
	return &DefaultURLNormalizer{}
}

// cleanTelegramGarbage removes corrupted emoji and Unicode patterns from Telegram
func cleanTelegramGarbage(input string) string {
	// Smart approach: Only keep characters that make sense in credential data
	// This is much more maintainable than blacklisting specific corruption patterns

	// Define what we want to KEEP (whitelist approach):
	// - ASCII printable: 0x20-0x7E (includes letters, numbers, symbols)
	// - Common Latin extended: À-ÿ but exclude the corruption range (À-Ã, Â)
	// - Dots, slashes, colons for URLs
	// - Basic punctuation that appears in passwords

	// First pass: Remove obvious corruption patterns
	// Characters in range U+00C0-U+00C3 (À, Á, Â, Ã) are telltale signs of UTF-8 corruption
	// when they appear repeatedly or with ¢ (U+00A2)
	result := regexp.MustCompile(`[À-Ã]+[¢Â]+|[Â¢]+[À-Ã]+|[ÀÁÂÃâ¢§¹°]+`).ReplaceAllString(input, "")

	// Second pass: Remove control characters and non-printable
	result = regexp.MustCompile(`[\x00-\x1F\x7F-\x9F]`).ReplaceAllString(result, "")

	// Third pass: Remove emoji ranges (these shouldn't be in credential data)
	// Using a more comprehensive approach for emoji blocks
	result = regexp.MustCompile(`[\x{1F000}-\x{1FFFF}]|[\x{2600}-\x{27BF}]|[\x{FE00}-\x{FE0F}]|[\x{1F900}-\x{1F9FF}]`).ReplaceAllString(result, "")

	// Fourth pass: Remove any remaining suspicious Unicode sequences
	// If we see multiple non-ASCII characters in a row that aren't part of a valid domain
	result = regexp.MustCompile(`[\x{0080}-\x{00BF}]{2,}`).ReplaceAllString(result, "")

	// Clean up whitespace
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	return strings.TrimSpace(result)
}

func (n *DefaultURLNormalizer) Normalize(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	// Clean up Telegram emoji garbage and corrupted Unicode
	normalized := cleanTelegramGarbage(rawURL)

	normalized = strings.ReplaceAll(normalized, "|", ":")

	normalized = strings.ReplaceAll(normalized, "\r", "")
	normalized = strings.ReplaceAll(normalized, "\n", "")
	normalized = strings.TrimSpace(normalized)

	// Handle different URL formats
	if strings.HasPrefix(normalized, "android://") {
		// For Android URLs, we need to handle the special format:
		// android://[token]@[package]/:username:password
		// We want to keep the android://[token]@[package]/ part intact
		if idx := strings.Index(normalized, "/:"); idx != -1 && idx > len("android://") {
			// Found the /: separator, everything before it is the URL
			return normalized
		}
		return normalized
	} else if strings.HasPrefix(normalized, "https://") || strings.HasPrefix(normalized, "http://") {
		// Remove protocol and www prefix, keep path
		re := regexp.MustCompile(`^https?://(www\.)?([^/:]+)(.*?):`)
		matches := re.FindStringSubmatch(normalized)
		if len(matches) >= 4 {
			domain := matches[2]
			path := matches[3]
			rest := normalized[len(matches[0])-1:] // Everything after the last ':'
			return domain + path + rest
		}
	} else if strings.HasPrefix(normalized, "www.") {
		// Remove www prefix, keep path
		re := regexp.MustCompile(`^www\.([^/:]+)(.*?):`)
		matches := re.FindStringSubmatch(normalized)
		if len(matches) >= 3 {
			domain := matches[1]
			path := matches[2]
			rest := normalized[len(matches[0])-1:] // Everything after the last ':'
			return domain + path + rest
		}
	}

	// For domain:user:pass format, ensure it's properly formatted
	parts := strings.Split(normalized, ":")
	if len(parts) >= 3 {
		// Check if first part looks like a domain with optional path
		domainPart := parts[0]
		if strings.Contains(domainPart, ".") || strings.Contains(domainPart, "/") {
			return normalized
		}
	}

	return normalized
}

// ExtractNormalizedDomain extracts the domain part for deduplication purposes
func ExtractNormalizedDomain(url string) string {
	domain := url

	// Remove common protocols
	if len(domain) >= 8 && domain[:8] == "https://" {
		domain = domain[8:]
	} else if len(domain) >= 7 && domain[:7] == "http://" {
		domain = domain[7:]
	}

	// Remove www prefix
	if len(domain) >= 4 && domain[:4] == "www." {
		domain = domain[4:]
	}

	return domain
}
