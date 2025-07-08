package context

import (
	"context"
	"errors"
	"testing"

	"github.com/synaptiq/standup-bot/config"
)

// Mock implementations for testing
type mockConfig struct {
	version string
}

func (m *mockConfig) Version() string                                    { return m.version }
func (m *mockConfig) BotToken() string                                   { return "xoxb-test" }
func (m *mockConfig) AppToken() string                                   { return "xapp-test" }
func (m *mockConfig) DatabaseTable() string                              { return "test-table" }
func (m *mockConfig) DatabaseRegion() string                             { return "us-east-1" }
func (m *mockConfig) Channels() []config.ChannelConfig                   { return nil }
func (m *mockConfig) ChannelByID(id string) (config.ChannelConfig, bool) { return nil, false }
func (m *mockConfig) IsFeatureEnabled(feature string) bool               { return false }
func (m *mockConfig) Reload() error                                      { return nil }

type mockConfigProvider struct {
	loadFunc func() (config.Config, error)
}

func (m *mockConfigProvider) Load() (config.Config, error) {
	if m.loadFunc != nil {
		return m.loadFunc()
	}
	return &mockConfig{version: "2.0"}, nil
}

func (m *mockConfigProvider) Watch(callback func(config.Config)) error {
	return nil
}

type mockDynamoDBClient struct{}

func (m *mockDynamoDBClient) PutItem(ctx context.Context, item interface{}) error { return nil }
func (m *mockDynamoDBClient) GetItem(ctx context.Context, key interface{}, item interface{}) error {
	return nil
}

func (m *mockDynamoDBClient) Query(ctx context.Context, params interface{}) ([]interface{}, error) {
	return nil, nil
}

type mockSecretsClient struct{}

func (m *mockSecretsClient) GetSecret(ctx context.Context, secretID string) (string, error) {
	return "secret-value", nil // pragma: allowlist secret
}

type mockSlackClient struct{}

func (m *mockSlackClient) PostMessage(ctx context.Context, channelID string, message string) error {
	return nil
}

func (m *mockSlackClient) SendDM(ctx context.Context, userID string, message string) error {
	return nil
}

func (m *mockSlackClient) OpenModal(ctx context.Context, triggerID string, view interface{}) error {
	return nil
}

type mockTracer struct {
	spans []string
}

func (m *mockTracer) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	m.spans = append(m.spans, name)
	return ctx, func() {}
}

func (m *mockTracer) AddAnnotation(ctx context.Context, key string, value interface{}) {}

type mockLogger struct {
	logs []string
}

func (m *mockLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	m.logs = append(m.logs, "DEBUG: "+msg)
}

func (m *mockLogger) Info(ctx context.Context, msg string, fields ...Field) {
	m.logs = append(m.logs, "INFO: "+msg)
}

func (m *mockLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	m.logs = append(m.logs, "WARN: "+msg)
}

func (m *mockLogger) Error(ctx context.Context, msg string, err error, fields ...Field) {
	m.logs = append(m.logs, "ERROR: "+msg)
}

func TestNewBotContext(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr error
	}{
		{
			name: "valid options",
			opts: Options{
				Config:         &mockConfig{version: "1.0"},
				DynamoDB:       &mockDynamoDBClient{},
				SecretsManager: &mockSecretsClient{},
				SlackClient:    &mockSlackClient{},
				Tracer:         &mockTracer{},
				Logger:         &mockLogger{},
			},
			wantErr: nil,
		},
		{
			name: "minimal options with config only",
			opts: Options{
				Config: &mockConfig{version: "1.0"},
			},
			wantErr: nil,
		},
		{
			name:    "missing config",
			opts:    Options{},
			wantErr: ErrConfigRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := New(tt.opts)
			if err != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr == nil && ctx == nil {
				t.Error("Expected non-nil context")
			}
		})
	}
}

func TestBotContextConfig(t *testing.T) {
	cfg := &mockConfig{version: "1.0"}
	ctx, err := New(Options{Config: cfg})
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	// Test Config() method
	if ctx.Config() != cfg {
		t.Error("Config() should return the same config instance")
	}

	// Test config properties
	if ctx.Config().Version() != "1.0" {
		t.Errorf("Expected version 1.0, got %s", ctx.Config().Version())
	}
}

