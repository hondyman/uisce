-- 20260911_corporate_actions_lifecycle.up.sql
-- Autonomous Corporate Actions Lifecycle Engine & Tax-Lot Re-Allocation Ledger

CREATE SCHEMA IF NOT EXISTS ledger_ca;

-- 1. Master Corporate Action Event Definitions
CREATE TABLE IF NOT EXISTS ledger_ca.event_definitions (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_key VARCHAR(150) NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- FORWARD_SPLIT, REVERSE_SPLIT, SPIN_OFF, MERGER_STOCK, CASH_DIVIDEND
    mandate_type VARCHAR(20) NOT NULL DEFAULT 'MANDATORY',
    target_security_node_id UUID NOT NULL,
    new_security_node_id UUID,
    
    announcement_date DATE NOT NULL,
    ex_date DATE NOT NULL,
    record_date DATE NOT NULL,
    payable_date DATE NOT NULL,
    
    split_ratio_numerator NUMERIC(18, 8) NOT NULL DEFAULT 1.00000000,
    split_ratio_denominator NUMERIC(18, 8) NOT NULL DEFAULT 1.00000000,
    cost_basis_allocation_pct NUMERIC(6, 4) NOT NULL DEFAULT 1.0000,
    cash_per_share NUMERIC(18, 6) NOT NULL DEFAULT 0.000000,
    fractional_share_treatment VARCHAR(30) NOT NULL DEFAULT 'CASH_IN_LIEU',
    
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_EX_DATE',
    merkle_event_seal VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_ca_event UNIQUE (tenant_id, event_key)
);

-- 2. Portfolio Entitlement Snapshot Ledger
CREATE TABLE IF NOT EXISTS ledger_ca.account_entitlements (
    entitlement_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES ledger_ca.event_definitions(event_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    account_node_id UUID NOT NULL,
    
    held_shares_at_record_date NUMERIC(28, 6) NOT NULL,
    calculated_entitled_shares NUMERIC(28, 6) NOT NULL,
    allocated_whole_shares NUMERIC(28, 6) NOT NULL,
    fractional_share_remainder NUMERIC(18, 8) NOT NULL,
    cash_in_lieu_amount_usd NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    
    execution_status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    processed_at TIMESTAMPTZ,
    CONSTRAINT uq_account_event_entitlement UNIQUE (tenant_id, event_id, account_node_id)
);

-- 3. Tax-Lot Adjustment Traceability Ledger
CREATE TABLE IF NOT EXISTS ledger_ca.lot_cost_basis_adjustments (
    adjustment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES ledger_ca.event_definitions(event_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    original_lot_id UUID NOT NULL,
    new_spinoff_lot_id UUID,
    
    pre_event_shares NUMERIC(28, 6) NOT NULL,
    post_event_shares NUMERIC(28, 6) NOT NULL,
    pre_event_cost_basis_per_share NUMERIC(18, 6) NOT NULL,
    post_event_cost_basis_per_share NUMERIC(18, 6) NOT NULL,
    
    cash_in_lieu_received_usd NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    realized_capital_gain_on_cil NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    merkle_leaf_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ca_event_timeline 
ON ledger_ca.event_definitions (tenant_id, ex_date, status);
