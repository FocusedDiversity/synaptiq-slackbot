#!/bin/bash

# Script to check licenses of dependencies

set -e

echo "Checking licenses for dependencies..."
echo ""

# Install go-licenses if not present
if ! command -v go-licenses &> /dev/null; then
    echo "Installing go-licenses..."
    go install github.com/google/go-licenses@latest
fi

# Function to check licenses for a module
check_module() {
    local module=$1
    echo "Checking $module..."
    
    # Generate license report, ignoring our own modules
    go-licenses report "$module" \
        --ignore github.com/synaptiq/standup-bot \
        --ignore github.com/synaptiq/standup-bot/config \
        --ignore github.com/synaptiq/standup-bot/context \
        2>/dev/null | sort | uniq > /tmp/licenses_report.txt
    
    # Check for forbidden licenses
    local forbidden_found=false
    while IFS=, read -r package url license; do
        case "$license" in
            "AGPL-3.0"|"GPL-2.0"|"GPL-3.0"|"LGPL-2.1"|"LGPL-3.0")
                echo "  ❌ FORBIDDEN: $package uses $license"
                forbidden_found=true
                ;;
            "Apache-2.0"|"BSD-2-Clause"|"BSD-3-Clause"|"ISC"|"MIT"|"MPL-2.0"|"Unlicense")
                # Allowed licenses - do nothing
                ;;
            *)
                echo "  ⚠️  UNKNOWN: $package uses $license (needs review)"
                ;;
        esac
    done < /tmp/licenses_report.txt
    
    if [ "$forbidden_found" = true ]; then
        return 1
    fi
    
    echo "  ✅ All licenses approved"
    return 0
}

# Check each command
echo "=== Checking Lambda Functions ==="
for cmd in cmd/*; do
    if [ -d "$cmd" ]; then
        check_module "./$cmd" || exit 1
    fi
done

echo ""
echo "=== License Summary ==="

# Generate full report
echo "Generating full license report..."
go-licenses report ./... \
    --ignore github.com/synaptiq/standup-bot \
    --ignore github.com/synaptiq/standup-bot/config \
    --ignore github.com/synaptiq/standup-bot/context \
    2>/dev/null | sort | uniq > licenses.txt

# Count licenses by type
echo ""
echo "License distribution:"
cut -d, -f3 licenses.txt | sort | uniq -c | sort -rn

echo ""
echo "Full license report saved to licenses.txt"

# Clean up
rm -f /tmp/licenses_report.txt

echo ""
echo "✅ License check complete!"