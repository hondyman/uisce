#!/usr/bin/env bash
# Auto-generated; do not edit.  Regenerate via gen_copy_via_pgdump.py.
#
# Step-3 (copy) driver: alpha -> crims via pg_dump | psql pipes.
#
# - Topologically sorts movers by alpha's pg_constraint FK graph so parents
#   load before dependents (defensive; pg_dump --inserts wouldn't enforce FKs
#   even out of order because they're restored in Step 3b).
# - Verifies pre-existing target schemas against alpha's columns; HALT on any
#   alpha column not present identically in crims.  Additive-compatible targets
#   (crims extra columns are OK) get INSERT-only.
# - Halts soft: per-table errors are logged to stderr and the script proceeds
#   with remaining movers.  Exit code is non-zero if any row failed.

export PGSSLMODE="verify-full"
export PGSSLCERT="/Users/eganpj/.uisce/certs/postgres-client.crt"
export PGSSLKEY="/Users/eganpj/.uisce/certs/postgres-client.key"
export PGSSLROOTCERT="/Users/eganpj/.uisce/certs/ca.crt"

ALPHA_HOST=100.84.50.65
CRIMS_HOST=100.84.50.65
ALPHA_USER=postgres
CRIMS_USER=postgres
ALPHA_DB=alpha
CRIMS_DB=crims

# ----------------------------------------------------------------------------
# mismatch_helper.py: invoked on the alpha vs crims column dumps per pre-existing
# target.  Outputs <missing column spec> one per line; empty = compatible.
# ----------------------------------------------------------------------------
MISMATCH_HELPER=/Users/eganpj/GitHub/uisce/backend/db/manual_fixes/mismatch_helper.py

# FK-strip filter for pg_dump CREATE pipes (see cp_fk_strip.py)
CP_FK_STRIP=/Users/eganpj/GitHub/uisce/backend/db/manual_fixes/cp_fk_strip.py

# ----------------------------------------------------------------------------
# Step 1: load movers from migration.plan.  The plan uses batch_order=100..900
# (FABRIC -> Anchor -> Satellite -> Golden -> Logs -> Exceptions -> Drop) and
# orders rows by FK depth within batch via the importer.  We rely on that; alpha
# --inserts without FK restore would not enforce FKs anyway.  This is belt and
# braces.  Domain filtering is via DOMAIN env (PRODUCT, PARTY, RATING, ...).
# ----------------------------------------------------------------------------
fetch_movers () {
  if [ "${DOMAIN:-ALL}" = "ALL" ]; then
    PG_WHERE_CLAUSE=""
  else
    PG_WHERE_CLAUSE="AND p.domain = '$DOMAIN'"
  fi
  psql -h "$CRIMS_HOST" -U "$CRIMS_USER" -d crims -At <<SQL
  SELECT p.source_schema||'.'||p.source_table || '|' || p.target_schema||'.'||p.target_table || '|' || p.batch_order
    FROM migration.plan p
   WHERE p.source_db='alpha'
     AND p.target_db='crims'
     AND p.classification IN ('DATA','DATA_LINEAGE','FABRIC_REF','FABRIC_RULE')
     AND p.batch_order >= 100
     $PG_WHERE_CLAUSE
   ORDER BY p.batch_order, p.source_schema, p.source_table;
SQL
}

