-- Migration 027: Transaction Master Schema
-- Bi-temporal, RLS-enabled, multi-tenant transaction ledger

CREATE SCHEMA IF NOT EXISTS edm;

-- ============================================
-- ROOT: Transaction Master
-- ============================================
CREATE TABLE IF NOT EXISTS edm.transaction_master (
    transaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Core Identity
    portfolio_id UUID NOT NULL,
    security_id UUID, -- Optional for cash-only transactions
    trade_date DATE NOT NULL,
    settlement_date DATE,
    booking_date DATE,
    
    -- Classification
    transaction_type TEXT NOT NULL, -- BUY, SELL, DIVIDEND, FEE, etc.
    transaction_subtype TEXT,
    
    -- Economics
    quantity NUMERIC(28,10),
    price NUMERIC(28,10),
    gross_amount NUMERIC(28,10),
    net_amount NUMERIC(28,10),
    commission NUMERIC(28,10),
    fees NUMERIC(28,10),
    taxes NUMERIC(28,10),
    accrued_interest NUMERIC(28,10),
    
    -- Currency / FX
    transaction_currency TEXT NOT NULL,
    settlement_currency TEXT,
    fx_rate NUMERIC(28,10),
    
    -- Counterparty / Account
    counterparty_id TEXT,
    broker_id TEXT,
    custody_account_id TEXT,
    
    -- Corporate Action Linkage
    corporate_action_id TEXT,
    
    -- Status / Audit
    status TEXT DEFAULT 'PENDING', -- PENDING, SETTLED, CANCELLED
    source_system TEXT NOT NULL,
    external_reference TEXT,
    
    -- Bi-temporal Versioning
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_to TIMESTAMPTZ DEFAULT 'infinity',
    system_from TIMESTAMPTZ DEFAULT NOW(),
    system_to TIMESTAMPTZ DEFAULT 'infinity',
    
    -- Multi-tenant Lineage
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    core_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE (portfolio_id, trade_date, external_reference, valid_from),
    CONSTRAINT chk_tx_type CHECK (transaction_type IN ('BUY', 'SELL', 'SHORT', 'COVER', 'DIVIDEND', 'INTEREST', 'FEE', 'TRANSFER', 'CORP_ACTION')),
    CONSTRAINT chk_currency_format CHECK (transaction_currency ~ '^[A-Z]{3}$')
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_tx_portfolio_date ON edm.transaction_master (portfolio_id, trade_date);
CREATE INDEX IF NOT EXISTS idx_tx_security ON edm.transaction_master (security_id, trade_date);
CREATE INDEX IF NOT EXISTS idx_tx_status ON edm.transaction_master (status, settlement_date);
CREATE INDEX IF NOT EXISTS idx_tx_tenant ON edm.transaction_master (tenant_id, portfolio_id);
CREATE INDEX IF NOT EXISTS idx_tx_valid ON edm.transaction_master (valid_from, valid_to) WHERE valid_to = 'infinity';

-- RLS Policies (Usice Architecture §6.2)
ALTER TABLE edm.transaction_master ENABLE ROW LEVEL SECURITY;

CREATE POLICY tx_tenant_isolation ON edm.transaction_master
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);

CREATE POLICY tx_core_read ON edm.transaction_master
    FOR SELECT USING (
        current_setting('app.tenant_scope', TRUE) IN ('multi', 'all') 
        OR tenant_id = current_setting('app.current_tenant', TRUE)::UUID
    );

-- ============================================
-- TRACE: Transaction → Position Impact
-- ============================================
CREATE TABLE IF NOT EXISTS edm.transaction_flow_trace (
    trace_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES edm.transaction_master(transaction_id),
    position_id UUID, -- Resulting position impact
    impact_type TEXT, -- OPEN, CLOSE, INCREASE, DECREASE
    quantity_delta NUMERIC(28,10),
    cost_basis_delta NUMERIC(28,10),
    realized_pl NUMERIC(28,10),
    processed_at TIMESTAMPTZ DEFAULT NOW(),
    tenant_id UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_trace_tx ON edm.transaction_flow_trace (transaction_id);

-- ============================================
-- GOLD TRACE: Survivorship Lineage
-- ============================================
CREATE TABLE IF NOT EXISTS edm.transaction_gold_trace (
    trace_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL,
    field_name TEXT NOT NULL,
    source_system TEXT NOT NULL,
    source_value JSONB,
    was_selected BOOLEAN,
    selection_reason TEXT,
    survivorship_run_id UUID,
    traced_at TIMESTAMPTZ DEFAULT NOW(),
    tenant_id UUID NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_gold_trace_tx ON edm.transaction_gold_trace (transaction_id, field_name);
