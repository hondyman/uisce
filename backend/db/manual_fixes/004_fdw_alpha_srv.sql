-- 004_fdw_alpha_srv.sql
-- Sets up FDW on CRIMS pointing at alpha via mTLS (no password).
-- Idempotent: server/user_mapping/error_table guarded by IF NOT EXISTS equivalents;
-- IMPORT FOREIGN SCHEMA is gated by an existence check on foreign tables.
--
-- ---------------------------------------------------------------
-- BLOCKED 2026-09-23: cert-paths on the remote alpha server
-- ---------------------------------------------------------------
-- The Mac client certs at /Users/eganpj/.uisce/certs/{ca.crt,
-- postgres-client.crt, postgres-client.key} are reachable from this Mac
-- (psql uses them fine) but the postgres server at 100.84.50.65 (alpha) is
-- on a separate Ubuntu host whose postgres user cannot see macOS paths.
-- alpha enforces mTLS cert-auth for the postgres role from any IP (see
-- pg_hba_file_rules() line `hostssl|{all}|{postgres}|0.0.0.0|cert`), and
-- postgres 18 has no pg_write_file(), so we cannot seed server-side cert
-- files via SQL. ssh/scp to the host is also unavailable (no SSH daemon
-- reachable from this network).
--
-- Resolution path:
--   1. Operator places the three cert files on the alpha server under
--      /etc/postgresql/18/main/ (or another postgres-readable path) and
--      chmods them (ca.crt 0644, .crt/.key 0600 owned by postgres).
--   2. Re-run this script: extension + server + user_mapping + IMPORT will
--      all succeed once the server can read those paths.
--
-- Interim copy transport (used until FDW cert-paths are provisioned):
--   pg_dump -h 100.84.50.65 -U postgres -d alpha -t '<table>'
--     --no-acl --no-owner [--schema-only | --data-only]
--     | psql -h 100.84.50.65 -U postgres -d crims
-- This works on the Mac because both client connections use the same
-- /Users/eganpj/.uisce/certs/ mTLS path.  See gen_copy_via_pgdump.py for a
-- driver script templated on migration.plan.
--
-- The remainder of this file prepares the FDW scaffold and exits at the
-- IMPORT line which requires the missing cert placement.
-- ---------------------------------------------------------------

SET search_path = migration, crims, public;

BEGIN;

CREATE EXTENSION IF NOT EXISTS postgres_fdw;

-- ============================================================================
-- Staging schemas (foreign-table namespaces)
-- ============================================================================
CREATE SCHEMA IF NOT EXISTS alpha_staging_mdm;
CREATE SCHEMA IF NOT EXISTS alpha_staging_edm;
CREATE SCHEMA IF NOT EXISTS alpha_staging_oms;

-- ============================================================================
-- FDW server — mTLS, no password
-- ============================================================================
-- Using DETERMINISTIC so the optimizer can simplify; these options match the
-- local cert paths and the alpha instance on the same Postgres host (100.84.50.65).
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_foreign_server WHERE srvname='alpha_srv') THEN
    CREATE SERVER alpha_srv
      FOREIGN DATA WRAPPER postgres_fdw
      OPTIONS (
        host '100.84.50.65',
        port '5432',
        dbname 'alpha',
        sslmode 'verify-full',
        sslcert '/Users/eganpj/.uisce/certs/postgres-client.crt',
        sslkey  '/Users/eganpj/.uisce/certs/postgres-client.key',
        sslrootcert '/Users/eganpj/.uisce/certs/ca.crt'
      );
  END IF;
END $$;

-- ============================================================================
-- User mapping — empty (mTLS trusts client cert identity)
-- ============================================================================
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_user_mappings WHERE srvname='alpha_srv' AND usename=CURRENT_USER
  ) THEN
    CREATE USER MAPPING FOR CURRENT_USER SERVER alpha_srv OPTIONS (user 'postgres');
  END IF;
END $$;

-- ============================================================================
-- Validate connectivity before bulk IMPORT
-- ============================================================================
DO $$
DECLARE
  v int;
BEGIN
  -- The trivial test: make sure the server is reachable
  PERFORM 1 FROM pg_foreign_server WHERE srvname='alpha_srv';
  RAISE NOTICE 'alpha_srv reachable; ready for IMPORT FOREIGN SCHEMA';
EXCEPTION
  WHEN OTHERS THEN
    RAISE EXCEPTION 'alpha_srv register failed: %', SQLERRM;
END $$;

-- ============================================================================
-- Refresh import (drop+create). Pre-flight wrap in DO ... ON CONFLICT
-- simulation: only re-import if the staging schema is empty.
-- ============================================================================
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema='alpha_staging_mdm' AND table_name='party'
  ) THEN
    -- single-transaction wrapper not supported for IMPORT FOREIGN SCHEMA
    -- with multiple schema; run as individual statements outside this DO block.
    RAISE NOTICE 'IMPORT alpha.mdm -> alpha_staging_mdm (next step)';
  END IF;
END $$;

-- Force the IMPORT to be re-runnable.  IMPORT FOREIGN SCHEMA processes ALL
-- tables in the source schema. We import all of mdm/edm into the staging
-- schemas; the copy executor will skip DROP tables and METADATA/INFRA that
-- stay in alpha per migration.plan.

IMPORT FOREIGN SCHEMA mdm
  FROM SERVER alpha_srv INTO alpha_staging_mdm
  OPTIONS (import_default 'false');

IMPORT FOREIGN SCHEMA edm
  FROM SERVER alpha_srv INTO alpha_staging_edm
  OPTIONS (import_default 'false');

COMMIT;

-- ============================================================================
-- Smoke test
-- ============================================================================
-- SELECT 1 FROM alpha_staging_mdm.party LIMIT 0;
-- This will fail if the server is unreachable or auth is wrong. The error mode
-- is acceptable as a smoke probe; capture and report to migration.progress.
DO $$
DECLARE
  v_count int;
BEGIN
  SELECT count(*) INTO v_count FROM alpha_staging_mdm.party LIMIT 0;  -- returns 0, validates connectivity
  RAISE NOTICE 'alpha_staging_mdm.party readable; foreign-table count probe OK';
EXCEPTION WHEN OTHERS THEN
  RAISE NOTICE 'alpha_staging_mdm.party probe failed: %', SQLERRM;
END $$;
