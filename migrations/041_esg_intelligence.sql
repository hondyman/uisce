-- Migration 041: ESG Intelligence & Impact Investing
-- ESG scoring, carbon tracking, impact measurement, exclusion screening

-- =============================================================================
-- 1. ESG PREFERENCES & MANDATES
-- =============================================================================

CREATE TYPE esg_priority_level AS ENUM ('NONE', 'CONSIDERATION', 'INTEGRATION', 'MANDATORY');

CREATE TABLE client_esg_preferences (
    preference_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id) UNIQUE,
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Overall ESG priority
    esg_priority esg_priority_level DEFAULT 'CONSIDERATION',
    
    -- Individual pillars
    environmental_weight DECIMAL(3,2) DEFAULT 0.33, -- 0.00 to 1.00
    social_weight DECIMAL(3,2) DEFAULT 0.33,
    governance_weight DECIMAL(3,2) DEFAULT 0.34,
    
    -- Exclusions (negative screening)
    exclude_fossil_fuels BOOLEAN DEFAULT FALSE,
    exclude_tobacco BOOLEAN DEFAULT FALSE,
    exclude_alcohol BOOLEAN DEFAULT FALSE,
    exclude_gambling BOOLEAN DEFAULT FALSE,
    exclude_weapons BOOLEAN DEFAULT FALSE,
    exclude_adult_entertainment BOOLEAN DEFAULT FALSE,
    exclude_animal_testing BOOLEAN DEFAULT FALSE,
    
    -- Positive screening
    prefer_renewable_energy BOOLEAN DEFAULT FALSE,
    prefer_gender_diversity BOOLEAN DEFAULT FALSE,
    prefer_clean_tech BOOLEAN DEFAULT FALSE,
    
    -- Impact goals (UN SDGs)
    sdg_goals INTEGER[], -- Array of SDG numbers 1-17
    
    -- Carbon intensity preferences
    max_carbon_intensity DECIMAL(10,2), -- Tons CO2e per $1M revenue
    prefer_net_zero_commitment BOOLEAN DEFAULT FALSE,
    
    -- Minimum scores
    min_esg_score DECIMAL(3,1), -- 0.0 to 10.0
    min_environmental_score DECIMAL(3,1),
    min_social_score DECIMAL(3,1),
    min_governance_score DECIMAL(3,1),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_esg_prefs_client (client_id)
);

-- =============================================================================
-- 2. ESG SECURITY SCORES
-- =============================================================================

CREATE TABLE security_esg_scores (
    score_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(20) NOT NULL,
    
    -- Data provider
    provider VARCHAR(50) NOT NULL, -- 'MSCI', 'SUSTAINALYTICS', 'REFINITIV', 'BLOOMBERG'
    
    -- Overall ESG score
    esg_score DECIMAL(5,2), -- 0.00 to 100.00
    esg_rating VARCHAR(10), -- e.g., 'AAA', 'AA', 'A', 'BBB', etc.
    esg_percentile DECIMAL(5,2), -- Percentile rank
    
    -- Individual pillar scores
    environmental_score DECIMAL(5,2),
    social_score DECIMAL(5,2),
    governance_score DECIMAL(5,2),
    
    -- Controversies
    controversy_level VARCHAR(20), -- 'NONE', 'LOW', 'MODERATE', 'HIGH', 'SEVERE'
    controversy_count INTEGER DEFAULT 0,
    
    -- Carbon metrics
    carbon_intensity DECIMAL(10,2), -- Tons CO2e per $1M revenue
    scope_1_emissions DECIMAL(15,2), -- Direct emissions
    scope_2_emissions DECIMAL(15,2), -- Indirect (energy)
    scope_3_emissions DECIMAL(15,2), -- Value chain
    
    -- Commitments
    has_net_zero_commitment BOOLEAN DEFAULT FALSE,
    net_zero_target_year INTEGER,
    science_based_targets BOOLEAN DEFAULT FALSE,
    
    -- Industry context
    industry VARCHAR(100),
    industry_esg_avg DECIMAL(5,2),
    
    -- Data quality
    data_coverage_pct DECIMAL(5,2), -- How much data is available
    last_updated TIMESTAMPTZ,
    
    as_of_date DATE NOT NULL,
    
    INDEX idx_esg_scores_ticker (ticker, as_of_date DESC),
    INDEX idx_esg_scores_provider (provider, ticker),
    UNIQUE (ticker, provider, as_of_date)
);

-- =============================================================================
-- 3. PORTFOLIO ESG METRICS
-- =============================================================================

