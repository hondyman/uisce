#!/usr/bin/env bash
# apply_crims_rating_slice.sh — Vertical slice 1: Rating
#
# End-to-end run on a fresh crims database:
#   * 0010_staging_rating_incoming DDL
#   * 0011_staging_load_run_crims DDL
#   * 0012_mdm_rating_internal_override DDL
#   * 0014..0017 DDL (party, rating reference, rating, rating_action)
#   * 019..025 seeds (5 rating references + party + 2 override action_types)
#   * cmd/rating-load master load (initial 7 CSV rows → mdm.rating)
#   * Override flow: pending → approved → replace → revert
#   * 4-row action trail assertion + rank-consistency assertion
#   * RLS enforcement test (non-superuser sees only own tenant)
#
# Required env (defaults match dev):
#   ALPHA_DSN     postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable
#   CRIMS_DSN     postgres://postgres:postgres@localhost:5432/crims?sslmode=disable
#   TENANT_ID     99e99e99-99e9-49e9-89e9-99e99e99e999
#   BACKEND_URL   http://localhost:8080
#   SKIP_API_STEPS=true   bypass steps 3-6 (JWT, upload, spec, trigger) and
#                         insert staging rows directly from the CSV. Use this
#                         on a dev box without a running backend.
#
# Apply + verify (fresh DB):
#   psql -h localhost -U postgres -c "DROP DATABASE IF EXISTS crims_fresh WITH (FORCE);"
#   psql -h localhost -U postgres -c "CREATE DATABASE crims_fresh;"
#   SKIP_API_STEPS=true ./apply_crims_rating_slice.sh

set -euo pipefail

ALPHA_DSN="${ALPHA_DSN:-postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable}"
CRIMS_DSN="${CRIMS_DSN:-postgres://postgres@localhost:5432/crims?sslmode=disable}"
TENANT_ID="${TENANT_ID:-99e99e99-99e9-49e9-89e9-99e99e99e999}"
BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"
SKIP_API_STEPS="${SKIP_API_STEPS:-false}"
RUN_REF="20261112-rating-bloomberg-01"
ISSACME_LEI='529900T8BM49AURSDO55'

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BACKEND_DIR="$ROOT/backend"
SEEDS_DIR="$BACKEND_DIR/db/seeds_crims"
CSV_PATH="$ROOT/seed/rating_bloomberg.csv"

step() { printf '\n\033[1;34m▶ %s\033[0m\n' "$*"; }
ok()   { printf '  \033[1;32m✓ %s\033[0m\n' "$*"; }
fail() { printf '  \033[1;31m✗ %s\033[0m\n' "$*" >&2; exit 1; }

# ── RLS test-role lifecycle ─────────────────────────────────────────────
# Step 10 needs a NOLOGIN role with SELECT grants. The role is cluster-level
# and its table grants are per-database, so a run that dies between step 10
# and teardown leaks it (the next run's IF NOT EXISTS then silently skips
# creation and teardown never learns it owns the role). Step 10 sets
# TEARDOWN_RLS_ROLE; the EXIT trap drops the role whether the run passed or
# aborted. Not conditional on pre-existence: the role survives DROP DATABASE,
# so "created it this run" is the wrong test — it would never fire on a retry.
RLS_ROLE="crims_tenant_user"
TEARDOWN_RLS_ROLE=0

# The DSN's own role (postgres) is not CREATEROLE on homebrew PG, and REVOKE
# only works for the grantor — both need the CREATEROLE superuser (here the
# OS user), not the DSN role.
ADMIN_DSN="$(printf '%s' "$CRIMS_DSN" | sed -E 's|//[^@]*@|//'"$(id -un)"'@|')"

cleanup_role() {
    [ "$TEARDOWN_RLS_ROLE" = "1" ] || return 0
    TEARDOWN_RLS_ROLE=0
    # psql keeps going after a failed statement (no ON_ERROR_STOP), so DROP
    # still runs when an early revoke hits a schema that step 1 never got to.
    psql "$ADMIN_DSN" <<SQL >/dev/null 2>&1 || true
ALTER DEFAULT PRIVILEGES IN SCHEMA mdm REVOKE SELECT ON TABLES FROM $RLS_ROLE;
REVOKE ALL ON SCHEMA mdm FROM $RLS_ROLE;
REVOKE ALL ON ALL TABLES IN SCHEMA mdm FROM $RLS_ROLE;
REVOKE ALL ON SCHEMA public FROM $RLS_ROLE;
DROP ROLE IF EXISTS $RLS_ROLE;
SQL
    # Judge by outcome, not psql's exit code: a no-op revoke still exits 3
    # even when the DROP below succeeded.
    if psql "$ADMIN_DSN" -Atq \
            -c "SELECT 1 FROM pg_roles WHERE rolname = '$RLS_ROLE'" 2>/dev/null | grep -q 1; then
        printf '  \033[1;31m! %s leaked — grants may live in another database. Drop it with:\n    psql -U %s -d postgres -c "DROP ROLE IF EXISTS %s"\033[0m\n' \
            "$RLS_ROLE" "$(id -un)" "$RLS_ROLE" >&2
    fi
}
trap cleanup_role EXIT

