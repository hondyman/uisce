#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

# Only modify source files under frontend/src
find frontend/src -type f \( -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.jsx' \) -print0 | xargs -0 sed -i "" 's/X-Tenant-Instance-ID/X-Tenant-Datasource-ID/g'
find frontend/src -type f \( -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.jsx' \) -print0 | xargs -0 sed -i "" 's/tenant_instance_id/datasource_id/g'

# Update localStorage keys if present (selected_datasource usually fine)

echo "Frontend src replacements complete. Please run the build and run tests."