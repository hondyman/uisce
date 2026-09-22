#!/usr/bin/env bash
# Plan/apply wrapper for fix_order_binding_driving_node.sql
# Usage:
#   ./fix_order_binding_driving_node.sh plan
#   ./fix_order_binding_driving_node.sh apply
set -euo pipefail

MODE="${1:-plan}"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SQL_FILE="$DIR/fix_order_binding_driving_node.sql"

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
  BACKUP="/tmp/order_binding_backup_${TS}.sql"
  echo "This will UPDATE driving_node_id on 1 business_object_binding row (order)."
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
