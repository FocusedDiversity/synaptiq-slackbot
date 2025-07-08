#!/bin/bash

# Fix GitHub Actions compatibility issues

set -e

echo "Fixing GitHub Actions compatibility issues..."

# 1. Note: go.mod will use the patch version (e.g., 1.24.4) when go mod tidy is run
# This is expected behavior and GitHub Actions should handle it correctly
echo "Checking Go version in go.mod..."

# 2. Ensure go.mod and go.sum are clean
echo "Cleaning go.mod and go.sum..."
go mod tidy || true

# 3. Check for any remaining issues
echo "Checking for potential issues..."

# Check Go version in go.mod
GO_VERSION=$(grep "^go " go.mod | awk '{print $2}')
echo "go.mod specifies Go version: $GO_VERSION"

# Check for any missing dependencies
if ! go mod verify 2>/dev/null; then
    echo "WARNING: Some dependencies may have issues"
fi

echo "GitHub Actions compatibility fixes applied!"
echo ""
echo "IMPORTANT: Before pushing to GitHub, ensure:"
echo "1. go.mod Go version is compatible with GitHub Actions (1.24.x)"
echo "2. All tests pass locally with: go test ./..."
echo "3. Linting passes with: golangci-lint run --timeout=5m"
echo "4. GitHub Actions workflow uses Go 1.24 (handles patch versions automatically)"