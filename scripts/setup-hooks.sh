#!/usr/bin/env bash
# Setup git hooks for the project

set -euo pipefail

HOOKS_DIR=".git/hooks"

echo "Setting up git hooks..."

# Create pre-commit hook
cat > "$HOOKS_DIR/pre-commit" << 'EOF'
#!/bin/sh
# Pre-commit hook that runs basic CI checks

# Allow skipping with SKIP_CI_CHECKS=1 or --no-verify
if [ "${SKIP_CI_CHECKS:-}" = "1" ]; then
    echo "Skipping CI checks (SKIP_CI_CHECKS=1)"
    exit 0
fi

echo "Running pre-commit checks..."

# Run the summary CI script for speed
if ./scripts/ci-summary.sh; then
    echo "Pre-commit checks passed!"
    exit 0
else
    echo
    echo "Pre-commit checks failed!"
    echo "To bypass (not recommended), use: git commit --no-verify"
    echo "Or set: SKIP_CI_CHECKS=1"
    exit 1
fi
EOF

chmod +x "$HOOKS_DIR/pre-commit"
echo "✅ Created pre-commit hook"

# Create pre-push hook
cat > "$HOOKS_DIR/pre-push" << 'EOF'
#!/bin/sh
# Pre-push hook that runs full CI checks

# Allow skipping with SKIP_CI_CHECKS=1
if [ "${SKIP_CI_CHECKS:-}" = "1" ]; then
    echo "Skipping CI checks (SKIP_CI_CHECKS=1)"
    exit 0
fi

echo "Running pre-push CI checks..."
echo "This may take a minute..."

# Run the full CI script
if ./scripts/ci-local.sh; then
    echo "Pre-push checks passed!"
    exit 0
else
    echo
    echo "Pre-push checks failed!"
    echo "Please fix the issues before pushing."
    echo "To bypass (not recommended), set: SKIP_CI_CHECKS=1"
    exit 1
fi
EOF

chmod +x "$HOOKS_DIR/pre-push"
echo "✅ Created pre-push hook"

echo
echo "Git hooks installed successfully!"
echo
echo "Hooks behavior:"
echo "- pre-commit: Runs quick checks (build, format, security)"
echo "- pre-push: Runs full CI suite including linting"
echo
echo "To skip hooks temporarily:"
echo "- For commit: git commit --no-verify"
echo "- For push: SKIP_CI_CHECKS=1 git push"
echo
echo "To uninstall hooks:"
echo "- rm .git/hooks/pre-commit .git/hooks/pre-push"