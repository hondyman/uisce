-- Migration 040: Alternative Investments Platform
-- Multi-asset class tracking for PE, hedge funds, real estate, private credit

-- =============================================================================
-- 1. ASSET CLASSES & TYPES
-- =============================================================================

CREATE TYPE alternative_asset_class AS ENUM (
    'PRIVATE_EQUITY',
    'VENTURE_CAPITAL',
    'HEDGE_FUND',
    'REAL_ESTATE',
    'PRIVATE_CREDIT',
    'INFRASTRUCTURE',
    'COMMODITIES',
    'COLLECTIBLES'
);

CREATE TYPE investment_status AS ENUM (
    'COMMITTED',
    'FUNDED',
    'ACTIVE',
    'DISTRIBUTING',
    'LIQUIDATED',
    'WRITTEN_OFF'
);

-- =============================================================================
-- 2. ALTERNATIVE INVESTMENTS
-- =============================================================================

CREATE TABLE alternative_investments (
    investment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Investment details
    investment_name TEXT NOT NULL,
    asset_class alternative_asset_class NOT NULL,
    fund_manager TEXT,
    fund_vintage_year INTEGER,
    
    -- Commitment & funding
    committed_capital DECIMAL(15,2) NOT NULL,
    funded_capital DECIMAL(15,2) DEFAULT 0,
    unfunded_commitment DECIMAL(15,2) GENERATED ALWAYS AS (committed_capital - funded_capital) STORED,
    
    -- Current values
    current_nav DECIMAL(15,2) DEFAULT 0, -- Net Asset Value
    current_fair_value DECIMAL(15,2) DEFAULT 0,
    
    -- Distributions
    total_distributions DECIMAL(15,2) DEFAULT 0,
    
    -- Performance metrics
    irr DECIMAL(8,4), -- Internal Rate of Return (percentage)
    moic DECIMAL(8,4), -- Multiple on Invested Capital
    tvpi DECIMAL(8,4), -- Total Value to Paid-In
    dpi DECIMAL(8,4), -- Distributions to Paid-In
    rvpi DECIMAL(8,4), -- Residual Value to Paid-In
    
    -- Fund details (JSONB for flexibility)
    fund_details JSONB DEFAULT '{}',
    /* Example for PE:
    {
        "strategy": "Growth Equity",
        "sector_focus": ["Technology", "Healthcare"],
        "geographic_focus": ["North America", "Europe"],
        "fund_size": 500000000,
        "management_fee": 0.02,
        "carried_interest": 0.20,
        "preferred_return": 0.08
    }
    */
    
    -- Status
    investment_status investment_status DEFAULT 'COMMITTED',
    
    -- Dates
    commitment_date DATE,
    first_close_date DATE,
    final_close_date DATE,
    anticipated_liquidation_date DATE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_alt_inv_client (client_id, asset_class),
    INDEX idx_alt_inv_status (investment_status, tenant_id)
);

-- =============================================================================
-- 3. CAPITAL CALLS (Drawdowns)
-- =============================================================================

CREATE TABLE capital_calls (
    call_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(investment_id) ON DELETE CASCADE,
    
    -- Call details
    call_number INTEGER NOT NULL,
    call_date DATE NOT NULL,
    due_date DATE NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    
    -- Purpose
    purpose TEXT, -- e.g., "Acquisition of ABC Corp", "Management fees"
    call_type VARCHAR(50), -- 'INVESTMENT', 'FEES', 'EXPENSES'
    
    -- Payment
    paid_amount DECIMAL(15,2) DEFAULT 0,
    paid_date DATE,
    payment_reference TEXT,
    
    -- Status
    call_status VARCHAR(20) DEFAULT 'PENDING', -- 'PENDING', 'PAID', 'OVERDUE', 'WAIVED'
    
    -- Notifications
    notice_received_date DATE,
    reminder_sent BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_capital_calls_investment (investment_id, call_date DESC),
    INDEX idx_capital_calls_due (due_date, call_status) WHERE call_status = 'PENDING'
);

-- =============================================================================
-- 4. DISTRIBUTIONS
-- =============================================================================

