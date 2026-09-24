#!/usr/bin/env bash
# Merges duplicate semantic/business terms that live in the same datasource (see the .sql for what it does and what it leaves alone).
#
#   bash merge_older_term_duplicates.sh          PLAN: does everything inside a transaction, prints the result and
#                                              rolls back. Changes nothing. Safe to run any time.
#   bash merge_older_term_duplicates.sh apply    shows the plan, backs up the affected tables, asks you to type MERGE,
#                                              then runs the same thing and commits. All or nothing.
set -euo pipefail
cd "$(dirname "$0")/../.."          # backend/
set -a; source .env 2>/dev/null || true; set +a
: "${DATABASE_URL:?DATABASE_URL missing}"
SQL=db/manual_fixes/merge_older_term_duplicates.sql
MODE="${1:-plan}"

echo "== PLAN (transaction rolled back, nothing changes)"
psql "$DATABASE_URL" -X -q -v mode=plan -f "$SQL"

if [ "$MODE" != "apply" ]; then
  echo; echo "That was only the plan. To apply it:  bash db/manual_fixes/merge_older_term_duplicates.sh apply"
  exit 0
fi

echo
read -r -p "Apply this now? A backup is taken first. Type MERGE: " ans
[ "$ans" = "MERGE" ] || { echo "aborted, nothing changed"; exit 1; }

TS=$(date +%Y%m%d_%H%M%S)
BK="/tmp/older_term_dups_backup_$TS.sql"
pg_dump "$DATABASE_URL" --no-owner --data-only \
  -t public.catalog_node -t public.catalog_edge -t public.abbreviations -t public.business_object_fields \
  -t public.semantic_mapping_suggestions -t public.semantic_term_validations -t public.suggestion_feedback \
  -f "$BK"
echo "backup: $BK ($(du -h "$BK" | cut -f1))"

psql "$DATABASE_URL" -X -q -v mode=apply -f "$SQL"
echo "done. Reload the glossary page to see the merged terms."