# ── 1. Apply DDL + seeds ────────────────────────────────────────────────
step "1/14 Apply DDL + seeds to crims"

# Apply order: 0011 first (creates staging._load_run + _mapping_error),
# then 0010 (staging.rating_incoming FKs to _load_run), then the rest.
for m in 0011_staging_load_run_crims \
         0010_staging_rating_incoming \
         0012_mdm_rating_internal_override \
         0014_mdm_party_crims \
         0015_mdm_rating_reference_crims \
         0016_mdm_rating_crims \
         0017_mdm_rating_action_crims; do
    f="$BACKEND_DIR/db/crims/${m}.up.sql"
    [ -f "$f" ] || fail "missing migration: $f"
    if ! psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f "$f" >/dev/null; then
        fail "migration $f failed"
    fi
    ok "$(basename "$f")"
done

for s in 019 020 021 022 023 024 025; do
    f=$(ls "$SEEDS_DIR/${s}_"*.sql 2>/dev/null | head -1)
    [ -n "$f" ] || fail "missing seed: ${s}_*.sql in $SEEDS_DIR"
    if ! psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f "$f" >/dev/null; then
        fail "seed $f failed"
    fi
    ok "$(basename "$f")"
done

# ── 2. Build cmd/rating-load ─────────────────────────────────────────────
step "2/14 Build cmd/rating-load"
( cd "$BACKEND_DIR" && go build -o /tmp/rating-load ./cmd/rating-load ) \
    && ok "/tmp/rating-load built"

# ── 3. Stage the CSV ────────────────────────────────────────────────────
step "3/14 Stage CSV (SKIP_API_STEPS=$SKIP_API_STEPS)"

[ -f "$CSV_PATH" ] || fail "missing $CSV_PATH"

# 3a. Create a staging._load_run row. SET app.current_tenant (session, not LOCAL)
# before the INSERT so RLS passes — SET LOCAL outside a transaction is a
# no-op with a warning.
LOAD_RUN_ID=$(psql "$CRIMS_DSN" -Atq <<SQL
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
INSERT INTO staging._load_run (id, source_system_cd, domain, run_ref, tenant_id, status)
VALUES (gen_random_uuid(), 'BLOOMBERG', 'RATING', '$RUN_REF', '$TENANT_ID'::uuid, 'RUNNING')
RETURNING id;
COMMIT;
SQL
)
# Filter out psql status lines (BEGIN, SET, COMMIT, INSERT 0 1) that bleed through.
LOAD_RUN_ID=$(echo "$LOAD_RUN_ID" | grep -E '^[0-9a-f-]{36}$' | head -1)
[ -n "$LOAD_RUN_ID" ] || fail "could not create _load_run (got: $LOAD_RUN_ID)"
[ -n "$LOAD_RUN_ID" ] || fail "could not create _load_run (got: $LOAD_RUN_ID)"
ok "load_run_id=$LOAD_RUN_ID"

if [ "$SKIP_API_STEPS" = "true" ]; then
    # Direct insert: read the CSV and INSERT each row with tenant_id +
    # source_system_id + load_run_id + the CSV columns.
    SOURCE_SYSTEM_ID="11111111-1111-1111-1111-111111111111"
    ok "loading 7 staging rows directly from CSV (SKIP_API_STEPS=true)"
    tail -n +2 "$CSV_PATH" | nl -ba -w1 -s, | while IFS=, read -r i rpt rpk agc rtc rsc rv oc wc cwd ice rd ef et sd; do
        psql "$CRIMS_DSN" -Atq <<SQL >/dev/null
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
INSERT INTO staging.rating_incoming (
    id, source_system_id, source_system_cd, source_row_id, load_run_id, tenant_id,
    rated_party_type, rated_party_key,
    agency_cd, rating_type_cd, rating_scale_cd, rating_value,
    outlook_cd, watch_cd, credit_watch_direction, is_credit_event,
    rating_date, effective_from, effective_to, source_document
) VALUES (
    gen_random_uuid(), '$SOURCE_SYSTEM_ID'::uuid, 'BLOOMBERG', 'bbg-row-$(printf %03d $i)',
    '$LOAD_RUN_ID'::uuid, '$TENANT_ID'::uuid,
    '$rpt', '$rpk', '$agc', '$rtc', '$rsc', '$rv',
    $([ -n "$oc"  ] && echo "'$oc'"  || echo 'NULL'),
    $([ -n "$wc"  ] && echo "'$wc'"  || echo 'NULL'),
    $([ -n "$cwd" ] && echo "'$cwd'" || echo 'NULL'),
    $ice,
    '$rd', '$ef', $([ -n "$et" ] && echo "'$et'" || echo 'NULL'),
    $([ -n "$sd" ] && echo "'$sd'" || echo 'NULL')
);
COMMIT;
SQL
    done
    ok "7 staging rows inserted"

    # Skip steps 4-6: no backend needed
    step "4/14 Upload CSV (SKIPPED — SKIP_API_STEPS=true)"
    ok "skipped"
    step "5/14 Register spec (SKIPPED)"
    ok "skipped"
    step "6/14 Trigger run (SKIPPED)"
    ok "skipped"
