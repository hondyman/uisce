-- 20260727000030_strict_tenant_rls.sql
-- Replaces 20260727000020_enable_tenant_rls.sql with STRICT enforcement.
-- Cardinal Rule 7: fail-closed; no permissive OR IS NULL bypass.
--
-- Active tenant context is read from session parameter 'uisce.current_tenant'.
-- All policies are RESTRICTIVE or have explicit WITH CHECK to prevent row-leakage.
-- The uisce_get_current_tenant() helper is also tightened to never return
-- a magic-bypass value.

-- 1. Strict helper function — returns NULL when GUC is absent, never silently bypasses
CREATE OR REPLACE FUNCTION uisce_get_current_tenant() RETURNS uuid AS $$
BEGIN
    RETURN NULLIF(current_setting('uisce.current_tenant', true), '')::uuid;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

-- +goose Down
DROP FUNCTION IF EXISTS uisce_get_current_tenant();

-- =============================================================================
-- 2. TENANT_INSTANCE — RLS
-- =============================================================================
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

-- +goose Down
ALTER TABLE tenant_instance DISABLE ROW LEVEL SECURITY;

-- =============================================================================
-- 3. TENANT_PRODUCT — RLS (replaces old policy)
-- =============================================================================
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

-- +goose Down
ALTER TABLE tenant_product DISABLE ROW LEVEL SECURITY;

-- =============================================================================
-- 4. TENANT_PRODUCT_DATASOURCE — RLS (requires tenant_id column from 20260727000025)
-- =============================================================================
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

-- +goose Down
ALTER TABLE tenant_product_datasource DISABLE ROW LEVEL SECURITY;

-- =============================================================================
-- 5. CONNECTIONS — RLS
-- =============================================================================
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

-- +goose Down
ALTER TABLE connections DISABLE ROW LEVEL SECURITY;

-- =============================================================================
-- 6. AUDIT_LOGS — RLS (RESTRICTIVE so it cannot be bypassed by other policies)
-- =============================================================================
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

-- +goose Down
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'audit_logs') THEN
        ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;
    END IF;
END $$;

-- =============================================================================
-- 7. Drop legacy permissive migration policies (belt-and-suspenders)
-- =============================================================================
DROP POLICY IF EXISTS tenant_isolation_policy ON tenant_product;
DROP POLICY IF EXISTS tenant_product_isolation_policy ON tenant_product;
