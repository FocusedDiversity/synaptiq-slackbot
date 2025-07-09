# CI/CD Setup Documentation

This document describes the CI/CD setup for the Synaptiq Standup Slackbot project.

## Overview

The project uses both GitHub Actions (remote CI) and local CI scripts to ensure code quality. The local scripts allow developers to run the same checks that run in GitHub Actions before committing or pushing code.

## Scripts

### 1. `scripts/ci-local.sh`

The main CI runner that mirrors GitHub Actions workflow. It runs:
- Tool availability checks
- Go module verification (`go mod tidy`)
- Code formatting (`gofumpt`, `goimports`)
- Linting (`golangci-lint`)
- Build verification
- Unit tests
- Security scanning (`gosec`)
- SAM template validation
- Optional checks (govulncheck, nancy, go-licenses)

Usage:
```bash
./scripts/ci-local.sh
```

### 2. `scripts/ci-summary.sh`

A lightweight CI script that runs only critical checks:
- Build verification
- Format checking
- Import organization
- Basic tests
- Security scan (gosec)
- SAM validation
- CodeQL security analysis (if installed)

This is used by the pre-commit hook for speed.

Usage:
```bash
./scripts/ci-summary.sh
```

### 3. `scripts/fix-lint.sh`

Automatically fixes common linting issues:
- Adds error handling for unchecked errors
- Fixes type assertions
- Runs formatters

Usage:
```bash
./scripts/fix-lint.sh
```

### 4. `scripts/setup-hooks.sh`

Installs git hooks for automated CI checks:
- **pre-commit**: Runs `ci-summary.sh` (quick checks)
- **pre-push**: Runs `ci-local.sh` (full CI suite)

Usage:
```bash
./scripts/setup-hooks.sh
```

## Git Hooks

Once installed, the hooks will:

1. **Pre-commit Hook**:
   - Runs automatically before each commit
   - Executes quick checks (build, format, security)
   - Can be skipped with `git commit --no-verify`

2. **Pre-push Hook**:
   - Runs automatically before pushing
   - Executes full CI suite including linting
   - Can be skipped with `SKIP_CI_CHECKS=1 git push`

## CodeQL Integration

The project includes CodeQL security analysis in both CI and pre-commit hooks:

### Local Setup
1. Install CodeQL via Homebrew: `brew install codeql`
2. CodeQL checks run automatically in pre-commit if installed
3. First run creates a database (may take ~30 seconds)
4. Subsequent runs are faster as the database is reused

### Pre-commit Hook
- Runs basic security analysis locally
- Non-blocking if CodeQL is not installed
- Results are cleaned up automatically

### GitHub Actions
- Full `security-and-quality` query suite runs in CI
- Results uploaded to GitHub Security tab
- Includes checks from gosec, Trivy, and Checkov

## GitHub Actions Integration

The `.github/workflows/ci.yml` file defines the remote CI pipeline with:
- Lint job (golangci-lint, go mod tidy)
- Security scan (govulncheck, gosec, nancy)
- Test matrix (Go 1.21, 1.22)
- Build verification
- Integration tests with DynamoDB local
- License checking
- Benchmark tests

## Known Issues

### Linting Warnings

The current codebase has some non-critical linting warnings:
- Line length exceeds 120 characters in some files
- Some error returns could use better handling
- Function parameters could be optimized (pass by pointer)
- Some comments don't follow exact Go conventions

These don't prevent the code from working but should be addressed over time.

### Test Coverage

Currently, only the `config` and `context` modules have tests. The main application code needs test coverage added.

## Best Practices

1. **Always run CI before committing**: The pre-commit hook does this automatically
2. **Fix lint issues promptly**: Use `./scripts/fix-lint.sh` for common fixes
3. **Keep CI green**: Don't push code that fails CI checks
4. **Update CI scripts**: When adding new tools or checks, update the CI scripts

## Troubleshooting

### Missing Tools

If CI fails due to missing tools:
```bash
# Install Go tools
go install mvdan.cc/gofumpt@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest

# Install SAM CLI
brew install aws-sam-cli
```

### Skipping CI

In emergency situations, you can skip CI:
```bash
# Skip pre-commit
git commit --no-verify

# Skip pre-push
SKIP_CI_CHECKS=1 git push
```

**Note**: This should be used sparingly and the issues should be fixed in the next commit.

## Future Improvements

1. Add more comprehensive test coverage
2. Fix all linting warnings
3. Add performance benchmarks to CI
4. Integrate with code coverage services
5. Add dependency vulnerability scanning