else
    # ── 3b. Mint JWT ──────────────────────────────────────────────────
    step "4/14 Mint dev JWT"
    ( cd "$BACKEND_DIR" && go run ./cmd/devjwt --tenant "$TENANT_ID" --user testuser@example.com ) \
        > /tmp/rating_slice.jwt
    JWT=$(tail -n1 /tmp/rating_slice.jwt)
    [ -n "$JWT" ] || fail "JWT mint failed"
    ok "JWT minted"

    # ── 4. Upload CSV ───────────────────────────────────────────────
    step "5/14 Upload CSV"
    UPLOAD_NAME="rating_bloomberg.csv"
    curl -fsS -X POST \
        "${BACKEND_URL}/api/data-pipelines/files/upload?name=${UPLOAD_NAME}" \
        -H "Authorization: Bearer $JWT" \
        -H "X-Tenant-ID: $TENANT_ID" \
        --data-binary "@$CSV_PATH" >/dev/null \
        && ok "uploaded $UPLOAD_NAME"

    # ── 5. Register spec ───────────────────────────────────────────
    step "6/14 Register spec"
    SPEC_FILE="$ROOT/seed/specs/rating_bloomberg.json"
    jq --arg uri "uploads/${UPLOAD_NAME}" \
       '.nodes[0].config.uri = $uri' "$SPEC_FILE" > /tmp/rating_spec.json
    NAME="Rating Ingest → staging.rating_incoming (file) [BLOOMBERG]"
    BODY=$(jq --arg name "$NAME" \
              --argjson spec "$(cat /tmp/rating_spec.json)" \
              '{name: $name, description: "Vertical slice 1: rating.", spec: $spec}')
    CREATE_RESP=$(curl -fsS -X POST "${BACKEND_URL}/api/data-pipelines/" \
        -H "Authorization: Bearer $JWT" \
        -H "X-Tenant-ID: $TENANT_ID" \
        -H "Content-Type: application/json" \
        -d "$BODY")
    SPEC_ID=$(echo "$CREATE_RESP" | jq -r '.ID // .id // empty')
    [ -n "$SPEC_ID" ] || fail "spec create returned no id: $CREATE_RESP"
    ok "spec id: $SPEC_ID"

    # ── 6. Trigger run ─────────────────────────────────────────────
    step "6/14 Trigger run (cont'd)"
    RUN_RESP=$(curl -fsS -X POST "${BACKEND_URL}/api/data-pipelines/${SPEC_ID}/runs" \
        -H "Authorization: Bearer $JWT" -H "X-Tenant-ID: $TENANT_ID")
    RUN_ID=$(echo "$RUN_RESP" | jq -r '.run_id // .id // empty')
    [ -n "$RUN_ID" ] || fail "no run_id in $RUN_RESP"
    for i in $(seq 1 60); do
        sleep 2
        STATUS=$(curl -fsS "${BACKEND_URL}/api/data-pipelines/runs/${RUN_ID}" \
            -H "Authorization: Bearer $JWT" -H "X-Tenant-ID: $TENANT_ID" \
            | jq -r '.status // empty')
        case "$STATUS" in
            COMPLETED|completed) ok "run completed in ${i} polls"; break ;;
            FAILED|failed)       fail "run failed; check backend logs for run $RUN_ID" ;;
        esac
        [ "$i" -eq 60 ] && fail "run did not complete in 120s"
    done
fi

# ── 7. Master load (initial) ────────────────────────────────────────────
step "7/14 Run cmd/rating-load (initial — no overrides yet)"
ALPHA_DSN="$ALPHA_DSN" CRIMS_DSN="$CRIMS_DSN" TENANT_ID="$TENANT_ID" \
    /tmp/rating-load || fail "rating-load failed"
ok "rating-load returned"

