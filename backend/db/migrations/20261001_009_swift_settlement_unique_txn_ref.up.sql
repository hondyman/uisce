-- Migration: Make settlement_transaction_ref unique so ON CONFLICT arbiter works.
--
-- Migration 008 created settlement_transaction_ref_tenant_idx as NON-UNIQUE.
-- settlement_writer.go uses:
--   ON CONFLICT (tenant_id, transaction_ref) WHERE transaction_ref IS NOT NULL DO NOTHING
-- PostgreSQL requires the target to be a UNIQUE or EXCLUSION constraint — without
-- this, every INSERT the pipeline produces fails at runtime with:
--   "no unique or exclusion constraint matching the ON CONFLICT specification"
-- Unit tests (sqlmock) cannot catch this because they never validate against schema.
--
-- Idempotency contract: (tenant_id, transaction_ref) — custodian_id was dropped
-- because a single instruction has exactly one (tenant, ref) pair per the :20: spec.
-- Custodian is stored in the row; it need not be in the conflict key.
-- HANDOFF_SWIFT_SETTLEMENT.md §Idempotency updated accordingly.

DROP INDEX IF EXISTS settlement_transaction_ref_tenant_idx;   -- was non-unique from 008

CREATE UNIQUE INDEX settlement_transaction_ref_tenant_uniq
    ON cash_flow.settlement (tenant_id, transaction_ref)
    WHERE transaction_ref IS NOT NULL;
