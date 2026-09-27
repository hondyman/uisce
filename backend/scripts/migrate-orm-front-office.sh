#!/usr/bin/env bash
# Apply front-office ORM migrations to alpha over mTLS.
# Wraps the existing Go migration runner (backend/cmd/migrate).
#
# Usage:
#   ./migrate-orm-front-office.sh status
#   ./migrate-orm-front-office.sh up
#   ./migrate-orm-front-office.sh up-no-rls
#   ./migrate-orm-front-office.sh verify
#
# Environment overrides:
#   PGHOST, PGPORT, PGUSER, PGDATABASE, PGSSLMODE,
#   PGSSLROOTCERT, PGSSLCERT, PGSSLKEY
#   GO_BIN       path to go binary (default: go)
#   SKIP_RLS     if set to 1, migration 015 (RLS) is not applied

set -euo pipefail

ACTION="${1:-status}"

: "${PGHOST:=100.84.50.65}"
: "${PGPORT:=5432}"
: "${PGUSER:=postgres}"
: "${PGDATABASE:=alpha}"
: "${PGSSLMODE:=verify-full}"
: "${PGSSLROOTCERT:=$HOME/.uisce/certs/ca.crt}"
: "${PGSSLCERT:=$HOME/.uisce/certs/postgres-client.crt}"
: "${PGSSLKEY:=$HOME/.uisce/certs/postgres-client.key}"
: "${GO_BIN:=go}"
: "${SKIP_RLS:=0}"

for f in "$PGSSLROOTCERT" "$PGSSLCERT" "$PGSSLKEY"; do
  if [ ! -f "$f" ]; then
    echo "cert not found: $f" >&2
    exit 2
  fi
done

DATABASE_URL="postgresql://${PGUSER}@${PGHOST}:${PGPORT}/${PGDATABASE}?sslmode=${PGSSLMODE}&sslcert=${PGSSLCERT}&sslkey=${PGSSLKEY}&sslrootcert=${PGSSLROOTCERT}"
export DATABASE_URL

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$BACKEND_DIR"

echo "-> DB    : $PGDATABASE @ $PGHOST:$PGPORT (mTLS)"
echo "-> Action: $ACTION"
echo

case "$ACTION" in
  status)
    "$GO_BIN" run ./cmd/migrate status
    ;;

  verify)
    "$GO_BIN" run ./cmd/migrate verify
    ;;

  up)
    echo "-- Pending before applying --"
    "$GO_BIN" run ./cmd/migrate status || true
    echo
    read -r -p "Apply pending migrations? [y/N] " CONFIRM
    if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
      echo "aborted"
      exit 0
    fi

    "$GO_BIN" run ./cmd/migrate up

    echo
    echo "-- Post-apply verify --"
    "$GO_BIN" run ./cmd/migrate verify
    ;;

  up-no-rls)
    # Apply only migrations 002..014 (skip 015).
    # Useful on first deploy when the backend does not yet set
    # app.current_tenant on every connection.
    # The runner does not support a --only flag, so this falls back to
    # direct psql for the front-office files in numeric order.
    echo "-- Applying 002..014 directly via psql (RLS skipped) --"
    for n in 002 003 004 005 006 007 008 009 010 011 012 013 014; do
      f=$(ls "db/migrations/20261026_${n}_"*.up.sql 2>/dev/null | head -1)
      [ -f "$f" ] || continue
      echo "  applying: $f"
      psql -v ON_ERROR_STOP=1 -f "$f"
    done

    echo
    echo "-- Recording front-office migrations as applied (mark only) --"
    for n in 002 003 004 005 006 007 008 009 010 011 012 013 014; do
      f=$(ls "db/migrations/20261026_${n}_"*.up.sql 2>/dev/null | head -1)
      [ -f "$f" ] || continue
      filename=$(basename "$f")
      sha=$(shasum -a 256 "$f" | awk '{print $1}')
      psql -At -c "
        INSERT INTO oms.migration_log (filename, sha256, applied_at)
        VALUES ('$filename', '$sha', now())
        ON CONFLICT (filename) DO NOTHING;
      " >/dev/null
      echo "  recorded: $filename"
    done
    ;;

  *)
    echo "usage: $0 <status|up|up-no-rls|verify>" >&2
    exit 2
    ;;
esac
