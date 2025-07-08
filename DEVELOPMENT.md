# Development Guide - Slack Stand-up Bot

This guide covers the development environment setup, workflows, and best practices for working on the Slack stand-up bot.

## Prerequisites

- Go 1.21 or later
- Docker Desktop
- AWS CLI configured
- GitHub account with SSH access
- macOS or Linux (Windows via WSL2)

## Initial Setup

### 1. Clone and Install Dependencies

```bash
git clone git@github.com:your-org/standup-bot.git
cd standup-bot
make setup  # Installs all tools and dependencies
```

### 2. Environment Configuration

```bash
cp .env.example .env.local
# Edit .env.local with your values:
# - SLACK_BOT_TOKEN
# - SLACK_SIGNING_SECRET
# - AWS_PROFILE (optional)
```

### 3. Local Services

```bash
make dev  # Starts DynamoDB local and other services
```

## Development Workflow

### Daily Development

1. **Start Development Environment**

   ```bash
   make dev
   ```

   This starts:
   - Local DynamoDB on port 8000
   - Air for hot reload
   - SAM local API on port 3000

2. **Make Changes**
   - Code changes auto-reload via Air
   - SAM template changes require restart

3. **Run Tests**

   ```bash
   make test        # All tests
   make test-watch  # Watch mode
   ```

4. **Lint and Format**

   ```bash
   make lint  # Runs all linters
   make fmt   # Formats code
   ```

### Testing Lambda Functions Locally

#### Method 1: SAM Local (Recommended)

```bash
# Start API Gateway locally
make test-lambda-local

# In another terminal, test endpoints
curl -X POST http://localhost:3000/slack/events \
  -H "Content-Type: application/json" \
  -d @events/command.json
```

#### Method 2: Direct Invocation

```bash
# Test specific function with event
sam local invoke WebhookFunction -e events/slack-command.json

# With debugging
sam local invoke WebhookFunction -e events/slack-command.json -d 5858
```

### Debugging

#### VS Code Debugging

1. Use the provided launch configurations
2. Set breakpoints in your code
3. Press F5 to start debugging

#### GoLand/IntelliJ Debugging

1. Create a new "Go Remote" configuration
2. Set host to `localhost:5858`
3. Run Lambda with `-d 5858` flag
4. Start debugging session

#### Delve Debugging

```bash
# Start Lambda with Delve
sam local start-api -d 5858

# Connect with Delve
dlv connect localhost:5858
```

### Working with DynamoDB Local

```bash
# Access DynamoDB shell
make dynamodb-shell

# List tables
aws dynamodb list-tables --endpoint-url http://localhost:8000

# Scan table
aws dynamodb scan \
  --table-name standup-bot \
  --endpoint-url http://localhost:8000
```

## Testing Strategies

### Unit Tests

- Located next to source files as `*_test.go`
- Mock external dependencies
- Focus on business logic

### Integration Tests

- Located in `*_integration_test.go` files
- Use build tag: `//go:build integration`
- Test with real AWS services

### Lambda Tests

- Use `testutil/lambda` helpers
- Mock API Gateway events
- Test error scenarios

### End-to-End Tests

```bash
# Deploy to test environment
make deploy-test

# Run E2E tests
make test-e2e
```

## Performance Profiling

### CPU Profiling

```bash
make profile-cpu FUNC=webhook
go tool pprof cpu.prof
```

### Memory Profiling

```bash
make profile-memory FUNC=webhook
go tool pprof mem.prof
```

### Benchmark Tests

```bash
make benchmark
```

## Security Scanning

### Before Committing

```bash
make security  # Runs all security checks
```

### Security Tools

- **govulncheck**: Go vulnerability database
- **gosec**: Security-focused linter
- **nancy**: Dependency vulnerability scanner
- **trivy**: Container/filesystem scanner

## Deployment

### Deploy to Development

```bash
make deploy-dev
```

### Deploy to Staging

```bash
make deploy-staging
```

### Deploy to Production

```bash
# Requires approval in GitHub
git tag v1.0.0
git push origin v1.0.0
```

## Architecture Decisions

### Why SAM over Serverless Framework?

- Native AWS service
- Better CloudFormation integration
- Local testing capabilities
- No additional dependencies

### Why DynamoDB over RDS?

- True serverless (scale to zero)
- No connection pool issues
- Predictable performance
- Lower operational overhead

### Why Separate Lambda Functions?

- Independent scaling
- Smaller deployment packages
- Easier debugging
- Clear separation of concerns

## Common Tasks

### Add a New Slack Command

1. Update webhook handler in `cmd/webhook/main.go`
2. Add command logic in `internal/slack/commands.go`
3. Add tests in `internal/slack/commands_test.go`
4. Update SAM template if needed

### Add a New Scheduled Job

1. Create new function in `cmd/newjob/main.go`
2. Add to SAM template with schedule
3. Implement logic in `internal/standup/`
4. Add tests

### Modify DynamoDB Schema

1. Update models in `internal/store/models.go`
2. Update queries in `internal/store/dynamodb.go`
3. Create migration script if needed
4. Test with local DynamoDB first

## Troubleshooting

### Lambda Not Updating

```bash
# Clear SAM cache
rm -rf .aws-sam/
make build-lambda
```

### DynamoDB Local Issues

```bash
# Reset local database
docker-compose down -v
docker-compose up -d
```

### Slow Tests

```bash
# Run tests in parallel
go test -parallel 4 ./...

# Skip integration tests
go test -short ./...
```

### Memory Issues

```bash
# Increase Lambda memory in template.yaml
MemorySize: 512  # Increase as needed
```

## Best Practices

### Code Organization

1. Keep handlers thin - delegate to internal packages
2. Use interfaces for external dependencies
3. Group related functionality in packages
4. Avoid circular dependencies

### Error Handling

1. Always check errors
2. Wrap errors with context
3. Log errors with structured fields
4. Return user-friendly messages

### Testing

1. Write tests first (TDD)
2. Aim for 80%+ coverage
3. Test edge cases
4. Use table-driven tests

### Performance

1. Initialize clients in `init()`
2. Reuse HTTP connections
3. Batch DynamoDB operations
4. Use context timeouts

### Security

1. Never log sensitive data
2. Validate all inputs
3. Use least-privilege IAM
4. Keep dependencies updated

## Useful Resources

- [AWS SAM Documentation](https://docs.aws.amazon.com/serverless-application-model/)
- [Go Best Practices](https://github.com/golang/go/wiki/CodeReviewComments)
- [Slack API Documentation](https://api.slack.com/)
- [DynamoDB Best Practices](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/best-practices.html)

## Getting Help

1. Check CLAUDE.md for quick reference
2. Search existing issues on GitHub
3. Ask in #standup-bot-dev Slack channel
4. Create detailed bug reports with:
   - Steps to reproduce
   - Expected vs actual behavior
   - Relevant logs
   - Environment details
