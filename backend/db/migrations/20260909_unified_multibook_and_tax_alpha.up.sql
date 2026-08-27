-- 20260909_unified_multibook_and_tax_alpha.up.sql
-- Unified Multi-Book Synchronization (IBOR / ABOR / PBOR) & WASM Tax-Alpha Rebalancing Kernel

CREATE SCHEMA IF NOT EXISTS ledger_multibook;
CREATE SCHEMA IF NOT EXISTS portfolio_alpha;

-- 1. Unified Multi-Book Event Seam (Bitemporal Ledger Entries)
CREATE TABLE IF NOT EXISTS ledger_multibook.bitemporal_entries (
    entry_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL,
    book_type VARCHAR(20) NOT NULL, -- IBOR_PROJECTED, ABOR_SETTLED, PBOR_PERFORMANCE
    event_type VARCHAR(50) NOT NULL, -- TRADE_FILL, CASH_SETTLEMENT, DIVIDEND_ACCRUAL, CORP_ACTION
    security_node_id UUID NOT NULL,
    lot_id UUID,
    quantity_delta NUMERIC(28, 8) NOT NULL,
    cash_delta NUMERIC(28, 4) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    event_time TIMESTAMPTZ NOT NULL, -- Te: Effective time in the real world
    knowledge_time TIMESTAMPTZ NOT NULL DEFAULT NOW(), -- Tk: System recording time
    settlement_status VARCHAR(30) NOT NULL DEFAULT 'PENDING', -- PENDING, SETTLED, ADJUSTED, RECON_BREAK
    merkle_entry_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Tri-Party Reconciliation & Auto-Healing Breaks Ledger
CREATE TABLE IF NOT EXISTS ledger_multibook.recon_break_queue (
    break_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL,
    security_node_id UUID NOT NULL,
    break_type VARCHAR(50) NOT NULL, -- POSITION_MISMATCH, CASH_UNSETTLED, ACCRUAL_DISCREPANCY
    internal_quantity NUMERIC(28, 8) NOT NULL,
    custodian_quantity NUMERIC(28, 8) NOT NULL,
    variance_bps NUMERIC(8, 4) NOT NULL,
    auto_heal_strategy VARCHAR(50), -- SYNTHETIC_JOURNAL_ADJUSTMENT, ESCALATE_MAKER_CHECKER
    status VARCHAR(30) NOT NULL DEFAULT 'OPEN', -- OPEN, AUTO_HEALED, MANUAL_RESOLVED
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Tax-Alpha Harvesting & Rebalancing Runs
CREATE TABLE IF NOT EXISTS portfolio_alpha.rebalance_runs (
    run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_id UUID NOT NULL,
    optimization_objective VARCHAR(50) NOT NULL, -- TAX_ALPHA_HARVESTING, RISK_PARITY, MEAN_VARIANCE
    gross_loss_harvested_usd NUMERIC(18, 2) NOT NULL DEFAULT 0.00,
    estimated_tax_savings_usd NUMERIC(18, 2) NOT NULL DEFAULT 0.00,
    wash_sale_conflicts_prevented INT NOT NULL DEFAULT 0,
    order_tickets_generated_count INT NOT NULL DEFAULT 0,
    solver_latency_ms INT NOT NULL DEFAULT 0,
    merkle_execution_seal VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 4. Generated FIX Tag 35=D Execution Tickets
CREATE TABLE IF NOT EXISTS portfolio_alpha.generated_order_tickets (
    ticket_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES portfolio_alpha.rebalance_runs(run_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL,
    security_node_id UUID NOT NULL,
    side VARCHAR(10) NOT NULL, -- BUY, SELL, SELL_SHORT
    quantity NUMERIC(18, 6) NOT NULL,
    order_type VARCHAR(20) NOT NULL DEFAULT 'LIMIT',
    limit_price NUMERIC(18, 6),
    fix_tag_35_payload TEXT NOT NULL,
    allocation_split_rule VARCHAR(100),
    status VARCHAR(30) NOT NULL DEFAULT 'READY_FOR_EMS', -- READY_FOR_EMS, TRANSMITTED, FILLED
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_multibook_account_time 
ON ledger_multibook.bitemporal_entries (tenant_id, account_id, book_type, event_time, knowledge_time);
