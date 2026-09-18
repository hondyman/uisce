-- 1) Drop redundant catalog_edge tenant_isolation_policy.
--    Gold-aware policy already covers tenant equality OR gold; the older
--    policy is plain tenant equality (no gold) — OR-combine makes it a no-op
--    when GUCs are set, and confusing archaeology when they aren't.
DROP POLICY IF EXISTS tenant_isolation_policy ON public.catalog_edge;

-- 2) FORCE RLS on live MDM queue (catalog_mdm, not the imagined mdm schema).
ALTER TABLE IF EXISTS catalog_mdm.universal_exception_queue ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS catalog_mdm.universal_exception_queue FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS catalog_mdm_exception_queue_tenant_policy ON catalog_mdm.universal_exception_queue;
CREATE POLICY catalog_mdm_exception_queue_tenant_policy ON catalog_mdm.universal_exception_queue
    FOR ALL
    USING (tenant_id = uisce_get_current_tenant())
    WITH CHECK (tenant_id = uisce_get_current_tenant());

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'uisce_mcp_app') THEN
    EXECUTE 'GRANT USAGE ON SCHEMA catalog_mdm TO uisce_mcp_app';
    EXECUTE 'GRANT SELECT ON catalog_mdm.universal_exception_queue TO uisce_mcp_app';
  END IF;
END $$;
