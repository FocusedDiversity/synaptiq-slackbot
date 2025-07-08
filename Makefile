.PHONY: help setup dev test lint build deploy clean

# Default target
.DEFAULT_GOAL := help

# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=standup-bot
SAM=sam
DOCKER_COMPOSE=docker-compose
AWS_REGION ?= us-east-1
STACK_NAME ?= standup-bot
ENVIRONMENT ?= dev

# Colors
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
NC := \033[0m # No Color

## help: Display this help message
help:
	@echo "Slack Stand-up Bot - Development Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## setup: Install all development dependencies and tools
setup:
	@echo "$(YELLOW)Setting up development environment...$(NC)"
	@./scripts/install-tools.sh
	@go mod download
	@pre-commit install || echo "Pre-commit not installed"
	@echo "$(GREEN)✓ Setup complete!$(NC)"

## dev: Start local development environment with hot reload
dev:
	@echo "$(YELLOW)Starting development environment...$(NC)"
	@$(DOCKER_COMPOSE) up -d
	@air -c .air.toml

## dev-stop: Stop local development environment
dev-stop:
	@echo "$(YELLOW)Stopping development environment...$(NC)"
	@$(DOCKER_COMPOSE) down

## test: Run all tests with coverage
test:
	@echo "$(YELLOW)Running tests...$(NC)"
	@gotestsum --format pkgname -- -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)✓ Tests complete. Coverage: $$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}')$(NC)"

## test-unit: Run unit tests only
test-unit:
	@echo "$(YELLOW)Running unit tests...$(NC)"
	@gotestsum --format pkgname -- -short -race ./...

## test-integration: Run integration tests
test-integration:
	@echo "$(YELLOW)Running integration tests...$(NC)"
	@gotestsum --format pkgname -- -race -tags=integration ./...

## test-watch: Run tests in watch mode
test-watch:
	@echo "$(YELLOW)Running tests in watch mode...$(NC)"
	@gow -c test ./...

## coverage: Generate and display coverage report
coverage: test
	@echo "$(YELLOW)Generating coverage report...$(NC)"
	@go tool cover -html=coverage.out -o coverage.html
	@go-cover-treemap -coverprofile coverage.out > coverage.svg
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(NC)"

## lint: Run all linters
lint:
	@echo "$(YELLOW)Running linters...$(NC)"
	@golangci-lint run
	@echo "$(GREEN)✓ Linting complete$(NC)"

## fmt: Format code
fmt:
	@echo "$(YELLOW)Formatting code...$(NC)"
	@gofumpt -l -w .
	@golines -w --max-len=120 --reformat-tags .
	@echo "$(GREEN)✓ Code formatted$(NC)"

## security: Run security scans
security:
	@echo "$(YELLOW)Running security scans...$(NC)"
	@gosec -fmt sarif -out gosec.sarif ./... || true
	@govulncheck ./...
	@nancy sleuth || true
	@echo "$(GREEN)✓ Security scan complete$(NC)"

## build: Build the application
build:
	@echo "$(YELLOW)Building application...$(NC)"
	@$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/...
	@echo "$(GREEN)✓ Build complete$(NC)"

## build-lambda: Build Lambda functions
build-lambda:
	@echo "$(YELLOW)Building Lambda functions...$(NC)"
	@$(SAM) build --use-container --parallel
	@echo "$(GREEN)✓ Lambda build complete$(NC)"

## test-lambda-local: Run Lambda functions locally
test-lambda-local: build-lambda
	@echo "$(YELLOW)Starting local Lambda API...$(NC)"
	@$(SAM) local start-api --env-vars local.env.json --warm-containers EAGER

## invoke-webhook: Test webhook function locally
invoke-webhook:
	@echo "$(YELLOW)Invoking webhook function...$(NC)"
	@$(SAM) local invoke WebhookFunction -e events/slack-command.json

## invoke-scheduler: Test scheduler function locally
invoke-scheduler:
	@echo "$(YELLOW)Invoking scheduler function...$(NC)"
	@$(SAM) local invoke SchedulerFunction

## deploy-dev: Deploy to development environment
deploy-dev: build-lambda
	@echo "$(YELLOW)Deploying to development...$(NC)"
	@$(SAM) deploy \
		--stack-name $(STACK_NAME)-dev \
		--s3-prefix dev \
		--parameter-overrides Environment=dev \
		--no-confirm-changeset \
		--no-fail-on-empty-changeset
	@echo "$(GREEN)✓ Deployed to development$(NC)"

## deploy-staging: Deploy to staging environment
deploy-staging: build-lambda
	@echo "$(YELLOW)Deploying to staging...$(NC)"
	@$(SAM) deploy \
		--stack-name $(STACK_NAME)-staging \
		--s3-prefix staging \
		--parameter-overrides Environment=staging \
		--confirm-changeset
	@echo "$(GREEN)✓ Deployed to staging$(NC)"

