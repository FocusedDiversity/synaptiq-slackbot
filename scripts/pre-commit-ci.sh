#!/usr/bin/env bash
# Pre-commit CI runner
# This is a lightweight wrapper for git pre-commit hooks

set -euo pipefail

# Check if CI checks should be skipped
if [ "${SKIP_CI_CHECKS:-}" = "1" ]; then
    echo "Skipping CI checks (SKIP_CI_CHECKS=1)"
    exit 0
fi

# Check for --no-verify flag
if [ "${GIT_PARAMS:-}" = "--no-verify" ]; then
    echo "Skipping CI checks (--no-verify)"
    exit 0
fi

# Run the full CI check suite
echo "Running pre-commit CI checks..."
echo "(To skip, use 'git commit --no-verify' or set SKIP_CI_CHECKS=1)"
echo

# Run the CI checks
if ./scripts/ci-local.sh; then
    exit 0
else
    echo
    echo "Commit blocked due to CI check failures."
    echo "To bypass (not recommended), use: git commit --no-verify"
    exit 1
fi