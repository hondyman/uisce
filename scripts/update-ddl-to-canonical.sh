#!/usr/bin/env bash
# Script to update DDL/seed files to canonical table names (preview mode by default)
# Usage: ./scripts/update-ddl-to-canonical.sh [--apply]

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
APPLY=false
if [ "${1:-}" = "--apply" ]; then
  APPLY=true
fi

FILES=(
  "$ROOT_DIR/backend/totalddl.sql"
  "$ROOT_DIR/backend/init-db.sql"
)

for f in "${FILES[@]}"; do
  if [ ! -f "$f" ]; then
    echo "Skipping missing $f"
    continue
  fi
  echo "Processing $f"
  tmp="${f}.canonical.tmp"
  # Replace singular catalog_edge_type with catalog_edge_types
  sed -E "s/\bcatalog_edge_type\b/catalog_edge_types/g" "$f" > "$tmp"
  if [ "$APPLY" = true ]; then
    mv "$tmp" "$f"
    echo "Applied canonicalization to $f"
  else
    echo "Preview written to $tmp (run with --apply to overwrite)"
  fi
done

echo "Done. Review the .tmp files before applying."