# ── 7.5a. Insert pending override ─────────────────────────────────────
step "7.5a/14 Insert pending override (loader should ignore it)"
psql "$CRIMS_DSN" -1 <<SQL >/dev/null
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
INSERT INTO mdm.rating_internal_override
    (tenant_id, rated_party_type, rated_party_key,
     rating_value, reason, approval_status,
     requested_by, effective_from, effective_to, is_active)
VALUES
    ('$TENANT_ID'::uuid, 'ISSUER', '$ISSACME_LEI',
     'BBB+', 'credit committee override A- → BBB+',
     'pending',
     '00000000-0000-0000-0000-000000000001'::uuid,
     CURRENT_DATE, NULL, true);
COMMIT;
SQL
ALPHA_DSN="$ALPHA_DSN" CRIMS_DSN="$CRIMS_DSN" TENANT_ID="$TENANT_ID" \
    /tmp/rating-load || fail "rating-load (pending) failed"
PENDING_VALUE=$(psql "$CRIMS_DSN" -Atq <<SQL
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
SELECT r.rating_value FROM mdm.rating r
JOIN mdm.rating_agency a ON a.id = r.agency_id
WHERE r.tenant_id = '$TENANT_ID'::uuid AND r.is_latest
  AND a.agency_cd = 'INTERNAL'
  AND r.rated_party_type = 'ISSUER'
  AND r.rated_party_key = '$ISSACME_LEI';
COMMIT;
SQL
)
# Filter out psql status lines that bleed through with `-q`.
PENDING_VALUE=$(echo "$PENDING_VALUE" | grep -vE '^(BEGIN|SET|COMMIT|ROLLBACK|INSERT|UPDATE|DELETE)' | head -1)
[ "$PENDING_VALUE" = "A-" ] \
    && ok "pending override ignored, master still A-" \
    || fail "pending override applied unexpectedly: $PENDING_VALUE"

# ── 7.5b. Approve override ──────────────────────────────────────────
step "7.5b/14 Approve override (loader should apply it)"
psql "$CRIMS_DSN" -1 <<SQL >/dev/null
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
UPDATE mdm.rating_internal_override
SET approval_status = 'approved',
    approved_by = '00000000-0000-0000-0000-000000000002'::uuid,
    approved_at = now()
WHERE tenant_id = '$TENANT_ID'::uuid
  AND rated_party_type = 'ISSUER'
  AND rated_party_key = '$ISSACME_LEI'
  AND approval_status = 'pending';
-- Reset loaded_at_master on the INTERNAL staging row so the loader
-- picks it up again with the approved override in place.
UPDATE staging.rating_incoming
SET loaded_at_master = NULL
WHERE tenant_id = '$TENANT_ID'::uuid
  AND rated_party_key = '$ISSACME_LEI'
  AND agency_cd = 'INTERNAL'
  AND is_valid = true;
COMMIT;
SQL
ALPHA_DSN="$ALPHA_DSN" CRIMS_DSN="$CRIMS_DSN" TENANT_ID="$TENANT_ID" \
    /tmp/rating-load || fail "rating-load (approved) failed"
APPROVED_VALUE=$(psql "$CRIMS_DSN" -Atq <<SQL
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
SELECT r.rating_value FROM mdm.rating r
JOIN mdm.rating_agency a ON a.id = r.agency_id
WHERE r.tenant_id = '$TENANT_ID'::uuid AND r.is_latest
  AND a.agency_cd = 'INTERNAL'
  AND r.rated_party_type = 'ISSUER'
  AND r.rated_party_key = '$ISSACME_LEI';
COMMIT;
SQL
)
APPROVED_VALUE=$(echo "$APPROVED_VALUE" | grep -vE '^(BEGIN|SET|COMMIT|ROLLBACK|INSERT|UPDATE|DELETE)' | head -1)
[ "$APPROVED_VALUE" = "BBB+" ] \
    && ok "approved override applied, master now BBB+" \
    || fail "approved override not applied: $APPROVED_VALUE"

RANK_OK=$(psql "$CRIMS_DSN" -Atq <<SQL
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
SELECT CASE WHEN r.rating_rank = ra.new_rank THEN 'OK' ELSE 'MISMATCH' END
FROM mdm.rating r
JOIN mdm.rating_action ra ON ra.rating_id = r.id
JOIN mdm.rating_agency a ON a.id = r.agency_id
WHERE r.is_latest AND a.agency_cd = 'INTERNAL'
  AND r.rated_party_type = 'ISSUER'
  AND r.rated_party_key = '$ISSACME_LEI'
ORDER BY ra.action_timestamp DESC, ra.id DESC LIMIT 1;
COMMIT;
SQL
)
[ "$RANK_OK" = "OK" ] \
    && ok "step 7.5b: rating_rank matches new_rank" \
    || fail "step 7.5b: rating_rank != new_rank (Bug 1 regression)"

