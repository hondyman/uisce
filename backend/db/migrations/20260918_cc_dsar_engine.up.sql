-- 20260918_cc_dsar_engine.up.sql
-- Continuous Compliance Drift Simulation & Automated Rebalancing (CC-DSAR)

CREATE SCHEMA IF NOT EXISTS catalog_cc_dsar;

-- 1. Intraday Portfolio Drift Telemetry
CREATE TABLE IF NOT EXISTS catalog_cc_dsar.intraday_drift_telemetry (
    telemetry_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    rule_id UUID NOT NULL,
    
    baseline_utilization_pct NUMERIC(8, 4) NOT NULL,
    projected_utilization_1h NUMERIC(8, 4) NOT NULL,
    projected_utilization_4h NUMERIC(8, 4) NOT NULL,
    drift_status VARCHAR(30) NOT NULL DEFAULT 'STABLE',
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Staged Autonomous Rebalancing Baskets
CREATE TABLE IF NOT EXISTS catalog_cc_dsar.rebalancing_baskets (
    basket_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    rule_id UUID NOT NULL,
    
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_APPROVAL',
    total_rebalance_turnover_pct NUMERIC(6, 4) NOT NULL,
    estimated_tax_impact_usd NUMERIC(28, 4) NOT NULL DEFAULT 0.0000,
    basket_payload_json JSONB NOT NULL,
    merkle_audit_seal VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cc_dsar_status 
ON catalog_cc_dsar.intraday_drift_telemetry (tenant_id, drift_status, calculated_at DESC);
