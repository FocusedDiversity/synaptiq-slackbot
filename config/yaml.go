package config

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// yamlConfig implements Config interface
type yamlConfig struct {
	mu       sync.RWMutex
	raw      *yamlSchema
	channels map[string]ChannelConfig
	features map[string]bool
}

// yamlSchema represents the YAML structure
type yamlSchema struct {
	Version  string `yaml:"version"`
	Bot      botSchema `yaml:"bot"`
	Database databaseSchema `yaml:"database"`
	Channels []channelSchema `yaml:"channels"`
	Features map[string]bool `yaml:"features"`
}

type botSchema struct {
	Token    string `yaml:"token"`
	AppToken string `yaml:"app_token"`
}

type databaseSchema struct {
	TableName string `yaml:"table_name"`
	Region    string `yaml:"region"`
}

type channelSchema struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	Enabled   bool   `yaml:"enabled"`
	Schedule  scheduleSchema `yaml:"schedule"`
	Users     []userSchema `yaml:"users"`
	Templates templateSchema `yaml:"templates"`
	Questions []string `yaml:"questions"`
}

type scheduleSchema struct {
	Timezone      string   `yaml:"timezone"`
	SummaryTime   string   `yaml:"summary_time"`
	ReminderTimes []string `yaml:"reminder_times"`
	ActiveDays    []string `yaml:"active_days"`
}

type userSchema struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Timezone string `yaml:"timezone"`
}

type templateSchema struct {
	Reminder      string `yaml:"reminder"`
	SummaryHeader string `yaml:"summary_header"`
	UserCompleted string `yaml:"user_completed"`
	UserMissing   string `yaml:"user_missing"`
}

// NewYAMLProvider creates a new YAML configuration provider
func NewYAMLProvider(path string) Provider {
	return &yamlProvider{path: path}
}

type yamlProvider struct {
	path string
}

func (p *yamlProvider) Load() (Config, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	content := os.ExpandEnv(string(data))

	var schema yamlSchema
	if err := yaml.Unmarshal([]byte(content), &schema); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	cfg := &yamlConfig{
		raw:      &schema,
		channels: make(map[string]ChannelConfig),
		features: schema.Features,
	}

	// Parse and validate channels
	for _, ch := range schema.Channels {
		channelCfg, err := parseChannelConfig(ch)
		if err != nil {
			return nil, fmt.Errorf("invalid channel config for %s: %w", ch.ID, err)
		}
		cfg.channels[ch.ID] = channelCfg
	}

	return cfg, nil
}

func (p *yamlProvider) Watch(callback func(Config)) error {
	// TODO: Implement file watching
	return fmt.Errorf("watch not implemented")
}

// Config interface implementation
func (c *yamlConfig) Version() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.raw.Version
}

func (c *yamlConfig) BotToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.raw.Bot.Token
}

func (c *yamlConfig) AppToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.raw.Bot.AppToken
}

func (c *yamlConfig) DatabaseTable() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.raw.Database.TableName
}

func (c *yamlConfig) DatabaseRegion() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.raw.Database.Region
}

func (c *yamlConfig) Channels() []ChannelConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	channels := make([]ChannelConfig, 0, len(c.channels))
	for _, ch := range c.channels {
		channels = append(channels, ch)
	}
	return channels
}

func (c *yamlConfig) ChannelByID(id string) (ChannelConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	ch, ok := c.channels[id]
	return ch, ok
}

func (c *yamlConfig) IsFeatureEnabled(feature string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	enabled, ok := c.features[feature]
	return ok && enabled
}

func (c *yamlConfig) Reload() error {
	// TODO: Implement reload logic
	return fmt.Errorf("reload not implemented")
}

