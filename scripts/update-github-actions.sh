#!/bin/bash

# Script to update GitHub Actions to their latest versions

set -e

echo "Updating GitHub Actions to latest versions..."
echo ""

# Create a temporary directory for backups
BACKUP_DIR=".github/workflows/backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Backup all workflow files
cp .github/workflows/*.yml "$BACKUP_DIR/"
echo "Backed up workflows to $BACKUP_DIR"
echo ""

# Update CI workflow
echo "Updating ci.yml..."
sed -i.bak \
    -e 's|golangci/golangci-lint-action@v3|golangci/golangci-lint-action@v6|g' \
    -e 's|codecov/codecov-action@v3|codecov/codecov-action@v5|g' \
    .github/workflows/ci.yml

# Update security workflow
echo "Updating security.yml..."
sed -i.bak \
    -e 's|github/codeql-action/init@v2|github/codeql-action/init@v3|g' \
    -e 's|github/codeql-action/autobuild@v2|github/codeql-action/autobuild@v3|g' \
    -e 's|github/codeql-action/analyze@v2|github/codeql-action/analyze@v3|g' \
    -e 's|github/codeql-action/upload-sarif@v2|github/codeql-action/upload-sarif@v3|g' \
    -e 's|aquasecurity/trivy-action@master|aquasecurity/trivy-action@0.28.0|g' \
    -e 's|bridgecrewio/checkov-action@master|bridgecrewio/checkov-action@v12|g' \
    -e 's|securego/gosec@master|securego/gosec@v2.21.4|g' \
    -e 's|trufflesecurity/trufflehog@main|trufflesecurity/trufflehog@v3|g' \
    -e 's|dependency-check/Dependency-Check_Action@main|dependency-check/Dependency-Check_Action@v1|g' \
    .github/workflows/security.yml

# Clean up backup files
rm -f .github/workflows/*.yml.bak


echo "GitHub Actions update complete!"
echo ""
echo "Summary of changes:"
echo "- Updated CodeQL actions from v2 to v3"
echo "- Updated golangci-lint-action from v3 to v6"
echo "- Updated codecov-action from v3 to v5"
echo "- Updated security scanners to specific versions instead of 'master'"
echo "- Updated trufflesecurity/trufflehog from main to v3"
echo ""
echo "Next steps:"
echo "1. Review the changes with: git diff .github/workflows/"
echo "2. Test workflows locally if possible"
echo "3. Commit the changes"
echo ""
echo "Note: Backup files are in $BACKUP_DIR"