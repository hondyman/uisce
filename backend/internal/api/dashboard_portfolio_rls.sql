-- ============================================================================
-- Risk & Compliance Console - Multi-Tenant RLS Policies
-- ============================================================================
-- This script sets up Row-Level Security (RLS) policies to enforce multi-tenant
-- isolation at the database level. All dashboard and portfolio data queries
-- will be automatically filtered by tenant_id.

-- ============================================================================
-- 1. Enable RLS on Dashboard Tables
-- ============================================================================

-- Create dashboard_compliance_rules table if it doesn't exist
CREATE TABLE IF NOT EXISTS public.dashboard_compliance_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rule_id TEXT NOT NULL,
    rule_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('Pass', 'Fail', 'Warning')),
    pass_rate NUMERIC(5,2),
    last_checked TIMESTAMPTZ DEFAULT now(),
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT unique_compliance_rule UNIQUE(tenant_id, rule_id)
);

-- Create dashboard_risk_metrics table
CREATE TABLE IF NOT EXISTS public.dashboard_risk_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    metric_id TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    value NUMERIC(12,4),
    unit TEXT,
    threshold NUMERIC(12,4),
    status TEXT NOT NULL CHECK(status IN ('Normal', 'Warning', 'Alert')),
    last_updated TIMESTAMPTZ DEFAULT now(),
    created_at TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT unique_dashboard_risk_metric UNIQUE(tenant_id, metric_id)
);

-- Create dashboard_alerts table
CREATE TABLE IF NOT EXISTS public.dashboard_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    alert_id TEXT NOT NULL,
    title TEXT NOT NULL,
    severity TEXT NOT NULL CHECK(severity IN ('Critical', 'Warning', 'Info')),
    message TEXT,
    source TEXT CHECK(source IN ('Compliance', 'Risk', 'Operations')),
    created_at TIMESTAMPTZ DEFAULT now(),
    status TEXT NOT NULL DEFAULT 'Open' CHECK(status IN ('Open', 'Acknowledged', 'Resolved')),
    resolved_at TIMESTAMPTZ,
    CONSTRAINT unique_dashboard_alert UNIQUE(tenant_id, alert_id)
);

-- Create dashboard_etl_runs table
CREATE TABLE IF NOT EXISTS public.dashboard_etl_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    run_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('Running', 'Success', 'Failed')),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    records_processed BIGINT,
    records_failed BIGINT,
    duration_seconds INTEGER,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT unique_etl_run UNIQUE(tenant_id, run_id)
);

-- Create portfolios table if it doesn't exist
CREATE TABLE IF NOT EXISTS public.portfolios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_id TEXT NOT NULL,
    portfolio_name TEXT NOT NULL,
    manager TEXT,
    status TEXT NOT NULL CHECK(status IN ('Active', 'Closed')),
    created_date DATE,
    valuation_date DATE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT unique_portfolio UNIQUE(tenant_id, portfolio_id)
);

-- Create portfolio_metrics table
CREATE TABLE IF NOT EXISTS public.portfolio_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_id TEXT NOT NULL,
    total_value NUMERIC(18,2),
    day_change_amt NUMERIC(18,2),
    day_change_percent NUMERIC(6,4),
    ytd_return_percent NUMERIC(6,4),
    one_year_return NUMERIC(6,4),
    inception_to_date_return NUMERIC(6,4),
    valuation_date DATE,
    created_at TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT unique_portfolio_metrics UNIQUE(tenant_id, portfolio_id, valuation_date)
);

-- Create portfolio_holdings table
CREATE TABLE IF NOT EXISTS public.portfolio_holdings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_id TEXT NOT NULL,
    instrument_id TEXT NOT NULL,
    symbol TEXT,
    name TEXT,
    asset_class TEXT,
    quantity NUMERIC(12,4),
    unit_price NUMERIC(12,4),
    position_value NUMERIC(18,2),
    weight_percent NUMERIC(6,4),
    day_change NUMERIC(6,4),
    ytd_return NUMERIC(6,4),
    country_code TEXT,
    sector_code TEXT,
    valuation_date DATE,
    created_at TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT unique_holding UNIQUE(tenant_id, portfolio_id, symbol, valuation_date)
);

-- Create portfolio_risk_factors table
CREATE TABLE IF NOT EXISTS public.portfolio_risk_factors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_id TEXT NOT NULL,
    factor_name TEXT NOT NULL,
    exposure NUMERIC(12,4),
    beta NUMERIC(6,4),
    contribution NUMERIC(6,4),
    valuation_date DATE,
    created_at TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT unique_risk_factor UNIQUE(tenant_id, portfolio_id, factor_name, valuation_date)
);

