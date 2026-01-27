package credential

import (
	"regexp"
	"strings"
)

var (
	reTelegramGarbage1 = regexp.MustCompile(`[À-Ã]+[¢Â]+|[Â¢]+[À-Ã]+|[ÀÁÂÃâ¢§¹°]+`)
	reTelegramGarbage2 = regexp.MustCompile(`[\x00-\x1F\x7F-\x9F]`)
	reTelegramGarbage3 = regexp.MustCompile(`[\x{1F000}-\x{1FFFF}]|[\x{2600}-\x{27BF}]|[\x{FE00}-\x{FE0F}]|[\x{1F900}-\x{1F9FF}]`)
	reTelegramGarbage4 = regexp.MustCompile(`[\x{0080}-\x{00BF}]{2,}`)
	reWhitespace       = regexp.MustCompile(`\s+`)
	reProtocol         = regexp.MustCompile(`^https?://(www\.)?([^/:]+)(.*?):`)
	reWww              = regexp.MustCompile(`^www\.([^/:]+)(.*?):`)
)

type DefaultURLNormalizer struct{}

func NewDefaultURLNormalizer() *DefaultURLNormalizer {
	return &DefaultURLNormalizer{}
}

func cleanTelegramGarbage(input string) string {
	result := reTelegramGarbage1.ReplaceAllString(input, "")
	result = reTelegramGarbage2.ReplaceAllString(result, "")
	result = reTelegramGarbage3.ReplaceAllString(result, "")
	result = reTelegramGarbage4.ReplaceAllString(result, "")
	result = reWhitespace.ReplaceAllString(result, " ")

	return strings.TrimSpace(result)
}

func (n *DefaultURLNormalizer) Normalize(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	normalized := cleanTelegramGarbage(rawURL)

	normalized = strings.ReplaceAll(normalized, "|", ":")

	normalized = strings.ReplaceAll(normalized, "\r", "")
	normalized = strings.ReplaceAll(normalized, "\n", "")
	normalized = strings.TrimSpace(normalized)

	if strings.HasPrefix(normalized, "android://") {
		if idx := strings.Index(normalized, "/:"); idx != -1 && idx > len("android://") {
			// Found the /: separator, everything before it is the URL
			return normalized
		}
		return normalized
	} else if strings.HasPrefix(normalized, "https://") || strings.HasPrefix(normalized, "http://") {
		// Remove protocol and www prefix, keep path
		matches := reProtocol.FindStringSubmatch(normalized)
		if len(matches) >= 4 {
			domain := matches[2]
			path := matches[3]
			rest := normalized[len(matches[0])-1:] // Everything after the last ':'
			return domain + path + rest
		}
	} else if strings.HasPrefix(normalized, "www.") {
		// Remove www prefix, keep path
		matches := reWww.FindStringSubmatch(normalized)
		if len(matches) >= 3 {
			domain := matches[1]
			path := matches[2]
			rest := normalized[len(matches[0])-1:] // Everything after the last ':'
			return domain + path + rest
		}
	}

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
