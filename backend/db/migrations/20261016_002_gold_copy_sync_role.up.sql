-- 20261016_002_gold_copy_sync_role.up.sql
-- Prerequisite for landing 20261016_001_strict_tenant_rls.up.sql safely.
--
-- That migration replaces a fail-open RLS policy with a strict, fail-closed
-- one on tenant_instance, tenant_product, tenant_product_datasource,
-- connections, and audit_logs. Auditing every direct query against those
-- five tables (grep across backend/**, checked for tenant-context setting)
-- found the gold-copy sync/provisioning subsystem is *structurally*
-- cross-tenant: it reads the gold-copy tenant's rows and writes into every
-- other tenant's rows in the same operation, often in loops
-- (internal/db/instance_clone.go's CloneGoldCopyInstance,
-- SyncGoldCopyConnectionToAllInstances, SyncAllConnectionsForInstance;
-- internal/temporal/activities/gold_copy_activities.go's
-- syncConnectionToTenant; internal/api/connection_sync_handler.go's
-- SyncConnectionsFromGoldCopy; internal/sync/tenant_worker.go's tenant
-- deprovisioning). None of these set a single tenant's session GUC, because
-- there isn't one — the whole point is touching every tenant.
--
-- Postgres's BYPASSRLS is a role-level attribute, not a per-table one — it
-- can't be scoped by grant the way SELECT/INSERT/UPDATE/DELETE can. So the
-- scoping here comes from what this role is granted, not from BYPASSRLS
-- itself: it gets BYPASSRLS plus explicit grants on exactly the five tables
-- this subsystem touches, and nothing else. Same pattern as
-- 20260916_001_app_admin_read_role.up.sql (monitoring reads); this is the
-- write-capable equivalent for gold-copy sync and tenant provisioning.
--
-- This migration only creates the role. Wiring the confirmed cross-tenant
-- call sites to use it (a second connection pool/DSN, or SET ROLE within
-- their existing transactions) is deliberately NOT done here — that's
-- real, security-sensitive production code across 4+ files and belongs in
-- its own reviewed change, not bundled into standing up the role.
-- 20261016_001_strict_tenant_rls.up.sql should not be applied to alpha
-- until that wiring lands and is verified.

CREATE ROLE uisce_gold_copy_sync WITH LOGIN NOCREATEDB NOCREATEROLE BYPASSRLS;

GRANT SELECT, INSERT, UPDATE, DELETE ON
    public.tenant_instance,
    public.tenant_product,
    public.tenant_product_datasource,
    public.connections
TO uisce_gold_copy_sync;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'audit_logs') THEN
        GRANT SELECT, INSERT ON public.audit_logs TO uisce_gold_copy_sync;
    END IF;
END $$;

COMMENT ON ROLE uisce_gold_copy_sync IS 'System role for gold-copy sync and tenant provisioning/deprovisioning: structurally cross-tenant operations that cannot run under a single tenant''s RLS context. BYPASSRLS, scoped in practice to tenant_instance, tenant_product, tenant_product_datasource, connections, and audit_logs via explicit grants only — no other table access.';
