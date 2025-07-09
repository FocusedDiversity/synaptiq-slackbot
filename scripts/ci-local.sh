#!/usr/bin/env bash
# Local CI Runner - Runs all CI checks before commits
# This script mirrors the GitHub Actions CI workflow

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions for pretty output
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Check if we're in the project root
if [ ! -f "go.mod" ]; then
    error "Must be run from project root (where go.mod is located)"
    exit 1
fi

# Track overall status
FAILED_CHECKS=()
WARNINGS=()

# Function to run a check and track failures
run_check() {
    local name="$1"
    local cmd="$2"
    
    info "Running: $name"
    if eval "$cmd"; then
        success "$name passed"
    else
        error "$name failed"
        FAILED_CHECKS+=("$name")
        return 1
    fi
}

# Function to run a check that's allowed to fail (warning only)
run_optional_check() {
    local name="$1"
    local cmd="$2"
    
    info "Running: $name (optional)"
    if eval "$cmd"; then
        success "$name passed"
    else
        warning "$name failed (non-blocking)"
        WARNINGS+=("$name")
    fi
}

echo "================================================"
echo "Running Local CI Checks"
echo "================================================"
echo

# 1. Check for required tools
info "Checking required tools..."
MISSING_TOOLS=()

check_tool() {
    if ! command -v "$1" &> /dev/null; then
        MISSING_TOOLS+=("$1")
        return 1
    fi
}

check_tool "go" && success "go installed"
check_tool "golangci-lint" && success "golangci-lint installed"
check_tool "gofumpt" && success "gofumpt installed"
check_tool "goimports" && success "goimports installed"
check_tool "gosec" && success "gosec installed"
check_tool "sam" && success "sam installed"

# Optional tools (don't add to MISSING_TOOLS)
if ! command -v "govulncheck" &> /dev/null; then
    warning "govulncheck not installed (optional)"
fi
if ! command -v "nancy" &> /dev/null; then
    warning "nancy not installed (optional)"
fi
if ! command -v "gotestsum" &> /dev/null; then
    warning "gotestsum not installed (optional)"
fi

if [ ${#MISSING_TOOLS[@]} -ne 0 ]; then
    error "Missing required tools: ${MISSING_TOOLS[*]}"
    echo
    info "Install missing tools with: make install-tools"
    exit 1
fi

echo

# 2. Go Module Checks
info "=== Go Module Checks ==="
run_check "go mod download" "go mod download"
run_check "go mod tidy" "go mod tidy && git diff --exit-code go.mod go.sum"

echo

# 3. Formatting Checks
info "=== Formatting Checks ==="
run_check "gofumpt" "gofumpt -l -d . | (! grep .)"
run_check "goimports" "goimports -l -d . | (! grep .)"

echo

# 4. Linting
info "=== Linting ==="
run_check "golangci-lint" "golangci-lint run --timeout=5m"

echo

# 5. Build Checks
info "=== Build Checks ==="
run_check "go build" "go build ./..."

echo

# 6. Test Suite
info "=== Test Suite ==="
if command -v gotestsum &> /dev/null; then
    run_check "unit tests" "gotestsum --format pkgname -- -race -coverprofile=coverage.out -covermode=atomic ./..."
else
    run_check "unit tests" "go test -race -coverprofile=coverage.out -covermode=atomic ./..."
fi

echo

# 7. Security Checks
info "=== Security Checks ==="
run_check "gosec" "gosec -quiet -fmt json ./..."

if command -v govulncheck &> /dev/null; then
    run_optional_check "govulncheck" "govulncheck ./..."
fi

if command -v nancy &> /dev/null; then
    run_optional_check "nancy" "go list -json -deps ./... | nancy sleuth"
fi

echo

# 8. SAM/CloudFormation Checks (if SAM is installed)
if command -v sam &> /dev/null; then
    info "=== SAM/CloudFormation Checks ==="
    run_check "SAM template validation" "sam validate"
    
    # Only try to build if we have Docker
    if command -v docker &> /dev/null && docker info &> /dev/null; then
        run_optional_check "SAM build" "sam build --use-container"
    else
        warning "Docker not running - skipping SAM build"
    fi
fi

echo

# 9. License Checks (optional)
if command -v go-licenses &> /dev/null; then
    info "=== License Checks ==="
    run_optional_check "license check" "go-licenses check ./..."
fi

echo
echo "================================================"
echo "CI Check Summary"
echo "================================================"

# Report results
if [ ${#FAILED_CHECKS[@]} -eq 0 ]; then
    success "All required checks passed! ✨"
    
    if [ ${#WARNINGS[@]} -ne 0 ]; then
        echo
        warning "Optional checks with warnings:"
        for check in "${WARNINGS[@]}"; do
            echo "  - $check"
        done
    fi
    
    echo
    success "Ready to commit!"
    exit 0
else
    error "Some checks failed:"
    for check in "${FAILED_CHECKS[@]}"; do
        echo "  - $check"
    done
    
    echo
    error "Please fix the issues before committing."
    exit 1
fi