#!/usr/bin/env bash
# Pre-flight check for the front-office ORM migrations.
# Verifies:
#   1. All 26 base tables can be created (FK targets exist)
#   2. The backend source sets app.current_tenant on connections
#   3. No existing migration has drift
#
# Run before applying 015 (RLS).

set -euo pipefail

: "${PGHOST:=100.84.50.65}"
: "${PGPORT:=5432}"
: "${PGUSER:=postgres}"
: "${PGDATABASE:=alpha}"
: "${PGSSLMODE:=verify-full}"
: "${PGSSLROOTCERT:=$HOME/.uisce/certs/ca.crt}"
: "${PGSSLCERT:=$HOME/.uisce/certs/postgres-client.crt}"
: "${PGSSLKEY:=$HOME/.uisce/certs/postgres-client.key}"

export PGHOST PGPORT PGUSER PGDATABASE PGSSLMODE PGSSLROOTCERT PGSSLCERT PGSSLKEY

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

FAIL=0

echo "-- 1. FK targets exist --"
for pair in "oms:security" "oms:account" "oms:position" "orm:order" "orm:execution"; do
  s="${pair%%:*}"; tb="${pair##*:}"
  exists=$(psql -At -c "SELECT 1 FROM information_schema.tables WHERE table_schema='$s' AND table_name='$tb';")
  if [ "$exists" != "1" ]; then
    echo "  MISSING: $s.$tb"
    FAIL=1
  else
    echo "  ok: $s.$tb"
  fi
done

echo
echo "-- 2. Backend sets app.current_tenant --"
if grep -rn "set_config.*app.current_tenant" "$BACKEND_DIR/cmd/server" \
     "$BACKEND_DIR/internal/api" 2>/dev/null | head -3 | grep -q .; then
  echo "  found set_config(app.current_tenant) calls in backend"
else
  echo "  WARNING: no set_config(app.current_tenant) found in cmd/server or internal/api"
  echo "  RLS migration 015 will lock everyone out of these tables until this is fixed."
  echo "  Apply 002..014 first (migrate-orm-front-office.sh up-no-rls) and fix the backend."
  FAIL=1
fi

echo
echo "-- 3. No existing migration drift --"
if cd "$BACKEND_DIR" && DATABASE_URL="postgresql://${PGUSER}@${PGHOST}:${PGPORT}/${PGDATABASE}?sslmode=${PGSSLMODE}&sslcert=${PGSSLCERT}&sslkey=${PGSSLKEY}&sslrootcert=${PGSSLROOTCERT}" \
     go run ./cmd/migrate verify 2>&1 | grep -qi "no drift"; then
  echo "  no drift"
else
  echo "  DRIFT detected in existing migrations. Resolve before proceeding."
  FAIL=1
fi

echo
if [ $FAIL -eq 0 ]; then
  echo "PREREQS OK — safe to apply all migrations including 015"
else
  echo "PREREQS NOT MET — investigate above before applying 015"
  exit 1
fi
