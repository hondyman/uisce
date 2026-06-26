#!/usr/bin/env bash
set -euo pipefail

echo "🔍 Scanning for Jest tests..."

# Patterns that represent Jest tests
PATTERNS=(
  "src/**/*.test.ts"
  "src/**/*.test.tsx"
  "src/**/*.spec.ts"
  "src/**/*.spec.tsx"
  "src/**/__tests__/*"
)

DEST="src/__jest__"

mkdir -p "$DEST"

for pattern in "${PATTERNS[@]}"; do
  for file in $(ls $pattern 2>/dev/null || true); do
    # Skip Vitest tests
    if [[ "$file" == src/vitest/* ]]; then
      continue
    fi

    echo "📦 Moving $file → $DEST/"
    mv "$file" "$DEST/"
  done
done

echo "✅ Jest tests moved to $DEST/"
echo "You can now run Jest and Vitest independently."