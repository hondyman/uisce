-- Slice 4: retire legacy report_schedules / burst tables.
-- Active rows were copied into public.schedules (kind=report) by
-- backend/cmd/migrate-report-schedules. Creates have returned 410 since PR #192.
--
-- report_executions.schedule_id is kept as a nullable historical column with
-- the FK removed so past execution rows stay queryable.

ALTER TABLE public.report_executions
    DROP CONSTRAINT IF EXISTS report_executions_schedule_id_fkey;

DROP TABLE IF EXISTS public.report_burst_artifacts;
DROP TABLE IF EXISTS public.report_burst_batches;
DROP TABLE IF EXISTS public.report_schedules;

COMMENT ON COLUMN public.report_executions.schedule_id IS
    'Legacy report_schedules id (FK removed 20261112); prefer parameters->>''core_schedule_id'' for the one scheduler.';
