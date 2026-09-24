-- 20260915_predictive_compliance_ai.up.sql
-- Predictive Compliance Forecasting, Sizing Engine & Remediation Ledger

CREATE SCHEMA IF NOT EXISTS catalog_compliance;

-- 1. Calibrated Machine Learning Feature Weights
CREATE TABLE IF NOT EXISTS catalog_compliance.feature_weights (
    weight_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rule_category VARCHAR(50) NOT NULL,
    intercept NUMERIC(10, 6) NOT NULL DEFAULT -4.500000,
    weight_utilization NUMERIC(10, 6) NOT NULL DEFAULT 5.200000,
    weight_volatility NUMERIC(10, 6) NOT NULL DEFAULT 2.800000,
    weight_momentum NUMERIC(10, 6) NOT NULL DEFAULT 1.950000,
    weight_reopen_count NUMERIC(10, 6) NOT NULL DEFAULT 0.650000,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_rule_category UNIQUE (tenant_id, rule_category)
);

-- 2. Passive Drift Forecast Ledger
CREATE TABLE IF NOT EXISTS catalog_compliance.passive_drift_forecasts (
    forecast_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_node_id UUID NOT NULL,
    rule_id UUID NOT NULL,
    security_node_id UUID,
    
    current_utilization_pct NUMERIC(8, 4) NOT NULL,
    forecasted_breach_probability NUMERIC(6, 4) NOT NULL,
    estimated_hours_to_breach NUMERIC(8, 2) NOT NULL,
    drift_velocity_usd_per_hour NUMERIC(18, 4) NOT NULL,
    
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE_WARNING',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Automated Trade Resizing & Proxy Recommendations
CREATE TABLE IF NOT EXISTS catalog_compliance.order_resizing_recommendations (
    recommendation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    ticket_id VARCHAR(100) NOT NULL,
    portfolio_node_id UUID NOT NULL,
    security_node_id UUID NOT NULL,
    
    proposed_shares NUMERIC(28, 6) NOT NULL,
    max_compliant_shares NUMERIC(28, 6) NOT NULL,
    shares_reduction_delta NUMERIC(28, 6) GENERATED ALWAYS AS (proposed_shares - max_compliant_shares) STORED,
    
    proxy_security_node_id UUID,
    proxy_ticker VARCHAR(30),
    proxy_correlation NUMERIC(6, 4),
    
    explain_why_diagnostic TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compliance_forecasts_lookup 
ON catalog_compliance.passive_drift_forecasts (tenant_id, portfolio_node_id, forecasted_breach_probability DESC);
