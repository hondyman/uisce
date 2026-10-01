#!/usr/bin/env bash
# Builds a real schema in an empty Postgres for CI jobs that run DB-backed tests.
#
# Mirrors the "gated-tests" job in .github/workflows/backend-gated-tests.yml:
#   1. create the roles the schema snapshot references
#   2. restore the schema + migration-log snapshots
#   3. seed the gold-copy tenant (BEFORE migrations: several insert into tables
#      with a tenant_id FK to public.tenants)
#   4. apply every migration not yet covered by the snapshot (`migrate up`)
#
# Without this the integration job ran against an EMPTY database, so any test
# needing a table or the gold-copy tenant failed ("no gold-copy tenant exists").
#
# Env (defaults match the CI Postgres service):
#   PGHOST=localhost PGPORT=5432 PGUSER=postgres PGPASSWORD=postgres PGDATABASE=alpha
# Run from the repo root.
set -euo pipefail

export PGHOST="${PGHOST:-localhost}" PGPORT="${PGPORT:-5432}" PGUSER="${PGUSER:-postgres}"
export PGPASSWORD="${PGPASSWORD:-postgres}" PGDATABASE="${PGDATABASE:-alpha}"
export DATABASE_URL="${DATABASE_URL:-postgres://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}/${PGDATABASE}?sslmode=disable}"

SNAP=backend/db/snapshots
psql_q() { psql -v ON_ERROR_STOP=1 -q "$@"; }

echo "== roles referenced by the schema snapshot"
# The dump's own GRANT/OWNER TO lines are the authoritative list of roles it needs.
roles=$(grep -ohE '(GRANT[[:print:]]*TO|OWNER TO)[[:space:]]+"?[A-Za-z_][A-Za-z0-9_]*"?[[:space:];]' "$SNAP/schema-snapshot.sql" \
  | sed -E 's/.*TO[[:space:]]+"?([A-Za-z_][A-Za-z0-9_]*)"?[[:space:];].*/\1/' \
  | sort -u | grep -vx -e PUBLIC -e postgres -e app_user -e app_admin_read || true)
echo "$roles"

# app_user / app_admin_read are created for real (migrations grant to them and
# assume they pre-exist); every other referenced role only needs to exist.
psql_q -c "CREATE USER app_user WITH PASSWORD 'app_user_password';"
psql_q -c "CREATE ROLE app_admin_read WITH LOGIN PASSWORD 'app_admin_read_password' NOCREATEDB NOCREATEROLE BYPASSRLS;"
while IFS= read -r role; do
  [ -z "$role" ] && continue
  psql_q -c "CREATE ROLE \"$role\";"
done <<< "$roles"

echo "== restore schema + migration-log snapshots"
psql_q -f "$SNAP/schema-snapshot.sql" 2>&1 | grep -v -e 'wal_level' -e '^HINT' || true
psql_q -f "$SNAP/migration-log-snapshot.sql" >/dev/null

echo "== seed gold-copy tenant (before migrations)"
psql_q -c "INSERT INTO public.tenants (id, name, display_name, gold_copy)
           VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'northwind', 'Northwind Traders', true)
           ON CONFLICT (id) DO NOTHING;"

echo "== migrate up"
# `migrate` resolves db/migrations relative to the working directory: run from backend/.
( cd backend && go build -o bin/migrate ./cmd/migrate && ./bin/migrate up )

echo "== prerequisites"
n=$(psql -At -c "SELECT count(*) FROM public.tenants WHERE gold_copy = true;")
[ "$n" -ge 1 ] || { echo "PREREQUISITE FAILED: no gold-copy tenant"; exit 1; }
psql -At -c "SELECT public.uisce_gold_copy_tenant_id();" >/dev/null \
  || { echo "PREREQUISITE FAILED: public.uisce_gold_copy_tenant_id() missing"; exit 1; }
echo "OK: database bootstrapped"
