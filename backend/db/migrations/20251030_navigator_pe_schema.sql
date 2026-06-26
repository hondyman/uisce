-- Navigator: Cash Flow Forecasting for Alternative Investments (PE, VC, Infrastructure, Debt)
-- Created: October 30, 2025
-- Scope: Tenant-scoped, multi-fund portfolio management, Yale model calibration, Monte Carlo forecasting, reconciliation

-- ============================================================================
-- 1. FUND MASTER DATA
-- ============================================================================

-- Strategy types for alternative investments
CREATE TYPE strategy_type AS ENUM (
  'buyout',
  'venture_capital',
  'growth_equity',
  'private_debt',
  'infrastructure',
  'real_estate',
  'mezzanine',
  'secondary',
  'other'
);

-- Geographic focuses
CREATE TYPE geography_type AS ENUM (
  'north_america',
  'europe',
  'asia',
  'latin_america',
  'middle_east',
  'africa',
  'global',
  'emerging_markets'
);

-- Fund lifecycle status
CREATE TYPE fund_status AS ENUM (
  'fundraising',
  'investing',
  'harvesting',
  'liquidating',
  'closed',
  'extended'
);

-- Capital events types
CREATE TYPE capital_event_type AS ENUM (
  'initial_investment',
  'capital_call',
  'follow_on_call',
  'distribution',
  'recallable_distribution',
  'fee_payment',
  'expense_reimbursement',
  'dividend',
  'return_of_capital'
);

-- Reconciliation status
CREATE TYPE reconciliation_status AS ENUM (
  'pending',
  'partial_match',
  'matched',
  'exception',
  'reconciled',
  'manual_override'
);

-- Document types
CREATE TYPE document_type AS ENUM (
  'capital_call_notice',
  'distribution_notice',
  'quarterly_statement',
  'annual_report',
  'fund_agreement',
  'other'
);

-- Forecast scenario type
CREATE TYPE scenario_type AS ENUM (
  'base_case',
  'upside',
  'downside',
  'accelerated_exit',
  'delayed_exit',
  'stress_case'
);

-- ==========================================================================
-- 2. FUND COMMITMENTS (Core Table)
-- ==========================================================================

CREATE TABLE IF NOT EXISTS fund_commitments (
  commitment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  fund_id VARCHAR(255) NOT NULL,
  fund_name VARCHAR(500) NOT NULL,
  fund_manager VARCHAR(255),
  strategy_type strategy_type NOT NULL,
  geography_focus geography_type,
  vintage_year INTEGER NOT NULL,
  fund_size_usd NUMERIC(18, 2),
  commitment_amount NUMERIC(18, 2) NOT NULL,
  commitment_date DATE NOT NULL,
  investment_period_end DATE,
  fund_termination_date DATE NOT NULL,
  fund_status fund_status DEFAULT 'investing',
  management_fee_pct NUMERIC(5, 2),
  carried_interest_pct NUMERIC(5, 2),
  hurdle_rate_pct NUMERIC(5, 2),
  target_irr_pct NUMERIC(5, 2),
  target_tvpi NUMERIC(8, 2),
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  created_by VARCHAR(255),
  
  CONSTRAINT valid_commitment_amount CHECK (commitment_amount > 0),
  CONSTRAINT valid_termination_date CHECK (fund_termination_date > commitment_date)
);

CREATE INDEX idx_fund_commitments_tenant ON fund_commitments(tenant_id);
CREATE INDEX idx_fund_commitments_strategy ON fund_commitments(strategy_type);
CREATE INDEX idx_fund_commitments_vintage ON fund_commitments(vintage_year);
CREATE INDEX idx_fund_commitments_status ON fund_commitments(fund_status);

-- ==========================================================================
-- 3. CAPITAL EVENTS (Historical + Reconciliation)
-- ==========================================================================

CREATE TABLE IF NOT EXISTS capital_events (
  event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  commitment_id UUID NOT NULL REFERENCES fund_commitments(commitment_id) ON DELETE CASCADE,
  event_type capital_event_type NOT NULL,
  event_date DATE NOT NULL,
  settlement_date DATE,
  notice_date DATE,
  amount_requested NUMERIC(18, 2),
  amount_settled NUMERIC(18, 2),
  currency VARCHAR(3) DEFAULT 'USD',
  fx_rate NUMERIC(10, 6) DEFAULT 1.0,
  status reconciliation_status DEFAULT 'pending',
  source_document_id UUID,
  bank_transaction_id VARCHAR(255),
  internal_ledger_id VARCHAR(255),
  notes TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT valid_amounts CHECK (amount_requested IS NULL OR amount_requested >= 0),
  CONSTRAINT valid_settlement CHECK (settlement_date IS NULL OR settlement_date >= event_date)
);

