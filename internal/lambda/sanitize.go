package lambda

import (
	"strings"
	"unicode"
)

// SanitizeLogValue sanitizes a value for safe logging by removing newlines
// and other control characters that could be used for log injection.
func SanitizeLogValue(value string) string {
	if value == "" {
		return ""
	}

	// Remove newlines, carriage returns, and other control characters
	sanitized := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, value)

	// Trim to reasonable length to prevent log flooding
	const maxLength = 200
	if len(sanitized) > maxLength {
		sanitized = sanitized[:maxLength] + "..."
	}

	return strings.TrimSpace(sanitized)
}
