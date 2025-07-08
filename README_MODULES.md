# Synaptiq Standup Bot - Module Documentation

This document describes the configuration and context modules that form the foundation of the Slack standup bot.

## Configuration Module

The configuration module (`config/`) provides a flexible, YAML-based configuration system with environment variable substitution and comprehensive validation.

### Features

- **YAML-based configuration** with clean, readable syntax
- **Environment variable substitution** using `${VAR_NAME}` syntax
- **Comprehensive validation** to catch errors early
- **Thread-safe access** for concurrent operations
- **Extensible design** with interfaces for custom providers

### Usage

```go
import "github.com/synaptiq/standup-bot/config"

// Load configuration from file
provider := config.NewYAMLProvider("config.yaml")
cfg, err := provider.Load()
if err != nil {
    log.Fatal(err)
}

// Validate configuration
validator := config.NewValidator()
if err := validator.Validate(cfg); err != nil {
    log.Fatal(err)
}

// Access configuration
fmt.Printf("Bot token: %s\n", cfg.BotToken())
fmt.Printf("Database: %s in %s\n", cfg.DatabaseTable(), cfg.DatabaseRegion())

// Check feature flags
if cfg.IsFeatureEnabled("threading_enabled") {
    // Use threading
}

// Access channel configuration
channel, found := cfg.ChannelByID("C1234567890")
if found && channel.IsEnabled() {
    users := channel.Users()
    questions := channel.Questions()
    // Process channel
}
```

### Configuration Schema

See `config.example.yaml` for a complete example. Key sections:

- `bot`: Slack authentication tokens
- `database`: DynamoDB settings
- `channels`: Per-channel standup configurations
- `features`: Feature flags for enabling/disabling functionality

## Context Module

The context module (`context/`) provides a centralized way to manage application state, dependencies, and request-scoped data.

### Features

- **Dependency injection** for AWS and Slack clients
- **Request-scoped data** (request ID, user ID, channel ID)
- **Configuration management** with hot-reload support
- **Structured logging** with automatic context enrichment
- **Distributed tracing** support

### Usage

```go
import (
    "github.com/synaptiq/standup-bot/config"
    "github.com/synaptiq/standup-bot/context"
)

// Create bot context
botCtx, err := context.New(context.Options{
    Config:         cfg,
    ConfigProvider: provider,
    DynamoDB:       dynamoClient,
    SecretsManager: secretsClient,
    SlackClient:    slackClient,
    Tracer:         tracer,
    Logger:         logger,
})

// Add request context
ctx := context.Background()
ctx = botCtx.WithRequestID(ctx, "req-123")
ctx = botCtx.WithUserID(ctx, "U1234567890")
ctx = botCtx.WithChannelID(ctx, "C1234567890")

// Use logger with automatic context
logger := botCtx.Logger()
logger.Info(ctx, "Processing standup",
    context.Field{Key: "action", Value: "submit"},
)

// Access clients
db := botCtx.DynamoDB()
slack := botCtx.SlackClient()

// Hot-reload configuration
err = botCtx.ReloadConfig()
```

### Context Interfaces

The module defines interfaces for all external dependencies:

- `DynamoDBClient`: DynamoDB operations
- `SecretsClient`: AWS Secrets Manager
- `SlackClient`: Slack API operations
- `Tracer`: Distributed tracing
- `Logger`: Structured logging

This design makes testing easy and allows swapping implementations.

## Integration Example

See `example/main.go` for a complete example showing how to:

1. Load and validate configuration
2. Create a bot context with dependencies
3. Handle a standup request with proper context
4. Access channel and user configurations
5. Use feature flags

## Testing

Both modules include comprehensive test suites:

```bash
# Test configuration module
cd config
go test -v

# Test context module
cd context
go test -v
```

The tests demonstrate:
- Configuration loading and validation
- Error handling
- Thread safety
- Mock implementations
- Edge cases

## Best Practices

1. **Always validate configuration** after loading
2. **Use environment variables** for sensitive data (tokens, secrets)
3. **Add request context** at the beginning of request handlers
4. **Check feature flags** before using optional features
5. **Handle missing configurations** gracefully
6. **Use structured logging** with proper context fields

## Next Steps

With these foundational modules in place, the next steps are:

1. Implement DynamoDB data layer using the defined schema
2. Create Slack client wrapper implementing the interface
3. Build Lambda handlers using the context module
4. Implement core standup logic (collection, reminders, summaries)
5. Add comprehensive integration tests