CREATE INDEX idx_capital_events_tenant ON capital_events(tenant_id);
CREATE INDEX idx_capital_events_commitment ON capital_events(commitment_id);
CREATE INDEX idx_capital_events_type ON capital_events(event_type);
CREATE INDEX idx_capital_events_status ON capital_events(status);
CREATE INDEX idx_capital_events_date ON capital_events(event_date);

-- ==========================================================================
-- 4. FUND POSITION SNAPSHOTS (Valuation Data)
-- ==========================================================================

CREATE TABLE IF NOT EXISTS fund_position_snapshots (
  snapshot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  commitment_id UUID NOT NULL REFERENCES fund_commitments(commitment_id) ON DELETE CASCADE,
  snapshot_date DATE NOT NULL,
  as_of_timestamp TIMESTAMP NOT NULL,
  
  -- Valuation metrics
  nav NUMERIC(18, 2) NOT NULL,
  unrealized_value NUMERIC(18, 2),
  
  -- Cumulative cash flows
  paid_in_capital NUMERIC(18, 2) NOT NULL,
  distributed_capital NUMERIC(18, 2) NOT NULL,
  recallable_capital NUMERIC(18, 2) DEFAULT 0,
  unfunded_commitment NUMERIC(18, 2) NOT NULL,
  
  -- Performance metrics
  dpi NUMERIC(8, 4),
  tvpi NUMERIC(8, 4),
  rvpi NUMERIC(8, 4),
  irr_bps INTEGER,
  irr_pct NUMERIC(5, 2),
  
  -- Fund position details
  holdings_count INTEGER,
  cash_position NUMERIC(18, 2),
  fees_accrued NUMERIC(18, 2),
  
  source VARCHAR(100),
  created_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT valid_nav CHECK (nav >= 0),
  CONSTRAINT valid_picc CHECK (paid_in_capital >= 0),
  CONSTRAINT valid_dcc CHECK (distributed_capital >= 0)
);

CREATE INDEX idx_position_snapshots_tenant ON fund_position_snapshots(tenant_id);
CREATE INDEX idx_position_snapshots_commitment ON fund_position_snapshots(commitment_id);
CREATE INDEX IF NOT EXISTS idx_position_snapshots_date ON fund_position_snapshots(snapshot_date);
-- Create a unique index for the latest snapshot per commitment. PostgreSQL does not allow IF NOT EXISTS on CREATE UNIQUE INDEX prior to newer versions,
-- so guard with a conditional DO block.
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind = 'i' AND c.relname = 'idx_position_snapshots_latest'
  ) THEN
    CREATE UNIQUE INDEX idx_position_snapshots_latest ON fund_position_snapshots(commitment_id, snapshot_date);
  END IF;
END$$;

-- ==========================================================================
-- 5. YALE MODEL CALIBRATION (Per Fund)
-- ==========================================================================

CREATE TABLE IF NOT EXISTS yale_model_calibration (
  calibration_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  commitment_id UUID NOT NULL REFERENCES fund_commitments(commitment_id) ON DELETE CASCADE,
  calibration_date TIMESTAMP DEFAULT NOW(),
  
  -- Yale Model Parameters
  call_rate_pct NUMERIC(5, 2) NOT NULL,              -- RC: quarterly call rate
  growth_rate_pct NUMERIC(5, 2) NOT NULL,            -- G: quarterly NAV growth
  yield_rate_pct NUMERIC(5, 2) DEFAULT 0,            -- Y: minimum quarterly distribution
  bow_factor NUMERIC(5, 2) NOT NULL,                 -- B: distribution timing curve
  termination_years INTEGER NOT NULL,                -- L: fund lifetime
  
  -- Calibration targets
  target_irr_pct NUMERIC(5, 2),
  target_tvpi NUMERIC(8, 2),
  
  -- Validation
  variance_pct NUMERIC(5, 2),                        -- Difference between forecast and actual (for mature funds)
  confidence_score NUMERIC(3, 2),                    -- 0.0-1.0
  notes TEXT,
  
  is_active BOOLEAN DEFAULT TRUE,
  created_by VARCHAR(255),
  
  CONSTRAINT valid_rates CHECK (call_rate_pct >= 0 AND growth_rate_pct >= 0 AND yield_rate_pct >= 0),
  CONSTRAINT valid_bow_factor CHECK (bow_factor > 0 AND bow_factor < 5)
);

CREATE INDEX idx_yale_calibration_tenant ON yale_model_calibration(tenant_id);
CREATE INDEX idx_yale_calibration_commitment ON yale_model_calibration(commitment_id);

