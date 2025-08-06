package credential

import (
	"regexp"
	"strings"
)

type DefaultURLNormalizer struct{}

func NewDefaultURLNormalizer() *DefaultURLNormalizer {
	return &DefaultURLNormalizer{}
}

func (n *DefaultURLNormalizer) Normalize(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	// Replace pipe characters with colons
	normalized := strings.ReplaceAll(rawURL, "|", ":")

	// Remove carriage returns and newlines
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
