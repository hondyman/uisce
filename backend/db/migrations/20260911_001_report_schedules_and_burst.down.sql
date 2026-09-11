-- =============================================================================
-- 20260911_001_report_schedules_and_burst.down.sql
--
-- Drops all tables and indexes created by the up migration.
-- Order: artifacts → batches → schedules → calendars (FK dependency order).
-- report_cache_metadata index dropped; table itself was pre-existing.
-- report_executions columns are NOT dropped (they were pre-existing on alpha).
-- =============================================================================

DROP INDEX IF EXISTS public.idx_rcm_tenant_template;

DROP TABLE IF EXISTS public.report_burst_artifacts;
DROP TABLE IF EXISTS public.report_burst_batches;

DROP INDEX IF EXISTS public.idx_report_schedules_active;
DROP INDEX IF EXISTS public.idx_report_schedules_owner;
DROP INDEX IF EXISTS public.idx_report_schedules_template;
DROP INDEX IF EXISTS public.idx_report_schedules_tenant;
DROP TABLE IF EXISTS public.report_schedules;

DROP TABLE IF EXISTS public.tenant_calendar_holidays;
DROP INDEX IF EXISTS public.idx_burst_batches_tenant_status;
DROP INDEX IF EXISTS public.idx_burst_batches_schedule;
DROP TABLE IF EXISTS public.tenant_exchange_calendars;

DO $$ BEGIN
    RAISE NOTICE 'report_schedules_and_burst migration rolled back';
END $$;
