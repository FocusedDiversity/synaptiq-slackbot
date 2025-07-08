package store

import (
	"context"
	"time"
)

// Store defines the interface for data persistence
type Store interface {
	// Workspace operations
	SaveWorkspaceConfig(ctx context.Context, config *WorkspaceConfig) error
	GetWorkspaceConfig(ctx context.Context, teamID string) (*WorkspaceConfig, error)

	// Channel configuration operations
	SaveChannelConfig(ctx context.Context, config *ChannelConfig) error
	GetChannelConfig(ctx context.Context, teamID, channelID string) (*ChannelConfig, error)
	ListChannelConfigs(ctx context.Context, teamID string) ([]*ChannelConfig, error)
	ListActiveChannelConfigs(ctx context.Context) ([]*ChannelConfig, error)

	// Session operations
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, channelID, date string) (*Session, error)
	UpdateSessionStatus(ctx context.Context, channelID, date string, status SessionStatus) error
	MarkSummaryPosted(ctx context.Context, channelID, date string) error

	// User response operations
	SaveUserResponse(ctx context.Context, response *UserResponse) error
	GetUserResponse(ctx context.Context, channelID, date, userID string) (*UserResponse, error)
	ListUserResponses(ctx context.Context, channelID, date string) ([]*UserResponse, error)
	IncrementReminderCount(ctx context.Context, channelID, date, userID string) error

	// Reminder operations
	SaveReminder(ctx context.Context, reminder *Reminder) error
	ListReminders(ctx context.Context, channelID, date string) ([]*Reminder, error)

	// Query operations
	GetPendingSessions(ctx context.Context, currentTime time.Time) ([]*Session, error)
	GetUsersWithoutResponse(ctx context.Context, channelID, date string, userIDs []string) ([]string, error)
}

// Errors
var (
	ErrNotFound        = &StoreError{Code: "NOT_FOUND", Message: "Item not found"}
	ErrAlreadyExists   = &StoreError{Code: "ALREADY_EXISTS", Message: "Item already exists"}
	ErrInvalidInput    = &StoreError{Code: "INVALID_INPUT", Message: "Invalid input provided"}
	ErrOperationFailed = &StoreError{Code: "OPERATION_FAILED", Message: "Operation failed"}
)

// StoreError represents a store-specific error
type StoreError struct {
	Code    string
	Message string
	Err     error
}

func (e *StoreError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *StoreError) Unwrap() error {
	return e.Err
}
