package context

import (
	"context"
	"sync"

	"github.com/synaptiq/standup-bot/config"
)

// Key type for context values
type contextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey contextKey = "request_id"
	
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "user_id"
	
	// ChannelIDKey is the context key for channel ID
	ChannelIDKey contextKey = "channel_id"
)

// BotContext provides shared state across the application
type BotContext interface {
	// Configuration access
	Config() config.Config
	ReloadConfig() error

	// AWS service clients
	DynamoDB() DynamoDBClient
	SecretsManager() SecretsClient

	// Slack client
	SlackClient() SlackClient

	// Tracing and monitoring
	Tracer() Tracer
	Logger() Logger

	// Request-scoped data
	WithRequestID(ctx context.Context, requestID string) context.Context
	RequestID(ctx context.Context) string
	
	// User-scoped data
	WithUserID(ctx context.Context, userID string) context.Context
	UserID(ctx context.Context) string
	
	// Channel-scoped data
	WithChannelID(ctx context.Context, channelID string) context.Context
	ChannelID(ctx context.Context) string
}

// DynamoDBClient interface for DynamoDB operations
type DynamoDBClient interface {
	// Add DynamoDB methods as needed
	PutItem(ctx context.Context, item interface{}) error
	GetItem(ctx context.Context, key interface{}, item interface{}) error
	Query(ctx context.Context, params interface{}) ([]interface{}, error)
}

// SecretsClient interface for AWS Secrets Manager
type SecretsClient interface {
	GetSecret(ctx context.Context, secretID string) (string, error)
}

// SlackClient interface for Slack operations
type SlackClient interface {
	PostMessage(ctx context.Context, channelID string, message string) error
	SendDM(ctx context.Context, userID string, message string) error
	OpenModal(ctx context.Context, triggerID string, view interface{}) error
}

// Tracer interface for distributed tracing
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, func())
	AddAnnotation(ctx context.Context, key string, value interface{})
}

// Logger interface for structured logging
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, err error, fields ...Field)
}

// Field represents a structured logging field
type Field struct {
	Key   string
	Value interface{}
}

// botContext is the default implementation
type botContext struct {
	mu             sync.RWMutex
	cfg            config.Config
	configProvider config.Provider
	dynamoDB       DynamoDBClient
	secrets        SecretsClient
	slack          SlackClient
	tracer         Tracer
	logger         Logger
}

// Options for creating a new BotContext
type Options struct {
	Config         config.Config
	ConfigProvider config.Provider
	DynamoDB       DynamoDBClient
	SecretsManager SecretsClient
	SlackClient    SlackClient
	Tracer         Tracer
	Logger         Logger
}

// New creates a new bot context
func New(opts Options) (BotContext, error) {
	if opts.Config == nil {
		return nil, ErrConfigRequired
	}

	ctx := &botContext{
		cfg:            opts.Config,
		configProvider: opts.ConfigProvider,
		dynamoDB:       opts.DynamoDB,
		secrets:        opts.SecretsManager,
		slack:          opts.SlackClient,
		tracer:         opts.Tracer,
		logger:         opts.Logger,
	}

	// Use default implementations if not provided
	if ctx.logger == nil {
		ctx.logger = &defaultLogger{}
	}

	if ctx.tracer == nil {
		ctx.tracer = &noopTracer{}
	}

	return ctx, nil
}

// Config returns the current configuration
func (c *botContext) Config() config.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg
}

// ReloadConfig reloads configuration from the provider
func (c *botContext) ReloadConfig() error {
	if c.configProvider == nil {
		return ErrNoConfigProvider
	}

	newConfig, err := c.configProvider.Load()
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.cfg = newConfig
	c.mu.Unlock()

	return nil
}

// DynamoDB returns the DynamoDB client
func (c *botContext) DynamoDB() DynamoDBClient {
	return c.dynamoDB
}

// SecretsManager returns the Secrets Manager client
func (c *botContext) SecretsManager() SecretsClient {
	return c.secrets
}

// SlackClient returns the Slack client
func (c *botContext) SlackClient() SlackClient {
	return c.slack
}

// Tracer returns the tracer
func (c *botContext) Tracer() Tracer {
	return c.tracer
}

// Logger returns the logger
func (c *botContext) Logger() Logger {
	return c.logger
}

// WithRequestID adds a request ID to the context
func (c *botContext) WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// RequestID retrieves the request ID from the context
func (c *botContext) RequestID(ctx context.Context) string {
	if v := ctx.Value(RequestIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// WithUserID adds a user ID to the context
func (c *botContext) WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// UserID retrieves the user ID from the context
func (c *botContext) UserID(ctx context.Context) string {
	if v := ctx.Value(UserIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// WithChannelID adds a channel ID to the context
func (c *botContext) WithChannelID(ctx context.Context, channelID string) context.Context {
	return context.WithValue(ctx, ChannelIDKey, channelID)
}

// ChannelID retrieves the channel ID from the context
func (c *botContext) ChannelID(ctx context.Context) string {
	if v := ctx.Value(ChannelIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}