# ── 7.6. Second approved override (must fail) ────────────────────────
step "7.6/14 Insert second approved override (expect unique violation)"
if psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 <<SQL 2>/tmp/rating_slice.err
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
INSERT INTO mdm.rating_internal_override
    (tenant_id, rated_party_type, rated_party_key,
     rating_value, reason, approval_status,
     requested_by, effective_from, effective_to, is_active)
VALUES
    ('$TENANT_ID'::uuid, 'ISSUER', '$ISSACME_LEI',
     'A', 'second override attempt — should be rejected',
     'approved',
     '00000000-0000-0000-0000-000000000003'::uuid,
     CURRENT_DATE, NULL, true);
COMMIT;
SQL
then
    fail "second approved override was accepted — partial unique index is broken"
else
    if grep -q "uq_rating_internal_override_active" /tmp/rating_slice.err; then
        ok "second approved override rejected by uq_rating_internal_override_active"
    else
        fail "second override failed but error didn't mention the unique index: $(cat /tmp/rating_slice.err)"
    fi
fi

# ── 7.7. Close first, insert replacement, approve ─────────────────────
step "7.7/14 Close first override (is_active=false), insert second, approve, re-run"
psql "$CRIMS_DSN" -1 <<SQL >/dev/null
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
UPDATE mdm.rating_internal_override
SET is_active = false, updated_at = now()
WHERE tenant_id = '$TENANT_ID'::uuid
  AND rated_party_type = 'ISSUER'
  AND rated_party_key = '$ISSACME_LEI'
  AND approval_status = 'approved'
  AND is_active;
INSERT INTO mdm.rating_internal_override
    (tenant_id, rated_party_type, rated_party_key,
     rating_value, reason, approval_status,
     requested_by, effective_from, effective_to, is_active)
VALUES
    ('$TENANT_ID'::uuid, 'ISSUER', '$ISSACME_LEI',
     'BB+', 'credit committee override BBB+ → BB+',
     'pending',
     '00000000-0000-0000-0000-000000000004'::uuid,
     CURRENT_DATE, NULL, true);
UPDATE mdm.rating_internal_override
SET approval_status = 'approved',
    approved_by = '00000000-0000-0000-0000-000000000005'::uuid,
    approved_at = now()
WHERE tenant_id = '$TENANT_ID'::uuid
  AND rated_party_type = 'ISSUER'
  AND rated_party_key = '$ISSACME_LEI'
  AND approval_status = 'pending';
UPDATE staging.rating_incoming
SET loaded_at_master = NULL
WHERE tenant_id = '$TENANT_ID'::uuid
  AND rated_party_key = '$ISSACME_LEI'
  AND agency_cd = 'INTERNAL'
  AND is_valid = true;
COMMIT;
SQL
ALPHA_DSN="$ALPHA_DSN" CRIMS_DSN="$CRIMS_DSN" TENANT_ID="$TENANT_ID" \
    /tmp/rating-load || fail "rating-load (replacement) failed"
REPLACED_VALUE=$(psql "$CRIMS_DSN" -Atq <<SQL
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
SELECT r.rating_value FROM mdm.rating r
JOIN mdm.rating_agency a ON a.id = r.agency_id
WHERE r.tenant_id = '$TENANT_ID'::uuid AND r.is_latest
  AND a.agency_cd = 'INTERNAL'
  AND r.rated_party_type = 'ISSUER'
  AND r.rated_party_key = '$ISSACME_LEI';
COMMIT;
SQL
)
REPLACED_VALUE=$(echo "$REPLACED_VALUE" | grep -vE '^(BEGIN|SET|COMMIT|ROLLBACK|INSERT|UPDATE|DELETE)' | head -1)
[ "$REPLACED_VALUE" = "BB+" ] \
    && ok "replacement override applied, master now BB+" \
    || fail "replacement override not applied: $REPLACED_VALUE"

RANK_OK=$(psql "$CRIMS_DSN" -Atq <<SQL
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
SELECT CASE WHEN r.rating_rank = ra.new_rank THEN 'OK' ELSE 'MISMATCH' END
FROM mdm.rating r
JOIN mdm.rating_action ra ON ra.rating_id = r.id
JOIN mdm.rating_agency a ON a.id = r.agency_id
WHERE r.is_latest AND a.agency_cd = 'INTERNAL'
  AND r.rated_party_type = 'ISSUER'
  AND r.rated_party_key = '$ISSACME_LEI'
ORDER BY ra.action_timestamp DESC, ra.id DESC LIMIT 1;
COMMIT;
SQL
)
[ "$RANK_OK" = "OK" ] \
    && ok "step 7.7: rating_rank matches new_rank" \
    || fail "step 7.7: rating_rank != new_rank (Bug 1 regression)"

