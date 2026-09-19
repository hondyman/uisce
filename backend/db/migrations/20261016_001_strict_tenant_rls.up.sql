-- 20261016_001_strict_tenant_rls.up.sql
-- Ports backend/migrations/20260727000030_strict_tenant_rls.sql (git ref
-- b9d00dc2c6, 2026-07-27) into this repo's actual migration runner path. That file was never
-- applied to alpha: it lives in backend/migrations/, one of the 8 directories
-- documented in backend/db/MIGRATION_DIRECTORY_DRIFT_AUDIT.md as invisible to
-- internal/migrations/runner.go, which reads only backend/db/migrations/*.up.sql.
-- It is also written in Goose's two-way format (six "-- +goose Down" markers with
-- the down-statement for each section interleaved immediately after it), a format
-- this repo's custom runner has no concept of.
--
-- Fixes a real, live security gap on alpha: tenant_product_isolation_policy and
-- tenant_product_datasource_isolation_policy currently read
--   (current_setting('uisce.current_tenant', true) IS NULL) OR (tenant_id = ...)
-- which is fail-OPEN — full read and write access to every tenant's rows whenever
-- the session GUC is unset. This migration replaces both (and adds equivalent
-- policies for tenant_instance, connections, and — if present — audit_logs) with
-- fail-closed enforcement keyed on uisce_get_current_tenant(), which returns NULL
-- (never a bypass value) when the GUC is absent, so `tenant_id = NULL` is never
-- true.
--
-- This is NOT a byte-for-byte copy of the original file's Up sections. The
-- original file's final section ("7. Drop legacy permissive migration policies")
-- re-issues `DROP POLICY IF EXISTS tenant_product_isolation_policy ON
-- tenant_product` — the exact policy section 3 just created two statements
-- earlier in the same file. Verified by actually running the original file's
-- Up-only content (every "-- +goose Down" block mechanically stripped) against a
-- scratch Postgres: it completes with zero errors, but leaves tenant_product with
-- FORCE ROW LEVEL SECURITY enabled and ZERO policies — which denies ALL access to
-- every non-owner role, not fail-open and not the intended fail-closed either.
-- Section 7 is redundant with section 3's own DROP-then-CREATE and actively
-- harmful in file order; it is dropped entirely from this port, not preserved.
--
-- Runtime note for whoever applies this to alpha: flipping these four/five tables
-- to fail-closed changes behavior for any session that touches them without
-- uisce.current_tenant set. The request path sets it via WithTenantTransaction,
-- but provisioning code, background workers, and ad-hoc scripts may not — grep for
-- direct queries against tenant_instance, tenant_product, tenant_product_datasource,
-- connections, and audit_logs and confirm each runs inside tenant context before
-- applying this to alpha. A loud permission/RLS failure replacing today's silent
-- hole is progress, but it should be anticipated, not discovered in an outage.

-- 1. Strict helper function — returns NULL when the GUC is absent, never a bypass value
CREATE OR REPLACE FUNCTION uisce_get_current_tenant() RETURNS uuid AS $$
BEGIN
    RETURN NULLIF(current_setting('uisce.current_tenant', true), '')::uuid;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

-- 2. TENANT_INSTANCE — RLS
ALTER TABLE tenant_instance ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_instance FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_instance_isolation_policy ON tenant_instance;
CREATE POLICY tenant_instance_isolation_policy ON tenant_instance
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
    )
    WITH CHECK (
        tenant_id = uisce_get_current_tenant()
    );

-- 3. TENANT_PRODUCT — RLS (replaces the fail-open policy)
ALTER TABLE tenant_product ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_product FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_product_isolation_policy ON tenant_product;
DROP POLICY IF EXISTS tenant_isolation_policy ON tenant_product;
CREATE POLICY tenant_product_isolation_policy ON tenant_product
    FOR ALL
    USING (
        datasource_id IN (
            SELECT id FROM tenant_instance WHERE tenant_id = uisce_get_current_tenant()
        )
    )
    WITH CHECK (
        datasource_id IN (
            SELECT id FROM tenant_instance WHERE tenant_id = uisce_get_current_tenant()
        )
    );

-- 4. TENANT_PRODUCT_DATASOURCE — RLS (replaces the fail-open policy)
ALTER TABLE tenant_product_datasource ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_product_datasource FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_product_datasource_isolation_policy ON tenant_product_datasource;
CREATE POLICY tenant_product_datasource_isolation_policy ON tenant_product_datasource
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
    )
    WITH CHECK (
        tenant_id = uisce_get_current_tenant()
    );

-- 5. CONNECTIONS — RLS
ALTER TABLE connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE connections FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS connections_isolation_policy ON connections;
CREATE POLICY connections_isolation_policy ON connections
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
    )
    WITH CHECK (
        tenant_id = uisce_get_current_tenant()
    );

-- 6. AUDIT_LOGS — RLS (RESTRICTIVE so it cannot be bypassed by other policies), only if present
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'audit_logs') THEN
        ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
        ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;

        DROP POLICY IF EXISTS audit_logs_isolation_policy ON audit_logs;
        CREATE POLICY audit_logs_isolation_policy ON audit_logs
            AS RESTRICTIVE
            FOR ALL
            USING (
                tenant_id = uisce_get_current_tenant()
            )
            WITH CHECK (
                tenant_id = uisce_get_current_tenant()
            );
    END IF;
END $$;
