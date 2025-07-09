package config

import (
	"time"
)

// Config represents the bot configuration
type Config interface {
	// Version information
	Version() string

	// Bot settings
	BotToken() string
	AppToken() string

	// Database settings
	DatabaseTable() string
	DatabaseRegion() string

	// Channel configurations
	Channels() []ChannelConfig
	ChannelByID(id string) (ChannelConfig, bool)

	// Feature flags
	IsFeatureEnabled(feature string) bool

	// Reload configuration from source
	Reload() error
}

// ChannelConfig represents per-channel configuration
type ChannelConfig interface {
	ID() string
	Name() string
	IsEnabled() bool

	// Schedule settings
	Timezone() *time.Location
	SummaryTime() time.Time
	ReminderTimes() []time.Time
	IsActiveDay(day time.Weekday) bool

	// User management
	Users() []UserConfig
	UserByID(id string) (UserConfig, bool)
	IsUserRequired(userID string) bool

	// Templates
	Templates() TemplateConfig

	// Questions
	Questions() []string
}

// UserConfig represents a user configuration
type UserConfig interface {
	ID() string
	Name() string
	Timezone() *time.Location
}

// TemplateConfig represents message templates
type TemplateConfig interface {
	Reminder() string
	SummaryHeader() string
	UserCompleted() string
	UserMissing() string
}

// Provider loads configuration from a source
type Provider interface {
	Load() (Config, error)
	Watch(callback func(Config)) error
}
