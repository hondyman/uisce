-- Migration: 20260913_001_report_executions_triggered_by.up.sql
-- Description: Add triggered_by column and index to report_executions table.
-- Supports Option (b) two-sided execution identity model:
--   - requested_by carries template owner identity (execution identity invariant)
--   - triggered_by carries caller user identity (attributable trigger action)
--   - Enables cross-tenant caller visibility for runs of shared gold-copy templates.

ALTER TABLE public.report_executions
    ADD COLUMN IF NOT EXISTS triggered_by TEXT;

CREATE INDEX IF NOT EXISTS idx_re_triggered_by
    ON public.report_executions(triggered_by);
