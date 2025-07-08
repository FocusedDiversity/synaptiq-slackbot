.PHONY: build clean deploy test lint dev

# Variables
STACK_NAME ?= synaptiq-standup-bot
ENVIRONMENT ?= dev
AWS_REGION ?= us-east-1
SAM_CONFIG_ENV ?= default

# Go build settings
GOOS := linux
GOARCH := amd64
CGO_ENABLED := 0
LDFLAGS := -ldflags="-s -w"

# Build all Lambda functions
build:
	@echo "Building Lambda functions..."
	@for func in webhook scheduler processor; do \
		echo "Building $$func..."; \
		GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build $(LDFLAGS) -o cmd/$$func/bootstrap cmd/$$func/main.go || exit 1; \
	done
	@echo "Build complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@find cmd -name bootstrap -delete
	@rm -rf .aws-sam/
	@echo "Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Tests complete! Coverage report: coverage.html"

# Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run
	@echo "Linting complete!"

# Format code
fmt:
	@echo "Formatting code..."
	@gofumpt -l -w .
	@echo "Formatting complete!"

# Run security checks
security:
	@echo "Running security checks..."
	@gosec -quiet ./...
	@echo "Security checks complete!"

# Local development with hot reload
dev:
	@echo "Starting local development..."
	@docker-compose up -d
	@air -c .air.toml

# Stop local development
dev-stop:
	@echo "Stopping local development..."
	@docker-compose down

# SAM local testing
local-api:
	@echo "Starting SAM local API..."
	@sam local start-api --env-vars env.json

# SAM build
sam-build:
	@echo "Building with SAM..."
	@sam build

# Deploy to AWS
deploy: build
	@echo "Deploying to $(ENVIRONMENT)..."
	@sam deploy \
		--stack-name $(STACK_NAME)-$(ENVIRONMENT) \
		--parameter-overrides \
			Environment=$(ENVIRONMENT) \
			SlackBotToken=$(SLACK_BOT_TOKEN) \
			SlackSigningSecret=$(SLACK_SIGNING_SECRET) \
		--config-env $(SAM_CONFIG_ENV) \
		--no-confirm-changeset \
		--no-fail-on-empty-changeset

# Deploy with guided prompts
deploy-guided:
	@echo "Deploying with guided prompts..."
	@sam deploy --guided

# View Lambda logs
logs-webhook:
	@sam logs -n WebhookFunction --stack-name $(STACK_NAME)-$(ENVIRONMENT) --tail

logs-scheduler:
	@sam logs -n SchedulerFunction --stack-name $(STACK_NAME)-$(ENVIRONMENT) --tail

logs-processor:
	@sam logs -n ProcessorFunction --stack-name $(STACK_NAME)-$(ENVIRONMENT) --tail

# Test Lambda functions locally
test-webhook:
	@echo "Testing webhook function..."
	@sam local invoke WebhookFunction -e events/webhook-event.json

test-scheduler:
	@echo "Testing scheduler function..."
	@sam local invoke SchedulerFunction -e events/scheduler-event.json

test-processor:
	@echo "Testing processor function..."
	@sam local invoke ProcessorFunction -e events/processor-event.json

# Generate test events
generate-events:
	@mkdir -p events
	@echo '{"body": "{\"type\": \"url_verification\", \"challenge\": \"test-challenge\"}"}' > events/webhook-event.json
	@echo '{"source": "aws.events", "detail-type": "Scheduled Event"}' > events/scheduler-event.json
	@echo '{"Records": [{"body": "{\"type\": \"test\", \"channel_id\": \"C123\"}"}]}' > events/processor-event.json

# Check Lambda package sizes
lambda-size:
	@echo "Lambda package sizes:"
	@for func in webhook scheduler processor; do \
		if [ -f cmd/$$func/bootstrap ]; then \
			size=$$(du -h cmd/$$func/bootstrap | cut -f1); \
			echo "  $$func: $$size"; \
		fi; \
	done

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./tests/integration/...

# Update dependencies
deps:
	@echo "Updating dependencies..."
	@go mod download
	@go mod tidy
	@cd config && go mod download && go mod tidy
	@cd context && go mod download && go mod tidy

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@brew install aws-sam-cli

# Validate SAM template
validate:
	@echo "Validating SAM template..."
	@sam validate

# Delete stack
delete:
	@echo "Deleting stack $(STACK_NAME)-$(ENVIRONMENT)..."
	@aws cloudformation delete-stack --stack-name $(STACK_NAME)-$(ENVIRONMENT)
	@echo "Waiting for stack deletion..."
	@aws cloudformation wait stack-delete-complete --stack-name $(STACK_NAME)-$(ENVIRONMENT)

# Show stack outputs
outputs:
	@echo "Stack outputs for $(STACK_NAME)-$(ENVIRONMENT):"
	@aws cloudformation describe-stacks \
		--stack-name $(STACK_NAME)-$(ENVIRONMENT) \
		--query 'Stacks[0].Outputs[*].[OutputKey,OutputValue]' \
		--output table

# Help
help:
	@echo "Synaptiq Standup Bot - Available commands:"
	@echo ""
	@echo "  make build          - Build Lambda functions"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make test           - Run tests"
	@echo "  make lint           - Run linters"
	@echo "  make fmt            - Format code"
	@echo "  make security       - Run security checks"
	@echo "  make dev            - Start local development"
	@echo "  make deploy         - Deploy to AWS"
	@echo "  make logs-webhook   - View webhook function logs"
	@echo "  make logs-scheduler - View scheduler function logs"
	@echo "  make logs-processor - View processor function logs"
	@echo "  make delete         - Delete the stack"
	@echo "  make help           - Show this help message"
