#!/bin/bash
set -euo pipefail

echo "🚀 Installing Go development tools for Slack Stand-up Bot..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go 1.21+ first.${NC}"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}✓ Go ${GO_VERSION} detected${NC}"

# Install Go tools
echo -e "\n${YELLOW}📦 Installing Go development tools...${NC}"

# Language server and core tools
go install golang.org/x/tools/gopls@latest
echo -e "${GREEN}✓ gopls (language server)${NC}"

go install github.com/go-delve/delve/cmd/dlv@latest
echo -e "${GREEN}✓ dlv (debugger)${NC}"

# Linting and code quality
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
echo -e "${GREEN}✓ golangci-lint${NC}"

go install honnef.co/go/tools/cmd/staticcheck@latest
echo -e "${GREEN}✓ staticcheck${NC}"

go install github.com/mgechev/revive@latest
echo -e "${GREEN}✓ revive${NC}"

# Security tools
go install golang.org/x/vuln/cmd/govulncheck@latest
echo -e "${GREEN}✓ govulncheck (vulnerability scanner)${NC}"

go install github.com/securego/gosec/v2/cmd/gosec@latest
echo -e "${GREEN}✓ gosec (security checker)${NC}"

# Testing tools
go install gotest.tools/gotestsum@latest
echo -e "${GREEN}✓ gotestsum (better test output)${NC}"

go install github.com/rakyll/gotest@latest
echo -e "${GREEN}✓ gotest (colorized output)${NC}"

go install github.com/nikolaydubina/go-cover-treemap@latest
echo -e "${GREEN}✓ go-cover-treemap (coverage visualization)${NC}"

go install github.com/orlangure/gocovsh@latest
echo -e "${GREEN}✓ gocovsh (coverage in terminal)${NC}"

# Development tools
go install github.com/cosmtrek/air@latest
echo -e "${GREEN}✓ air (hot reload)${NC}"

go install github.com/mitranim/gow@latest
echo -e "${GREEN}✓ gow (file watcher)${NC}"

# Code generation and documentation
go install github.com/jfeliu007/goplantuml/cmd/goplantuml@latest
echo -e "${GREEN}✓ goplantuml (diagram generation)${NC}"

go install golang.org/x/tools/cmd/godoc@latest
echo -e "${GREEN}✓ godoc (documentation server)${NC}"

# Formatting
go install mvdan.cc/gofumpt@latest
echo -e "${GREEN}✓ gofumpt (stricter gofmt)${NC}"

go install github.com/segmentio/golines@latest
echo -e "${GREEN}✓ golines (long line formatter)${NC}"

# Dependency management
go install github.com/psampaz/go-mod-outdated@latest
echo -e "${GREEN}✓ go-mod-outdated${NC}"

go install github.com/sonatype-nexus-community/nancy@latest
echo -e "${GREEN}✓ nancy (dependency vulnerability scanner)${NC}"

# Code navigation and analysis
if command -v brew &> /dev/null; then
    echo -e "\n${YELLOW}🍺 Installing tools via Homebrew...${NC}"

    # Install universal-ctags for code indexing
    if ! command -v ctags &> /dev/null; then
        brew install universal-ctags
        echo -e "${GREEN}✓ universal-ctags${NC}"
    else
        echo -e "${GREEN}✓ ctags already installed${NC}"
    fi

    # Install AWS tools
    if ! command -v aws &> /dev/null; then
        brew install awscli
        echo -e "${GREEN}✓ AWS CLI${NC}"
    else
        echo -e "${GREEN}✓ AWS CLI already installed${NC}"
    fi

    if ! command -v sam &> /dev/null; then
        brew install aws-sam-cli
        echo -e "${GREEN}✓ SAM CLI${NC}"
    else
        echo -e "${GREEN}✓ SAM CLI already installed${NC}"
    fi

    # Install other useful tools
    brew list jq &>/dev/null || brew install jq
    echo -e "${GREEN}✓ jq (JSON processor)${NC}"

    brew list yq &>/dev/null || brew install yq
    echo -e "${GREEN}✓ yq (YAML processor)${NC}"

    brew list pre-commit &>/dev/null || brew install pre-commit
    echo -e "${GREEN}✓ pre-commit${NC}"

    brew list act &>/dev/null || brew install act
    echo -e "${GREEN}✓ act (run GitHub Actions locally)${NC}"

    # CodeQL for security analysis (matches GitHub Actions workflow)
    brew list codeql &>/dev/null || brew install codeql
    echo -e "${GREEN}✓ codeql (semantic code analysis)${NC}"