func TestBotContextReloadConfig(t *testing.T) {
	initialCfg := &mockConfig{version: "1.0"}
	provider := &mockConfigProvider{}

	ctx, err := New(Options{
		Config:         initialCfg,
		ConfigProvider: provider,
	})
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	// Initial version should be 1.0
	if ctx.Config().Version() != "1.0" {
		t.Errorf("Expected initial version 1.0, got %s", ctx.Config().Version())
	}

	// Reload config
	err = ctx.ReloadConfig()
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	// Version should now be 2.0 (from mock provider)
	if ctx.Config().Version() != "2.0" {
		t.Errorf("Expected reloaded version 2.0, got %s", ctx.Config().Version())
	}

	// Test reload without provider
	ctx2, _ := New(Options{Config: initialCfg})
	err = ctx2.ReloadConfig()
	if err != ErrNoConfigProvider {
		t.Errorf("Expected ErrNoConfigProvider, got %v", err)
	}

	// Test reload with error
	provider.loadFunc = func() (config.Config, error) {
		return nil, errors.New("load error")
	}
	err = ctx.ReloadConfig()
	if err == nil || err.Error() != "load error" {
		t.Errorf("Expected load error, got %v", err)
	}
}

func TestBotContextClients(t *testing.T) {
	dynamoDB := &mockDynamoDBClient{}
	secrets := &mockSecretsClient{}
	slack := &mockSlackClient{}

	ctx, err := New(Options{
		Config:         &mockConfig{},
		DynamoDB:       dynamoDB,
		SecretsManager: secrets,
		SlackClient:    slack,
	})
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	// Test client accessors
	if ctx.DynamoDB() != dynamoDB {
		t.Error("DynamoDB() should return the same instance")
	}

	if ctx.SecretsManager() != secrets {
		t.Error("SecretsManager() should return the same instance")
	}

	if ctx.SlackClient() != slack {
		t.Error("SlackClient() should return the same instance")
	}
}

func TestBotContextTracerAndLogger(t *testing.T) {
	tracer := &mockTracer{}
	logger := &mockLogger{}

	ctx, err := New(Options{
		Config: &mockConfig{},
		Tracer: tracer,
		Logger: logger,
	})
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	// Test tracer
	if ctx.Tracer() != tracer {
		t.Error("Tracer() should return the same instance")
	}

	// Test logger
	if ctx.Logger() != logger {
		t.Error("Logger() should return the same instance")
	}

	// Test default implementations when not provided
	ctx2, _ := New(Options{Config: &mockConfig{}})
	if ctx2.Tracer() == nil {
		t.Error("Tracer() should return default implementation")
	}
	if ctx2.Logger() == nil {
		t.Error("Logger() should return default implementation")
	}
}

func TestBotContextRequestScoping(t *testing.T) {
	botCtx, err := New(Options{Config: &mockConfig{}})
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	ctx := context.Background()

	// Test request ID
	ctx = botCtx.WithRequestID(ctx, "req-123")
	if botCtx.RequestID(ctx) != "req-123" {
		t.Errorf("Expected request ID req-123, got %s", botCtx.RequestID(ctx))
	}

	// Test user ID
	ctx = botCtx.WithUserID(ctx, "U123456")
	if botCtx.UserID(ctx) != "U123456" {
		t.Errorf("Expected user ID U123456, got %s", botCtx.UserID(ctx))
	}

	// Test channel ID
	ctx = botCtx.WithChannelID(ctx, "C789012")
	if botCtx.ChannelID(ctx) != "C789012" {
		t.Errorf("Expected channel ID C789012, got %s", botCtx.ChannelID(ctx))
	}

	// Test empty context
	emptyCtx := context.Background()
	if botCtx.RequestID(emptyCtx) != "" {
		t.Error("Expected empty request ID for empty context")
	}
	if botCtx.UserID(emptyCtx) != "" {
		t.Error("Expected empty user ID for empty context")
	}
	if botCtx.ChannelID(emptyCtx) != "" {
		t.Error("Expected empty channel ID for empty context")
	}
}

func TestDefaultLogger(t *testing.T) {
	logger := &defaultLogger{}
	ctx := context.WithValue(context.Background(), RequestIDKey, "req-123")

	// Just ensure these don't panic
	logger.Debug(ctx, "debug message", Field{Key: "key", Value: "value"})
	logger.Info(ctx, "info message")
	logger.Warn(ctx, "warn message")
	logger.Error(ctx, "error message", errors.New("test error"))
}

func TestNoopTracer(t *testing.T) {
	tracer := &noopTracer{}
	ctx := context.Background()

	// Test StartSpan
	newCtx, done := tracer.StartSpan(ctx, "test-span")
	if newCtx != ctx {
		t.Error("NoopTracer should return same context")
	}
	done() // Should not panic

	// Test AddAnnotation
	tracer.AddAnnotation(ctx, "key", "value") // Should not panic
}

func TestConcurrentAccess(t *testing.T) {
	cfg := &mockConfig{version: "1.0"}
	provider := &mockConfigProvider{}

	ctx, err := New(Options{
		Config:         cfg,
		ConfigProvider: provider,
	})
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	// Test concurrent config access and reload
	done := make(chan bool)

	// Multiple readers
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = ctx.Config().Version()
			}
			done <- true
		}()
	}

	// Config reloader
	go func() {
		for i := 0; i < 50; i++ {
			_ = ctx.ReloadConfig()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 11; i++ {
		<-done
	}
}