## deploy-prod: Deploy to production environment
deploy-prod:
	@echo "$(RED)Production deployment must be done via GitHub Actions$(NC)"
	@echo "Push a tag: git tag v1.0.0 && git push origin v1.0.0"

## logs-webhook: Tail webhook function logs
logs-webhook:
	@echo "$(YELLOW)Tailing webhook logs...$(NC)"
	@$(SAM) logs -n WebhookFunction --stack-name $(STACK_NAME)-$(ENVIRONMENT) --tail

## logs-scheduler: Tail scheduler function logs
logs-scheduler:
	@echo "$(YELLOW)Tailing scheduler logs...$(NC)"
	@$(SAM) logs -n SchedulerFunction --stack-name $(STACK_NAME)-$(ENVIRONMENT) --tail

## dynamodb-shell: Open DynamoDB local shell
dynamodb-shell:
	@echo "$(YELLOW)Opening DynamoDB shell...$(NC)"
	@aws dynamodb scan --table-name $(STACK_NAME) --endpoint-url http://localhost:8000

## clean: Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf .aws-sam/
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html coverage.svg
	@rm -f gosec.sarif
	@find . -name "bootstrap" -type f -delete
	@echo "$(GREEN)✓ Clean complete$(NC)"

## ci-local: Run CI pipeline locally
ci-local:
	@echo "$(YELLOW)Running CI pipeline locally...$(NC)"
	@act -j lint
	@act -j test
	@act -j build
	@echo "$(GREEN)✓ CI pipeline complete$(NC)"

## validate: Validate SAM template
validate:
	@echo "$(YELLOW)Validating SAM template...$(NC)"
	@$(SAM) validate
	@echo "$(GREEN)✓ Template valid$(NC)"

## benchmark: Run benchmarks
benchmark:
	@echo "$(YELLOW)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. -benchmem ./...

## profile-cpu: Profile CPU usage
profile-cpu:
	@echo "$(YELLOW)Profiling CPU...$(NC)"
	@$(GOTEST) -cpuprofile=cpu.prof -bench=. ./...
	@go tool pprof cpu.prof

## profile-memory: Profile memory usage
profile-memory:
	@echo "$(YELLOW)Profiling memory...$(NC)"
	@$(GOTEST) -memprofile=mem.prof -bench=. ./...
	@go tool pprof mem.prof

## index: Generate code index with ctags
index:
	@echo "$(YELLOW)Generating code index...$(NC)"
	@ctags -R --languages=go --exclude=.git --exclude=.aws-sam --exclude=vendor .
	@echo "$(GREEN)✓ Index generated$(NC)"

## docs: Start local documentation server
docs:
	@echo "$(YELLOW)Starting documentation server...$(NC)"
	@godoc -http=:6060

## release: Create a new release (interactive)
release:
	@echo "$(YELLOW)Creating new release...$(NC)"
	@read -p "Version (e.g., v1.0.0): " VERSION; \
	git tag -a $$VERSION -m "Release $$VERSION"; \
	git push origin $$VERSION
	@echo "$(GREEN)✓ Release created$(NC)"

## install-hooks: Install git hooks
install-hooks:
	@echo "$(YELLOW)Installing git hooks...$(NC)"
	@pre-commit install
	@echo "$(GREEN)✓ Git hooks installed$(NC)"

## update-deps: Update Go dependencies
update-deps:
	@echo "$(YELLOW)Updating dependencies...$(NC)"
	@$(GOGET) -u ./...
	@$(GOCMD) mod tidy
	@go-mod-outdated
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

## lambda-size: Check Lambda function sizes
lambda-size: build-lambda
	@echo "$(YELLOW)Checking Lambda function sizes...$(NC)"
	@find .aws-sam/build -name "bootstrap" -exec ls -lh {} \;

## smoke-test: Run smoke tests against deployed endpoint
smoke-test:
	@echo "$(YELLOW)Running smoke tests...$(NC)"
	@./scripts/smoke-test.sh $$(aws cloudformation describe-stacks \
		--stack-name $(STACK_NAME)-$(ENVIRONMENT) \
		--query 'Stacks[0].Outputs[?OutputKey==`WebhookUrl`].OutputValue' \
		--output text)

## rollback: Rollback to previous deployment
rollback:
	@echo "$(YELLOW)Rolling back deployment...$(NC)"
	@aws cloudformation cancel-update-stack --stack-name $(STACK_NAME)-$(ENVIRONMENT) || \
	aws cloudformation update-stack \
		--stack-name $(STACK_NAME)-$(ENVIRONMENT) \
		--use-previous-template \
		--capabilities CAPABILITY_IAM
	@echo "$(GREEN)✓ Rollback initiated$(NC)"
