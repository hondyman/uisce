#!/usr/bin/env bash
# Regenerates schema-snapshot.sql and migration-log-snapshot.sql as an atomic
# pair from the same alpha state, closing the desync class documented in
# migration-log-snapshot.sql's own header (schema-snapshot.sql already
# reflecting 20260915_001/20260916_001 while the log snapshot didn't know it,
# because the two files were previously captured from alpha at different
# moments by separate, unsynchronized pg_dump invocations).
#
# The fix isn't "run pg_dump twice in a row" — two separate connections can
# still straddle a commit between them. It's pg_dump's own cross-session
# consistency primitive: export one transaction snapshot ID from a single
# long-lived connection, then have both pg_dump invocations import that same
# ID (--snapshot=<id>), so both see byte-identical database state regardless
# of what commits on alpha in between. The long-lived connection is a bash
# coprocess so the exporting transaction stays open for the whole script.
#
# Usage:
#   DATABASE_URL=postgresql://...@alpha-host:5432/alpha?sslmode=... \
#     backend/db/snapshots/regenerate.sh
#
# Requires: psql, pg_dump (matching or newer than alpha's server version —
# this repo's CI uses pgvector/pgvector:pg18, so a pg_dump from a Postgres 18
# client is the safe choice).

set -euo pipefail

: "${DATABASE_URL:?DATABASE_URL must be set to a connection string for alpha}"

SNAPSHOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_OUT="$SNAPSHOT_DIR/schema-snapshot.sql"
LOG_OUT="$SNAPSHOT_DIR/migration-log-snapshot.sql"

cleanup() {
  if [ -n "${COPROC_PID:-}" ] && kill -0 "$COPROC_PID" 2>/dev/null; then
    exec {COPROC[1]}>&- 2>/dev/null || true
    wait "$COPROC_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "Exporting a shared transaction snapshot from alpha..." >&2

# coproc keeps one psql process's stdin/stdout open as pipes for the life of
# this script, so the exporting transaction (and its snapshot ID) stays
# valid across both pg_dump calls below.
coproc COPROC { psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -Atq; }

echo "BEGIN ISOLATION LEVEL REPEATABLE READ;" >&"${COPROC[1]}"
echo "SELECT pg_export_snapshot();" >&"${COPROC[1]}"

SNAPSHOT_ID=""
if ! read -r -t 15 SNAPSHOT_ID <&"${COPROC[0]}"; then
  echo "Timed out waiting for a snapshot ID from alpha; aborting without touching either file." >&2
  exit 1
fi

if [ -z "$SNAPSHOT_ID" ]; then
  echo "Failed to obtain a snapshot ID from alpha; aborting without touching either file." >&2
  exit 1
fi

echo "Snapshot ID: $SNAPSHOT_ID" >&2

echo "Dumping schema (schema-only) under this snapshot..." >&2
pg_dump "$DATABASE_URL" --snapshot="$SNAPSHOT_ID" --schema-only \
  > "$SCHEMA_OUT.tmp"

echo "Dumping oms.migration_log (data-only) under the same snapshot..." >&2
pg_dump "$DATABASE_URL" --snapshot="$SNAPSHOT_ID" --data-only --table=oms.migration_log \
  > "$LOG_OUT.tmp"

echo "COMMIT;" >&"${COPROC[1]}"
echo "\\q" >&"${COPROC[1]}"
exec {COPROC[1]}>&-
wait "$COPROC_PID" 2>/dev/null || true

mv "$SCHEMA_OUT.tmp" "$SCHEMA_OUT"
mv "$LOG_OUT.tmp" "$LOG_OUT"

echo "Done. Both files now reflect the exact same alpha state (snapshot $SNAPSHOT_ID)." >&2
echo "Review the diff before committing — in particular, re-check backend-gated-tests.yml's" >&2
echo "derived-role-stub step still covers every role the new schema-snapshot.sql references" >&2
echo "(it derives the list at CI runtime, so this is a sanity check, not a hand-edit)." >&2
