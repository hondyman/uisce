#!/bin/bash

# Generate Go types from JSON schema with version awareness
# Requires: quicktype

set -e

SCHEMA_FILE="schemas/upgrade-artifacts-data.schema.json"
GO_OUTPUT="backend/internal/types/upgrade.go"

echo "Generating Go types from $SCHEMA_FILE..."

# Create output directory if it doesn't exist
mkdir -p "$(dirname "$GO_OUTPUT")"

# Generate Go types with version awareness
quicktype \
  --src "$SCHEMA_FILE" \
  --lang go \
  --out "$GO_OUTPUT" \
  --package types \
  --top-level UpgradeArtifacts

echo "Go types generated at $GO_OUTPUT"

# Format the generated Go code
gofmt -w "$GO_OUTPUT"

echo "Go code formatted"
