-- 20260908_multibook_synchronization_engine.up.sql
-- Unified Multi-Book Synchronization Engine (IBOR <-> ABOR <-> PBOR) & Tri-Party Recon Ledger

CREATE SCHEMA IF NOT EXISTS ledger_multi;

-- 1. Front-Office Real-Time IBOR Position State
CREATE TABLE IF NOT EXISTS ledger_multi.ibor_intraday_positions (
    position_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    security_node_id UUID NOT NULL,
    account_node_id UUID NOT NULL,
    settled_shares NUMERIC(28, 6) NOT NULL DEFAULT 0.000000,
    open_buy_shares NUMERIC(28, 6) NOT NULL DEFAULT 0.000000,
    open_sell_shares NUMERIC(28, 6) NOT NULL DEFAULT 0.000000,
    projected_net_shares NUMERIC(28, 6) GENERATED ALWAYS AS (settled_shares + open_buy_shares - open_sell_shares) STORED,
    projected_cash_usd NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    knowledge_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_ibor_position UNIQUE (tenant_id, portfolio_node_id, security_node_id, account_node_id)
);

-- 2. Back-Office ABOR Double-Entry General Ledger (WORM Compliant)
CREATE TABLE IF NOT EXISTS ledger_multi.abor_general_ledger_entries (
    entry_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    journal_batch_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    gl_account_code VARCHAR(50) NOT NULL, -- e.g., '1010-CASH', '1200-EQUITY-INVESTMENTS', '2010-PAYABLES'
    debit_amount NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    credit_amount NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    currency VARCHAR(10) NOT NULL DEFAULT 'USD',
    effective_date DATE NOT NULL, -- Te (Accounting Period)
    knowledge_time TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Tk (System Record Time)
    entry_type VARCHAR(30) NOT NULL, -- TRADE_SETTLEMENT, DIVIDEND_ACCRUAL, MGMT_FEE, RECON_ADJUSTMENT
    is_reversal BOOLEAN DEFAULT FALSE,
    merkle_leaf_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. PBOR Performance & Valuation Snapshots
CREATE TABLE IF NOT EXISTS ledger_multi.pbor_performance_snapshots (
    snapshot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    valuation_date DATE NOT NULL,
    beginning_nav NUMERIC(28, 4) NOT NULL,
    net_contributions NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    net_withdrawals NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    ending_nav NUMERIC(28, 4) NOT NULL,
    gross_twr NUMERIC(12, 8) NOT NULL,
    net_twr NUMERIC(12, 8) NOT NULL,
    rolling_xirr NUMERIC(12, 8),
    knowledge_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_pbor_snapshot UNIQUE (tenant_id, portfolio_node_id, valuation_date, knowledge_time)
);

-- 4. Tri-Party Reconciliation Ingestion & Break Ledger
CREATE TABLE IF NOT EXISTS ledger_multi.recon_break_instances (
    break_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    security_node_id UUID,
    break_type VARCHAR(50) NOT NULL, -- POSITION_QUANTITY, CASH_BALANCE, ACCRUED_INTEREST, UNMATCHED_TRADE
    internal_ibor_value NUMERIC(28, 6) NOT NULL,
    custodian_abor_value NUMERIC(28, 6) NOT NULL,
    variance_amount NUMERIC(28, 6) GENERATED ALWAYS AS (internal_ibor_value - custodian_abor_value) STORED,
    tolerance_threshold NUMERIC(28, 6) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'OPEN', -- OPEN, AUTO_RESOLVED, ESCALATED_FOUR_EYES, WRITTEN_OFF
    statement_source VARCHAR(100) NOT NULL, -- e.g., 'BNY_MELLON_MT535_20260825'
    resolution_journal_batch_id UUID,
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_abor_gl_bitemporal 
ON ledger_multi.abor_general_ledger_entries (tenant_id, portfolio_node_id, effective_date, knowledge_time);
