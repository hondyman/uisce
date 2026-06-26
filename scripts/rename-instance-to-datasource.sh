#!/usr/bin/env bash
set -euo pipefail

# WARNING: This script performs a project-wide text replacement. Run in a branch and review changes.
# Usage: scripts/rename-instance-to-datasource.sh

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

# Replace header string
find backend -type f -name '*.go' -print0 | xargs -0 sed -i "" 's/X-Tenant-Instance-ID/X-Tenant-Datasource-ID/g'

# Replace JSON/db/query param naming occurrences
find backend -type f -name '*.go' -print0 | xargs -0 sed -i "" 's/tenant_instance_id/datasource_id/g'
find backend -type f -name '*.sql' -print0 | xargs -0 sed -i "" 's/tenant_instance_id/datasource_id/g'

# Update CORS and header lists (case-insensitive where needed)
# Note: review Vary and Access-Control-Allow-Headers updates manually after running

# Print diff summary
git add -A
git status --porcelain

echo "Done. Please review changes, run go vet and go test, and commit when satisfied."