# ── 7.8. Close replacement, expect revert ────────────────────────────
step "7.8/14 Close replacement, re-run (expect OVERRIDE_EXPIRED revert to A-)"
psql "$CRIMS_DSN" -1 <<SQL >/dev/null
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
UPDATE mdm.rating_internal_override
SET is_active = false, updated_at = now()
WHERE tenant_id = '$TENANT_ID'::uuid
  AND rated_party_type = 'ISSUER'
  AND rated_party_key = '$ISSACME_LEI'
  AND approval_status = 'approved'
  AND is_active;
UPDATE staging.rating_incoming
SET loaded_at_master = NULL
WHERE tenant_id = '$TENANT_ID'::uuid
  AND rated_party_key = '$ISSACME_LEI'
  AND agency_cd = 'INTERNAL'
  AND is_valid = true;
COMMIT;
SQL
ALPHA_DSN="$ALPHA_DSN" CRIMS_DSN="$CRIMS_DSN" TENANT_ID="$TENANT_ID" \
    /tmp/rating-load || fail "rating-load (revert) failed"
REVERTED_VALUE=$(psql "$CRIMS_DSN" -Atq <<SQL
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
SELECT r.rating_value FROM mdm.rating r
JOIN mdm.rating_agency a ON a.id = r.agency_id
WHERE r.tenant_id = '$TENANT_ID'::uuid AND r.is_latest
  AND a.agency_cd = 'INTERNAL'
  AND r.rated_party_type = 'ISSUER'
  AND r.rated_party_key = '$ISSACME_LEI';
COMMIT;
SQL
)
REVERTED_VALUE=$(echo "$REVERTED_VALUE" | grep -vE '^(BEGIN|SET|COMMIT|ROLLBACK|INSERT|UPDATE|DELETE)' | head -1)
[ "$REVERTED_VALUE" = "A-" ] \
    && ok "revert applied, master back to A-" \
    || fail "revert did not happen: $REVERTED_VALUE"

RANK_OK=$(psql "$CRIMS_DSN" -Atq <<SQL
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
SELECT CASE WHEN r.rating_rank = ra.new_rank THEN 'OK' ELSE 'MISMATCH' END
FROM mdm.rating r
JOIN mdm.rating_action ra ON ra.rating_id = r.id
JOIN mdm.rating_agency a ON a.id = r.agency_id
WHERE r.is_latest AND a.agency_cd = 'INTERNAL'
  AND r.rated_party_type = 'ISSUER'
  AND r.rated_party_key = '$ISSACME_LEI'
ORDER BY ra.action_timestamp DESC, ra.id DESC LIMIT 1;
COMMIT;
SQL
)
[ "$RANK_OK" = "OK" ] \
    && ok "step 7.8: rating_rank matches new_rank" \
    || fail "step 7.8: rating_rank != new_rank (Bug 1 regression)"

# ── 8. Verify mdm.rating counts ────────────────────────────────────────
step "8/14 Verify counts"
psql "$CRIMS_DSN" -Atq <<SQL > /tmp/slice_counts.txt
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
SELECT
    (SELECT count(*) FROM mdm.rating
       WHERE tenant_id = '$TENANT_ID'::uuid AND is_latest)
        AS master_rows,
    (SELECT count(*) FROM staging.rating_incoming
       WHERE tenant_id = '$TENANT_ID'::uuid AND loaded_at_master IS NOT NULL
         AND is_valid = true)
        AS staging_loaded,
    (SELECT count(*) FROM staging.rating_incoming
       WHERE tenant_id = '$TENANT_ID'::uuid AND is_valid = false)
        AS staging_rejected;
COMMIT;
SQL
cat /tmp/slice_counts.txt
ok "done"

# ── 9. 4-row action trail for INTERNAL tuple ──────────────────────────
step "9/14 4-row action trail (INTERNAL, ISSUER, ISSACME_LEI)"
TRAIL=$(psql "$CRIMS_DSN" -Atq <<SQL
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
SELECT at.action_cd || '|' || COALESCE(ra.prior_value, '<null>') || '|' || ra.new_value || '|' || COALESCE(ra.prior_rank::text, '<null>') || '|' || ra.new_rank || '|' || COALESCE(ra.model_value, '<null>')
FROM mdm.rating_action ra
JOIN mdm.rating_action_type at ON at.id = ra.action_type_id
JOIN mdm.rating r ON r.id = ra.rating_id
JOIN mdm.rating_agency a ON a.id = r.agency_id
WHERE a.agency_cd = 'INTERNAL'
  AND r.rated_party_type = 'ISSUER'
  AND r.rated_party_key = '$ISSACME_LEI'
ORDER BY ra.action_timestamp, ra.id;
COMMIT;
SQL
)
# Filter out psql status lines that bleed through.
TRAIL=$(echo "$TRAIL" | grep -vE '^(BEGIN|SET|COMMIT|ROLLBACK|INSERT|UPDATE|DELETE)$')
echo "$TRAIL"
TRAIL_ROWS=$(echo "$TRAIL" | grep -c '^')
[ "$TRAIL_ROWS" = "4" ] \
    && ok "trail has exactly 4 rows" \
    || fail "expected 4 trail rows, got $TRAIL_ROWS"