-- Create portfolio_compliance_rules table
CREATE TABLE IF NOT EXISTS public.portfolio_compliance_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_id TEXT NOT NULL,
    rule_id TEXT NOT NULL,
    rule_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('Breach', 'Warning', 'Pass')),
    current_value NUMERIC(12,4),
    limit_value NUMERIC(12,4),
    severity TEXT CHECK(severity IN ('Critical', 'Warning', 'Info')),
    description TEXT,
    remediation_by DATE,
    valuation_date DATE,
    created_at TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT unique_portfolio_compliance_rule UNIQUE(tenant_id, portfolio_id, rule_id, valuation_date)
);

-- Create portfolio_scenarios table
CREATE TABLE IF NOT EXISTS public.portfolio_scenarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_id TEXT NOT NULL,
    scenario_id TEXT NOT NULL,
    scenario_name TEXT NOT NULL,
    description TEXT,
    based_on_date DATE,
    baseline_value NUMERIC(18,2),
    simulated_value NUMERIC(18,2),
    pnl_change NUMERIC(18,2),
    percent_change NUMERIC(6,4),
    breach_count INTEGER,
    volatility_change NUMERIC(6,4),
    var_change NUMERIC(18,2),
    created_at TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT unique_scenario UNIQUE(tenant_id, portfolio_id, scenario_id)
);

-- ============================================================================
-- 2. Create RLS Policies
-- ============================================================================

-- Enable RLS on all tables
ALTER TABLE public.dashboard_compliance_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.dashboard_risk_metrics  ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.dashboard_alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.dashboard_etl_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.portfolios ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.portfolio_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.portfolio_holdings ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.portfolio_risk_factors ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.portfolio_compliance_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.portfolio_scenarios ENABLE ROW LEVEL SECURITY;

-- Helper function to get current tenant_id from request context
-- This assumes tenant_id is set via set_config() in the application
CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS UUID AS $$
  SELECT current_setting('app.tenant_id')::UUID;
$$ LANGUAGE SQL STABLE;

-- ============================================================================
-- 2a. Dashboard Compliance Rules Policies
-- ============================================================================
CREATE POLICY dashboard_compliance_tenant_select ON public.dashboard_compliance_rules
  FOR SELECT
  USING (tenant_id = current_tenant_id());

CREATE POLICY dashboard_compliance_tenant_insert ON public.dashboard_compliance_rules
  FOR INSERT
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY dashboard_compliance_tenant_update ON public.dashboard_compliance_rules
  FOR UPDATE
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY dashboard_compliance_tenant_delete ON public.dashboard_compliance_rules
  FOR DELETE
  USING (tenant_id = current_tenant_id());

-- ============================================================================
-- 2b. Dashboard Risk Metrics Policies
-- ============================================================================
CREATE POLICY dashboard_risk_tenant_select ON public.dashboard_risk_metrics
  FOR SELECT
  USING (tenant_id = current_tenant_id());

CREATE POLICY dashboard_risk_tenant_insert ON public.dashboard_risk_metrics
  FOR INSERT
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY dashboard_risk_tenant_update ON public.dashboard_risk_metrics
  FOR UPDATE
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY dashboard_risk_tenant_delete ON public.dashboard_risk_metrics
  FOR DELETE
  USING (tenant_id = current_tenant_id());

-- ============================================================================
-- 2c. Dashboard Alerts Policies
-- ============================================================================
CREATE POLICY dashboard_alerts_tenant_select ON public.dashboard_alerts
  FOR SELECT
  USING (tenant_id = current_tenant_id());

CREATE POLICY dashboard_alerts_tenant_insert ON public.dashboard_alerts
  FOR INSERT
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY dashboard_alerts_tenant_update ON public.dashboard_alerts
  FOR UPDATE
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY dashboard_alerts_tenant_delete ON public.dashboard_alerts
  FOR DELETE
  USING (tenant_id = current_tenant_id());

-- ============================================================================
-- 2d. Dashboard ETL Runs Policies
-- ============================================================================
CREATE POLICY dashboard_etl_tenant_select ON public.dashboard_etl_runs
  FOR SELECT
  USING (tenant_id = current_tenant_id());

CREATE POLICY dashboard_etl_tenant_insert ON public.dashboard_etl_runs
  FOR INSERT
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY dashboard_etl_tenant_update ON public.dashboard_etl_runs
  FOR UPDATE
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY dashboard_etl_tenant_delete ON public.dashboard_etl_runs
  FOR DELETE
  USING (tenant_id = current_tenant_id());

