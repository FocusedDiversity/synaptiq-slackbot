package validation

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// Slack ID format validation.
	userIDRegex    = regexp.MustCompile(`^[UW][A-Z0-9]{8,}$`)
	channelIDRegex = regexp.MustCompile(`^[CG][A-Z0-9]{8,}$`)
	teamIDRegex    = regexp.MustCompile(`^T[A-Z0-9]{8,}$`)

	// Maximum lengths based on Slack documentation.
	maxIDLength = 50

	// ErrInvalidUserID is returned when a Slack user ID has an invalid format.
	ErrInvalidUserID = errors.New("invalid user ID format")
	// ErrInvalidChannelID is returned when a Slack channel ID has an invalid format.
	ErrInvalidChannelID = errors.New("invalid channel ID format")
	// ErrInvalidTeamID is returned when a Slack team ID has an invalid format.
	ErrInvalidTeamID = errors.New("invalid team ID format")
	// ErrIDTooLong is returned when a Slack ID exceeds the maximum allowed length.
	ErrIDTooLong = errors.New("ID exceeds maximum length")
	// ErrEmptyID is returned when a Slack ID is empty.
	ErrEmptyID = errors.New("ID cannot be empty")
	// ErrInvalidCharacter is returned when a Slack ID contains invalid characters.
	ErrInvalidCharacter = errors.New("ID contains invalid characters")
)

// ValidateSlackID validates any Slack ID based on type.
func ValidateSlackID(id, idType string) error {
	// Check for empty
	if id == "" {
		return ErrEmptyID
	}

	// Check length
	if len(id) > maxIDLength {
		return ErrIDTooLong
	}

	// Check for dangerous characters that could break our key structure
	if strings.Contains(id, "#") || strings.Contains(id, "/") || strings.Contains(id, "\\") {
		return ErrInvalidCharacter
	}

	// Validate format based on type
	switch idType {
	case "user":
		if !userIDRegex.MatchString(id) {
			return ErrInvalidUserID
		}
	case "channel":
		if !channelIDRegex.MatchString(id) {
			return ErrInvalidChannelID
		}
	case "team":
		if !teamIDRegex.MatchString(id) {
			return ErrInvalidTeamID
		}
	default:
		return fmt.Errorf("unknown ID type: %s", idType)
	}

	return nil
}

// ValidateUserID validates a Slack user ID.
func ValidateUserID(userID string) error {
	return ValidateSlackID(userID, "user")
}

// ValidateChannelID validates a Slack channel ID.
func ValidateChannelID(channelID string) error {
	return ValidateSlackID(channelID, "channel")
}

// ValidateTeamID validates a Slack team ID.
func ValidateTeamID(teamID string) error {
	return ValidateSlackID(teamID, "team")
}

// ValidateDate validates a date string in YYYY-MM-DD format.
func ValidateDate(date string) error {
	if date == "" {
		return errors.New("date cannot be empty")
	}

	// Simple validation for YYYY-MM-DD format
	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !dateRegex.MatchString(date) {
		return errors.New("invalid date format, expected YYYY-MM-DD")
	}

	// Check for dangerous characters
	if strings.Contains(date, "#") || strings.Contains(date, "/") || strings.Contains(date, "\\") {
		return errors.New("date contains invalid characters")
	}

	return nil
}

// SanitizeForKey removes or replaces characters that could break DynamoDB keys.
func SanitizeForKey(input string) string {
	// Replace dangerous characters with safe alternatives
	sanitized := strings.ReplaceAll(input, "#", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")

	// Limit length
	if len(sanitized) > maxIDLength {
		sanitized = sanitized[:maxIDLength]
	}

	return sanitized
}