echo "$TRAIL" | grep -q '^AFFIRMATION|.*|A-' \
    && ok "step 7 init: AFFIRMATION NULL → A-" \
    || fail "missing AFFIRMATION NULL → A- row"
echo "$TRAIL" | grep -q '^OVERRIDE_APPLIED|A-|BBB+|14|13|.*' \
    && ok "step 7.5b: OVERRIDE_APPLIED A- → BBB+ rank 13" \
    || fail "missing step 7.5b row"
echo "$TRAIL" | grep -q '^OVERRIDE_APPLIED|BBB+|BB+|13|10|A-' \
    && ok "step 7.7: OVERRIDE_APPLIED BBB+ → BB+ rank 10, model=A-" \
    || fail "missing step 7.7 row"
echo "$TRAIL" | grep -q '^OVERRIDE_EXPIRED|BB+|A-|10|14|.*' \
    && ok "step 7.8: OVERRIDE_EXPIRED BB+ → A- rank 14" \
    || fail "missing step 7.8 row"

# ── 10. RLS enforcement test (non-superuser) ──────────────────────────
step "10/14 RLS enforcement (non-superuser sees only own tenant)"

# Create the test role idempotently. Grants run against $CRIMS_DSN; the role
# DDL itself needs CREATEROLE, which the DSN's default role does not have on
# homebrew PG, so it goes through $ADMIN_DSN (the OS superuser).
DSN_ROLE="$(printf '%s' "$CRIMS_DSN" | sed -E 's|^[^:]*//([^:/@]+).*|\1|')"

psql "$ADMIN_DSN" <<SQL >/dev/null 2>&1 || true
DO \$\$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '$RLS_ROLE') THEN
        CREATE ROLE $RLS_ROLE NOLOGIN;
    END IF;
END \$\$;
GRANT $RLS_ROLE TO $DSN_ROLE;
GRANT USAGE ON SCHEMA mdm, public TO $RLS_ROLE;
GRANT SELECT ON ALL TABLES IN SCHEMA mdm TO $RLS_ROLE;
ALTER DEFAULT PRIVILEGES IN SCHEMA mdm
    GRANT SELECT ON TABLES TO $RLS_ROLE;
SQL

# The role may already exist from a run that leaked it — IF NOT EXISTS above
# handles that, and the grants below are idempotent either way.
if ! psql "$ADMIN_DSN" -Atq \
        -c "SELECT 1 FROM pg_roles WHERE rolname = '$RLS_ROLE'" 2>/dev/null | grep -q 1; then
    fail "could not create $RLS_ROLE — need a CREATEROLE-capable superuser at $ADMIN_DSN"
fi
# This run used the role, so teardown owns dropping it (even if a prior run
# created it) — otherwise a leaked role never gets cleaned up.
TEARDOWN_RLS_ROLE=1

WRONG_TENANT="aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

# Run RLS test as the non-superuser role. Two separate psql calls
# (one per tenant) — easier to parse than one combined call.
WRONG_HALF=$(psql "$CRIMS_DSN" -Atq <<SQL 2>&1
SET ROLE crims_tenant_user;
SET app.current_tenant = '$WRONG_TENANT';
SELECT 'rating_agency' AS t, count(*) AS rows FROM mdm.rating_agency
UNION ALL SELECT 'rating_scale', count(*) FROM mdm.rating_scale
UNION ALL SELECT 'rating_outlook', count(*) FROM mdm.rating_outlook
UNION ALL SELECT 'rating_watch', count(*) FROM mdm.rating_watch
UNION ALL SELECT 'rating_type', count(*) FROM mdm.rating_type
UNION ALL SELECT 'rating_action_type', count(*) FROM mdm.rating_action_type
UNION ALL SELECT 'party', count(*) FROM mdm.party
UNION ALL SELECT 'rating', count(*) FROM mdm.rating
UNION ALL SELECT 'rating_action', count(*) FROM mdm.rating_action;
RESET ROLE;
SQL
)
RIGHT_HALF=$(psql "$CRIMS_DSN" -Atq <<SQL 2>&1
SET ROLE crims_tenant_user;
SET app.current_tenant = '$TENANT_ID';
SELECT 'rating_agency' AS t, count(*) AS rows FROM mdm.rating_agency
UNION ALL SELECT 'rating_scale', count(*) FROM mdm.rating_scale
UNION ALL SELECT 'rating_outlook', count(*) FROM mdm.rating_outlook
UNION ALL SELECT 'rating_watch', count(*) FROM mdm.rating_watch
UNION ALL SELECT 'rating_type', count(*) FROM mdm.rating_type
UNION ALL SELECT 'rating_action_type', count(*) FROM mdm.rating_action_type
UNION ALL SELECT 'party', count(*) FROM mdm.party
UNION ALL SELECT 'rating', count(*) FROM mdm.rating
UNION ALL SELECT 'rating_action', count(*) FROM mdm.rating_action;
RESET ROLE;
SQL
)