else
    echo -e "${YELLOW}⚠️  Homebrew not found. Please install AWS CLI, SAM CLI, and other tools manually.${NC}"
fi

# Install Node.js tools for Lambda event mocking
if command -v npm &> /dev/null; then
    echo -e "\n${YELLOW}📦 Installing Node.js tools...${NC}"
    npm install -g @serverless/event-mocks
    echo -e "${GREEN}✓ serverless event mocks${NC}"
else
    echo -e "${YELLOW}⚠️  npm not found. Skipping Node.js tools.${NC}"
fi

# Set up git hooks
echo -e "\n${YELLOW}🔧 Setting up git hooks...${NC}"
if [ -f .pre-commit-config.yaml ]; then
    pre-commit install
    echo -e "${GREEN}✓ Pre-commit hooks installed${NC}"
else
    echo -e "${YELLOW}⚠️  .pre-commit-config.yaml not found. Skipping pre-commit setup.${NC}"
fi

# Create necessary directories
echo -e "\n${YELLOW}📁 Creating project directories...${NC}"
mkdir -p .aws-sam/build
mkdir -p events
mkdir -p coverage
echo -e "${GREEN}✓ Project directories created${NC}"

# Download common Lambda event examples
echo -e "\n${YELLOW}📥 Downloading Lambda event examples...${NC}"
if [ ! -f events/api-gateway-proxy.json ]; then
    curl -s https://raw.githubusercontent.com/aws/aws-lambda-go/main/events/testdata/api-gateway-proxy-request.json \
        -o events/api-gateway-proxy.json
    echo -e "${GREEN}✓ API Gateway proxy event${NC}"
fi

if [ ! -f events/slack-command.json ]; then
    cat > events/slack-command.json << 'EOF'
{
  "body": "token=test&team_id=T1234&team_domain=test&channel_id=C1234&channel_name=general&user_id=U1234&user_name=test&command=%2Fstandup&text=start&response_url=https%3A%2F%2Fhooks.slack.com%2Fcommands%2Ftest&trigger_id=123.456.789",
  "headers": {
    "Content-Type": "application/x-www-form-urlencoded",
    "X-Slack-Request-Timestamp": "1234567890",
    "X-Slack-Signature": "v0=test"
  },
  "httpMethod": "POST",
  "path": "/slack/events"
}
EOF
    echo -e "${GREEN}✓ Slack command event${NC}"
fi

# Final setup steps
echo -e "\n${YELLOW}🔍 Running go mod tidy...${NC}"
if [ -f go.mod ]; then
    go mod tidy
    echo -e "${GREEN}✓ Dependencies tidied${NC}"
else
    echo -e "${YELLOW}⚠️  go.mod not found. Run 'go mod init' first.${NC}"
fi

echo -e "\n${GREEN}✅ Installation complete!${NC}"
echo -e "\nNext steps:"
echo -e "  1. Copy ${YELLOW}.env.example${NC} to ${YELLOW}.env.local${NC} and fill in your values"
echo -e "  2. Run ${YELLOW}make dev${NC} to start development environment"
echo -e "  3. Run ${YELLOW}make test${NC} to verify everything is working"
echo -e "\nFor more information, see ${YELLOW}DEVELOPMENT.md${NC}"
