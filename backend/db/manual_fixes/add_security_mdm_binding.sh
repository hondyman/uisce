#!/usr/bin/env bash
# Plan/apply wrapper for add_security_mdm_binding.sql
# Usage:
#   ./add_security_mdm_binding.sh plan
#   ./add_security_mdm_binding.sh apply
set -euo pipefail

MODE="${1:-plan}"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SQL_FILE="$DIR/add_security_mdm_binding.sql"

if [ -z "${DATABASE_URL:-}" ]; then
  echo "DATABASE_URL is not set" >&2
  exit 1
fi

if [ "$MODE" == "plan" ]; then
  echo "== PLAN =="
  psql "$DATABASE_URL" -v mode=plan -f "$SQL_FILE"
  exit 0
fi

if [ "$MODE" == "apply" ]; then
  TS=$(date +%Y%m%d_%H%M%S)
  BACKUP="/tmp/security_mdm_binding_backup_${TS}.sql"
  echo "This will INSERT a new business_object_binding row on the Security BO."
  echo "A backup will be written to: $BACKUP"
  read -r -p "Type MERGE to confirm: " CONFIRM
  if [ "$CONFIRM" != "MERGE" ]; then
    echo "Aborted (confirmation did not match)."
    exit 1
  fi
  pg_dump "$DATABASE_URL" -t public.business_object_binding > "$BACKUP"
  echo "Backup written to $BACKUP"
  echo "== APPLY =="
  psql "$DATABASE_URL" -v mode=apply -f "$SQL_FILE"
  exit 0
fi

echo "Usage: $0 [plan|apply]" >&2
exit 1
