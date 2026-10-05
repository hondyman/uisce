#!/usr/bin/env bash
# Prepares a THROWAWAY Postgres cluster for the real-database tests (tenant onboarding gate).
# Never point this at a shared cluster: it revokes CONNECT on postgres/template1 and creates
# scratch databases. Prints the env file the tests read on stdout (KEY=VALUE lines).
set -euo pipefail
: "${PGHOST:?}" "${PGPORT:?}" "${PGUSER:?}" "${PGPASSWORD:?}"
# The superuser password is rotated to a random one for this run, and masked in CI logs, so no fixed credential is
# ever used by the tests. PGPASSWORD is the bootstrap password the cluster was started with.
pw="$(python3 -c 'import secrets;print(secrets.token_hex(16))')"
[ -z "${GITHUB_ACTIONS:-}" ] || echo "::add-mask::${pw}" >&2
psql -v ON_ERROR_STOP=1 -X -q -d postgres -c "ALTER ROLE ${PGUSER} PASSWORD '${pw}'"
export PGPASSWORD="${pw}"
q() { psql -v ON_ERROR_STOP=1 -X -q -d postgres "$@"; }

# Provisioning proves a tenant role can reach no other database, so nothing may be open to PUBLIC.
q -c "REVOKE CONNECT ON DATABASE postgres FROM PUBLIC" -c "REVOKE CONNECT ON DATABASE template1 FROM PUBLIC"
q -c "CREATE ROLE saga_app LOGIN PASSWORD 'saga_app' NOSUPERUSER NOBYPASSRLS"
q -c "CREATE ROLE uisce_gold_copy_sync NOLOGIN BYPASSRLS" -c "GRANT uisce_gold_copy_sync TO saga_app"
for db in saga_alpha tdb_migrate; do
  q -c "CREATE DATABASE $db"
  q -c "REVOKE CONNECT ON DATABASE $db FROM PUBLIC"
done
q -c "GRANT CONNECT ON DATABASE saga_alpha TO saga_app" -c "GRANT ALL ON SCHEMA public TO saga_app" 2>/dev/null || true
psql -v ON_ERROR_STOP=1 -X -q -d saga_alpha -c "GRANT ALL ON SCHEMA public TO saga_app"

base="postgres://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}"
cat <<ENV
SAGA_TEST_ALPHA_ADMIN_DSN=${base}/saga_alpha?sslmode=disable
SAGA_TEST_ALPHA_APP_DSN=postgres://saga_app:saga_app@${PGHOST}:${PGPORT}/saga_alpha?sslmode=disable
SAGA_TEST_PG_HOST=${PGHOST}
SAGA_TEST_PG_PORT=${PGPORT}
SAGA_TEST_PG_USER=${PGUSER}
SAGA_TEST_PG_PASSWORD=${PGPASSWORD}
SCANNER_TEST_ADMIN_DSN=${base}/postgres?sslmode=disable
TENANTSCHEMA_TEST_ADMIN_DSN=${base}/postgres?sslmode=disable
TENANT_MIGRATE_TEST_DSN=${base}/tdb_migrate?sslmode=disable
# CandidateDDL creates/drops its own database against the maintenance DB.
TENANT_DDL_TEST_ADMIN_DSN=${base}/postgres?sslmode=disable
ENV
