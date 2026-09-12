-- 20260924_corporate_actions_engine.up.sql
-- Corporate Action & Event Propagation Engine (CA-PE)

CREATE SCHEMA IF NOT EXISTS catalog_ca;

-- 1. Declarative Corporate Action Rules (Rule 1: Config-Before-Code)
CREATE TABLE IF NOT EXISTS catalog_ca.action_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    target_asset_class VARCHAR(50) NOT NULL DEFAULT 'ALL',
    adjustment_formula_json JSONB NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_action_rule UNIQUE (tenant_id, action_type, target_asset_class)
);

-- 2. Corporate Action Events Ledger
CREATE TABLE IF NOT EXISTS catalog_ca.corporate_action_events (
    action_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    security_node_id UUID NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    effective_date DATE NOT NULL,
    announcement_source VARCHAR(50) NOT NULL,
    terms_payload JSONB NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_PROPAGATION',
    merkle_audit_seal VARCHAR(64) NOT NULL,
    propagated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Position Adjustment Audit Log (OLTP Accounting Boundary)
CREATE TABLE IF NOT EXISTS catalog_ca.position_adjustment_audit (
    adjustment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    action_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    security_node_id UUID NOT NULL,
    pre_adjustment_shares NUMERIC(28, 6) NOT NULL,
    post_adjustment_shares NUMERIC(28, 6) NOT NULL,
    cash_adjustment_usd NUMERIC(18, 4) DEFAULT 0.0000,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ca_events_lookup 
ON catalog_ca.corporate_action_events (tenant_id, effective_date, status);
