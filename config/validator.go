package config

import (
	"fmt"
	"strings"
	"time"
)

// Validator validates configuration
type Validator interface {
	Validate(cfg Config) error
}

// NewValidator creates a new configuration validator
func NewValidator() Validator {
	return &validator{}
}

type validator struct{}

func (v *validator) Validate(cfg Config) error {
	// Validate version
	if cfg.Version() == "" {
		return fmt.Errorf("configuration version is required")
	}

	// Validate bot settings
	if err := v.validateBotSettings(cfg); err != nil {
		return fmt.Errorf("bot settings validation failed: %w", err)
	}

	// Validate database settings
	if err := v.validateDatabaseSettings(cfg); err != nil {
		return fmt.Errorf("database settings validation failed: %w", err)
	}

	// Validate channels
	if err := v.validateChannels(cfg); err != nil {
		return fmt.Errorf("channel validation failed: %w", err)
	}

	return nil
}

func (v *validator) validateBotSettings(cfg Config) error {
	if cfg.BotToken() == "" {
		return fmt.Errorf("bot token is required")
	}

	if !strings.HasPrefix(cfg.BotToken(), "xoxb-") {
		return fmt.Errorf("bot token must start with 'xoxb-'")
	}

	if cfg.AppToken() != "" && !strings.HasPrefix(cfg.AppToken(), "xapp-") {
		return fmt.Errorf("app token must start with 'xapp-' when provided")
	}

	return nil
}

func (v *validator) validateDatabaseSettings(cfg Config) error {
	if cfg.DatabaseTable() == "" {
		return fmt.Errorf("database table name is required")
	}

	if cfg.DatabaseRegion() == "" {
		return fmt.Errorf("database region is required")
	}

	// Validate region format
	validRegions := map[string]bool{
		"us-east-1": true, "us-east-2": true, "us-west-1": true, "us-west-2": true,
		"eu-west-1": true, "eu-west-2": true, "eu-central-1": true,
		"ap-northeast-1": true, "ap-southeast-1": true, "ap-southeast-2": true,
	}

	if !validRegions[cfg.DatabaseRegion()] {
		return fmt.Errorf("invalid AWS region: %s", cfg.DatabaseRegion())
	}

	return nil
}

func (v *validator) validateChannels(cfg Config) error {
	channels := cfg.Channels()
	if len(channels) == 0 {
		return fmt.Errorf("at least one channel must be configured")
	}

	seenIDs := make(map[string]bool)
	for i, ch := range channels {
		// Check for duplicate IDs
		if seenIDs[ch.ID()] {
			return fmt.Errorf("duplicate channel ID: %s", ch.ID())
		}
		seenIDs[ch.ID()] = true

		// Validate channel
		if err := v.validateChannel(ch); err != nil {
			return fmt.Errorf("channel[%d] %s: %w", i, ch.ID(), err)
		}
	}

	return nil
}

func (v *validator) validateChannel(ch ChannelConfig) error {
	// Validate basic fields
	if ch.ID() == "" {
		return fmt.Errorf("channel ID is required")
	}

	if !strings.HasPrefix(ch.ID(), "C") {
		return fmt.Errorf("channel ID must start with 'C'")
	}

	if ch.Name() == "" {
		return fmt.Errorf("channel name is required")
	}

	// Validate timezone
	if ch.Timezone() == nil {
		return fmt.Errorf("timezone is required")
	}

	// Validate schedule
	if err := v.validateSchedule(ch); err != nil {
		return fmt.Errorf("schedule validation failed: %w", err)
	}

	// Validate users
	if err := v.validateUsers(ch); err != nil {
		return fmt.Errorf("user validation failed: %w", err)
	}

	// Validate templates
	if err := v.validateTemplates(ch.Templates()); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Validate questions
	if len(ch.Questions()) == 0 {
		return fmt.Errorf("at least one question is required")
	}

	return nil
}

func (v *validator) validateSchedule(ch ChannelConfig) error {
	// Check if at least one active day
	hasActiveDay := false
	for i := 0; i < 7; i++ {
		if ch.IsActiveDay(time.Weekday(i)) {
			hasActiveDay = true
			break
		}
	}
	if !hasActiveDay {
		return fmt.Errorf("at least one active day is required")
	}

	// Validate reminder times are before summary time
	summaryHour := ch.SummaryTime().Hour()
	summaryMin := ch.SummaryTime().Minute()
	
	for _, rt := range ch.ReminderTimes() {
		reminderHour := rt.Hour()
		reminderMin := rt.Minute()
		
		if reminderHour > summaryHour || (reminderHour == summaryHour && reminderMin >= summaryMin) {
			return fmt.Errorf("reminder time %02d:%02d must be before summary time %02d:%02d",
				reminderHour, reminderMin, summaryHour, summaryMin)
		}
	}

	return nil
}

func (v *validator) validateUsers(ch ChannelConfig) error {
	users := ch.Users()
	if len(users) == 0 {
		return fmt.Errorf("at least one user must be configured")
	}

	seenIDs := make(map[string]bool)
	for _, u := range users {
		// Check for duplicates
		if seenIDs[u.ID()] {
			return fmt.Errorf("duplicate user ID: %s", u.ID())
		}
		seenIDs[u.ID()] = true

		// Validate user fields
		if u.ID() == "" {
			return fmt.Errorf("user ID is required")
		}

		if !strings.HasPrefix(u.ID(), "U") {
			return fmt.Errorf("user ID must start with 'U': %s", u.ID())
		}

		if u.Name() == "" {
			return fmt.Errorf("user name is required for %s", u.ID())
		}
	}

	return nil
}

func (v *validator) validateTemplates(tmpl TemplateConfig) error {
	if tmpl.Reminder() == "" {
		return fmt.Errorf("reminder template is required")
	}

	if tmpl.SummaryHeader() == "" {
		return fmt.Errorf("summary header template is required")
	}

	if tmpl.UserCompleted() == "" {
		return fmt.Errorf("user completed template is required")
	}

	if tmpl.UserMissing() == "" {
		return fmt.Errorf("user missing template is required")
	}

	// Validate template variables
	requiredVars := map[string][]string{
		"reminder":       {"{{.UserName}}", "{{.ChannelName}}"},
		"summary_header": {"{{.Date}}"},
		"user_completed": {"{{.UserName}}", "{{.Time}}"},
		"user_missing":   {"{{.UserName}}"},
	}

	templates := map[string]string{
		"reminder":       tmpl.Reminder(),
		"summary_header": tmpl.SummaryHeader(),
		"user_completed": tmpl.UserCompleted(),
		"user_missing":   tmpl.UserMissing(),
	}

	for name, template := range templates {
		for _, required := range requiredVars[name] {
			if !strings.Contains(template, required) {
				return fmt.Errorf("%s template must contain %s", name, required)
			}
		}
	}

	return nil
}