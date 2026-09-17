#!/bin/bash
# Dev password drift detector — Phase 2 Step 1 supersession fix.
#
# Catches the failure mode that recurred twice in this workstream:
# the DB hash drifting from the committed migration hash because a
# earlier migration (20260916_fix_dev_user_password_hashes) was
# applied without the newer (20260916_regenerate_dev_user_password)
# running after it. The earlier file is now a no-op (see its comment
# block); this script verifies the DB and migration are still in
# sync.
#
# Usage:
#   DB_HOST=100.84.50.65 bash backend/scripts/dev_password_drift_check.sh
#
# Exit codes:
#   0  DB hash matches migration hash (in sync)
#   1  DB hash missing or migration missing
#   2  DB hash does NOT match migration hash (DRIFT — apply migration
#      backend/db/migrations/20260916_regenerate_dev_user_password.up.sql
#      to bring DB in sync)
#   3  Live login with the recorded dev password returns non-200 (auth
#      chain broken — the password is one rotation cycle away from
#      working)

set -e

DB_HOST="${DB_HOST:-100.84.50.65}"
CERT_DIR="${CERT_DIR:-/home/eganpj/.uisce/certs}"
BACKEND_URL="${BACKEND_URL:-http://${DB_HOST}:8080}"
TEST_EMAIL="${TEST_EMAIL:-testuser@example.com}"
TEST_TENANT="${TEST_TENANT:-99e99e99-99e9-49e9-89e9-99e99e99e999}"
MIG_FILE="${MIG_FILE:-backend/db/migrations/20260916_regenerate_dev_user_password.up.sql}"

echo "=== Drift check: dev-user bcrypt hash (DB vs migration) ==="

# Migration hash from committed file
MIG_HASH=$(grep -oE '\$2a\$10\$[A-Za-z0-9./]+' "$MIG_FILE" 2>/dev/null | head -1 || echo "")
if [ -z "$MIG_HASH" ]; then
    echo "FAIL: migration file $MIG_FILE missing or contains no bcrypt hash" >&2
    exit 1
fi
echo "  migration ($MIG_FILE): $MIG_HASH"

# DB hash via psql (requires PG access via ssh + mTLS certs)
DB_HASH=$(ssh "eganpj@${DB_HOST}" "PGPASSWORD=postgres psql -h localhost -U postgres -d alpha -t -c \"SELECT password_hash FROM public.app_user WHERE email = '${TEST_EMAIL}';\"" 2>/dev/null | tr -d ' ' | head -1 || echo "")
if [ -z "$DB_HASH" ]; then
    echo "FAIL: could not read password_hash from DB for ${TEST_EMAIL}" >&2
    exit 1
fi
echo "  DB live ($TEST_EMAIL):    $DB_HASH"

# Compare
if [ "$MIG_HASH" != "$DB_HASH" ]; then
    echo "FAIL: DRIFT — DB hash and migration hash do not match" >&2
    echo "  apply migration: ssh eganpj@${DB_HOST} 'PGPASSWORD=postgres psql -h localhost -U postgres -d alpha' < ${MIG_FILE}" >&2
    exit 2
fi
echo "  hashes match: ✓"

# Live login check (best-effort — only runs if the password is in
# the operator's credential store or a known temp file).
echo ""
echo "=== Live login probe (best-effort, skips if preimage not locatable) ==="
PW_FILE="${PW_FILE:-/tmp/new_dev_user_password.txt}"
if [ -f "$PW_FILE" ]; then
    LOGIN_STATUS=$(curl -sS -X POST "${BACKEND_URL}/api/auth/login" \
        -H 'Content-Type: application/json' \
        --data-binary @"$PW_FILE" 2>/dev/null | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    print('200' if d.get('access_token') else 'no-token')
except Exception as e:
    print(f'parse-error: {e}')
")
    if [ "$LOGIN_STATUS" != "200" ]; then
        echo "FAIL: login returned non-200 status (auth chain broken)" >&2
        exit 3
    fi
    echo "  login: 200 ✓"
else
    echo "  preimage not at $PW_FILE — skipping login probe"
    echo "  (operator's credential store is the durable home; the temp file is post-cleanup expected)"
fi

echo ""
echo "OK: DB hash matches migration; dev-user password is in sync."