CREATE TABLE alternative_distributions (
    distribution_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(investment_id) ON DELETE CASCADE,
    
    -- Distribution details
    distribution_date DATE NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    
    -- Type
    distribution_type VARCHAR(50), -- 'INCOME', 'RETURN_OF_CAPITAL', 'CAPITAL_GAIN', 'LIQUIDATION'
    
    -- Tax reporting
    is_qualified_dividend BOOLEAN DEFAULT FALSE,
    is_capital_gain BOOLEAN DEFAULT FALSE,
    capital_gain_type VARCHAR(20), -- 'SHORT_TERM', 'LONG_TERM'
    
    -- Recallable
    is_recallable BOOLEAN DEFAULT FALSE,
    recall_period_end_date DATE,
    
    -- Payment
    received_date DATE,
    payment_reference TEXT,
    
    -- Tax documents
    k1_document_id UUID, -- Reference to document storage
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_distributions_investment (investment_id, distribution_date DESC)
);

-- =============================================================================
-- 5. VALUATIONS
-- =============================================================================

CREATE TABLE alternative_valuations (
    valuation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(investment_id) ON DELETE CASCADE,
    
    -- Valuation details
    valuation_date DATE NOT NULL,
    nav_per_share DECIMAL(12,6),
    total_nav DECIMAL(15,2) NOT NULL,
    fair_value DECIMAL(15,2),
    
    -- Valuation methodology
    valuation_method VARCHAR(50), -- 'MARKET', 'COST', 'DCF', 'COMPARABLE', 'APPRAISAL'
    
    -- Source
    source VARCHAR(50), -- 'FUND_MANAGER', 'THIRD_PARTY', 'INTERNAL'
    verified BOOLEAN DEFAULT FALSE,
    
    -- Change from previous
    previous_nav DECIMAL(15,2),
    change_amount DECIMAL(15,2),
    change_percentage DECIMAL(8,4),
    
    -- Supporting documents
    valuation_report_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_valuations_investment (investment_id, valuation_date DESC),
    UNIQUE (investment_id, valuation_date)
);

-- =============================================================================
-- 6. REAL ESTATE SPECIFIC
-- =============================================================================

CREATE TABLE real_estate_properties (
    property_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(investment_id) ON DELETE CASCADE,
    
    -- Property details
    property_name TEXT NOT NULL,
    property_type VARCHAR(50), -- 'MULTIFAMILY', 'OFFICE', 'RETAIL', 'INDUSTRIAL', 'HOSPITALITY'
    
    -- Location
    address TEXT,
    city TEXT,
    state VARCHAR(2),
    zip_code VARCHAR(10),
    country VARCHAR(2) DEFAULT 'US',
    
    -- Financials
    acquisition_price DECIMAL(15,2),
    acquisition_date DATE,
    current_value DECIMAL(15,2),
    
    -- Size
    square_footage INTEGER,
    units INTEGER, -- For multifamily
    occupancy_rate DECIMAL(5,2), -- Percentage
    
    -- Income
    annual_noi DECIMAL(15,2), -- Net Operating Income
    cap_rate DECIMAL(5,2), -- Capitalization rate
    
    -- Opportunity Zone
    is_opportunity_zone BOOLEAN DEFAULT FALSE,
    oz_designation_date DATE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_properties_investment (investment_id)
);

-- =============================================================================
-- 7. HEDGE FUND SPECIFIC
-- =============================================================================

CREATE TABLE hedge_fund_details (
    fund_detail_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(investment_id) ON DELETE CASCADE,
    
    -- Strategy
    strategy VARCHAR(100), -- 'LONG_SHORT_EQUITY', 'EVENT_DRIVEN', 'MACRO', 'RELATIVE_VALUE'
    sub_strategy TEXT,
    
    -- Terms
    management_fee DECIMAL(5,4), -- e.g., 0.0200 for 2%
    performance_fee DECIMAL(5,4), -- e.g., 0.2000 for 20%
    high_water_mark DECIMAL(15,2),
    hurdle_rate DECIMAL(5,4),
    
    -- Liquidity
    redemption_frequency VARCHAR(50), -- 'MONTHLY', 'QUARTERLY', 'ANNUALLY'
    redemption_notice_days INTEGER,
    lockup_period_months INTEGER,
    lockup_end_date DATE,
    
    -- Side pockets
    has_side_pocket BOOLEAN DEFAULT FALSE,
    side_pocket_value DECIMAL(15,2) DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_hedge_fund_investment (investment_id)
);

-- =============================================================================
-- 8. HELPER FUNCTIONS
-- =============================================================================

-- Calculate investment performance metrics
CREATE OR REPLACE FUNCTION calculate_investment_metrics(p_investment_id UUID)
RETURNS VOID AS $$
DECLARE
    v_investment RECORD;
    v_funded DECIMAL;
    v_distributions DECIMAL;
    v_current_value DECIMAL;
