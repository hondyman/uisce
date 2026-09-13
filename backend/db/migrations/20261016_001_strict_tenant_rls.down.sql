-- 20261016_001_strict_tenant_rls.down.sql
-- Reverts 20261016_001_strict_tenant_rls.up.sql. Restores the RLS-disabled
-- state, not the prior fail-open policies — those are gone, on purpose.
-- If this is ever run, tenant_product and tenant_product_datasource have NO
-- tenant isolation at all until the up migration (or a replacement) reapplies.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'audit_logs') THEN
        ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS audit_logs_isolation_policy ON audit_logs;
    END IF;
END $$;

ALTER TABLE connections DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS connections_isolation_policy ON connections;

ALTER TABLE tenant_product_datasource DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_product_datasource_isolation_policy ON tenant_product_datasource;

ALTER TABLE tenant_product DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_product_isolation_policy ON tenant_product;

ALTER TABLE tenant_instance DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_instance_isolation_policy ON tenant_instance;

DROP FUNCTION IF EXISTS uisce_get_current_tenant();