echo "RLS counts (wrong tenant '$WRONG_TENANT', expect 0 for all):"
echo "$WRONG_HALF"
WRONG_NONZERO=$(echo "$WRONG_HALF" | awk -F'|' '$2 != "0" && $2 != "" {print $1, $2}' | wc -l | tr -d ' ')
[ "$WRONG_NONZERO" = "0" ] \
    && ok "RLS filters all 9 tables to 0 for wrong tenant" \
    || fail "RLS leaked $WRONG_NONZERO tables for wrong tenant"

echo "RLS counts (right tenant '$TENANT_ID', expect seeded counts):"
echo "$RIGHT_HALF"
RATING_5=$(echo "$RIGHT_HALF" | awk -F'|' '$1=="rating_agency"{print $2}')
RATING_79=$(echo "$RIGHT_HALF" | awk -F'|' '$1=="rating_scale"{print $2}')
RATING_OUTLOOK_5=$(echo "$RIGHT_HALF" | awk -F'|' '$1=="rating_outlook"{print $2}')
RATING_WATCH_4=$(echo "$RIGHT_HALF" | awk -F'|' '$1=="rating_watch"{print $2}')
RATING_TYPE_7=$(echo "$RIGHT_HALF" | awk -F'|' '$1=="rating_type"{print $2}')
RATING_ACTION_TYPE_8=$(echo "$RIGHT_HALF" | awk -F'|' '$1=="rating_action_type"{print $2}')
PARTY_3=$(echo "$RIGHT_HALF" | awk -F'|' '$1=="party"{print $2}')
RATING_9=$(echo "$RIGHT_HALF" | awk -F'|' '$1=="rating"{print $2}')
RATING_ACTION_9=$(echo "$RIGHT_HALF" | awk -F'|' '$1=="rating_action"{print $2}')

[ "$RATING_5" = "5" ]           && ok "rating_agency=5" || fail "rating_agency=$RATING_5 (expected 5)"
[ "$RATING_79" = "79" ]          && ok "rating_scale=79" || fail "rating_scale=$RATING_79 (expected 79)"
[ "$RATING_OUTLOOK_5" = "5" ]    && ok "rating_outlook=5" || fail "rating_outlook=$RATING_OUTLOOK_5 (expected 5)"
[ "$RATING_WATCH_4" = "4" ]      && ok "rating_watch=4" || fail "rating_watch=$RATING_WATCH_4 (expected 4)"
[ "$RATING_TYPE_7" = "7" ]       && ok "rating_type=7" || fail "rating_type=$RATING_TYPE_7 (expected 7)"
[ "$RATING_ACTION_TYPE_8" = "8" ] && ok "rating_action_type=8" || fail "rating_action_type=$RATING_ACTION_TYPE_8 (expected 8)"
[ "$PARTY_3" = "3" ]             && ok "party=3" || fail "party=$PARTY_3 (expected 3)"
[ "$RATING_9" = "9" ]            && ok "rating=9" || fail "rating=$RATING_9 (expected 9)"
[ "$RATING_ACTION_9" = "9" ]     && ok "rating_action=9" || fail "rating_action=$RATING_ACTION_9 (expected 9)"

# ── 11. Sample mdm.rating ─────────────────────────────────────────────
step "11/14 Sample mdm.rating rows"
psql "$CRIMS_DSN" -Atq <<SQL
BEGIN;
SET LOCAL app.current_tenant = '$TENANT_ID';
SELECT a.agency_cd || '|' || rt.type_cd || '|' || r.rating_value || '|' || COALESCE(rs.scale_cd, '') || '|' || r.rating_rank
FROM mdm.rating r
JOIN mdm.rating_agency a ON a.id = r.agency_id
JOIN mdm.rating_type rt  ON rt.id = r.rating_type_id
LEFT JOIN mdm.rating_scale rs ON rs.id = r.rating_scale_id
WHERE r.tenant_id = '$TENANT_ID'::uuid AND r.is_latest
ORDER BY a.agency_cd, rt.type_cd;
COMMIT;
SQL

ok "slice 1.5 done"