// parseChannelConfig creates a ChannelConfig from schema
func parseChannelConfig(schema channelSchema) (ChannelConfig, error) {
	// Parse timezone
	tz, err := time.LoadLocation(schema.Schedule.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", schema.Schedule.Timezone, err)
	}

	// Parse summary time
	summaryTime, err := time.Parse("15:04", schema.Schedule.SummaryTime)
	if err != nil {
		return nil, fmt.Errorf("invalid summary time %s: %w", schema.Schedule.SummaryTime, err)
	}

	// Parse reminder times
	var reminderTimes []time.Time
	for _, rt := range schema.Schedule.ReminderTimes {
		t, err := time.Parse("15:04", rt)
		if err != nil {
			return nil, fmt.Errorf("invalid reminder time %s: %w", rt, err)
		}
		reminderTimes = append(reminderTimes, t)
	}

	// Parse active days
	activeDays := make(map[time.Weekday]bool)
	for _, day := range schema.Schedule.ActiveDays {
		weekday, err := parseWeekday(day)
		if err != nil {
			return nil, fmt.Errorf("invalid active day %s: %w", day, err)
		}
		activeDays[weekday] = true
	}

	// Parse users
	users := make(map[string]UserConfig)
	for _, u := range schema.Users {
		userCfg, err := parseUserConfig(u)
		if err != nil {
			return nil, fmt.Errorf("invalid user config for %s: %w", u.ID, err)
		}
		users[u.ID] = userCfg
	}

	return &channelConfig{
		id:            schema.ID,
		name:          schema.Name,
		enabled:       schema.Enabled,
		timezone:      tz,
		summaryTime:   summaryTime,
		reminderTimes: reminderTimes,
		activeDays:    activeDays,
		users:         users,
		templates:     &templateConfig{schema: schema.Templates},
		questions:     schema.Questions,
	}, nil
}

// parseUserConfig creates a UserConfig from schema
func parseUserConfig(schema userSchema) (UserConfig, error) {
	var tz *time.Location
	if schema.Timezone != "" {
		loc, err := time.LoadLocation(schema.Timezone)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %s: %w", schema.Timezone, err)
		}
		tz = loc
	}

	return &userConfig{
		id:       schema.ID,
		name:     schema.Name,
		timezone: tz,
	}, nil
}

// parseWeekday converts string to time.Weekday
func parseWeekday(day string) (time.Weekday, error) {
	switch strings.ToLower(day) {
	case "sun", "sunday":
		return time.Sunday, nil
	case "mon", "monday":
		return time.Monday, nil
	case "tue", "tuesday":
		return time.Tuesday, nil
	case "wed", "wednesday":
		return time.Wednesday, nil
	case "thu", "thursday":
		return time.Thursday, nil
	case "fri", "friday":
		return time.Friday, nil
	case "sat", "saturday":
		return time.Saturday, nil
	default:
		return 0, fmt.Errorf("unknown day: %s", day)
	}
}

// channelConfig implements ChannelConfig
type channelConfig struct {
	id            string
	name          string
	enabled       bool
	timezone      *time.Location
	summaryTime   time.Time
	reminderTimes []time.Time
	activeDays    map[time.Weekday]bool
	users         map[string]UserConfig
	templates     TemplateConfig
	questions     []string
}

func (c *channelConfig) ID() string                      { return c.id }
func (c *channelConfig) Name() string                    { return c.name }
func (c *channelConfig) IsEnabled() bool                 { return c.enabled }
func (c *channelConfig) Timezone() *time.Location        { return c.timezone }
func (c *channelConfig) SummaryTime() time.Time          { return c.summaryTime }
func (c *channelConfig) ReminderTimes() []time.Time      { return c.reminderTimes }
func (c *channelConfig) IsActiveDay(day time.Weekday) bool { return c.activeDays[day] }
func (c *channelConfig) Templates() TemplateConfig       { return c.templates }
func (c *channelConfig) Questions() []string             { return c.questions }

func (c *channelConfig) Users() []UserConfig {
	users := make([]UserConfig, 0, len(c.users))
	for _, u := range c.users {
		users = append(users, u)
	}
	return users
}

func (c *channelConfig) UserByID(id string) (UserConfig, bool) {
	u, ok := c.users[id]
	return u, ok
}

func (c *channelConfig) IsUserRequired(userID string) bool {
	_, ok := c.users[userID]
	return ok
}

// userConfig implements UserConfig
type userConfig struct {
	id       string
	name     string
	timezone *time.Location
}

func (u *userConfig) ID() string                { return u.id }
func (u *userConfig) Name() string              { return u.name }
func (u *userConfig) Timezone() *time.Location  { return u.timezone }

// templateConfig implements TemplateConfig
type templateConfig struct {
	schema templateSchema
}

func (t *templateConfig) Reminder() string      { return t.schema.Reminder }
func (t *templateConfig) SummaryHeader() string { return t.schema.SummaryHeader }
func (t *templateConfig) UserCompleted() string { return t.schema.UserCompleted }
func (t *templateConfig) UserMissing() string   { return t.schema.UserMissing }