-- ==========================================================================
-- 6. CASH FLOW FORECASTS (Yale Model Output)
-- ==========================================================================

CREATE TABLE IF NOT EXISTS cash_flow_forecasts (
  forecast_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  commitment_id UUID NOT NULL REFERENCES fund_commitments(commitment_id) ON DELETE CASCADE,
  calibration_id UUID REFERENCES yale_model_calibration(calibration_id),
  
  forecast_date DATE NOT NULL,
  scenario scenario_type NOT NULL,
  probability_weight NUMERIC(3, 2),                  -- For stochastic scenarios
  
  -- Projected cash flows
  projected_calls NUMERIC(18, 2),
  projected_distributions NUMERIC(18, 2),
  projected_net_cashflow NUMERIC(18, 2),
  
  -- Projected position
  projected_picc NUMERIC(18, 2),
  projected_dcc NUMERIC(18, 2),
  projected_nav NUMERIC(18, 2),
  projected_tvpi NUMERIC(8, 4),
  projected_irr_pct NUMERIC(5, 2),
  
  -- Confidence intervals (for probabilistic forecasts)
  p5_percentile NUMERIC(18, 2),
  p25_percentile NUMERIC(18, 2),
  p75_percentile NUMERIC(18, 2),
  p95_percentile NUMERIC(18, 2),
  
  created_at TIMESTAMP DEFAULT NOW(),
  model_version VARCHAR(50)
);

CREATE INDEX idx_forecasts_tenant ON cash_flow_forecasts(tenant_id);
CREATE INDEX idx_forecasts_commitment ON cash_flow_forecasts(commitment_id);
CREATE INDEX idx_forecasts_date ON cash_flow_forecasts(forecast_date);
CREATE INDEX idx_forecasts_scenario ON cash_flow_forecasts(scenario);

-- ==========================================================================
-- 7. BENCHMARK DATA (Industry Comparables)
-- ==========================================================================

CREATE TABLE IF NOT EXISTS pe_benchmarks (
  benchmark_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  strategy_type strategy_type NOT NULL,
  vintage_year INTEGER NOT NULL,
  fund_size_bucket VARCHAR(50),                      -- e.g., "$100M-$500M"
  geography geography_type,
  
  age_in_quarters INTEGER NOT NULL,
  
  -- Aggregated metrics
  avg_picc_pct_of_commitment NUMERIC(5, 2),         -- Average PICC as % of commitment at this age
  avg_dpi NUMERIC(8, 4),
  avg_tvpi NUMERIC(8, 4),
  avg_irr_pct NUMERIC(5, 2),
  
  -- Call pattern (cumulative % of commitment called by age)
  call_rate_pattern NUMERIC(5, 2),
  
  -- Distribution pattern
  dist_rate_pattern NUMERIC(5, 2),
  
  data_source VARCHAR(100),                          -- e.g., "Preqin", "Burgiss", "Internal"
  sample_size INTEGER,
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT valid_percentiles CHECK (avg_picc_pct_of_commitment >= 0 AND avg_picc_pct_of_commitment <= 100)
);

CREATE UNIQUE INDEX idx_benchmarks_key ON pe_benchmarks(strategy_type, vintage_year, fund_size_bucket, geography, age_in_quarters);

-- ==========================================================================
-- 8. RECONCILIATION RECORDS
-- ==========================================================================

