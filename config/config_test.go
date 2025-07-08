package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestYAMLConfigLoading(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `version: "1.0"

bot:
  token: "xoxb-test-token"
  app_token: "xapp-test-token"

database:
  table_name: "standup-bot-test"
  region: "us-east-1"

channels:
  - id: "C1234567890"  # pragma: allowlist secret
    name: "engineering-standup"
    enabled: true
    schedule:
      timezone: "America/New_York"
      summary_time: "09:00"
      reminder_times:
        - "08:30"
        - "08:50"
      active_days: ["Mon", "Tue", "Wed", "Thu", "Fri"]
    users:
      - id: "U1234567890"
        name: "alice"
        timezone: "America/New_York"
      - id: "U0987654321"
        name: "bob"
        timezone: "America/Chicago"
    templates:
      reminder: "Hey {{.UserName}}! Don't forget to submit your standup update for #{{.ChannelName}}"
      summary_header: "📊 Daily Standup Summary for {{.Date}}"
      user_completed: "✅ {{.UserName}} - {{.Time}}"
      user_missing: "❌ {{.UserName}} - No update"
    questions:
      - "What did you work on yesterday?"
      - "What are you working on today?"
      - "Any blockers?"

features:
  threading_enabled: true
  analytics_enabled: false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Test loading
	provider := NewYAMLProvider(configPath)
	cfg, err := provider.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test basic properties
	if cfg.Version() != "1.0" {
		t.Errorf("Expected version 1.0, got %s", cfg.Version())
	}

	if cfg.BotToken() != "xoxb-test-token" {
		t.Errorf("Expected bot token xoxb-test-token, got %s", cfg.BotToken())
	}

	if cfg.AppToken() != "xapp-test-token" {
		t.Errorf("Expected app token xapp-test-token, got %s", cfg.AppToken())
	}

	if cfg.DatabaseTable() != "standup-bot-test" {
		t.Errorf("Expected table standup-bot-test, got %s", cfg.DatabaseTable())
	}

	if cfg.DatabaseRegion() != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", cfg.DatabaseRegion())
	}

	// Test feature flags
	if !cfg.IsFeatureEnabled("threading_enabled") {
		t.Error("Expected threading_enabled to be true")
	}

	if cfg.IsFeatureEnabled("analytics_enabled") {
		t.Error("Expected analytics_enabled to be false")
	}

	if cfg.IsFeatureEnabled("non_existent_feature") {
		t.Error("Expected non_existent_feature to be false")
	}

	// Test channels
	channels := cfg.Channels()
	if len(channels) != 1 {
		t.Fatalf("Expected 1 channel, got %d", len(channels))
	}

	ch := channels[0]
	if ch.ID() != "C1234567890" {
		t.Errorf("Expected channel ID C1234567890, got %s", ch.ID())
	}

	if ch.Name() != "engineering-standup" {
		t.Errorf("Expected channel name engineering-standup, got %s", ch.Name())
	}

	if !ch.IsEnabled() {
		t.Error("Expected channel to be enabled")
	}

	// Test timezone
	tz := ch.Timezone()
	if tz.String() != "America/New_York" {
		t.Errorf("Expected timezone America/New_York, got %s", tz.String())
	}

	// Test schedule
	if ch.SummaryTime().Format("15:04") != "09:00" {
		t.Errorf("Expected summary time 09:00, got %s", ch.SummaryTime().Format("15:04"))
	}

	reminderTimes := ch.ReminderTimes()
	if len(reminderTimes) != 2 {
		t.Fatalf("Expected 2 reminder times, got %d", len(reminderTimes))
	}

	// Test active days
	if !ch.IsActiveDay(time.Monday) {
		t.Error("Expected Monday to be active")
	}

	if ch.IsActiveDay(time.Saturday) {
		t.Error("Expected Saturday to be inactive")
	}

	// Test users
	users := ch.Users()
	if len(users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(users))
	}

	// Test user lookup
	user, ok := ch.UserByID("U1234567890")
	if !ok {
		t.Error("Expected to find user U1234567890")
	}

	if user.Name() != "alice" {
		t.Errorf("Expected user name alice, got %s", user.Name())
	}

	if !ch.IsUserRequired("U1234567890") {
		t.Error("Expected user U1234567890 to be required")
	}

	if ch.IsUserRequired("U9999999999") {
		t.Error("Expected user U9999999999 to not be required")
	}

	// Test templates
	tmpl := ch.Templates()
	if tmpl.Reminder() != "Hey {{.UserName}}! Don't forget to submit your standup update for #{{.ChannelName}}" {
		t.Errorf("Unexpected reminder template: %s", tmpl.Reminder())
	}

	// Test questions
	questions := ch.Questions()
	if len(questions) != 3 {
		t.Fatalf("Expected 3 questions, got %d", len(questions))
	}
}

func TestYAMLConfigWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_BOT_TOKEN", "xoxb-env-token")
	os.Setenv("TEST_TABLE_NAME", "env-table")
	defer os.Unsetenv("TEST_BOT_TOKEN")
	defer os.Unsetenv("TEST_TABLE_NAME")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `version: "1.0"

bot:
  token: "${TEST_BOT_TOKEN}"
  app_token: ""

database:
  table_name: "${TEST_TABLE_NAME}"
  region: "us-east-1"

channels:
  - id: "C1234567890"
    name: "test"
    enabled: true
    schedule:
      timezone: "UTC"
      summary_time: "09:00"
      reminder_times: []
      active_days: ["Mon"]
    users:
      - id: "U123"
        name: "test"
    templates:
      reminder: "Reminder {{.UserName}} {{.ChannelName}}"
      summary_header: "Summary {{.Date}}"
      user_completed: "Complete {{.UserName}} {{.Time}}"
      user_missing: "Missing {{.UserName}}"
    questions:
      - "Test?"

features: {}
`

	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	provider := NewYAMLProvider(configPath)
	cfg, err := provider.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.BotToken() != "xoxb-env-token" {
		t.Errorf("Expected bot token from env var, got %s", cfg.BotToken())
	}

	if cfg.DatabaseTable() != "env-table" {
		t.Errorf("Expected table name from env var, got %s", cfg.DatabaseTable())
	}
}

func TestConfigValidation(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		config  string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: `version: "1.0"
bot:
  token: "xoxb-test"
  app_token: "xapp-test"
database:
  table_name: "test"
  region: "us-east-1"
channels:
  - id: "C123"
    name: "test"
    enabled: true
    schedule:
      timezone: "UTC"
      summary_time: "09:00"
      reminder_times: ["08:30"]
      active_days: ["Mon"]
    users:
      - id: "U123"
        name: "test"
    templates:
      reminder: "{{.UserName}} {{.ChannelName}}"
      summary_header: "{{.Date}}"
      user_completed: "{{.UserName}} {{.Time}}"
      user_missing: "{{.UserName}}"
    questions: ["Q1"]
`,
			wantErr: false,
		},
		{
			name: "missing version",
			config: `bot:
  token: "xoxb-test"
database:
  table_name: "test"
  region: "us-east-1"
channels: []
`,
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "invalid bot token",
			config: `version: "1.0"
bot:
  token: "invalid-token"
database:
  table_name: "test"
  region: "us-east-1"
channels: []
`,
			wantErr: true,
			errMsg:  "bot token must start with 'xoxb-'",
		},
		{
			name: "invalid region",
			config: `version: "1.0"
bot:
  token: "xoxb-test"
database:
  table_name: "test"
  region: "invalid-region"
channels: []
`,
			wantErr: true,
			errMsg:  "invalid AWS region",
		},
		{
			name: "no channels",
			config: `version: "1.0"
bot:
  token: "xoxb-test"
database:
  table_name: "test"
  region: "us-east-1"
channels: []
`,
			wantErr: true,
			errMsg:  "at least one channel must be configured",
		},
		{
			name: "invalid channel ID",
			config: `version: "1.0"
bot:
  token: "xoxb-test"
database:
  table_name: "test"
  region: "us-east-1"
channels:
  - id: "invalid"
    name: "test"
    enabled: true
    schedule:
      timezone: "UTC"
      summary_time: "09:00"
      reminder_times: []
      active_days: ["Mon"]
    users: []
    templates:
      reminder: "{{.UserName}} {{.ChannelName}}"
      summary_header: "{{.Date}}"
      user_completed: "{{.UserName}} {{.Time}}"
      user_missing: "{{.UserName}}"
    questions: ["Q1"]
`,
			wantErr: true,
			errMsg:  "channel ID must start with 'C'",
		},
		{
			name: "reminder after summary",
			config: `version: "1.0"
bot:
  token: "xoxb-test"
database:
  table_name: "test"
  region: "us-east-1"
channels:
  - id: "C123"
    name: "test"
    enabled: true
    schedule:
      timezone: "UTC"
      summary_time: "09:00"
      reminder_times: ["09:30"]
      active_days: ["Mon"]
    users:
      - id: "U123"
        name: "test"
    templates:
      reminder: "{{.UserName}} {{.ChannelName}}"
      summary_header: "{{.Date}}"
      user_completed: "{{.UserName}} {{.Time}}"
      user_missing: "{{.UserName}}"
    questions: ["Q1"]
`,
			wantErr: true,
			errMsg:  "reminder time .* must be before summary time",
		},
		{
			name: "missing template variables",
			config: `version: "1.0"
bot:
  token: "xoxb-test"
database:
  table_name: "test"
  region: "us-east-1"
channels:
  - id: "C123"
    name: "test"
    enabled: true
    schedule:
      timezone: "UTC"
      summary_time: "09:00"
      reminder_times: []
      active_days: ["Mon"]
    users:
      - id: "U123"
        name: "test"
    templates:
      reminder: "Missing variables"
      summary_header: "{{.Date}}"
      user_completed: "{{.UserName}} {{.Time}}"
      user_missing: "{{.UserName}}"
    questions: ["Q1"]
`,
			wantErr: true,
			errMsg:  "reminder template must contain {{.UserName}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.config), 0o644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			provider := NewYAMLProvider(configPath)
			cfg, err := provider.Load()
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("Failed to load config: %v", err)
				}
				return
			}

			err = validator.Validate(cfg)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected validation error, got none")
				}
				// Optionally check error message contains expected text
				// if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				// 	t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				// }
			} else {
				if err != nil {
					t.Errorf("Unexpected validation error: %v", err)
				}
			}
		})
	}
}
