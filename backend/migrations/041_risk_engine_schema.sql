-- Migration 041: Risk Engine Schema
-- Factor model, VaR, stress testing
-- Aligns with Whitepaper §7: Rules Engine uses Semantic Terms

CREATE SCHEMA IF NOT EXISTS edm;

-- ============================================
-- RISK FACTOR CATALOG
-- ============================================
CREATE TABLE edm.risk_factor (
    factor_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Identity
    factor_code TEXT NOT NULL UNIQUE,
    factor_name TEXT NOT NULL,
    category TEXT, -- EQUITY, FIXED_INCOME, FX, COMMODITY, CREDIT
    description TEXT,
    
    -- Factor Metadata
    factor_type TEXT CHECK (factor_type IN ('SYSTEMATIC', 'IDIOSYNCRATIC', 'MACRO')),
    unit TEXT, -- %, BP, USD, etc.
    
    -- Bi-temporal + Tenant
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_to TIMESTAMPTZ DEFAULT 'infinity',
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    core_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE (factor_code, valid_from)
);

CREATE INDEX idx_risk_factor_category ON edm.risk_factor (category);
CREATE INDEX idx_risk_factor_tenant ON edm.risk_factor (tenant_id);

-- ============================================
-- SECURITY FACTOR EXPOSURE (SCD2)
-- ============================================
CREATE TABLE edm.security_factor_exposure (
    exposure_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Keys
    security_id UUID NOT NULL,
    factor_id UUID NOT NULL REFERENCES edm.risk_factor(factor_id),
    
    -- Exposure
    exposure NUMERIC(28,10) NOT NULL, -- Beta, duration, etc.
    exposure_type TEXT, -- BETA, DURATION, SPREAD_SENSITIVITY
    
    -- As-of Date
    as_of_date DATE NOT NULL,
    
    -- Bi-temporal + Tenant
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_to TIMESTAMPTZ DEFAULT 'infinity',
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    core_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE (security_id, factor_id, as_of_date, valid_from)
);

CREATE INDEX idx_security_factor_security ON edm.security_factor_exposure (security_id, as_of_date);
CREATE INDEX idx_security_factor_factor ON edm.security_factor_exposure (factor_id);

-- ============================================
-- PORTFOLIO RISK MEASURES
-- ============================================
CREATE TABLE edm.portfolio_risk (
    portfolio_risk_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Keys
    portfolio_id UUID NOT NULL,
    valuation_date DATE NOT NULL,
    
    -- Risk Measures
    total_volatility NUMERIC(28,10), -- Annualized %
    tracking_error NUMERIC(28,10),
    var_95 NUMERIC(28,10), -- 95% VaR
    var_99 NUMERIC(28,10), -- 99% VaR
    expected_shortfall_97_5 NUMERIC(28,10),
    cvar_95 NUMERIC(28,10), -- Conditional VaR
    
    -- Factor Contributions (JSON)
    factor_contributions JSONB,
    
    -- Methodology
    var_method TEXT DEFAULT 'PARAMETRIC', -- PARAMETRIC, HISTORICAL, MONTE_CARLO
    confidence_level NUMERIC(5,4),
    holding_period_days INTEGER DEFAULT 1,
    
    -- Audit
    calculated_at TIMESTAMPTZ DEFAULT NOW(),
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    
    UNIQUE (portfolio_id, valuation_date)
);

CREATE INDEX idx_portfolio_risk_portfolio ON edm.portfolio_risk (portfolio_id, valuation_date);
CREATE INDEX idx_portfolio_risk_tenant ON edm.portfolio_risk (tenant_id);

-- ============================================
-- RISK SCENARIO (Stress Testing)
-- ============================================
CREATE TABLE edm.risk_scenario (
    scenario_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Identity
    scenario_code TEXT NOT NULL UNIQUE,
    scenario_name TEXT NOT NULL,
    description TEXT,
    scenario_type TEXT, -- HISTORICAL, HYPOTHETICAL, REGULATORY
    
    -- Shocks (JSON)
    shocks JSONB NOT NULL,
    
    -- Status
    status TEXT DEFAULT 'ACTIVE',
    
    -- Bi-temporal + Tenant
    valid_from TIMESTAMPTZ DEFAULT NOW(),
    valid_to TIMESTAMPTZ DEFAULT 'infinity',
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    core_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE (scenario_code, valid_from)
);

-- ============================================
-- SCENARIO RESULT
-- ============================================
CREATE TABLE edm.risk_scenario_result (
    scenario_result_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Keys
    scenario_id UUID NOT NULL REFERENCES edm.risk_scenario(scenario_id),
    portfolio_id UUID NOT NULL,
    valuation_date DATE NOT NULL,
    
    -- Results
    pnl NUMERIC(28,10), -- Scenario P&L
    pnl_pct NUMERIC(10,4),
    var_impact NUMERIC(28,10),
    
    -- Breakdown (JSON)
    details JSONB,
    
    -- Audit
    calculated_at TIMESTAMPTZ DEFAULT NOW(),
    tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    
    UNIQUE (scenario_id, portfolio_id, valuation_date)
);

CREATE INDEX idx_scenario_result_portfolio ON edm.risk_scenario_result (portfolio_id, valuation_date);

-- RLS for all risk tables
ALTER TABLE edm.portfolio_risk ENABLE ROW LEVEL SECURITY;
ALTER TABLE edm.risk_scenario ENABLE ROW LEVEL SECURITY;
ALTER TABLE edm.risk_scenario_result ENABLE ROW LEVEL SECURITY;

CREATE POLICY risk_tenant_isolation ON edm.portfolio_risk
    FOR ALL USING (tenant_id = current_setting('app.current_tenant', TRUE)::UUID);
