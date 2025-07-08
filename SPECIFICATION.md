# Synaptiq Standup Slackbot Specification

## Overview

A serverless Slack bot application built with Go and deployed on AWS Lambda that automates daily standup management. The bot tracks standup updates from team members, sends reminders, and posts daily summaries.

## Core Requirements

### 1. Standup Update Tracking
- Track when each user submits their standup update
- Store update history for reporting and analytics
- Support multiple channels with different configurations

### 2. Reminder System
- Send DM reminders to users who haven't submitted updates
- Configurable reminder times and frequencies
- Respect user timezone preferences

### 3. Daily Summary
- Post a summary to the configured channel each day
- List users who have submitted updates
- Highlight users who haven't submitted updates
- Configurable summary posting time

### 4. Configuration-Driven Design
- YAML-based configuration file
- Hot-reloadable configuration updates
- Per-channel configuration support

## Architecture

### Module Structure

```
synaptiq-standup-slackbot/
├── config/                    # Configuration module (separate go.mod)
│   ├── go.mod
│   ├── config.go             # Configuration interface
│   ├── yaml.go               # YAML parser implementation
│   ├── validator.go          # Configuration validation
│   └── config_test.go        # Configuration tests
├── context/                   # Shared context module (separate go.mod)
│   ├── go.mod
│   ├── context.go            # Context interface and implementation
│   └── context_test.go       # Context tests
├── cmd/                       # Lambda function entry points
│   ├── webhook/              # Slack event handler
│   ├── scheduler/            # Cron job for triggers and summaries
│   └── processor/            # Async message processor
├── internal/                  # Private application code
│   ├── slack/                # Slack API integration
│   ├── standup/              # Core business logic
│   ├── store/                # DynamoDB data access
│   └── lambda/               # Lambda utilities
└── pkg/                       # Public packages
    └── models/               # Shared data models
```

### Configuration Schema

```yaml
# config.yaml
version: "1.0"

# Global settings
bot:
  token: "${SLACK_BOT_TOKEN}"  # Environment variable substitution
  app_token: "${SLACK_APP_TOKEN}"
  
# Database configuration
database:
  table_name: "standup-bot"
  region: "us-east-1"

# Channel configurations
channels:
  - id: "C1234567890"
    name: "engineering-standup"
    enabled: true
    
    # Standup schedule
    schedule:
      timezone: "America/New_York"
      summary_time: "09:00"  # Time to post summary
      reminder_times:        # Times to send reminders
        - "08:30"
        - "08:50"
      active_days: ["Mon", "Tue", "Wed", "Thu", "Fri"]
    
    # Users required to submit updates
    users:
      - id: "U1234567890"
        name: "alice"
        timezone: "America/New_York"
      - id: "U0987654321"
        name: "bob"
        timezone: "America/Chicago"
    
    # Message templates
    templates:
      reminder: "Hey {{.UserName}}! Don't forget to submit your standup update for #{{.ChannelName}}"
      summary_header: "📊 Daily Standup Summary for {{.Date}}"
      user_completed: "✅ {{.UserName}} - {{.Time}}"
      user_missing: "❌ {{.UserName}} - No update"
    
    # Questions for standup
    questions:
      - "What did you work on yesterday?"
      - "What are you working on today?"
      - "Any blockers?"

# Feature flags
features:
  threading_enabled: true
  analytics_enabled: true
  vacation_mode: true
```

### Shared Context Interface

```go
// context/context.go
package context

import (
    "context"
    "github.com/synaptiq/standup-bot/config"
)

// BotContext provides shared state across the application
type BotContext interface {
    // Configuration access
    Config() config.Config
    
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
}

// New creates a new bot context
func New(cfg config.Config) (BotContext, error) {
    // Implementation
}
```

### Configuration Interface

```go
// config/config.go
package config

import "time"

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
    
    // Reload configuration
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
    ReminderTemplate() string
    SummaryTemplate() string
    
    // Questions
    Questions() []string
}
```

## Data Models

### DynamoDB Schema

```
# Workspace/Channel Configuration
PK: WORKSPACE#<team_id>
SK: CONFIG#<channel_id>
Attributes:
  - enabled: boolean
  - schedule: map
  - users: list
  - templates: map
  - questions: list
  - updated_at: timestamp

# Standup Session
PK: SESSION#<channel_id>#<date>
SK: SESSION#<channel_id>#<date>
Attributes:
  - session_id: string
  - channel_id: string
  - date: string (YYYY-MM-DD)
  - status: string (pending|in_progress|completed)
  - summary_posted: boolean
  - created_at: timestamp
  - completed_at: timestamp

# User Response
PK: SESSION#<channel_id>#<date>
SK: USER#<user_id>
Attributes:
  - user_id: string
  - user_name: string
  - responses: map<question_id, answer>
  - submitted_at: timestamp
  - reminder_count: number

# Reminder Tracking
PK: REMINDER#<channel_id>#<date>
SK: USER#<user_id>#<time>
Attributes:
  - sent_at: timestamp
  - message_ts: string
```

## Implementation Phases

### Phase 1: Foundation (Current Focus)
1. Initialize Go modules structure
2. Implement configuration module with YAML support
3. Implement shared context module
4. Set up basic Lambda handlers
5. Implement DynamoDB data layer

### Phase 2: Core Functionality
1. Slack event webhook handler
2. Standup response collection via modal
3. Response storage in DynamoDB
4. Basic daily summary generation

### Phase 3: Reminder System
1. Scheduler Lambda for reminder triggers
2. DM reminder sending
3. Reminder tracking and rate limiting
4. Timezone-aware scheduling

### Phase 4: Enhanced Features
1. Threading support for responses
2. Analytics and reporting
3. Vacation mode
4. Multi-workspace support
5. Configuration hot-reload

## Testing Strategy

### Unit Tests
- Configuration parsing and validation
- Context creation and management
- Business logic in standup package
- Data access layer

### Integration Tests
- Lambda handler integration
- Slack API integration
- DynamoDB operations
- End-to-end workflows

### Load Tests
- Concurrent user submissions
- High-volume reminder sending
- DynamoDB throughput limits

## Security Considerations

1. **Slack Request Verification**: All incoming webhooks verified using Slack signing secret
2. **Token Management**: Bot tokens stored in AWS Secrets Manager
3. **Data Encryption**: All data encrypted at rest in DynamoDB
4. **IAM Policies**: Least-privilege access for Lambda functions
5. **Input Validation**: All user inputs sanitized before storage

## Monitoring and Observability

1. **AWS X-Ray**: Distributed tracing for all Lambda invocations
2. **CloudWatch Logs**: Structured logging with request IDs
3. **Custom Metrics**:
   - Response submission rate
   - Reminder effectiveness
   - Lambda cold start frequency
   - DynamoDB consumed capacity

## Success Metrics

1. **User Engagement**:
   - Daily active users submitting standups
   - Response time after reminders
   - Completion rate per channel

2. **System Performance**:
   - Lambda execution time < 1s
   - Reminder delivery accuracy > 99%
   - Zero data loss

3. **Cost Efficiency**:
   - Cost per user per month < $0.10
   - True scale-to-zero during weekends

## Future Enhancements

1. **AI-Powered Insights**: Analyze standup patterns and provide team insights
2. **Integration Hub**: Connect with Jira, GitHub, etc. for automatic updates
3. **Mobile App**: Native mobile experience for standup submission
4. **Voice Integration**: Submit standups via voice message
5. **Team Analytics Dashboard**: Web dashboard for managers