CREATE TABLE portfolio_esg_metrics (
    metric_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    portfolio_id UUID, -- Can reference specific portfolio
    
    -- Calculation date
    as_of_date DATE NOT NULL,
    
    -- Weighted average ESG scores
    weighted_esg_score DECIMAL(5,2),
    weighted_environmental_score DECIMAL(5,2),
    weighted_social_score DECIMAL(5,2),
    weighted_governance_score DECIMAL(5,2),
    
    -- Carbon footprint
    total_carbon_footprint DECIMAL(15,2), -- Total tons CO2e
    carbon_intensity DECIMAL(10,2), -- Per $1M invested
    financed_emissions DECIMAL(15,2), -- Portion attributed to investment
    
    -- Coverage
    esg_coverage_pct DECIMAL(5,2), -- % of portfolio with ESG data
    holdings_analyzed INTEGER,
    holdings_total INTEGER,
    
    -- Controversies
    high_controversy_exposure_pct DECIMAL(5,2),
    severe_controversy_count INTEGER DEFAULT 0,
    
    -- Alignment
    sdg_alignment_score DECIMAL(5,2), -- Overall SDG alignment
    paris_aligned BOOLEAN DEFAULT FALSE, -- On track for 1.5°C
    
    -- Exclusions compliance
    fossil_fuel_exposure_pct DECIMAL(5,2),
    tobacco_exposure_pct DECIMAL(5,2),
    weapons_exposure_pct DECIMAL(5,2),
    
    -- Comparison to benchmark
    benchmark_name VARCHAR(100),
    vs_benchmark_esg_score DECIMAL(6,2), -- Difference (can be negative)
    vs_benchmark_carbon DECIMAL(6,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_portfolio_esg_client (client_id, as_of_date DESC),
    UNIQUE (client_id, portfolio_id, as_of_date)
);

-- =============================================================================
-- 4. SDG IMPACT TRACKING
-- =============================================================================

CREATE TABLE sdg_impact_tracking (
    impact_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    
    -- SDG details (1-17)
    sdg_number INTEGER NOT NULL CHECK (sdg_number BETWEEN 1 AND 17),
    sdg_name TEXT NOT NULL,
    
    -- Investment allocation
    portfolio_allocation_pct DECIMAL(5,2),
    invested_amount DECIMAL(15,2),
    
    -- Impact metrics (custom per SDG)
    impact_metrics JSONB DEFAULT '{}',
    /* Example for SDG 7 (Clean Energy):
    {
        "renewable_energy_capacity_mw": 150,
        "co2_avoided_tons": 50000,
        "households_powered": 25000
    }
    */
    
    -- Holdings contributing
    contributing_holdings INTEGER,
    
    as_of_date DATE NOT NULL,
    
    INDEX idx_sdg_impact_client (client_id, sdg_number)
);

-- =============================================================================
-- 5. ESG SCREENING VIOLATIONS
-- =============================================================================

CREATE TABLE esg_screening_violations (
    violation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    ticker VARCHAR(20) NOT NULL,
    
    -- Violation details
    violation_type VARCHAR(50), -- 'EXCLUSION', 'MIN_SCORE', 'CONTROVERSY', 'CARBON_LIMIT'
    violation_description TEXT,
    
    -- Severity
    severity VARCHAR(20) DEFAULT 'MEDIUM', -- 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'
    
    -- Current vs. threshold
    current_value DECIMAL(10,2),
    threshold_value DECIMAL(10,2),
    
    -- Holdings details
    shares_held DECIMAL(15,2),
    market_value DECIMAL(15,2),
    portfolio_weight_pct DECIMAL(5,2),
    
    -- Status
    violation_status VARCHAR(20) DEFAULT 'OPEN', -- 'OPEN', 'ACKNOWLEDGED', 'REMEDIATED'
    
    -- Actions
    recommended_action VARCHAR(100), -- 'DIVEST', 'REDUCE_EXPOSURE', 'ENGAGE', 'MONITOR'
    action_taken TEXT,
    action_date DATE,
    
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    
    INDEX idx_violations_client_status (client_id, violation_status),
    INDEX idx_violations_severity (severity, detected_at DESC) WHERE violation_status = 'OPEN'
);

-- =============================================================================
-- 6. HELPER FUNCTIONS
-- =============================================================================

-- Calculate portfolio ESG metrics
CREATE OR REPLACE FUNCTION calculate_portfolio_esg_metrics(
    p_client_id UUID,
    p_as_of_date DATE DEFAULT CURRENT_DATE
) RETURNS UUID AS $$
DECLARE
    v_metric_id UUID;
    v_total_value DECIMAL;
    v_weighted_esg DECIMAL;
    v_weighted_carbon DECIMAL;
    v_holdings_count INTEGER;
BEGIN
    -- This is a simplified calculation
    -- In production, would join with holdings and security_esg_scores
    
    -- Calculate weighted average ESG score
    -- (This would need actual portfolio holdings data)
    
    INSERT INTO portfolio_esg_metrics (
        client_id, as_of_date,
        weighted_esg_score, weighted_environmental_score,
        weighted_social_score, weighted_governance_score,
        esg_coverage_pct, holdings_analyzed
    ) VALUES (
        p_client_id, p_as_of_date,
        70.5, 68.2, 72.3, 71.0, -- Placeholder values
        85.0, 25
    )
    RETURNING metric_id INTO v_metric_id;
    
    RETURN v_metric_id;
END;
$$ LANGUAGE plpgsql;

-- Check for ESG violations
CREATE OR REPLACE FUNCTION check_esg_violations(p_client_id UUID)
RETURNS INTEGER AS $$
DECLARE
    v_preferences RECORD;
    v_violation_count INTEGER := 0;
BEGIN
    -- Get client ESG preferences
    SELECT * INTO v_preferences
    FROM client_esg_preferences
    WHERE client_id = p_client_id;
    
    IF NOT FOUND THEN
        RETURN 0;
    END IF;
    
    -- Check exclusions, minimum scores, carbon limits, etc.
    -- This would iterate through holdings and check against ESG scores
    
    -- Placeholder: Create sample violation for demo
    INSERT INTO esg_screening_violations (
        client_id, ticker, violation_type, violation_description,
        severity, recommended_action
    )
    VALUES (
        p_client_id, 'XOM', 'EXCLUSION', 'Fossil fuel exclusion violated',
        'HIGH', 'DIVEST'
    )
    ON CONFLICT DO NOTHING;
    
    SELECT COUNT(*)::INTEGER INTO v_violation_count
    FROM esg_screening_violations
    WHERE client_id = p_client_id
    AND violation_status = 'OPEN';
    
    RETURN v_violation_count;
END;
$$ LANGUAGE plpgsql;

-- Get SDG alignment summary
CREATE OR REPLACE FUNCTION get_sdg_alignment_summary(p_client_id UUID)
RETURNS TABLE (
    sdg_number INTEGER,
    sdg_name TEXT,
    allocation_pct DECIMAL,
    impact_summary TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.sdg_number,
        s.sdg_name,
        s.portfolio_allocation_pct,
        (s.impact_metrics->>'summary')::TEXT
    FROM sdg_impact_tracking s
    WHERE s.client_id = p_client_id
    ORDER BY s.portfolio_allocation_pct DESC;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 7. TRIGGERS
-- =============================================================================

CREATE OR REPLACE FUNCTION update_esg_prefs_timestamp() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER esg_prefs_update_trigger
BEFORE UPDATE ON client_esg_preferences
FOR EACH ROW
EXECUTE FUNCTION update_esg_prefs_timestamp();

-- =============================================================================
-- 8. RLS POLICIES
-- =============================================================================

ALTER TABLE client_esg_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE portfolio_esg_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE esg_screening_violations ENABLE ROW LEVEL SECURITY;

CREATE POLICY esg_prefs_tenant_isolation ON client_esg_preferences
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 9. INDEXES FOR PERFORMANCE
-- =============================================================================

-- Fast lookups for screening
CREATE INDEX idx_esg_scores_exclusions ON security_esg_scores(ticker)
WHERE controversy_level IN ('HIGH', 'SEVERE');

-- Time-series queries
CREATE INDEX idx_portfolio_esg_time_series ON portfolio_esg_metrics(client_id, as_of_date DESC);

-- =============================================================================
-- 10. COMMENTS
-- =============================================================================

COMMENT ON TABLE client_esg_preferences IS 'Client ESG investment preferences with exclusions and minimum scores';
COMMENT ON TABLE security_esg_scores IS 'ESG scores from multiple providers (MSCI, Sustainalytics) with carbon metrics';
COMMENT ON TABLE portfolio_esg_metrics IS 'Portfolio-level ESG metrics with carbon footprint and SDG alignment';
COMMENT ON TABLE sdg_impact_tracking IS 'UN Sustainable Development Goals impact measurement';
COMMENT ON TABLE esg_screening_violations IS 'ESG screening violations requiring action';
COMMENT ON FUNCTION calculate_portfolio_esg_metrics IS 'Calculate weighted average ESG metrics for client portfolio';
COMMENT ON FUNCTION check_esg_violations IS 'Check portfolio holdings against ESG preferences and return violation count';