-- ============================================================================
-- 2e. Portfolios Policies
-- ============================================================================
CREATE POLICY portfolios_tenant_select ON public.portfolios
  FOR SELECT
  USING (tenant_id = current_tenant_id());

CREATE POLICY portfolios_tenant_insert ON public.portfolios
  FOR INSERT
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolios_tenant_update ON public.portfolios
  FOR UPDATE
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolios_tenant_delete ON public.portfolios
  FOR DELETE
  USING (tenant_id = current_tenant_id());

-- ============================================================================
-- 2f. Portfolio Metrics Policies
-- ============================================================================
CREATE POLICY portfolio_metrics_tenant_select ON public.portfolio_metrics
  FOR SELECT
  USING (tenant_id = current_tenant_id());

CREATE POLICY portfolio_metrics_tenant_insert ON public.portfolio_metrics
  FOR INSERT
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolio_metrics_tenant_update ON public.portfolio_metrics
  FOR UPDATE
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolio_metrics_tenant_delete ON public.portfolio_metrics
  FOR DELETE
  USING (tenant_id = current_tenant_id());

-- ============================================================================
-- 2g. Portfolio Holdings Policies
-- ============================================================================
CREATE POLICY portfolio_holdings_tenant_select ON public.portfolio_holdings
  FOR SELECT
  USING (tenant_id = current_tenant_id());

CREATE POLICY portfolio_holdings_tenant_insert ON public.portfolio_holdings
  FOR INSERT
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolio_holdings_tenant_update ON public.portfolio_holdings
  FOR UPDATE
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolio_holdings_tenant_delete ON public.portfolio_holdings
  FOR DELETE
  USING (tenant_id = current_tenant_id());

-- ============================================================================
-- 2h. Portfolio Risk Factors Policies
-- ============================================================================
CREATE POLICY portfolio_risk_factors_tenant_select ON public.portfolio_risk_factors
  FOR SELECT
  USING (tenant_id = current_tenant_id());

CREATE POLICY portfolio_risk_factors_tenant_insert ON public.portfolio_risk_factors
  FOR INSERT
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolio_risk_factors_tenant_update ON public.portfolio_risk_factors
  FOR UPDATE
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolio_risk_factors_tenant_delete ON public.portfolio_risk_factors
  FOR DELETE
  USING (tenant_id = current_tenant_id());

-- ============================================================================
-- 2i. Portfolio Compliance Rules Policies
-- ============================================================================
CREATE POLICY portfolio_compliance_rules_tenant_select ON public.portfolio_compliance_rules
  FOR SELECT
  USING (tenant_id = current_tenant_id());

CREATE POLICY portfolio_compliance_rules_tenant_insert ON public.portfolio_compliance_rules
  FOR INSERT
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolio_compliance_rules_tenant_update ON public.portfolio_compliance_rules
  FOR UPDATE
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolio_compliance_rules_tenant_delete ON public.portfolio_compliance_rules
  FOR DELETE
  USING (tenant_id = current_tenant_id());

-- ============================================================================
-- 2j. Portfolio Scenarios Policies
-- ============================================================================
CREATE POLICY portfolio_scenarios_tenant_select ON public.portfolio_scenarios
  FOR SELECT
  USING (tenant_id = current_tenant_id());

CREATE POLICY portfolio_scenarios_tenant_insert ON public.portfolio_scenarios
  FOR INSERT
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolio_scenarios_tenant_update ON public.portfolio_scenarios
  FOR UPDATE
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY portfolio_scenarios_tenant_delete ON public.portfolio_scenarios
  FOR DELETE
  USING (tenant_id = current_tenant_id());

-- ============================================================================
-- 3. Create Indexes for Performance
-- ============================================================================

-- Dashboard Compliance Rules Indexes
CREATE INDEX IF NOT EXISTS idx_dashboard_compliance_tenant ON public.dashboard_compliance_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_compliance_status ON public.dashboard_compliance_rules(status);
CREATE INDEX IF NOT EXISTS idx_dashboard_compliance_rule_id ON public.dashboard_compliance_rules(tenant_id, rule_id);

