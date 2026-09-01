-- 20260909_tax_alpha_optimizer_kernel.up.sql
-- WASM Tax-Alpha & Parametric Portfolio Optimization Kernel

CREATE SCHEMA IF NOT EXISTS portfolio_opt;

-- 1. Optimization Strategy & Risk Profiles
CREATE TABLE IF NOT EXISTS portfolio_opt.strategy_profiles (
    profile_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    profile_key VARCHAR(100) NOT NULL,
    profile_name VARCHAR(150) NOT NULL,
    objective_type VARCHAR(50) NOT NULL, -- MEAN_VARIANCE, MIN_TRACKING_ERROR, RISK_PARITY, TAX_HARVEST_MAX
    risk_aversion_lambda NUMERIC(10, 6) NOT NULL DEFAULT 1.000000,
    max_turnover_pct NUMERIC(6, 4) NOT NULL DEFAULT 0.2000,
    max_single_stock_weight NUMERIC(6, 4) NOT NULL DEFAULT 0.0500,
    short_term_tax_rate NUMERIC(6, 4) NOT NULL DEFAULT 0.3700,
    long_term_tax_rate NUMERIC(6, 4) NOT NULL DEFAULT 0.2000,
    wash_sale_lookback_days INT NOT NULL DEFAULT 30,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_strategy UNIQUE (tenant_id, profile_key)
);

-- 2. Granular Tax-Lot Inventory (OLTP Ledger)
CREATE TABLE IF NOT EXISTS portfolio_opt.tax_lot_inventory (
    lot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    security_node_id UUID NOT NULL,
    account_node_id UUID NOT NULL,
    acquisition_date DATE NOT NULL,
    cost_basis_per_share NUMERIC(18, 6) NOT NULL,
    current_shares NUMERIC(28, 6) NOT NULL,
    current_market_price NUMERIC(18, 6) NOT NULL,
    unrealized_gain_loss NUMERIC(28, 4) GENERATED ALWAYS AS ((current_market_price - cost_basis_per_share) * current_shares) STORED,
    tax_term_status VARCHAR(20) NOT NULL DEFAULT 'SHORT_TERM',
    is_wash_sale_blocked BOOLEAN DEFAULT FALSE,
    blocked_until_date DATE,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Optimization & Rebalance Execution Runs
CREATE TABLE IF NOT EXISTS portfolio_opt.rebalance_execution_runs (
    run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    profile_id UUID NOT NULL REFERENCES portfolio_opt.strategy_profiles(profile_id) ON DELETE CASCADE,
    status VARCHAR(30) NOT NULL DEFAULT 'COMPLETED',
    gross_tax_harvested_usd NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    projected_turnover_pct NUMERIC(6, 4) NOT NULL,
    solver_latency_ms NUMERIC(8, 3) NOT NULL,
    total_orders_generated INT NOT NULL DEFAULT 0,
    merkle_execution_seal VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 4. Generated Order Intent Tickets (FIX Tag 35=D Pre-Stage)
CREATE TABLE IF NOT EXISTS portfolio_opt.rebalance_order_tickets (
    ticket_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES portfolio_opt.rebalance_execution_runs(run_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    security_node_id UUID NOT NULL,
    target_lot_id UUID REFERENCES portfolio_opt.tax_lot_inventory(lot_id) ON DELETE SET NULL,
    order_side VARCHAR(10) NOT NULL, -- BUY, SELL
    order_shares NUMERIC(28, 6) NOT NULL,
    limit_price NUMERIC(18, 6),
    estimated_tax_impact_usd NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    is_substitute_asset BOOLEAN DEFAULT FALSE,
    substitute_for_security_id UUID,
    fix_clordid VARCHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tax_lot_unrealized 
ON portfolio_opt.tax_lot_inventory (tenant_id, portfolio_node_id, unrealized_gain_loss ASC);
