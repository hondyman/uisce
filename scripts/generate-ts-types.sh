#!/bin/bash

# Generate TypeScript types from JSON schema with version awareness
# Requires: json-schema-to-typescript (npm install -g json-schema-to-typescript)

set -e

SCHEMA_FILE="schemas/upgrade-artifacts-data.schema.json"
TS_OUTPUT="frontend/src/types/upgrade-generated.ts"

echo "Generating TypeScript types from $SCHEMA_FILE..."

# Create output directory if it doesn't exist
mkdir -p "$(dirname "$TS_OUTPUT")"

# Generate TypeScript types with version awareness
npx json-schema-to-typescript \
  "$SCHEMA_FILE" \
  --output "$TS_OUTPUT" \
  --style.singleQuote \
  --style.semi \
  --style.trailingComma es5

echo "TypeScript types generated at $TS_OUTPUT"
