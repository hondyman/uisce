-- Migration: 20261018_prewarm_job_tracking.down.sql
-- Rollback for prewarm job tracking extensions.
--
-- PRINCIPLE: down migrations never block on accumulated data. The pre-20261018
-- handler always wrote a non-NULL bo_id, so the legacy NOT NULL constraint is
-- decorative for the code it pairs with — the constraint restoration is an
-- operator decision that requires pruning run-level (NULL bo_id) rows, which is
-- out of scope for automated rollback.
--
-- Safety on populated tables: this file drops the new schema additions and
-- leaves bo_id nullable. It does NOT re-add the NOT NULL constraint. After
-- rollback, a populated ledger with run-level (NULL bo_id) rows survives intact.

DROP INDEX IF EXISTS finops.idx_prewarm_ledger_tenant_job;

ALTER TABLE finops.prewarm_execution_ledger
    DROP COLUMN IF EXISTS job_id,
    DROP COLUMN IF EXISTS triggered_by;

-- bo_id intentionally left nullable: run-level rows (PENDING with NULL bo_id
-- inserted by the synchronous CreatePendingExecution, plus SKIPPED_NO_TARGETS
-- and SKIPPED_BELOW_THRESHOLD terminal rows) legitimately carry NULL.
-- Restoring NOT NULL requires manual row pruning and is the operator's choice.
