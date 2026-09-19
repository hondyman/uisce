DROP INDEX IF EXISTS settlement_transaction_ref_tenant_uniq;

-- Restore non-unique index from migration 008.
CREATE INDEX IF NOT EXISTS settlement_transaction_ref_tenant_idx
    ON cash_flow.settlement (tenant_id, transaction_ref)
    WHERE transaction_ref IS NOT NULL;
