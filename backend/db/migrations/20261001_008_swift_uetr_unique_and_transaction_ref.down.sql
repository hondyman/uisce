-- Rollback for 20261001_008_swift_uetr_unique_and_transaction_ref.up.sql

-- Remove indexes added in the up migration
DROP INDEX IF EXISTS swift_session_log_uetr_tenant_uniq;
DROP INDEX IF EXISTS settlement_transaction_ref_tenant_idx;
DROP INDEX IF EXISTS settlement_tenant_updated_at_idx;

-- Remove transaction_ref column from cash_flow.settlement.
-- WARNING: this drops any transaction_ref data that was written by the
-- swift_settlement_writer tile. Only apply if rolling back the full SWIFT
-- pipeline deployment.
ALTER TABLE cash_flow.settlement DROP COLUMN IF EXISTS transaction_ref;
