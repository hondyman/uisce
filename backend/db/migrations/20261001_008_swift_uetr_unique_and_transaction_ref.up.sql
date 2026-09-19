-- Migration: UETR uniqueness on swift_session_log + transaction_ref on cash_flow.settlement.
-- 
-- Issue 1: UETR uniqueness on swift_session_log
--   SWIFTNet retransmits with the same UETR must be discarded at the log level.
--   A unique index on (tenant_id, uetr) enforces idempotency for inbound messages.
--   Partial index excludes NULL UETRs (MT category 5 messages may omit UETR).
--
-- Issue 2: transaction_ref column on cash_flow.settlement
--   Required by PersistSettlementStatusActivity (GSIFI UPDATE) and the
--   GET /api/cash-flow/swift/instructions/{id} endpoint.
--   Column is nullable to preserve compatibility with existing settlement rows
--   that pre-date the SWIFT pipeline.
--
-- Both are additive-only (no data changes) — safe to apply hot.

-- 1. Unique index on swift_session_log(tenant_id, uetr) — partial (UETR NOT NULL).
--    Retransmission guard: same UETR from the same tenant is silently discarded
--    at the log-insertion level. The settlement_writer tile checks for duplicates
--    before writing to cash_flow.settlement; this index adds a second line of defence.
CREATE UNIQUE INDEX IF NOT EXISTS swift_session_log_uetr_tenant_uniq
    ON swift_session_log (tenant_id, uetr)
    WHERE uetr IS NOT NULL;

-- 2. transaction_ref column on cash_flow.settlement.
--    Enables PersistSettlementStatusActivity to UPDATE by (tenant_id, transaction_ref)
--    instead of scanning all recent rows.
--    Nullable: pre-existing settlement rows (non-SWIFT) remain valid.
ALTER TABLE cash_flow.settlement
    ADD COLUMN IF NOT EXISTS transaction_ref TEXT;

-- 3. Index on (tenant_id, transaction_ref) for the GSIFI UPDATE and GET endpoint.
--    Partial: only index rows with a non-null transaction_ref.
CREATE INDEX IF NOT EXISTS settlement_transaction_ref_tenant_idx
    ON cash_flow.settlement (tenant_id, transaction_ref)
    WHERE transaction_ref IS NOT NULL;

-- 4. Index on (tenant_id, updated_at DESC) for the reconciliation report query.
CREATE INDEX IF NOT EXISTS settlement_tenant_updated_at_idx
    ON cash_flow.settlement (tenant_id, updated_at DESC)
    WHERE valid_to IS NULL;

COMMENT ON COLUMN cash_flow.settlement.transaction_ref IS
    'SWIFT :20: transaction reference (MT541/543/548) or ISO 20022 EndToEndId (pacs.008). '
    'NULL for non-SWIFT settlement rows.';