CREATE TABLE IF NOT EXISTS reconciliation_records (
  reconciliation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  commitment_id UUID NOT NULL REFERENCES fund_commitments(commitment_id) ON DELETE CASCADE,
  reconciliation_period DATE NOT NULL,               -- Month/quarter being reconciled
  
  -- Source matching
  fund_statement_events INTEGER,
  bank_transaction_count INTEGER,
  internal_ledger_records INTEGER,
  
  -- Matching results
  fully_matched INTEGER,
  partial_matches INTEGER,
  exceptions_count INTEGER,
  unmatched_count INTEGER,
  
  -- Amounts
  fund_statement_total NUMERIC(18, 2),
  bank_total NUMERIC(18, 2),
  internal_total NUMERIC(18, 2),
  
  -- Variances
  variance_amount NUMERIC(18, 2),
  variance_pct NUMERIC(5, 2),
  fx_variance NUMERIC(18, 2),
  timing_variance NUMERIC(18, 2),
  
  status reconciliation_status DEFAULT 'pending',
  notes TEXT,
  reconciled_by VARCHAR(255),
  reconciled_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_reconciliation_tenant ON reconciliation_records(tenant_id);
CREATE INDEX idx_reconciliation_commitment ON reconciliation_records(commitment_id);
CREATE INDEX idx_reconciliation_period ON reconciliation_records(reconciliation_period);

-- ==========================================================================
-- 9. DOCUMENT REPOSITORY
-- ==========================================================================

CREATE TABLE IF NOT EXISTS document_repository (
  document_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  commitment_id UUID NOT NULL REFERENCES fund_commitments(commitment_id) ON DELETE CASCADE,
  
  document_type document_type NOT NULL,
  document_name VARCHAR(500) NOT NULL,
  document_date DATE,
  file_path VARCHAR(1000),
  file_size INTEGER,
  
  -- AI Extraction
  extraction_status VARCHAR(50),                      -- pending, processing, success, failed
  extraction_confidence NUMERIC(3, 2),                -- 0.0-1.0
  extracted_data JSONB,                               -- Structured data extracted by AI
  extraction_error TEXT,
  human_verified BOOLEAN DEFAULT FALSE,
  verified_by VARCHAR(255),
  verified_at TIMESTAMP,
  
  notes TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_documents_tenant ON document_repository(tenant_id);
CREATE INDEX idx_documents_commitment ON document_repository(commitment_id);
CREATE INDEX idx_documents_type ON document_repository(document_type);

-- ==========================================================================
-- 10. AUDIT & COMPLIANCE
-- ==========================================================================

CREATE TABLE IF NOT EXISTS navigator_audit_trail (
  audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  commitment_id UUID,
  action VARCHAR(100) NOT NULL,                      -- forecast_generated, calibration_updated, reconciliation_completed, etc.
  actor_email VARCHAR(255),
  actor_role VARCHAR(100),
  changes JSONB,                                      -- What changed
  reason TEXT,
  ip_address INET,
  user_agent TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_audit_tenant ON navigator_audit_trail(tenant_id);
CREATE INDEX idx_audit_commitment ON navigator_audit_trail(commitment_id);
CREATE INDEX idx_audit_timestamp ON navigator_audit_trail(created_at);

-- ==========================================================================
-- 11. MATERIALIZED VIEWS FOR DASHBOARD & REPORTING
-- ==========================================================================

-- Portfolio-level exposure summary
CREATE MATERIALIZED VIEW v_portfolio_exposure_summary AS
SELECT 
  fc.tenant_id,
  fc.commitment_id,
  fc.fund_name,
  fc.strategy_type,
  fc.vintage_year,
  fc.commitment_amount,
  
  -- Current position (from latest snapshot)
  COALESCE(fps.paid_in_capital, 0) as paid_in_capital,
  COALESCE(fps.distributed_capital, 0) as distributed_capital,
  COALESCE(fps.nav, 0) as current_nav,
  COALESCE(fps.unfunded_commitment, 0) as unfunded_commitment,
  COALESCE(fps.tvpi, 0) as current_tvpi,
  COALESCE(fps.dpi, 0) as current_dpi,
  COALESCE(fps.irr_pct, 0) as current_irr_pct,
  
  -- Forecast (12-month ahead)
  COALESCE(
    SUM(CASE 
      WHEN ccf.forecast_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '12 months'
      AND ccf.scenario = 'base_case'
      THEN ccf.projected_calls 
      ELSE 0 
    END), 0
  ) as projected_calls_12m,
  
  COALESCE(
    SUM(CASE 
      WHEN ccf.forecast_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '12 months'
      AND ccf.scenario = 'base_case'
      THEN ccf.projected_distributions 
      ELSE 0 
    END), 0
  ) as projected_distributions_12m,
  
  fc.fund_status,
  fc.created_at
  
FROM fund_commitments fc
LEFT JOIN LATERAL (
  SELECT * FROM fund_position_snapshots 
  WHERE commitment_id = fc.commitment_id 
  ORDER BY snapshot_date DESC 
  LIMIT 1
) fps ON TRUE
LEFT JOIN cash_flow_forecasts ccf ON fc.commitment_id = ccf.commitment_id
GROUP BY 
  fc.tenant_id, fc.commitment_id, fc.fund_name, fc.strategy_type, 
  fc.vintage_year, fc.commitment_amount, fc.fund_status, fc.created_at,
  fps.paid_in_capital, fps.distributed_capital, fps.nav, 
  fps.unfunded_commitment, fps.tvpi, fps.dpi, fps.irr_pct;

CREATE UNIQUE INDEX idx_exposure_summary_key ON v_portfolio_exposure_summary(tenant_id, commitment_id);

-- Liquidity needs projection (rolling 12 months by month)
CREATE MATERIALIZED VIEW v_liquidity_needs_projection AS
SELECT 
  tenant_id,
  DATE_TRUNC('month', forecast_date)::DATE as month,
  scenario,
  COUNT(*) as fund_count,
  SUM(projected_calls) as total_calls,
  SUM(projected_distributions) as total_distributions,
  SUM(projected_calls) - SUM(projected_distributions) as net_needs,
  SUM(p95_percentile) as max_probable_calls_95th
FROM cash_flow_forecasts
WHERE forecast_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '24 months'
GROUP BY tenant_id, DATE_TRUNC('month', forecast_date), scenario;

CREATE INDEX idx_liquidity_needs_tenant ON v_liquidity_needs_projection(tenant_id);
CREATE INDEX idx_liquidity_needs_month ON v_liquidity_needs_projection(month);

-- Reconciliation status dashboard
CREATE MATERIALIZED VIEW v_reconciliation_status AS
SELECT 
  tenant_id,
  reconciliation_period,
  COUNT(*) as total_funds,
  SUM(CASE WHEN status = 'reconciled' THEN 1 ELSE 0 END) as reconciled_count,
  SUM(CASE WHEN status = 'partial_match' THEN 1 ELSE 0 END) as partial_matches,
  SUM(CASE WHEN status = 'exception' THEN 1 ELSE 0 END) as exceptions,
  ROUND(100.0 * SUM(CASE WHEN status = 'reconciled' THEN 1 ELSE 0 END) / COUNT(*), 2) as reconciliation_rate_pct,
  SUM(variance_amount) as total_variance,
  MAX(reconciled_at) as last_reconciled_at
FROM reconciliation_records
GROUP BY tenant_id, reconciliation_period;

CREATE INDEX idx_recon_status_tenant ON v_reconciliation_status(tenant_id);

-- ==========================================================================
-- 12. TRIGGERS
-- ==========================================================================

CREATE OR REPLACE FUNCTION update_commitment_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_commitment_ts
BEFORE UPDATE ON fund_commitments
FOR EACH ROW
EXECUTE FUNCTION update_commitment_timestamp();

-- ==========================================================================
-- 13. SAMPLE DATA (Optional - Remove for production)
-- ==========================================================================

-- Sample PE fund commitment
INSERT INTO fund_commitments (
  tenant_id, fund_id, fund_name, fund_manager, strategy_type, geography_focus,
  vintage_year, fund_size_usd, commitment_amount, commitment_date, investment_period_end,
  fund_termination_date, fund_status, management_fee_pct, carried_interest_pct,
  hurdle_rate_pct, target_irr_pct, target_tvpi, created_by
) VALUES (
  '00000000-0000-0000-0000-000000000000'::uuid,
  'FUND001',
  'Acme Buyout Fund IV',
  'Acme Partners',
  'buyout',
  'north_america',
  2020,
  500000000,
  25000000,
  '2020-06-15'::date,
  '2025-06-15'::date,
  '2030-06-15'::date,
  'harvesting',
  2.00,
  20.00,
  8.00,
  15.00,
  2.50,
  'system'
) ON CONFLICT DO NOTHING;

-- ==========================================================================
-- RLS Policies (Templates - Enable as needed)
-- ==========================================================================

/*
-- Tenant isolation for fund_commitments
CREATE POLICY tenant_isolation_commitments ON fund_commitments
  FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
ALTER TABLE fund_commitments ENABLE ROW LEVEL SECURITY;

-- Apply similar policies to other tables:
-- capital_events, fund_position_snapshots, yale_model_calibration,
-- cash_flow_forecasts, document_repository, reconciliation_records, navigator_audit_trail
*/

-- ==========================================================================
-- Grants (Customize per your RBAC model)
-- ==========================================================================

GRANT SELECT, INSERT, UPDATE ON fund_commitments TO postgres;
GRANT SELECT, INSERT, UPDATE ON capital_events TO postgres;
GRANT SELECT, INSERT, UPDATE ON fund_position_snapshots TO postgres;
GRANT SELECT, INSERT ON yale_model_calibration TO postgres;
GRANT SELECT, INSERT ON cash_flow_forecasts TO postgres;
GRANT SELECT ON pe_benchmarks TO postgres;
GRANT SELECT, INSERT, UPDATE ON reconciliation_records TO postgres;
GRANT SELECT, INSERT, UPDATE ON document_repository TO postgres;
GRANT INSERT ON navigator_audit_trail TO postgres;
GRANT SELECT ON v_portfolio_exposure_summary TO postgres;
GRANT SELECT ON v_liquidity_needs_projection TO postgres;
GRANT SELECT ON v_reconciliation_status TO postgres;
