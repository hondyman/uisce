#!/bin/bash
# seed_tenant_secrets.sh
# One-shot bootstrap script that reads existing plaintext passwords from the
# connections table and writes them into Infisical at the derived secret paths.
#
# This is a transitional tool: it allows the system to continue scanning with
# existing passwords while the team migrates to Infisical-managed credentials.
# After running this script, operators should:
#   1. Verify the secrets exist in Infisical
#   2. Remove plaintext passwords from the DB (handled by the migration)
#   3. Switch to using provision_tenant_secrets.sh for new connections
#
# Usage:
#   DATABASE_URL="postgres://..." \
#   INFISICAL_PROJECT_ID="..." \
#   INFISICAL_ENVIRONMENT="dev" \
#     ./seed_tenant_secrets.sh [--dry-run]
#
set -euo pipefail

DATABASE_URL="${DATABASE_URL:?required}"
INFISICAL_PROJECT_ID="${INFISICAL_PROJECT_ID:?required}"
INFISICAL_ENVIRONMENT="${INFISICAL_ENVIRONMENT:-dev}"
DRY_RUN="${1:-}"

if [ "$DRY_RUN" = "--dry-run" ]; then
  echo "[DRY RUN] No changes will be made"
  DRY_RUN=1
else
  DRY_RUN=0
fi

INFISICAL="${INFISICAL:-infisical}"

echo "[*] Fetching connections with stored passwords from alpha DB..."
QUERY="
SELECT
  c.id,
  c.tenant_id,
  c.name,
  c.username,
  c.password,
  c.database,
  c.type,
  '/connections/tenant-' || c.tenant_id::text || '/' || lower(regexp_replace(c.name, '[^a-zA-Z0-9_-]', '_', 'g')) AS secret_path
FROM public.connections c
WHERE c.password IS NOT NULL AND c.password != ''
ORDER BY c.tenant_id, c.name;
"

ROWS="$(psql "$DATABASE_URL" -t -A -F $'\t' <<< "$QUERY")"

if [ -z "$ROWS" ]; then
  echo "[*] No connections with stored passwords found. Nothing to do."
  exit 0
fi

TOTAL=0
SEEDED=0
SKIPPED=0

while IFS=$'\t' read -r conn_id tenant_id conn_name db_user db_pass db_name conn_type secret_path; do
  TOTAL=$((TOTAL + 1))
  echo ""
  echo "[$TOTAL] Connection: ${conn_name} (${conn_type})"
  echo "    tenant:    ${tenant_id}"
  echo "    database: ${db_name}"
  echo "    user:     ${db_user:-<none>}"
  echo "    path:     ${secret_path}"

  if [ "$DRY_RUN" -eq 1 ]; then
    echo "    [DRY RUN] Would set DB_PASSWORD at ${secret_path}"
    SKIPPED=$((SKIPPED + 1))
    continue
  fi

  # Create folder
  infisical secrets folders create "${secret_path}" \
    --env="${INFISICAL_ENVIRONMENT}" \
    --projectId="${INFISICAL_PROJECT_ID}" 2>/dev/null || true

  # Set DB_USERNAME if provided
  if [ -n "${db_user}" ]; then
    infisical secrets set "DB_USERNAME=${db_user}" \
      --env="${INFISICAL_ENVIRONMENT}" \
      --path="${secret_path}" \
      --projectId="${INFISICAL_PROJECT_ID}" && \
      echo "    [+] DB_USERNAME set" || \
      echo "    [!] DB_USERNAME failed (check token permissions)"
  fi

  # Set DB_PASSWORD
  infisical secrets set "DB_PASSWORD=${db_pass}" \
    --env="${INFISICAL_ENVIRONMENT}" \
    --path="${secret_path}" \
    --projectId="${INFISICAL_PROJECT_ID}" && \
    echo "    [+] DB_PASSWORD seeded (${#db_pass} chars)" || \
    echo "    [!] DB_PASSWORD failed (check token permissions)"

  SEEDED=$((SEEDED + 1))
done <<< "$ROWS"

echo ""
echo "========================================"
echo "[*] Seed complete"
echo "    Total connections found: ${TOTAL}"
if [ "$DRY_RUN" -eq 1 ]; then
  echo "    [DRY RUN] Would have seeded: ${SEEDED}"
else
  echo "    Successfully seeded:   ${SEEDED}"
  echo "    Failed:                $((TOTAL - SEEDED))"
fi
echo ""
echo "[!] Next steps:"
echo "    1. Verify secrets in Infisical UI at the paths above"
echo "    2. Run migration: migrate -path ./backend/migrations -database \"\$DATABASE_URL\" up"
echo "    3. The migration will NULL out the plaintext password column"
echo "    4. Restart the backend server to pick up the new schema"
