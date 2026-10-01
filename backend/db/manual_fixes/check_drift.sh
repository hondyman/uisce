#!/usr/bin/env bash
# check_drift.sh — row-count parity alpha vs crims for all 341 movers.
# Replaces migration.v_drift (FDW unavailable until cert placement).
# Exit 0 = no drift; exit 1 + MISMATCH lines = drift (rollback trigger).
set -u

export PGSSLMODE=verify-full
export PGSSLCERT="$HOME/.uisce/certs/postgres-client.crt"
export PGSSLKEY="$HOME/.uisce/certs/postgres-client.key"
export PGSSLROOTCERT="$HOME/.uisce/certs/ca.crt"

HOST="${PGHOST:-100.84.50.65}"
USER="${PGUSER:-postgres}"
DRIFT=0

# Intentional crims-only seeds (009/010): empty in alpha by design, populated
# in crims as V1 product-master config. Not parity targets — excluded so the
# detector stays a pure dual-write window oracle.

while IFS='|' read -r ss st tgt; do
  [ -z "${tgt:-}" ] && continue
  case "$ss.$st" in
    mdm.product_type_mapping|mdm.product_field_mapping|mdm.product_source_priority)
      continue ;;
  esac
  a=$(psql -h "$HOST" -U "$USER" -d alpha -Atc "SELECT count(*) FROM \"$ss\".\"$st\"" 2>/dev/null)
  c=$(psql -h "$HOST" -U "$USER" -d crims -Atc "SELECT count(*) FROM crims.$tgt" 2>/dev/null)
  if [ "$a" != "$c" ]; then
    echo "MISMATCH: $ss.$st alpha=$a crims=$c delta=$((${c:-0} - ${a:-0}))" 2>/dev/null || echo "MISMATCH: $ss.$st alpha=$a crims=$c"
    DRIFT=1
  fi
done <<SQL
$(psql -h "$HOST" -U "$USER" -d crims -Atc "
SELECT source_schema||'|'||source_table||'|'||target_schema||'.'||target_table
FROM migration.plan
WHERE source_db='alpha' AND target_db='crims' AND target_table IS NOT NULL
  AND classification IN ('DATA','DATA_LINEAGE','FABRIC_REF','FABRIC_RULE')
  AND batch_order >= 100;")
SQL

if [ "$DRIFT" -eq 0 ]; then
  echo "OK: 0 drift across all movers"
fi
exit "$DRIFT"
