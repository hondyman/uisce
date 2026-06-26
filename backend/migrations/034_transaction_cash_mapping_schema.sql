-- Migration 034: Transaction to Cash Ledger Mapping Schema
-- Per Whitepaper §7: Semantic Execution Fabric
-- ============================================

CREATE TABLE IF NOT EXISTS edm.transaction_cash_mapping (
    mapping_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES edm.transaction_master(transaction_id),
    cash_ledger_id UUID REFERENCES edm.cash_ledger(cash_ledger_id),
    mapping_type TEXT NOT NULL, -- SETTLEMENT, COMMISSION, FEE, TAX, INCOME
    amount NUMERIC(28,10) NOT NULL,
    currency TEXT NOT NULL,
    value_date DATE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    
    UNIQUE (transaction_id, mapping_type)
);

CREATE INDEX IF NOT EXISTS idx_tx_cash_map_tx ON edm.transaction_cash_mapping (transaction_id);
CREATE INDEX IF NOT EXISTS idx_tx_cash_map_ledger ON edm.transaction_cash_mapping (cash_ledger_id);
