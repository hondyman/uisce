-- migrations/001_uma_tables.sql
-- UMA Account Management Tables

-- ============================================================================
-- UMA Accounts Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS uma_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, inactive, archived
    aum DECIMAL(19, 2) NOT NULL DEFAULT 0,
    target_allocation JSONB, -- {"equities": 0.60, "fixed_income": 0.30, ...}
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_rebalanced TIMESTAMP,
    created_by UUID,
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE INDEX idx_uma_accounts_tenant_id ON uma_accounts(tenant_id);
CREATE INDEX idx_uma_accounts_status ON uma_accounts(status);
CREATE INDEX idx_uma_accounts_created_at ON uma_accounts(created_at DESC);

-- ============================================================================
-- UMA Sleeves Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS uma_sleeves (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uma_account_id UUID NOT NULL,
    model VARCHAR(100) NOT NULL, -- "Growth", "Conservative", "Alternatives"
    sleeve_type VARCHAR(100) NOT NULL, -- "equities", "fixed_income", "alternatives"
    target_allocation DECIMAL(5, 4) NOT NULL, -- 0.6000 = 60%
    current_allocation DECIMAL(5, 4) NOT NULL DEFAULT 0,
    drift DECIMAL(5, 4) NOT NULL DEFAULT 0, -- current - target
    min_drift_threshold DECIMAL(5, 4) DEFAULT 0.05, -- 5%
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, pending, rebalancing
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_uma_account FOREIGN KEY (uma_account_id) REFERENCES uma_accounts(id) ON DELETE CASCADE
);

CREATE INDEX idx_uma_sleeves_uma_account_id ON uma_sleeves(uma_account_id);
CREATE INDEX idx_uma_sleeves_status ON uma_sleeves(status);
CREATE INDEX idx_uma_sleeves_drift ON uma_sleeves(drift);

-- ============================================================================
-- UMA Holdings Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS uma_holdings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sleeve_id UUID NOT NULL,
    cusip VARCHAR(9) NOT NULL,
    security_id VARCHAR(50) NOT NULL,
    security_name VARCHAR(255) NOT NULL,
    quantity DECIMAL(19, 8) NOT NULL,
    unit_cost DECIMAL(19, 6) NOT NULL,
    market_price DECIMAL(19, 6) NOT NULL,
    market_value DECIMAL(19, 2) NOT NULL,
    unrealized_gain DECIMAL(19, 2) NOT NULL, -- market_value - cost_basis
    cost_basis DECIMAL(19, 2) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_sleeve FOREIGN KEY (sleeve_id) REFERENCES uma_sleeves(id) ON DELETE CASCADE
);

CREATE INDEX idx_uma_holdings_sleeve_id ON uma_holdings(sleeve_id);
CREATE INDEX idx_uma_holdings_cusip ON uma_holdings(cusip);
CREATE INDEX idx_uma_holdings_security_id ON uma_holdings(security_id);
CREATE INDEX idx_uma_holdings_updated_at ON uma_holdings(updated_at DESC);

-- ============================================================================
-- UMA Rebalance Requests Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS uma_rebalance_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    uma_account_id UUID NOT NULL,
    request_type VARCHAR(50) NOT NULL, -- "drift", "manual", "scheduled"
    reason TEXT,
    initiated_by UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, approved, executing, completed, failed
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_uma_account_req FOREIGN KEY (uma_account_id) REFERENCES uma_accounts(id) ON DELETE CASCADE
);

CREATE INDEX idx_uma_rebalance_requests_tenant_id ON uma_rebalance_requests(tenant_id);
CREATE INDEX idx_uma_rebalance_requests_uma_account_id ON uma_rebalance_requests(uma_account_id);
CREATE INDEX idx_uma_rebalance_requests_status ON uma_rebalance_requests(status);
CREATE INDEX idx_uma_rebalance_requests_created_at ON uma_rebalance_requests(created_at DESC);

-- ============================================================================
-- UMA Rebalance Plans Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS uma_rebalance_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id UUID,
    uma_account_id UUID NOT NULL,
    total_tax_impact DECIMAL(19, 2) NOT NULL DEFAULT 0,
    total_cost DECIMAL(19, 2) NOT NULL DEFAULT 0,
    trades JSONB NOT NULL DEFAULT '[]', -- Array of trade objects
    status VARCHAR(50) NOT NULL DEFAULT 'draft', -- draft, pending_approval, approved, executing, completed
    approved_at TIMESTAMP,
    approved_by UUID,
    executed_at TIMESTAMP,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_uma_account_plan FOREIGN KEY (uma_account_id) REFERENCES uma_accounts(id) ON DELETE CASCADE,
    CONSTRAINT fk_rebalance_request FOREIGN KEY (request_id) REFERENCES uma_rebalance_requests(id) ON DELETE SET NULL
);

CREATE INDEX idx_uma_rebalance_plans_uma_account_id ON uma_rebalance_plans(uma_account_id);
CREATE INDEX idx_uma_rebalance_plans_status ON uma_rebalance_plans(status);
CREATE INDEX idx_uma_rebalance_plans_created_at ON uma_rebalance_plans(created_at DESC);

-- ============================================================================
-- UMA Rebalance History Table
-- ============================================================================
CREATE TABLE IF NOT EXISTS uma_rebalance_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL,
    uma_account_id UUID NOT NULL,
    completed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    total_trade_count INT NOT NULL DEFAULT 0,
    success_count INT NOT NULL DEFAULT 0,
    failure_count INT NOT NULL DEFAULT 0,
    total_tax_impact DECIMAL(19, 2) NOT NULL DEFAULT 0,
    total_cost DECIMAL(19, 2) NOT NULL DEFAULT 0,
    pre_drift JSONB, -- {"sleeves": [{"sleeve_id": "...", "drift": 0.05}, ...]}
    post_drift JSONB,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_plan_history FOREIGN KEY (plan_id) REFERENCES uma_rebalance_plans(id) ON DELETE SET NULL,
    CONSTRAINT fk_uma_account_history FOREIGN KEY (uma_account_id) REFERENCES uma_accounts(id) ON DELETE CASCADE
);

CREATE INDEX idx_uma_rebalance_history_uma_account_id ON uma_rebalance_history(uma_account_id);
CREATE INDEX idx_uma_rebalance_history_completed_at ON uma_rebalance_history(completed_at DESC);

-- ============================================================================
-- Add audit columns to UMA tables for ABAC temporal policies
-- ============================================================================

ALTER TABLE uma_accounts ADD COLUMN IF NOT EXISTS location VARCHAR(50); -- For location-based ABAC
ALTER TABLE uma_accounts ADD COLUMN IF NOT EXISTS approval_required BOOLEAN DEFAULT FALSE;
ALTER TABLE uma_sleeves ADD COLUMN IF NOT EXISTS last_adjustment TIMESTAMP;
ALTER TABLE uma_rebalance_requests ADD COLUMN IF NOT EXISTS tenant_datasource_id VARCHAR(100); -- For multi-tenant queries

-- ============================================================================
-- Triggers for audit trail
-- ============================================================================

CREATE OR REPLACE FUNCTION audit_uma_rebalance_requests()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        NEW.updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_audit_uma_rebalance_requests
    BEFORE UPDATE ON uma_rebalance_requests
    FOR EACH ROW
    EXECUTE FUNCTION audit_uma_rebalance_requests();

CREATE OR REPLACE FUNCTION audit_uma_accounts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        NEW.updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_audit_uma_accounts
    BEFORE UPDATE ON uma_accounts
    FOR EACH ROW
    EXECUTE FUNCTION audit_uma_accounts();
