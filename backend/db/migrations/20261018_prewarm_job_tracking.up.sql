-- Migration: 20261018_prewarm_job_tracking.up.sql
-- Off-Peak Prewarm Execution Ledger Job Tracking, Trigger Attribution & Nullable BO Root
--
-- Depends on: 20261016_predictive_finops_and_smoothing.up.sql

ALTER TABLE finops.prewarm_execution_ledger
    ADD COLUMN IF NOT EXISTS job_id UUID,
    ADD COLUMN IF NOT EXISTS triggered_by UUID;

ALTER TABLE finops.prewarm_execution_ledger
    ALTER COLUMN bo_id DROP NOT NULL;

ALTER TABLE finops.prewarm_execution_ledger
    ALTER COLUMN target_metric DROP NOT NULL;

ALTER TABLE finops.prewarm_execution_ledger
    ALTER COLUMN target_metric SET DEFAULT 'ALL';

CREATE INDEX IF NOT EXISTS idx_prewarm_ledger_tenant_job
    ON finops.prewarm_execution_ledger (tenant_id, job_id);
