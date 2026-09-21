#!/usr/bin/env bash
# create_agg_writer_roles.sh
# Creates the dedicated Postgres roles for the CDC aggregate consumer.
# Run ONCE per environment (alpha, crims) before starting the consumers.
# Passwords are read from .env — never hardcoded.
#
# What this script does (in order):
#   1. CREATE LOGIN ROLE + password (alpha + crims)
#   2. Pre-create the agg schema + tables that the consumer writes to
#      (consumer_dedupe, agg.security_fund_access_change, agg.metrics_registry_changed)
#   3. Grant minimum privileges for the consumer to function
#
# Grants live ONLY in the migration 20260920_003_agg_writer_grants — this script
# does not grant; 003 is the source of truth for the access model so the
# migration history records the full access surface.
#
# Usage:
#   source .env && ./scripts/create_agg_writer_roles.sh
set -euo pipefail

CERTS_DIR="${HOME}/.uisce/certs"
PGHOST="${PGHOST:-100.84.50.65}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-postgres}"

export PGSSLMODE=verify-full
export PGSSLROOTCERT="${CERTS_DIR}/ca.crt"
export PGSSLCERT="${CERTS_DIR}/postgres-client.crt"
export PGSSLKEY="${CERTS_DIR}/postgres-client.key"

# ---------------------------------------------------------------------------
# Alpha
# ---------------------------------------------------------------------------
create_alpha_role() {
  local pw="$1"
  echo "[alpha] creating agg_writer_alpha..."
  psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d alpha <<'EOSQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'agg_writer_alpha') THEN
    CREATE ROLE agg_writer_alpha LOGIN;
  END IF;
END
$$;
EOSQL
  # Update password separately so it doesn't appear in any command logs
  PGPASSWORD="$pw" psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d alpha \
    -c "ALTER ROLE agg_writer_alpha PASSWORD '${pw}'"
  echo "[alpha] role ready"
}

# Pre-create schema + tables so the role can be granted access to existing objects
bootstrap_alpha_schema() {
  echo "[alpha] bootstrapping agg schema and consumer_dedupe table..."
  psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d alpha <<'EOSQL'
-- agg schema (not created by any migration yet)
CREATE SCHEMA IF NOT EXISTS agg;

-- consumer_dedupe: dedupe store for the alpha consumer
CREATE TABLE IF NOT EXISTS public.consumer_dedupe (
    handler      TEXT        NOT NULL,
    source_lsn   TEXT        NOT NULL,
    table_name   TEXT        NOT NULL,
    op           CHAR(1)     NOT NULL,
    processed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (handler, source_lsn, table_name, op)
);
CREATE INDEX IF NOT EXISTS idx_consumer_dedupe_processed_at
    ON public.consumer_dedupe (processed_at);

-- agg.security_fund_access_change: written by notify_events handler
CREATE TABLE IF NOT EXISTS agg.security_fund_access_change (
    tenant_id    UUID,
    user_id      UUID,
    action       TEXT,
    occurred_at  TIMESTAMPTZ DEFAULT NOW(),
    source_lsn   TEXT,
    source_table TEXT,
    op           CHAR(1),
    PRIMARY KEY (tenant_id, user_id, action, occurred_at, source_lsn)
);

-- agg.metrics_registry_changed: written by notify_events handler
CREATE TABLE IF NOT EXISTS agg.metrics_registry_changed (
    tenant_id    UUID,
    metric_id    UUID,
    changed_at   TIMESTAMPTZ DEFAULT NOW(),
    source_lsn   TEXT,
    source_table TEXT,
    op           CHAR(1),
    PRIMARY KEY (tenant_id, metric_id, changed_at, source_lsn)
);

-- Give the role access to the objects it needs to write
GRANT USAGE ON SCHEMA public, agg TO agg_writer_alpha;
GRANT SELECT, INSERT, DELETE ON public.consumer_dedupe            TO agg_writer_alpha;
GRANT INSERT ON agg.security_fund_access_change                   TO agg_writer_alpha;
GRANT INSERT ON agg.metrics_registry_changed                     TO agg_writer_alpha;
EOSQL
  echo "[alpha] schema bootstrap complete"
}

# ---------------------------------------------------------------------------
# Crims
# ---------------------------------------------------------------------------
create_crims_role() {
  local pw="$1"
  echo "[crims] creating agg_writer_crims..."
  psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d crims <<'EOSQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'agg_writer_crims') THEN
    CREATE ROLE agg_writer_crims LOGIN;
  END IF;
END
$$;
EOSQL
  PGPASSWORD="$pw" psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d crims \
    -c "ALTER ROLE agg_writer_crims PASSWORD '${pw}'"
  echo "[crims] role ready"
}

bootstrap_crims_schema() {
  echo "[crims] bootstrapping consumer_dedupe table..."
  psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d crims <<'EOSQL'
CREATE TABLE IF NOT EXISTS public.consumer_dedupe (
    handler      TEXT        NOT NULL,
    source_lsn   TEXT        NOT NULL,
    table_name   TEXT        NOT NULL,
    op           CHAR(1)     NOT NULL,
    processed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (handler, source_lsn, table_name, op)
);
CREATE INDEX IF NOT EXISTS idx_consumer_dedupe_processed_at
    ON public.consumer_dedupe (processed_at);
GRANT USAGE  ON SCHEMA public, orm TO agg_writer_crims;
GRANT SELECT, INSERT, DELETE ON public.consumer_dedupe TO agg_writer_crims;
EOSQL
  echo "[crims] schema bootstrap complete"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
if [[ -z "${ALPHA_ROLE_PASSWORD:-}" ]] || [[ -z "${CRIMS_ROLE_PASSWORD:-}" ]]; then
  echo "ERROR: ALPHA_ROLE_PASSWORD and CRIMS_ROLE_PASSWORD must be set"
  echo "  source .env && $0"
  exit 1
fi

create_alpha_role    "$ALPHA_ROLE_PASSWORD"
bootstrap_alpha_schema
create_crims_role    "$CRIMS_ROLE_PASSWORD"
bootstrap_crims_schema

echo ""
echo "Done. Remaining grants (schema-only, no passwords) in:"
echo "  backend/db/migrations/20260920_003_agg_writer_grants.up.sql"
echo ""
echo "Verify:"
echo "  PGPASSWORD='\$ALPHA_ROLE_PASSWORD' psql -h $PGHOST -U agg_writer_alpha -d alpha -c 'SELECT 1'"
echo "  PGPASSWORD='\$CRIMS_ROLE_PASSWORD' psql -h $PGHOST -U agg_writer_crims  -d crims  -c 'SELECT 1'"