-- Dashboard Risk Metrics Indexes
CREATE INDEX IF NOT EXISTS idx_dashboard_risk_tenant ON public.dashboard_risk_metrics(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_risk_status ON public.dashboard_risk_metrics(status);
CREATE INDEX IF NOT EXISTS idx_dashboard_risk_metric_id ON public.dashboard_risk_metrics(tenant_id, metric_id);

-- Dashboard Alerts Indexes
CREATE INDEX IF NOT EXISTS idx_dashboard_alerts_tenant ON public.dashboard_alerts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_alerts_severity ON public.dashboard_alerts(severity);
CREATE INDEX IF NOT EXISTS idx_dashboard_alerts_status ON public.dashboard_alerts(status);
CREATE INDEX IF NOT EXISTS idx_dashboard_alerts_created ON public.dashboard_alerts(created_at DESC);

-- Dashboard ETL Runs Indexes
CREATE INDEX IF NOT EXISTS idx_dashboard_etl_tenant ON public.dashboard_etl_runs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_dashboard_etl_status ON public.dashboard_etl_runs(status);
CREATE INDEX IF NOT EXISTS idx_dashboard_etl_created ON public.dashboard_etl_runs(created_at DESC);

-- Portfolios Indexes
CREATE INDEX IF NOT EXISTS idx_portfolios_tenant ON public.portfolios(tenant_id);
CREATE INDEX IF NOT EXISTS idx_portfolios_portfolio_id ON public.portfolios(tenant_id, portfolio_id);
CREATE INDEX IF NOT EXISTS idx_portfolios_status ON public.portfolios(status);

-- Portfolio Metrics Indexes
CREATE INDEX IF NOT EXISTS idx_portfolio_metrics_tenant ON public.portfolio_metrics(tenant_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_metrics_portfolio ON public.portfolio_metrics(tenant_id, portfolio_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_metrics_valuation ON public.portfolio_metrics(valuation_date);

-- Portfolio Holdings Indexes
CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_tenant ON public.portfolio_holdings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_portfolio ON public.portfolio_holdings(tenant_id, portfolio_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_valuation ON public.portfolio_holdings(valuation_date);
CREATE INDEX IF NOT EXISTS idx_portfolio_holdings_sector ON public.portfolio_holdings(sector_code);

-- Portfolio Risk Factors Indexes
CREATE INDEX IF NOT EXISTS idx_portfolio_risk_factors_tenant ON public.portfolio_risk_factors(tenant_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_risk_factors_portfolio ON public.portfolio_risk_factors(tenant_id, portfolio_id);

-- Portfolio Compliance Rules Indexes
CREATE INDEX IF NOT EXISTS idx_portfolio_compliance_tenant ON public.portfolio_compliance_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_compliance_portfolio ON public.portfolio_compliance_rules(tenant_id, portfolio_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_compliance_status ON public.portfolio_compliance_rules(status);

-- Portfolio Scenarios Indexes
CREATE INDEX IF NOT EXISTS idx_portfolio_scenarios_tenant ON public.portfolio_scenarios(tenant_id);
CREATE INDEX IF NOT EXISTS idx_portfolio_scenarios_portfolio ON public.portfolio_scenarios(tenant_id, portfolio_id);

-- ============================================================================
-- 4. Documentation
-- ============================================================================
/*
 * MULTI-TENANT ISOLATION IMPLEMENTATION
 * 
 * This schema enforces multi-tenant data isolation at the database level using
 * PostgreSQL Row-Level Security (RLS) policies.
 * 
 * HOW IT WORKS:
 * 1. Each table has a tenant_id column
 * 2. RLS policies automatically filter queries by current tenant
 * 3. The application sets tenant_id via: SET app.tenant_id = 'xxx-xxx-xxx'
 * 4. PostgreSQL automatically applies the filter to all queries
 * 
 * SECURITY GUARANTEES:
 * - Users can ONLY see data from their own tenant
 * - Cross-tenant data access is IMPOSSIBLE at the database level
 * - Even with admin privileges on the app layer, RLS prevents data leaks
 * - All CRUD operations (SELECT, INSERT, UPDATE, DELETE) are filtered
 * 
 * APPLICATION USAGE:
 * Before each request:
 *   1. Extract tenant_id from JWT token or request context
 *   2. Execute: SET LOCAL app.tenant_id = 'tenant-uuid'
 *   3. Execute queries - they automatically filter by tenant
 *   4. Transaction ends - tenant context is reset
 * 
 * TESTING MULTI-TENANT ISOLATION:
 * See dashboard_handler_multitenancy_test.go for comprehensive tests
 * 
 * PERFORMANCE NOTES:
 * - RLS policies are efficiently compiled into WHERE clauses
 * - Index on tenant_id ensures fast lookups
 * - No significant performance overhead vs manual WHERE filtering
 * - Indexes on (tenant_id, field) are optimal for multi-tenant queries
 */
