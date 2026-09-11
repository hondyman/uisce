-- 20260915_001_report_executions_schedule_id.up.sql
-- Phase 2 of monitoring feature: add schedule_id column to report_executions
-- for joinable schedule-scoped read path (ListScheduleExecutions).
--
-- ON DELETE SET NULL rationale: consistent with the soft-delete model for
-- report_schedules (deleted_at, not hard delete). When a schedule is
-- soft-deleted, its execution history is preserved but the FK link is severed.
-- This is the same convention as the report_execution_events ON DELETE CASCADE
-- comment in 20260913_002.
--
-- Backfill: existing rows may have schedule_id in parameters->>'schedule_id'.
-- Pre-instrumentation rows have neither — they remain NULL and are filtered out
-- of the schedule-scoped endpoint (the schedule was tracked externally).

ALTER TABLE public.report_executions
    ADD COLUMN IF NOT EXISTS schedule_id UUID
    REFERENCES public.report_schedules(id) ON DELETE SET NULL;

-- Backfill: only update where the referenced schedule actually exists.
-- Pre-instrumentation rows have no valid schedule FK — they stay NULL and are
-- excluded from schedule-scoped queries (their schedule was tracked externally).
UPDATE public.report_executions e
SET schedule_id = (e.parameters->>'schedule_id')::uuid
FROM public.report_schedules s
WHERE e.schedule_id IS NULL
  AND e.parameters ? 'schedule_id'
  AND (e.parameters->>'schedule_id') ~ '^[0-9a-fA-F-]{36}$'
  AND s.id = (e.parameters->>'schedule_id')::uuid;

CREATE INDEX IF NOT EXISTS idx_report_executions_schedule_id
    ON public.report_executions(schedule_id)
    WHERE schedule_id IS NOT NULL;

COMMENT ON COLUMN public.report_executions.schedule_id IS
    'FK to report_schedules; denormalized from parameters->>''schedule_id'' for joinable read paths.';
