-- =============================================================================
-- 20260917_001_export_artifacts_and_events.down.sql
--
-- Reverts: export_artifacts and export_artifact_events tables, all indexes,
-- all RLS policies, and all grants.
-- The report_templates table and its data are not affected.
-- =============================================================================

-- export_artifact_events cleanup first (FK depends on export_artifacts)
REVOKE INSERT, SELECT ON public.export_artifact_events FROM app_user;
REVOKE UPDATE, DELETE ON public.export_artifact_events FROM PUBLIC;
REVOKE UPDATE, DELETE ON public.export_artifact_events FROM app_user;
DROP POLICY IF EXISTS eae_tenant_isolation ON public.export_artifact_events;
ALTER TABLE public.export_artifact_events DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS public.idx_eae_tenant_id_created_at;
DROP INDEX IF EXISTS public.idx_eae_export_created_at;
DROP INDEX IF EXISTS public.idx_eae_export_id_created_at;

DROP TABLE IF EXISTS public.export_artifact_events;

-- export_artifacts
REVOKE INSERT, SELECT ON public.export_artifacts FROM app_user;
REVOKE UPDATE ON public.export_artifacts FROM app_user;
REVOKE DELETE ON public.export_artifacts FROM PUBLIC;
REVOKE DELETE ON public.export_artifacts FROM app_user;
DROP POLICY IF EXISTS ea_tenant_isolation ON public.export_artifacts;
ALTER TABLE public.export_artifacts DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS public.idx_ea_status;
DROP INDEX IF EXISTS public.idx_ea_template_id;
DROP INDEX IF EXISTS public.idx_ea_exporter_user_id;
DROP INDEX IF EXISTS public.idx_ea_tenant_id_created_at;

DROP TABLE IF EXISTS public.export_artifacts;

DO $$ BEGIN
    RAISE NOTICE 'export_artifacts and export_artifact_events migration rolled back';
END $$;
