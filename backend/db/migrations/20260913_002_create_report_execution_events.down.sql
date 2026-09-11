-- =============================================================================
-- 20260913_002_create_report_execution_events.down.sql
--
-- Reverts the report_execution_events table and all its indexes and policies.
-- The report_executions table and its data are not affected.
-- =============================================================================

DROP POLICY IF EXISTS ree_tenant_isolation ON public.report_execution_events;
ALTER TABLE public.report_execution_events DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS public.idx_ree_tenant_id_created_at;
DROP INDEX IF EXISTS public.idx_ree_created_at;
DROP INDEX IF EXISTS public.idx_ree_execution_id_created_at;

DROP TABLE IF EXISTS public.report_execution_events;

DO $$ BEGIN
    RAISE NOTICE 'report_execution_events migration rolled back';
END $$;
