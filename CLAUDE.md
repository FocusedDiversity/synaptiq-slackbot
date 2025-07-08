# Claude Code Instructions - Slack Stand-up Bot

This document provides quick reference and patterns for Claude Code to efficiently work with this Go-based Slack stand-up bot deployed on AWS Lambda.

## Project Overview

- **Language**: Go 1.21+
- **Deployment**: AWS Lambda with SAM
- **Database**: DynamoDB (single-table design)
- **Architecture**: Serverless, event-driven
- **Main Components**: Webhook handler, Scheduler, Processor

## Quick Commands

### Development

```bash
make dev          # Start local development with hot reload
make test         # Run all tests with coverage
make lint         # Run linters and security checks
make build-lambda # Build Lambda functions
```

### Deployment

```bash
make deploy-dev   # Deploy to development environment
make deploy-prod  # Deploy to production (requires confirmation)
make rollback     # Rollback to previous version
```

### Testing

```bash
make test-unit       # Unit tests only
make test-integration # Integration tests
make test-lambda-local # Test Lambda functions locally
make coverage-report  # Generate coverage visualization
```

## Project Structure

```
.
├── cmd/                    # Lambda function entry points
│   ├── webhook/           # Slack event handler
│   ├── scheduler/         # Cron job for triggering stand-ups
│   └── processor/         # Async message processor
├── internal/              # Private application code
│   ├── slack/            # Slack API integration
│   ├── standup/          # Core business logic
│   ├── store/            # DynamoDB data access
│   └── lambda/           # Lambda utilities
├── pkg/                   # Public packages (if any)
├── testutil/             # Test utilities and mocks
├── scripts/              # Build and deployment scripts
├── template.yaml         # SAM template
└── Makefile             # Task automation
```

## Lambda Function Patterns

### Basic Lambda Handler

```go
func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // 1. Validate request
    if err := validateSlackRequest(request); err != nil {
        return events.APIGatewayProxyResponse{StatusCode: 401}, nil
    }

    // 2. Process event
    // 3. Return response
    return events.APIGatewayProxyResponse{
        StatusCode: 200,
        Body:       "Success",
    }, nil
}
```

### Error Handling Pattern

```go
// Always use structured errors
type StandupError struct {
    Code    string
    Message string
    Err     error
}

// Log errors with context
log.WithError(err).WithFields(log.Fields{
    "user_id": userID,
    "session_id": sessionID,
}).Error("Failed to store response")
```

## DynamoDB Access Patterns

### Single Table Design Keys

```
# Workspace
PK: WORKSPACE#<team_id>
SK: WORKSPACE#<team_id>

# Configuration
PK: WORKSPACE#<team_id>
SK: CONFIG#<channel_id>

# Session
PK: SESSION#<session_id>
SK: SESSION#<session_id>

# Response
PK: SESSION#<session_id>
SK: USER#<user_id>
```

### Common Queries

```go
// Get all active configurations
configs, err := store.QueryByGSI1("ACTIVE#true")

// Get responses for a session
responses, err := store.QueryByPK("SESSION#" + sessionID)

// Store response (with idempotency)
err := store.PutItemIdempotent(response)
```

## Slack API Integration

### Block Kit Patterns

```go
// Always use Block Kit for rich formatting
blocks := []slack.Block{
    slack.NewHeaderBlock(&slack.TextBlockObject{
        Type: slack.PlainTextType,
        Text: "Daily Stand-up",
    }),
    slack.NewSectionBlock(textBlock, nil, accessory),
}
```

### Modal Handling

```go
// Open modal for stand-up responses
_, err := client.OpenView(triggerID, modalView)

// Handle modal submission
payload := &slack.InteractionCallback{}
json.Unmarshal([]byte(request.Body), payload)
```

## Testing Patterns

### Table-Driven Tests

```go
func TestStandupTrigger(t *testing.T) {
    tests := []struct {
        name     string
        config   StandupConfig
        expected bool
    }{
        {"should trigger", activeConfig, true},
        {"should not trigger", inactiveConfig, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := shouldTrigger(tt.config)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Mock Patterns

```go
// Use interfaces for dependencies
type SlackClient interface {
    PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
}

// Mock in tests
mockClient := &MockSlackClient{}
mockClient.On("PostMessage", "C123", mock.Anything).Return("ts", "", nil)
```

## Performance Considerations

1. **Initialize SDK clients in init()** to reuse across invocations
2. **Use connection pooling** for HTTP clients
3. **Batch DynamoDB operations** when possible
4. **Keep Lambda packages small** - vendor only what's needed
5. **Use context for timeouts** in all operations

## Security Best Practices

1. **Always verify Slack signatures** on incoming requests
2. **Use AWS Secrets Manager** for tokens (never hardcode)
3. **Apply least privilege IAM** policies
4. **Sanitize user input** before storage
5. **Enable AWS X-Ray** for tracing

## Common Troubleshooting

### Lambda Cold Starts

- Check function size with `make lambda-size`
- Consider provisioned concurrency for critical functions

### DynamoDB Throttling

- Check metrics in CloudWatch
- Consider on-demand billing or increase provisioned capacity

### Slack Rate Limits

- Implement exponential backoff
- Use batch operations where available

## Debugging Commands

```bash
# View Lambda logs
make logs-webhook
make logs-scheduler

# Test with mock events
make test-event EVENT=slack-command

# Local DynamoDB shell
make dynamodb-shell

# Profile memory usage
make profile-memory FUNC=webhook
```

## Code Style Guidelines

1. Use `gofumpt` for formatting (enforced by pre-commit)
2. Keep functions under 50 lines
3. Use meaningful variable names (no single letters except loops)
4. Document exported functions
5. Return early on errors

## Making Changes

When modifying code:

1. Run `make lint` before committing
2. Add tests for new functionality
3. Update SAM template if adding resources
4. Run `make test-lambda-local` to verify
5. Check `make security` passes

## Useful Snippets

### Add New Slash Command

```go
// In cmd/webhook/main.go
case "/standup-report":
    handleStandupReport(ctx, cmd)
```

### Add New DynamoDB Query

```go
// In internal/store/dynamodb.go
func (s *Store) GetRecentSessions(channelID string, limit int) ([]Session, error) {
    // Implementation
}
```

### Add New Scheduled Job

```yaml
# In template.yaml
NewScheduledFunction:
  Type: Schedule
  Properties:
    Schedule: cron(0 17 ? * MON-FRI *)  # 5 PM weekdays
```