BEGIN
    -- Get investment
    SELECT * INTO v_investment FROM alternative_investments WHERE investment_id = p_investment_id;
    
    -- Calculate funded capital from capital calls
    SELECT COALESCE(SUM(paid_amount), 0) INTO v_funded
    FROM capital_calls
    WHERE investment_id = p_investment_id
    AND call_status = 'PAID';
    
    -- Calculate total distributions
    SELECT COALESCE(SUM(amount), 0) INTO v_distributions
    FROM alternative_distributions
    WHERE investment_id = p_investment_id;
    
    -- Get current NAV from latest valuation
    SELECT COALESCE(total_nav, 0) INTO v_current_value
    FROM alternative_valuations
    WHERE investment_id = p_investment_id
    ORDER BY valuation_date DESC
    LIMIT 1;
    
    -- Update investment with calculated metrics
    UPDATE alternative_investments
    SET funded_capital = v_funded,
        total_distributions = v_distributions,
        current_nav = v_current_value,
        tvpi = CASE WHEN v_funded > 0 THEN (v_distributions + v_current_value) / v_funded ELSE NULL END,
        dpi = CASE WHEN v_funded > 0 THEN v_distributions / v_funded ELSE NULL END,
        rvpi = CASE WHEN v_funded > 0 THEN v_current_value / v_funded ELSE NULL END,
        moic = CASE WHEN v_funded > 0 THEN (v_distributions + v_current_value) / v_funded ELSE NULL END,
        updated_at = NOW()
    WHERE investment_id = p_investment_id;
END;
$$ LANGUAGE plpgsql;

-- Get pending capital calls
CREATE OR REPLACE FUNCTION get_pending_capital_calls(p_client_id UUID)
RETURNS TABLE (
    investment_name TEXT,
    call_amount DECIMAL,
    due_date DATE,
    days_until_due INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ai.investment_name,
        cc.amount,
        cc.due_date,
        (cc.due_date - CURRENT_DATE)::INTEGER AS days_until_due
    FROM capital_calls cc
    JOIN alternative_investments ai ON cc.investment_id = ai.investment_id
    WHERE ai.client_id = p_client_id
    AND cc.call_status = 'PENDING'
    AND cc.due_date >= CURRENT_DATE
    ORDER BY cc.due_date;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 9. TRIGGERS
-- =============================================================================

-- Auto-update investment metrics on capital call payment
CREATE OR REPLACE FUNCTION update_metrics_on_capital_call() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.call_status = 'PAID' AND OLD.call_status != 'PAID' THEN
        PERFORM calculate_investment_metrics(NEW.investment_id);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER capital_call_metrics_trigger
AFTER UPDATE ON capital_calls
FOR EACH ROW
EXECUTE FUNCTION update_metrics_on_capital_call();

-- Auto-update metrics on distribution
CREATE OR REPLACE FUNCTION update_metrics_on_distribution() RETURNS TRIGGER AS $$
BEGIN
    PERFORM calculate_investment_metrics(NEW.investment_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER distribution_metrics_trigger
AFTER INSERT ON alternative_distributions
FOR EACH ROW
EXECUTE FUNCTION update_metrics_on_distribution();

-- =============================================================================
-- 10. RLS POLICIES
-- =============================================================================

ALTER TABLE alternative_investments ENABLE ROW LEVEL SECURITY;
ALTER TABLE capital_calls ENABLE ROW LEVEL SECURITY;
ALTER TABLE alternative_distributions ENABLE ROW LEVEL SECURITY;
ALTER TABLE alternative_valuations ENABLE ROW LEVEL SECURITY;

CREATE POLICY alt_investments_tenant_isolation ON alternative_investments
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 11. COMMENTS
-- =============================================================================

COMMENT ON TABLE alternative_investments IS 'Multi-asset class alternative investments with performance tracking';
COMMENT ON TABLE capital_calls IS 'Capital call (drawdown) tracking with payment status';
COMMENT ON TABLE alternative_distributions IS 'Distribution tracking with tax classification';
COMMENT ON TABLE alternative_valuations IS 'Quarterly valuations with methodology tracking';
COMMENT ON FUNCTION calculate_investment_metrics IS 'Auto-calculate IRR, MOIC, TVPI, DPI, RVPI metrics';
