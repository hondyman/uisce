-- Grants for non-superuser role uisce_mcp_app (NOBYPASSRLS).
-- Role is created out-of-band (live cutover); this migration is idempotent.
-- Scope: MCP-read tables with gold-aware FORCE policies + helpers + schemas.
-- Full-backend DSN cutover still requires broader grants + BeginTx→GUC migration
-- (~100+ files open txs without ApplyTenantGUCs).

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'uisce_mcp_app') THEN
    CREATE ROLE uisce_mcp_app WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
    COMMENT ON ROLE uisce_mcp_app IS 'Non-superuser app/MCP role for FORCE RLS. Password set out-of-band; DSN in gitignored .env (UISCE_APP_DSN).';
  END IF;
END $$;

GRANT USAGE ON SCHEMA public TO uisce_mcp_app;
GRANT SELECT ON public.page_definitions TO uisce_mcp_app;
GRANT SELECT ON public.business_objects TO uisce_mcp_app;
GRANT SELECT ON public.business_object_fields TO uisce_mcp_app;
GRANT SELECT ON public.catalog_edge TO uisce_mcp_app;
GRANT SELECT ON public.tenants TO uisce_mcp_app;

-- Helpers used by gold-aware policies
GRANT EXECUTE ON FUNCTION uisce_get_current_tenant() TO uisce_mcp_app;
GRANT EXECUTE ON FUNCTION uisce_get_gold_tenant() TO uisce_mcp_app;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'catalog_drift') THEN
    EXECUTE 'GRANT USAGE ON SCHEMA catalog_drift TO uisce_mcp_app';
    IF to_regclass('catalog_drift.schema_drift_proposals') IS NOT NULL THEN
      EXECUTE 'GRANT SELECT ON catalog_drift.schema_drift_proposals TO uisce_mcp_app';
    END IF;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'mdm') THEN
    EXECUTE 'GRANT USAGE ON SCHEMA mdm TO uisce_mcp_app';
    IF to_regclass('mdm.universal_exception_queue') IS NOT NULL THEN
      EXECUTE 'GRANT SELECT ON mdm.universal_exception_queue TO uisce_mcp_app';
    END IF;
  END IF;
  -- MCP audit writer (CallTool ledger) — INSERT only when table exists
  IF to_regclass('catalog_mdm_ai.mcp_tool_execution_logs') IS NOT NULL THEN
    EXECUTE 'GRANT USAGE ON SCHEMA catalog_mdm_ai TO uisce_mcp_app';
    EXECUTE 'GRANT INSERT, SELECT ON catalog_mdm_ai.mcp_tool_execution_logs TO uisce_mcp_app';
  END IF;
END $$;
