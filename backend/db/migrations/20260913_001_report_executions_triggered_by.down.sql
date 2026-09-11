-- Migration: 20260913_001_report_executions_triggered_by.down.sql
-- Description: Revert triggered_by column and index from report_executions table.

DROP INDEX IF EXISTS public.idx_re_triggered_by;
ALTER TABLE public.report_executions DROP COLUMN IF EXISTS triggered_by;
