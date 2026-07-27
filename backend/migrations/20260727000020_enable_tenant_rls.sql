-- 20260727000020_enable_tenant_rls.sql
-- Enforces PostgreSQL Row-Level Security (RLS) across multi-tenant tables
-- to satisfy Rule 7 (Zero-Tolerance Security Mandate).
--
-- Active tenant context is read from session parameter 'uisce.current_tenant'.
-- Master/core copy tenant (gold_copy = true) bypasses or evaluates via policy.

-- 1. Helper function for resolving current tenant session setting
CREATE OR REPLACE FUNCTION uisce_get_current_tenant() RETURNS uuid AS $$
BEGIN
    RETURN nullif(current_setting('uisce.current_tenant', true), '')::uuid;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

-- 2. Apply Row Level Security to tenant_product
ALTER TABLE tenant_product ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_product FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_policy ON tenant_product;
DROP POLICY IF EXISTS tenant_product_isolation_policy ON tenant_product;

CREATE POLICY tenant_product_isolation_policy ON tenant_product
    FOR ALL
    USING (
        nullif(current_setting('uisce.current_tenant', true), '') IS NULL
        OR tenant_id = nullif(current_setting('uisce.current_tenant', true), '')::uuid
    )
    WITH CHECK (
        nullif(current_setting('uisce.current_tenant', true), '') IS NULL
        OR tenant_id = nullif(current_setting('uisce.current_tenant', true), '')::uuid
    );

-- 3. Apply Row Level Security to tenant_product_datasource
ALTER TABLE tenant_product_datasource ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_product_datasource FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_product_datasource_isolation_policy ON tenant_product_datasource;
CREATE POLICY tenant_product_datasource_isolation_policy ON tenant_product_datasource
    FOR ALL
    USING (
        nullif(current_setting('uisce.current_tenant', true), '') IS NULL
        OR tenant_id = nullif(current_setting('uisce.current_tenant', true), '')::uuid
    )
    WITH CHECK (
        nullif(current_setting('uisce.current_tenant', true), '') IS NULL
        OR tenant_id = nullif(current_setting('uisce.current_tenant', true), '')::uuid
    );

-- 4. Apply Row Level Security to audit_logs (if present)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'audit_logs') THEN
        ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

        EXECUTE 'DROP POLICY IF EXISTS audit_logs_isolation_policy ON audit_logs;';
        EXECUTE 'CREATE POLICY audit_logs_isolation_policy ON audit_logs
            AS RESTRICTIVE
            FOR ALL
            USING (
                tenant_id = uisce_get_current_tenant()
                OR uisce_get_current_tenant() IS NULL
            )
            WITH CHECK (
                tenant_id = uisce_get_current_tenant()
            );';
    END IF;
END $$;
