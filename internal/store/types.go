package store

import (
	"time"
)

// SessionStatus represents the status of a standup session.
type SessionStatus string

// Session status constants define the lifecycle states of a standup session.
const (
	SessionPending    SessionStatus = "pending"     // Session created but not started
	SessionInProgress SessionStatus = "in_progress" // Session active and accepting responses
	SessionCompleted  SessionStatus = "completed"   // Session finished
)

// Session represents a daily standup session for a channel.
type Session struct {
	SessionID     string        `dynamodbav:"session_id"`
	ChannelID     string        `dynamodbav:"channel_id"`
	Date          string        `dynamodbav:"date"` // YYYY-MM-DD format
	Status        SessionStatus `dynamodbav:"status"`
	SummaryPosted bool          `dynamodbav:"summary_posted"`
	CreatedAt     time.Time     `dynamodbav:"created_at"`
	CompletedAt   *time.Time    `dynamodbav:"completed_at,omitempty"`
}

// UserResponse represents a user's standup response.
type UserResponse struct {
	SessionID     string            `dynamodbav:"session_id"`
	ChannelID     string            `dynamodbav:"channel_id"`
	Date          string            `dynamodbav:"date"`
	UserID        string            `dynamodbav:"user_id"`
	UserName      string            `dynamodbav:"user_name"`
	Responses     map[string]string `dynamodbav:"responses"`
	SubmittedAt   time.Time         `dynamodbav:"submitted_at"`
	ReminderCount int               `dynamodbav:"reminder_count"`
}

// Reminder represents a reminder sent to a user.
type Reminder struct {
	ChannelID string    `dynamodbav:"channel_id"`
	Date      string    `dynamodbav:"date"`
	UserID    string    `dynamodbav:"user_id"`
	Time      string    `dynamodbav:"time"` // HH:MM format
	SentAt    time.Time `dynamodbav:"sent_at"`
	MessageTS string    `dynamodbav:"message_ts"`
}

// WorkspaceConfig represents workspace-level configuration.
type WorkspaceConfig struct {
	TeamID      string    `dynamodbav:"team_id"`
	TeamName    string    `dynamodbav:"team_name"`
	BotToken    string    `dynamodbav:"bot_token"`
	AppToken    string    `dynamodbav:"app_token,omitempty"`
	InstalledAt time.Time `dynamodbav:"installed_at"`
	UpdatedAt   time.Time `dynamodbav:"updated_at"`
}

// ChannelConfig represents channel-specific standup configuration.
type ChannelConfig struct {
	TeamID      string            `dynamodbav:"team_id"`
	ChannelID   string            `dynamodbav:"channel_id"`
	ChannelName string            `dynamodbav:"channel_name"`
	Enabled     bool              `dynamodbav:"enabled"`
	Schedule    ScheduleConfig    `dynamodbav:"schedule"`
	Users       []string          `dynamodbav:"users"`
	Templates   map[string]string `dynamodbav:"templates"`
	Questions   []string          `dynamodbav:"questions"`
	UpdatedAt   time.Time         `dynamodbav:"updated_at"`
}

// ScheduleConfig represents scheduling configuration.
type ScheduleConfig struct {
	Timezone      string   `dynamodbav:"timezone"`
	SummaryTime   string   `dynamodbav:"summary_time"`   // HH:MM format
	ReminderTimes []string `dynamodbav:"reminder_times"` // HH:MM format
	ActiveDays    []string `dynamodbav:"active_days"`    // Mon, Tue, etc.
}

// DynamoDBItem represents the base structure for all DynamoDB items.
type DynamoDBItem struct {
	PK  string `dynamodbav:"PK"`
	SK  string `dynamodbav:"SK"`
	TTL *int64 `dynamodbav:"TTL,omitempty"`
	// GSI1 indexes for queries
	GSI1PK string `dynamodbav:"GSI1PK,omitempty"`
	GSI1SK string `dynamodbav:"GSI1SK,omitempty"`
}
