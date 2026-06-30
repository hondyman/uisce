-- Migration: Create Alternative Investments Schema
-- Description: Comprehensive schema for alternative investments including PE, VC, hedge funds, real estate, etc.
-- Author: Semlayer Platform
-- Date: 2025-11-27

-- ============================================================================
-- CORE ALTERNATIVE INVESTMENTS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS alternative_investments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    client_id UUID NOT NULL,
    account_id UUID,
    
    -- Investment identification
    fund_name VARCHAR(255) NOT NULL,
    fund_manager VARCHAR(255) NOT NULL,
    asset_class VARCHAR(50) NOT NULL CHECK (asset_class IN (
        'PRIVATE_EQUITY',
        'VENTURE_CAPITAL', 
        'HEDGE_FUND',
        'REAL_ESTATE',
        'PRIVATE_CREDIT',
        'INFRASTRUCTURE',
        'COLLECTIBLES',
        'COMMODITIES',
        'PRIVATE_DEBT'
    )),
    sub_strategy VARCHAR(100), -- e.g., 'Growth Equity', 'Buyout', 'Distressed Debt'
    
    -- Investment terms
    vintage_year INTEGER,
    commitment_amount DECIMAL(15,2) NOT NULL CHECK (commitment_amount >= 0),
    commitment_currency VARCHAR(3) DEFAULT 'USD',
    capital_called DECIMAL(15,2) DEFAULT 0 CHECK (capital_called >= 0),
    capital_distributed DECIMAL(15,2) DEFAULT 0 CHECK (capital_distributed >= 0),
    unfunded_commitment DECIMAL(15,2) GENERATED ALWAYS AS (commitment_amount - capital_called) STORED,
    
    -- Valuation
    current_nav DECIMAL(15,2) DEFAULT 0 CHECK (current_nav >= 0),
    last_valuation_date DATE,
    valuation_method VARCHAR(50) CHECK (valuation_method IN (
        'AUDITED',
        'GP_ESTIMATE',
        'THIRD_PARTY',
        'FAIR_VALUE'
    )),
    
    -- Fee structure (integrates with fee_billing)
    management_fee_pct DECIMAL(5,4) CHECK (management_fee_pct >= 0 AND management_fee_pct <= 1),
    performance_fee_pct DECIMAL(5,4) CHECK (performance_fee_pct >= 0 AND performance_fee_pct <= 1),
    hurdle_rate_pct DECIMAL(5,4) CHECK (hurdle_rate_pct >= 0),
    has_high_water_mark BOOLEAN DEFAULT FALSE,
    has_catch_up BOOLEAN DEFAULT TRUE,
    
    -- Tax and compliance
    tax_entity_type VARCHAR(50) CHECK (tax_entity_type IN (
        'PASS_THROUGH',
        'CORPORATE',
        'OFFSHORE',
        'PARTNERSHIP',
        'LLC'
    )),
    k1_received BOOLEAN DEFAULT FALSE,
    k1_received_date DATE,
    
    -- Metadata
    inception_date DATE NOT NULL,
    expected_term_years INTEGER,
    maturity_date DATE,
    notes TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT check_capital_called_lte_commitment CHECK (capital_called <= commitment_amount)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_alt_investments_tenant_client ON alternative_investments(tenant_id, client_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alt_investments_asset_class ON alternative_investments(asset_class) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alt_investments_vintage_year ON alternative_investments(vintage_year) WHERE deleted_at IS NULL;

-- ============================================================================
-- CAPITAL CALLS (Money requested by GP)
-- ============================================================================

CREATE TABLE IF NOT EXISTS capital_calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(id) ON DELETE CASCADE,
    
    -- Call details
    call_number INTEGER NOT NULL CHECK (call_number > 0),
    call_date DATE NOT NULL,
    due_date DATE NOT NULL,
    amount_requested DECIMAL(15,2) NOT NULL CHECK (amount_requested > 0),
    
    -- Status tracking
    status VARCHAR(50) DEFAULT 'PENDING' CHECK (status IN (
        'PENDING',
        'FUNDED',
        'PARTIALLY_FUNDED',
        'LATE',
        'DEFAULTED',
        'CANCELLED'
    )),
    amount_funded DECIMAL(15,2) DEFAULT 0 CHECK (amount_funded >= 0),
    funded_date DATE,
    
    -- Cash management integration
    liquidity_check_status VARCHAR(50) CHECK (liquidity_check_status IN (
        'SUFFICIENT',
        'MARGINAL',
        'INSUFFICIENT',
        'NOT_CHECKED'
    )),
    recommended_funding_source_account_id UUID,
    alert_sent BOOLEAN DEFAULT FALSE,
    alert_sent_at TIMESTAMPTZ,
    
    -- Document reference
    notice_document_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT check_amount_funded_lte_requested CHECK (amount_funded <= amount_requested),
    CONSTRAINT check_due_date_gte_call_date CHECK (due_date >= call_date),
    CONSTRAINT unique_call_number_per_investment UNIQUE(investment_id, call_number)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_capital_calls_status ON capital_calls(status, due_date);
CREATE INDEX IF NOT EXISTS idx_capital_calls_investment ON capital_calls(investment_id);
CREATE INDEX IF NOT EXISTS idx_capital_calls_due_date ON capital_calls(due_date) WHERE status IN ('PENDING', 'PARTIALLY_FUNDED');

-- ============================================================================
-- CAPITAL DISTRIBUTIONS (Money returned to LP)
-- ============================================================================

CREATE TABLE IF NOT EXISTS capital_distributions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(id) ON DELETE CASCADE,
    
    -- Distribution details
    distribution_date DATE NOT NULL,
    amount DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    distribution_type VARCHAR(50) CHECK (distribution_type IN (
        'RETURN_OF_CAPITAL',
        'CAPITAL_GAIN',
        'INCOME',
        'RECALLABLE',
        'DIVIDEND'
    )),
    is_recallable BOOLEAN DEFAULT FALSE,
    
    -- Tax implications
    ordinary_income DECIMAL(15,2) DEFAULT 0 CHECK (ordinary_income >= 0),
    long_term_capital_gain DECIMAL(15,2) DEFAULT 0 CHECK (long_term_capital_gain >= 0),
    short_term_capital_gain DECIMAL(15,2) DEFAULT 0 CHECK (short_term_capital_gain >= 0),
    return_of_capital DECIMAL(15,2) DEFAULT 0 CHECK (return_of_capital >= 0),
    
    -- Document reference
    notice_document_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT check_distribution_amounts_sum CHECK (
        ordinary_income + long_term_capital_gain + short_term_capital_gain + return_of_capital <= amount
    )
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_distributions_investment ON capital_distributions(investment_id, distribution_date DESC);
CREATE INDEX IF NOT EXISTS idx_distributions_date ON capital_distributions(distribution_date DESC);

-- ============================================================================
-- PERFORMANCE METRICS (Calculated periodically)
-- ============================================================================

CREATE TABLE IF NOT EXISTS alternative_investment_performance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(id) ON DELETE CASCADE,
    
    -- Calculation period
    as_of_date DATE NOT NULL,
    
    -- Core metrics
    irr_since_inception DECIMAL(8,5), -- Internal Rate of Return (e.g., 0.15234 = 15.234%)
    tvpi DECIMAL(8,4), -- Total Value / Paid-In
    dpi DECIMAL(8,4), -- Distributions / Paid-In
    rvpi DECIMAL(8,4), -- Residual Value / Paid-In (RVPI = TVPI - DPI)
    moic DECIMAL(8,4), -- Multiple on Invested Capital
    
    -- PME (Public Market Equivalent) benchmarking
    pme_kaplan_schoar DECIMAL(8,4), -- vs S&P 500
    pme_direct_alpha DECIMAL(8,4),
    benchmark_index VARCHAR(50) DEFAULT 'SP500',
    
    -- J-curve analysis
    j_curve_position VARCHAR(50) CHECK (j_curve_position IN (
        'INVESTMENT',
        'HARVESTING',
        'MATURE'
    )),
    
    -- Benchmark comparison
    peer_median_irr DECIMAL(8,5),
    peer_top_quartile_irr DECIMAL(8,5),
    percentile_rank INTEGER CHECK (percentile_rank >= 1 AND percentile_rank <= 100),
    
    -- Cash flow details
    total_called DECIMAL(15,2),
    total_distributed DECIMAL(15,2),
    nav_value DECIMAL(15,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_investment_date UNIQUE(investment_id, as_of_date)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_performance_investment_date ON alternative_investment_performance(investment_id, as_of_date DESC);
CREATE INDEX IF NOT EXISTS idx_performance_as_of_date ON alternative_investment_performance(as_of_date DESC);

-- ============================================================================
-- CAPITAL CALL FORECASTS (ML-powered predictions)
-- ============================================================================

CREATE TABLE IF NOT EXISTS capital_call_forecasts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(id) ON DELETE CASCADE,
    
    -- Forecast details
    forecasted_call_date DATE NOT NULL,
    estimated_amount DECIMAL(15,2) NOT NULL CHECK (estimated_amount > 0),
    confidence_score DECIMAL(3,2) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    
    -- Model metadata
    model_version VARCHAR(50),
    model_type VARCHAR(50) DEFAULT 'STATISTICAL',
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Alert configuration
    days_notice_before_due INTEGER DEFAULT 14 CHECK (days_notice_before_due > 0),
    alert_triggered BOOLEAN DEFAULT FALSE,
    alert_triggered_at TIMESTAMPTZ,
    
    -- Actual outcome (for model training)
    actual_call_id UUID REFERENCES capital_calls(id),
    forecast_accuracy_score DECIMAL(3,2)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_forecasts_investment ON capital_call_forecasts(investment_id, forecasted_call_date);
CREATE INDEX IF NOT EXISTS idx_forecasts_date ON capital_call_forecasts(forecasted_call_date) WHERE alert_triggered = FALSE;

-- ============================================================================
-- DOCUMENT STORAGE AND PROCESSING
-- ============================================================================

CREATE TABLE IF NOT EXISTS alternative_investment_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(id) ON DELETE CASCADE,
    
    -- Document metadata
    document_type VARCHAR(50) CHECK (document_type IN (
        'K1',
        'CAPITAL_CALL',
        'DISTRIBUTION_NOTICE',
        'QUARTERLY_STATEMENT',
        'ANNUAL_REPORT',
        'SUBSCRIPTION_DOCS',
        'SIDE_LETTER',
        'AUDITED_FINANCIALS'
    )),
    file_name VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    file_size_bytes BIGINT,
    mime_type VARCHAR(100),
    
    -- AI processing status
    processing_status VARCHAR(50) DEFAULT 'PENDING' CHECK (processing_status IN (
        'PENDING',
        'PROCESSING',
        'COMPLETED',
        'FAILED',
        'NEEDS_REVIEW',
        'REVIEWED'
    )),
    processed_at TIMESTAMPTZ,
    processing_error TEXT,
    
    -- Extracted data (JSONB for flexibility)
    extracted_data JSONB,
    confidence_scores JSONB, -- Confidence scores for each extracted field
    
    -- Human review
    requires_review BOOLEAN DEFAULT FALSE,
    reviewed_by UUID,
    reviewed_at TIMESTAMPTZ,
    review_notes TEXT,
    review_status VARCHAR(50) CHECK (review_status IN (
        'APPROVED',
        'REJECTED',
        'NEEDS_CORRECTION',
        'PENDING'
    )),
    
    uploaded_at TIMESTAMPTZ DEFAULT NOW(),
    uploaded_by UUID
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_documents_investment_type ON alternative_investment_documents(investment_id, document_type);
CREATE INDEX IF NOT EXISTS idx_documents_status ON alternative_investment_documents(processing_status) 
    WHERE processing_status IN ('PENDING', 'NEEDS_REVIEW');
CREATE INDEX IF NOT EXISTS idx_documents_uploaded_at ON alternative_investment_documents(uploaded_at DESC);

-- ============================================================================
-- ASSET-SPECIFIC KPIs (Polymorphic by asset class)
-- ============================================================================

CREATE TABLE IF NOT EXISTS alternative_investment_kpis (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(id) ON DELETE CASCADE,
    
    -- Reporting period
    period_end_date DATE NOT NULL,
    
    -- KPIs stored as JSONB for flexibility
    kpis JSONB NOT NULL,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_investment_kpi_period UNIQUE(investment_id, period_end_date)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_kpis_investment ON alternative_investment_kpis(investment_id, period_end_date DESC);
CREATE INDEX IF NOT EXISTS idx_kpis_period ON alternative_investment_kpis(period_end_date DESC);
CREATE INDEX IF NOT EXISTS idx_kpis_jsonb ON alternative_investment_kpis USING GIN (kpis);

-- ============================================================================
-- COMMENTS TABLE FOR KPIs (Example JSONB structures)
-- ============================================================================

COMMENT ON TABLE alternative_investment_kpis IS 
'Stores asset-class-specific KPIs in JSONB format. Examples:

PRIVATE_EQUITY:
{
  "revenue_growth_rate": 0.25,
  "ebitda_margin": 0.35,
  "leverage_ratio": 4.5,
  "exit_multiple": 8.2,
  "hold_period_months": 48
}

REAL_ESTATE:
{
  "occupancy_rate": 0.95,
  "noi_growth": 0.08,
  "cap_rate": 0.065,
  "debt_yield": 0.12,
  "cash_on_cash_return": 0.11
}

VENTURE_CAPITAL:
{
  "arr_growth": 2.5,
  "burn_multiple": 1.2,
  "magic_number": 0.8,
  "revenue_per_employee": 150000,
  "customer_acquisition_cost": 5000
}';

-- ============================================================================
-- TRIGGER: Update updated_at timestamp
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_alternative_investments_updated_at ON alternative_investments;
CREATE TRIGGER update_alternative_investments_updated_at
    BEFORE UPDATE ON alternative_investments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_capital_calls_updated_at ON capital_calls;
CREATE TRIGGER update_capital_calls_updated_at
    BEFORE UPDATE ON capital_calls
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- TRIGGER: Auto-update capital_called on alternative_investments
-- ============================================================================

CREATE OR REPLACE FUNCTION update_capital_called_on_funding()
RETURNS TRIGGER AS $$
BEGIN
    -- Update capital_called when a capital call is funded
    IF NEW.status IN ('FUNDED', 'PARTIALLY_FUNDED') AND 
       (OLD.status IS NULL OR OLD.status NOT IN ('FUNDED', 'PARTIALLY_FUNDED')) THEN
        
        UPDATE alternative_investments
        SET capital_called = capital_called + NEW.amount_funded
        WHERE id = NEW.investment_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_capital_called ON capital_calls;
CREATE TRIGGER trigger_update_capital_called
    AFTER UPDATE ON capital_calls
    FOR EACH ROW
    EXECUTE FUNCTION update_capital_called_on_funding();

-- ============================================================================
-- TRIGGER: Auto-update capital_distributed on alternative_investments
-- ============================================================================

CREATE OR REPLACE FUNCTION update_capital_distributed_on_distribution()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE alternative_investments
    SET capital_distributed = capital_distributed + NEW.amount
    WHERE id = NEW.investment_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_capital_distributed
    AFTER INSERT ON capital_distributions
    FOR EACH ROW
    EXECUTE FUNCTION update_capital_distributed_on_distribution();

-- ============================================================================
-- GRANT PERMISSIONS
-- ============================================================================

-- Grant appropriate permissions (adjust role names as needed)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO semlayer_app;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO semlayer_app;