# ----------------------------------------------------------------------------
# Step 2: per-mover column comparison.  Reads alpha's information_schema into
#   /tmp/mh_alpha.txt and crims's into /tmp/mh_crims.txt, invokes mismatch_helper.
#   Pipe output: empty = compatible, non-empty = list of mismatched alpha cols.
# ----------------------------------------------------------------------------
verify_schema_compat () {
  local SRC=$1 TGT=$2
  local SRC_SCHEMA=${SRC%.*} SRC_TABLE=${SRC#*.}
  local TGT_SCHEMA=${TGT%.*} TGT_TABLE=${TGT#*.}

  psql -h "$ALPHA_HOST" -U "$ALPHA_USER" -d alpha -At <<SQL > /tmp/mh_alpha.txt
SELECT column_name || ':' || data_type || ':' || COALESCE(character_maximum_length::text,'')
  FROM information_schema.columns
 WHERE table_schema='${SRC_SCHEMA}' AND table_name='${SRC_TABLE}'
 ORDER BY ordinal_position;
SQL

  psql -h "$CRIMS_HOST" -U "$CRIMS_USER" -d crims -At <<SQL > /tmp/mh_crims.txt
SELECT column_name || ':' || data_type || ':' || COALESCE(character_maximum_length::text,'')
  FROM information_schema.columns
 WHERE table_schema='${TGT_SCHEMA}' AND table_name='${TGT_TABLE}'
 ORDER BY ordinal_position;
SQL

  python3 "$MISMATCH_HELPER"
}

# ----------------------------------------------------------------------------
# Step 3: per-mover pipe.  Decision tree per row:
#   target exists, additive-compat  -> pg_dump --data-only --inserts   -> psql
#   target missing                  -> pg_dump --schema-only           -> psql
#                                                            then data pipe
#   target exists, mismatch         -> print and skip with FAIL_LIST entry
# ----------------------------------------------------------------------------
ALL_FAILURES=()
if [ "$DOMAIN" != "ALL" ]; then
  FILTER_TERM="$DOMAIN"
else
  FILTER_TERM="ANY"
fi

MOVERS=$(fetch_movers)

while IFS='|' read -r SRC TGT BATCH; do
  [ -z "$SRC" ] && continue
  TGT_EXISTS=$(psql -h "$CRIMS_HOST" -U "$CRIMS_USER" -d crims -At -c "SELECT to_regclass('$TGT') IS NOT NULL;")

  if [ "$TGT_EXISTS" != "t" ]; then
    echo "[CREATE]  $SRC -> $TGT"
    # Filter out FK constraints (inline + standalone) and rewrite schema to target
    # via cp_fk_strip.py; FKs are re-added later in Step 3b.
    if ! pg_dump -h "$ALPHA_HOST" -U "$ALPHA_USER" -d alpha -t "$SRC" \
        --schema-only --no-acl --no-owner 2>>/tmp/cp_err.log \
        | TGT_SCHEMA="${TGT%.*}" SRC_SCHEMA="${SRC%%.*}" python3 "$CP_FK_STRIP" \
        | psql -h "$CRIMS_HOST" -U "$CRIMS_USER" -d crims -v ON_ERROR_STOP=on -At -e 2>>/tmp/cp_err.log; then
      ALL_FAILURES+=("$SRC -> $TGT (CREATE)")
      continue
    fi
  else
    # target exists: verify schema compat
    MISMATCH=$(verify_schema_compat "$SRC" "$TGT")
    if [ -n "$MISMATCH" ]; then
      echo "[MISMATCH]  $SRC -> $TGT" >&2
      echo "  alpha columns not present (or with differing type/length) in $TGT:" >&2
      echo "$MISMATCH" | sed 's/^/    /' >&2
      ALL_FAILURES+=("$SRC -> $TGT (MISMATCH)")
      continue
    fi
    echo "[INSERT-ONLY]  $SRC -> $TGT (additive-compat)"
  fi

  # data pipe
  echo "[DATA]  $SRC -> $TGT"
  ROWS=$(pg_dump -h "$ALPHA_HOST" -U "$ALPHA_USER" -d alpha -t "$SRC" \
    --data-only --inserts --no-acl --no-owner 2>>/tmp/cp_err.log \
    | sed -E 's|INSERT INTO [^.]+\.([^ ]+) |INSERT INTO '"${TGT%.*}"'.\1 |' \
    | psql -h "$CRIMS_HOST" -U "$CRIMS_USER" -d crims -v ON_ERROR_STOP=off -At -e 2>>/tmp/cp_err.log | wc -l)
  psql -h "$CRIMS_HOST" -U "$CRIMS_USER" -d crims -At -c "INSERT INTO migration.progress (plan_id, status, rows_copied, started_at, finished_at) SELECT id, 'done', $ROWS, now(), now() FROM migration.plan WHERE source_db='alpha' AND source_schema||'.'||source_table='$SRC' ON CONFLICT (plan_id) DO UPDATE SET status='done', rows_copied=$ROWS, finished_at=now();"
done <<< "$MOVERS"

# Final report
if [ "${#ALL_FAILURES[@]}" -eq 0 ]; then
  echo "OK: all movers copied successfully."
  echo "Next: re-add FKs (skipped during pipe).  See gen_fk_recreate.py [next step]."
else
  echo "FAILURES: ${#ALL_FAILURES[@]}" >&2
  printf ' - %s\n' "${ALL_FAILURES[@]}" >&2
  exit 1
fi
