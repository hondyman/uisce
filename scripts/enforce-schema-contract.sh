#!/bin/bash

# CI/CD script to enforce schema contract with version awareness
# Run this in CI/CD pipeline to ensure schema consistency and version bumping

set -e

SCHEMA_FILE="schemas/upgrade-artifacts-data.schema.json"

echo "🔍 Validating schema contract with version awareness..."

# Check if schema file exists
if [ ! -f "$SCHEMA_FILE" ]; then
  echo "❌ Schema file not found: $SCHEMA_FILE"
  exit 1
fi

# Validate JSON schema syntax
if command -v jq &> /dev/null; then
  echo "✅ Validating JSON schema syntax..."
  jq . "$SCHEMA_FILE" > /dev/null
else
  echo "⚠️  jq not available, skipping JSON validation"
fi

# Check if schema_version is present and valid
SCHEMA_VERSION=$(jq -r '.schema_version' "$SCHEMA_FILE")
if [ "$SCHEMA_VERSION" = "null" ] || [ -z "$SCHEMA_VERSION" ]; then
  echo "❌ schema_version field is missing from schema"
  exit 1
fi

echo "📋 Current schema version: $SCHEMA_VERSION"

# Check if changelog exists
CHANGELOG_LENGTH=$(jq '.changelog | length' "$SCHEMA_FILE")
if [ "$CHANGELOG_LENGTH" = "null" ]; then
  echo "⚠️  changelog field is missing, initializing..."
  jq '.changelog = []' "$SCHEMA_FILE" > "$SCHEMA_FILE.tmp"
  mv "$SCHEMA_FILE.tmp" "$SCHEMA_FILE"
fi

# Auto-bump schema version if schema has changed
if [ -n "$GIT_COMMIT_MESSAGE" ]; then
  echo "🔄 Checking if schema needs version bump..."
  if git diff --quiet HEAD~1 -- "$SCHEMA_FILE"; then
    echo "✅ Schema unchanged, no version bump needed"
  else
    echo "📈 Schema changed, auto-bumping version..."
    BUMP_TYPE=${BUMP_TYPE:-patch}
    CHANGE_DESC=${GIT_COMMIT_MESSAGE:-"Schema updated"}
    node scripts/bump-schema.js
  fi
fi

# Regenerate types
echo "🔄 Regenerating types from schema..."

# Generate Go types
echo "  → Generating Go types..."
bash scripts/generate-go-types.sh

# Generate TypeScript types
echo "  → Generating TypeScript types..."
bash scripts/generate-ts-types.sh

# Check if any files changed
if git diff --quiet --exit-code; then
  echo "✅ No changes detected - schema is up to date"
else
  echo "⚠️  Schema changes detected. Files have been regenerated:"
  git diff --name-only
  echo ""
  echo "📝 Please review and commit the regenerated files"
  echo "💡 Schema version has been auto-bumped to maintain contract consistency"
fi

# Validate that Go types include schema_version field
echo "🔍 Validating Go type consistency..."
if ! grep -q "SchemaVersion" backend/internal/types/upgrade.go; then
  echo "❌ SchemaVersion field missing from Go types"
  exit 1
fi

# Validate that TypeScript types include schema_version field
echo "� Validating TypeScript type consistency..."
if ! grep -q "schema_version" frontend/src/types/upgrade-generated.ts; then
  echo "❌ schema_version field missing from TypeScript types"
  exit 1
fi

echo "🎉 Schema validation complete! Version: $SCHEMA_VERSION"
