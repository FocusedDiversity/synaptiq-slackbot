# Synaptiq Standup Slackbot

A serverless Slack bot built with Go and AWS Lambda that automates daily standup meetings, tracks team updates, sends reminders, and posts daily summaries.

## 🚀 Overview

This bot helps distributed teams run efficient asynchronous standups by:

- 📅 Automatically collecting daily standup updates from team members
- 🔔 Sending personalized DM reminders to team members who haven't submitted
- 📊 Posting daily summaries showing who has/hasn't submitted updates
- 🌍 Supporting multiple timezones and flexible schedules
- ⚙️ Providing per-channel configuration for different teams

## 🏗️ Architecture

The application follows a serverless architecture deployed on AWS:

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Slack     │────▶│ API Gateway  │────▶│   Lambda    │
│  Workspace  │     │              │     │  Functions  │
└─────────────┘     └──────────────┘     └─────────────┘
                                                 │
                                                 ▼
                                          ┌─────────────┐
                                          │  DynamoDB   │
                                          └─────────────┘
```

### Key Components

- **Lambda Functions**: Three main functions handle webhooks, scheduling, and async processing
- **DynamoDB**: Stores configurations, sessions, and user responses
- **EventBridge**: Triggers scheduled tasks for reminders and summaries
- **Secrets Manager**: Securely stores Slack tokens

## 📁 Project Structure

```
.
├── config/                 # Configuration module (separate go.mod)
│   ├── config.go          # Configuration interfaces
│   ├── yaml.go            # YAML parser implementation
│   └── validator.go       # Configuration validation
├── context/               # Shared context module (separate go.mod)
│   ├── context.go         # Context interface and implementation
│   └── defaults.go        # Default implementations
├── cmd/                   # Lambda function entry points
│   ├── webhook/           # Handles Slack events and commands
│   ├── scheduler/         # Triggers reminders and summaries
│   └── processor/         # Processes async tasks
├── internal/              # Private application code
│   ├── slack/            # Slack API integration
│   ├── standup/          # Core business logic
│   ├── store/            # DynamoDB data access
│   └── lambda/           # Lambda utilities
├── example/              # Example usage
├── scripts/              # Build and deployment scripts
├── template.yaml         # AWS SAM template
└── Makefile             # Task automation
```

## 🚦 Getting Started

### Prerequisites

- Go 1.21+
- AWS CLI configured with appropriate credentials
- AWS SAM CLI for local development
- Docker (for local DynamoDB)
- Slack workspace with admin access

### Local Development

1. **Clone the repository**

   ```bash
   git clone https://github.com/synaptiq/standup-bot.git
   cd standup-bot
   ```

2. **Copy environment configuration**

   ```bash
   cp .env.example .env
   # Edit .env with your Slack tokens and AWS settings
   ```

3. **Copy and configure the bot**

   ```bash
   cp config.example.yaml config.yaml
   # Edit config.yaml with your channel IDs and team settings
   ```

4. **Start local development environment**

   ```bash
   make dev
   ```

   This starts:
   - DynamoDB Local
   - Lambda function with hot reload
   - LocalStack for AWS services

5. **Run tests**

   ```bash
   make test
   ```

### Configuration

The bot is configured via YAML file. See `config.example.yaml` for a complete example:

```yaml
channels:
  - id: "C1234567890"  # pragma: allowlist secret
    name: "engineering-standup"
    schedule:
      timezone: "America/New_York"
      summary_time: "09:00"
      reminder_times: ["08:30", "08:50"]
    users:
      - id: "U1234567890"
        name: "alice"
        timezone: "America/New_York"
```

## 📚 Module Documentation

This codebase uses a modular architecture with two foundational modules:

### Configuration Module (`config/`)

- YAML-based configuration with environment variable support
- Comprehensive validation
- Thread-safe operations
- Hot-reload capability

### Context Module (`context/`)

- Centralized dependency injection
- Request-scoped data management
- Structured logging and tracing
- AWS and Slack client interfaces

See [README_MODULES.md](README_MODULES.md) for detailed module documentation.

## 🔧 Development

### Available Make Commands

```bash
make dev              # Start local development environment
make test             # Run all tests with coverage
make lint             # Run linters and security checks
make build-lambda     # Build Lambda functions
make deploy-dev       # Deploy to development environment
make deploy-prod      # Deploy to production
make logs-webhook     # View webhook Lambda logs
make logs-scheduler   # View scheduler Lambda logs
```

### Testing Strategy

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test interactions with AWS services
- **End-to-End Tests**: Test complete workflows using local stack

### Code Quality

The project uses:

- `golangci-lint` for comprehensive linting
- `gofumpt` for consistent formatting
- Pre-commit hooks for automated checks
- GitHub Actions for CI/CD

### Running CI Checks Locally

The project includes scripts to run the same CI checks that run in GitHub Actions:

```bash
# Run full CI suite (all checks)
./scripts/ci-local.sh

# Run critical checks only (faster)
./scripts/ci-summary.sh

# Auto-fix common lint issues
./scripts/fix-lint.sh

# Install git hooks
./scripts/setup-hooks.sh
```

The git hooks will:
- **pre-commit**: Run quick checks (build, format, security)
- **pre-push**: Run full CI suite including linting

To skip hooks temporarily:
```bash
git commit --no-verify
SKIP_CI_CHECKS=1 git push
```

## 🚀 Deployment

### Deploy to AWS

1. **Configure AWS credentials**

   ```bash
   aws configure
   ```

2. **Deploy to development**

   ```bash
   make deploy-dev
   ```

3. **Deploy to production** (requires confirmation)

   ```bash
   make deploy-prod
   ```

### Infrastructure

The application uses AWS SAM for infrastructure as code. Key resources:

- 3 Lambda functions (webhook, scheduler, processor)
- DynamoDB table with on-demand billing
- API Gateway for Slack webhooks
- EventBridge rules for scheduling
- CloudWatch logs and X-Ray tracing

## 📖 For Claude Code Users

This codebase includes a `CLAUDE.md` file with specific instructions and patterns for Claude Code. Key points:

1. **Architecture Decisions**: Serverless, event-driven design optimized for cost and scalability
2. **Development Patterns**: Table-driven tests, interface-based design, structured error handling
3. **AWS Best Practices**: Least privilege IAM, connection pooling, efficient DynamoDB queries
4. **Debugging Tools**: Comprehensive logging, distributed tracing, local development environment

When working with Claude Code:

- Run `make lint` before committing changes
- Add tests for new functionality
- Follow existing patterns for consistency
- Check `CLAUDE.md` for specific coding guidelines

## 🔒 Security

- Slack request signatures are verified on all incoming webhooks
- AWS Secrets Manager stores all sensitive tokens
- IAM policies follow least-privilege principle
- All data is encrypted at rest in DynamoDB
- No credentials are stored in code or configuration files

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Built with the [Slack API](https://api.slack.com/)
- Deployed using [AWS SAM](https://aws.amazon.com/serverless/sam/)
- Inspired by distributed team needs at Synaptiq

## 📞 Support

For issues and questions:

- Check the [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) guide
- Open an issue on GitHub
- Contact the maintainers

---

**Note**: This is a work in progress. See [SPECIFICATION.md](SPECIFICATION.md) for the complete roadmap and planned features.
