-- WealthVision Phase 1 Database Schema
-- Migration: Add tables for tax optimization, multi-generational planning, 
-- alternative investments, AI intelligence, and ESG analytics

-- ==============================================================================
-- ==============================================================================
-- FAMILY OFFICES (Base Table)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS family_offices (
    family_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ==============================================================================
-- TAX STRATEGIES
-- ==============================================================================

CREATE TABLE IF NOT EXISTS tax_strategies (
    strategy_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    strategy_type VARCHAR(100) NOT NULL, -- STATE_RESIDENCY, NIIT, CHARITABLE_BUNCHING, SALT, FTC
    strategy_name VARCHAR(255) NOT NULL,
    total_tax_savings NUMERIC(18,2) NOT NULL,
    implementation_complexity INT CHECK (implementation_complexity BETWEEN 1 AND 10),
    implementation_steps JSONB,
    estimated_cost NUMERIC(18,2),
    risk_level VARCHAR(50), -- LOW, MEDIUM, HIGH
    recommendation_rationale TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255),
    CONSTRAINT fk_tax_strategy_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_tax_strategies_family ON tax_strategies(family_id);
CREATE INDEX idx_tax_strategies_type ON tax_strategies(strategy_type);
CREATE INDEX idx_tax_strategies_created ON tax_strategies(created_at DESC);

-- State Residency Comparisons
CREATE TABLE IF NOT EXISTS state_residency_comparisons (
    comparison_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    current_state VARCHAR(2) NOT NULL,
    analysis_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    gross_income NUMERIC(18,2),
    investment_income NUMERIC(18,2),
    capital_gains NUMERIC(18,2),
    estate_value NUMERIC(18,2),
    state_comparisons JSONB NOT NULL, -- Array of state details
    top_recommendations JSONB, -- Top 3 recommended states
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_residency_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_residency_family ON state_residency_comparisons(family_id);

-- NIIT Calculations
CREATE TABLE IF NOT EXISTS niit_calculations (
    calculation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    member_id UUID,
    tax_year INT NOT NULL,
    filing_status VARCHAR(50) NOT NULL,
    modified_agi NUMERIC(18,2) NOT NULL,
    investment_income_breakdown JSONB NOT NULL,
    niit_threshold NUMERIC(18,2) NOT NULL,
    taxable_nii NUMERIC(18,2) NOT NULL,
    niit_tax NUMERIC(18,2) NOT NULL,
    mitigation_strategies JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_niit_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_niit_family ON niit_calculations(family_id);
CREATE INDEX idx_niit_year ON niit_calculations(tax_year DESC);

-- Charitable Bunching Analysis
CREATE TABLE IF NOT EXISTS charitable_bunching_analyses (
    analysis_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    member_id UUID,
    analysis_years INT NOT NULL,
    annual_giving NUMERIC(18,2) NOT NULL,
    baseline_scenario JSONB NOT NULL,
    bunching_scenario JSONB NOT NULL,
    estimated_tax_savings NUMERIC(18,2) NOT NULL,
    recommended_strategy VARCHAR(50), -- ANNUAL, BUNCHING_3YR
    daf_recommendation JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_bunching_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_bunching_family ON charitable_bunching_analyses(family_id);

-- ==============================================================================
-- MULTI-GENERATIONAL PLANNING
-- ==============================================================================

-- Dynasty Trust Simulations
CREATE TABLE IF NOT EXISTS dynasty_trust_simulations (
    simulation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    trust_name VARCHAR(255) NOT NULL,
    initial_funding NUMERIC(18,2) NOT NULL,
    assumed_growth_rate NUMERIC(5,4) NOT NULL,
    assumed_tax_rate NUMERIC(5,4) NOT NULL,
    generation_count INT NOT NULL,
    years_per_generation INT NOT NULL,
    generations JSONB NOT NULL, -- Array of generation projections
    total_tax_savings NUMERIC(18,2) NOT NULL,
    wealth_multiplier NUMERIC(10,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_dynasty_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_dynasty_family ON dynasty_trust_simulations(family_id);
CREATE INDEX idx_dynasty_created ON dynasty_trust_simulations(created_at DESC);

-- 529 Education Plans
CREATE TABLE IF NOT EXISTS education_529_plans (
    plan_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    student_member_id UUID,
    student_name VARCHAR(255),
    student_age INT NOT NULL,
    years_until_college INT NOT NULL,
    target_funding NUMERIC(18,2) NOT NULL,
    current_savings NUMERIC(18,2) NOT NULL,
    monthly_contribution NUMERIC(18,2) NOT NULL,
    projected_value NUMERIC(18,2) NOT NULL,
    overfunded BOOLEAN DEFAULT FALSE,
    gap NUMERIC(18,2),
    state_tax_benefit NUMERIC(18,2),
    recommended_state VARCHAR(2),
    total_tax_benefit_lifetime NUMERIC(18,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_529_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_529_family ON education_529_plans(family_id);
CREATE INDEX idx_529_student ON education_529_plans(student_member_id);

-- Legacy Impact Calculations
CREATE TABLE IF NOT EXISTS legacy_impact_calculations (
    impact_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    philanthropic_focus VARCHAR(100),
    annual_giving NUMERIC(18,2) NOT NULL,
    total_projected_giving NUMERIC(18,2) NOT NULL,
    generations_impacted INT,
    direct_beneficiaries_est INT,
    indirect_beneficiaries_est INT,
    legacy_rating VARCHAR(50), -- MODEST, SIGNIFICANT, TRANSFORMATIVE, GENERATIONAL
    recommended_structures JSONB,
    estimated_tax_deductions NUMERIC(18,2),
    net_cost_after_tax NUMERIC(18,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_legacy_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_legacy_family ON legacy_impact_calculations(family_id);

-- ==============================================================================
-- ALTERNATIVE INVESTMENTS
-- ==============================================================================

-- Private Equity Investments
CREATE TABLE IF NOT EXISTS private_equity_investments (
    investment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    fund_name VARCHAR(255) NOT NULL,
    general_partner VARCHAR(255),
    vintage_year INT,
    commitment_amount NUMERIC(18,2) NOT NULL,
    capital_called NUMERIC(18,2) DEFAULT 0,
    distributions NUMERIC(18,2) DEFAULT 0,
    current_nav NUMERIC(18,2) DEFAULT 0,
    irr NUMERIC(10,4), -- Internal Rate of Return
    moic NUMERIC(10,2), -- Multiple on Invested Capital
    dpi NUMERIC(10,2), -- Distributions to Paid-In
    rvpi NUMERIC(10,2), -- Residual Value to Paid-In
    tvpi NUMERIC(10,2), -- Total Value to Paid-In
    j_curve_phase VARCHAR(50), -- DRAWDOWN, TROUGH, RECOVERY, MATURE
    cash_flow_projection JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_pe_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_pe_family ON private_equity_investments(family_id);
CREATE INDEX idx_pe_vintage ON private_equity_investments(vintage_year DESC);

-- Venture Capital Investments
CREATE TABLE IF NOT EXISTS venture_capital_investments (
    investment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    company_name VARCHAR(255) NOT NULL,
    round VARCHAR(50), -- SEED, SERIES_A, SERIES_B, etc.
    investment_date DATE NOT NULL,
    initial_investment NUMERIC(18,2) NOT NULL,
    shares_owned BIGINT,
    current_valuation NUMERIC(18,2),
    ownership_pct NUMERIC(5,4),
    exit_scenarios JSONB, -- Array of exit scenarios
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_vc_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_vc_family ON venture_capital_investments(family_id);
CREATE INDEX idx_vc_company ON venture_capital_investments(company_name);

-- Art & Collectibles
CREATE TABLE IF NOT EXISTS art_collectibles (
    asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    artist_name VARCHAR(255),
    artwork_title VARCHAR(500),
    medium VARCHAR(100),
    year_created INT,
    acquisition_date DATE NOT NULL,
    acquisition_price NUMERIC(18,2) NOT NULL,
    current_valuation NUMERIC(18,2) NOT NULL,
    valuation_date DATE NOT NULL,
    appraisal_firm VARCHAR(255),
    insurance_value NUMERIC(18,2),
    location VARCHAR(255),
    provenance TEXT,
    condition VARCHAR(50), -- EXCELLENT, GOOD, FAIR, POOR
    annual_appreciation NUMERIC(10,4), -- CAGR %
    fractional_ownership BOOLEAN DEFAULT FALSE,
    ownership_pct NUMERIC(5,4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_art_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_art_family ON art_collectibles(family_id);
CREATE INDEX idx_art_artist ON art_collectibles(artist_name);

-- Real Estate Syndications
CREATE TABLE IF NOT EXISTS real_estate_syndications (
    investment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    property_name VARCHAR(255) NOT NULL,
    property_type VARCHAR(100), -- MULTIFAMILY, OFFICE, RETAIL, INDUSTRIAL
    location VARCHAR(255),
    sponsor VARCHAR(255),
    initial_investment NUMERIC(18,2) NOT NULL,
    ownership_pct NUMERIC(5,4),
    current_value NUMERIC(18,2),
    annual_cash_flow NUMERIC(18,2),
    total_depreciation NUMERIC(18,2),
    k1_received BOOLEAN DEFAULT FALSE,
    k1_document_id VARCHAR(255),
    exchange_1031_eligible BOOLEAN DEFAULT FALSE,
    expected_exit_year INT,
    projected_irr NUMERIC(10,4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_re_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_re_family ON real_estate_syndications(family_id);
CREATE INDEX idx_re_property ON real_estate_syndications(property_name);

-- ==============================================================================
-- AI INTELLIGENCE
-- ==============================================================================

-- Client Churn Predictions
CREATE TABLE IF NOT EXISTS churn_predictions (
    prediction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    family_name VARCHAR(255),
    churn_risk VARCHAR(50) NOT NULL, -- LOW, MEDIUM, HIGH, CRITICAL
    churn_probability NUMERIC(5,4) NOT NULL,
    risk_score NUMERIC(5,2) NOT NULL, -- 0-100
    risk_factors JSONB NOT NULL,
    protective_factors JSONB,
    recommended_actions JSONB,
    at_risk_revenue NUMERIC(18,2),
    prediction_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_churn_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_churn_family ON churn_predictions(family_id);
CREATE INDEX idx_churn_risk ON churn_predictions(churn_risk, risk_score DESC);
CREATE INDEX idx_churn_date ON churn_predictions(prediction_date DESC);

-- Meeting Preparations
CREATE TABLE IF NOT EXISTS meeting_preparations (
    prep_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    meeting_date DATE NOT NULL,
    meeting_type VARCHAR(50), -- QUARTERLY_REVIEW, ANNUAL_PLANNING, AD_HOC
    key_topics JSONB,
    talking_points JSONB,
    risk_alerts JSONB,
    opportunities JSONB,
    action_items JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_prep_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_prep_family ON meeting_preparations(family_id);
CREATE INDEX idx_prep_date ON meeting_preparations(meeting_date DESC);

-- ==============================================================================
-- ESG INTELLIGENCE
-- ==============================================================================

-- Carbon Footprint Calculations
CREATE TABLE IF NOT EXISTS carbon_footprint_calculations (
    calculation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    portfolio_value NUMERIC(18,2) NOT NULL,
    total_carbon_emissions NUMERIC(18,2) NOT NULL, -- Metric tons CO2e
    carbon_intensity NUMERIC(18,2) NOT NULL, -- Tons CO2e per $1M
    scope_breakdown JSONB NOT NULL, -- Scope 1, 2, 3
    asset_class_breakdown JSONB,
    highest_emitters JSONB,
    benchmark_comparison NUMERIC(10,2), -- % vs S&P 500
    reduction_opportunities JSONB,
    net_zero_alignment VARCHAR(50), -- ALIGNED, ON_TRACK, OFF_TRACK
    calculation_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_carbon_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_carbon_family ON carbon_footprint_calculations(family_id);
CREATE INDEX idx_carbon_date ON carbon_footprint_calculations(calculation_date DESC);

-- ESG Portfolio Scores
CREATE TABLE IF NOT EXISTS esg_portfolio_scores (
    score_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    portfolio_value NUMERIC(18,2) NOT NULL,
    overall_esg_score NUMERIC(5,2) NOT NULL, -- 0-100
    environmental_score NUMERIC(5,2),
    social_score NUMERIC(5,2),
    governance_score NUMERIC(5,2),
    msci_esg_rating VARCHAR(10), -- AAA, AA, A, BBB, BB, B, CCC
    sustainalytics_rating NUMERIC(5,2),
    holdings_breakdown JSONB,
    controversy_exposure JSONB,
    sdg_alignment JSONB, -- UN SDG goals
    impact_metrics JSONB,
    score_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_esg_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_esg_family ON esg_portfolio_scores(family_id);
CREATE INDEX idx_esg_rating ON esg_portfolio_scores(msci_esg_rating);
CREATE INDEX idx_esg_date ON esg_portfolio_scores(score_date DESC);

-- Impact Investments
CREATE TABLE IF NOT EXISTS impact_investments (
    investment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL,
    investment_name VARCHAR(255) NOT NULL,
    investment_amount NUMERIC(18,2) NOT NULL,
    impact_theme VARCHAR(100), -- CLEAN_ENERGY, EDUCATION, HEALTHCARE, etc.
    sdg_targets JSONB,
    impact_metrics JSONB, -- Quantified impact measurements
    financial_return NUMERIC(10,4),
    impact_return NUMERIC(10,2), -- SROI (Social Return on Investment)
    impact_verification VARCHAR(50), -- THIRD_PARTY, SELF_REPORTED
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_impact_family FOREIGN KEY (family_id) 
        REFERENCES family_offices(family_id) ON DELETE CASCADE
);

CREATE INDEX idx_impact_family ON impact_investments(family_id);
CREATE INDEX idx_impact_theme ON impact_investments(impact_theme);

-- ==============================================================================
-- AUDIT & TRACKING
-- ==============================================================================

COMMENT ON TABLE tax_strategies IS 'Tax optimization strategies and recommendations';
COMMENT ON TABLE state_residency_comparisons IS 'Multi-state tax residency analysis';
COMMENT ON TABLE niit_calculations IS 'Net Investment Income Tax (3.8%) calculations';
COMMENT ON TABLE charitable_bunching_analyses IS 'Charitable giving bunching vs annual strategies';
COMMENT ON TABLE dynasty_trust_simulations IS 'Multi-generational wealth projections';
COMMENT ON TABLE education_529_plans IS '529 college savings plan optimizations';
COMMENT ON TABLE legacy_impact_calculations IS 'Philanthropic legacy impact modeling';
COMMENT ON TABLE private_equity_investments IS 'PE fund investments and performance metrics';
COMMENT ON TABLE venture_capital_investments IS 'VC startup investments and exit scenarios';
COMMENT ON TABLE art_collectibles IS 'Art and collectible asset tracking';
COMMENT ON TABLE real_estate_syndications IS 'RE syndication investments and 1031 exchanges';
COMMENT ON TABLE churn_predictions IS 'AI-powered client churn risk predictions';
COMMENT ON TABLE meeting_preparations IS 'AI-generated meeting preparation materials';
COMMENT ON TABLE carbon_footprint_calculations IS 'Portfolio carbon emissions analysis';
COMMENT ON TABLE esg_portfolio_scores IS 'ESG ratings and scores aggregation';
COMMENT ON TABLE impact_investments IS 'Impact investing tracking and SROI';
