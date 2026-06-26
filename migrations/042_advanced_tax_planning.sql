-- Migration 042: Advanced Tax Planning & Optimization
-- Multi-state allocation, AMT, NIIT, opportunity zones, QBI deduction

-- =============================================================================
-- 1. CLIENT TAX PROFILE
-- =============================================================================

CREATE TABLE client_tax_profile (
    profile_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id) UNIQUE,
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Filing status
    filing_status VARCHAR(50), -- 'SINGLE', 'MARRIED_JOINT', 'MARRIED_SEPARATE', 'HEAD_OF_HOUSEHOLD'
    
    -- State residency
    primary_state VARCHAR(2), -- e.g., 'CA', 'NY'
    additional_states VARCHAR(2)[], -- States with income
    
    -- Federal brackets
    estimated_federal_rate DECIMAL(5,2), -- Current marginal rate
    estimated_amt_rate DECIMAL(5,2), -- AMT rate if applicable
    
    -- NIIT (Net Investment Income Tax)
    subject_to_niit BOOLEAN DEFAULT FALSE,
    magi_threshold DECIMAL(15,2), -- Modified AGI threshold
    
    -- QBI (Qualified Business Income)
    has_qbi BOOLEAN DEFAULT FALSE,
    qbi_eligible_income DECIMAL(15,2),
    
    -- Opportunity Zones
    has_oz_investments BOOLEAN DEFAULT FALSE,
    
    -- Tax year
    current_tax_year INTEGER DEFAULT EXTRACT(YEAR FROM CURRENT_DATE),
    
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- 2. MULTI-STATE TAX ALLOCATION
-- =============================================================================

CREATE TABLE state_tax_allocations (
    allocation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tax_year INTEGER NOT NULL,
    state VARCHAR(2) NOT NULL,
    
    -- Income allocation
    wages DECIMAL(15,2) DEFAULT 0,
    business_income DECIMAL(15,2) DEFAULT 0,
    rental_income DECIMAL(15,2) DEFAULT 0,
    investment_income DECIMAL(15,2) DEFAULT 0,
    k1_income DECIMAL(15,2) DEFAULT 0, -- From partnerships
    
    -- Deductions
    state_deductions DECIMAL(15,2) DEFAULT 0,
    
    -- Credits
    state_tax_credits DECIMAL(15,2) DEFAULT 0,
    
    -- Calculated tax
    estimated_state_tax DECIMAL(15,2),
    state_tax_rate DECIMAL(5,2),
    
    -- Withholding
    state_withholding DECIMAL(15,2) DEFAULT 0,
    estimated_payment DECIMAL(15,2) DEFAULT 0,
    
    -- Allocation method
    allocation_method VARCHAR(50), -- 'DAYS_METHOD', 'INCOME_SOURCE', 'APPORTIONMENT'
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_state_tax_client_year (client_id, tax_year),
    UNIQUE (client_id, tax_year, state)
);

-- =============================================================================
-- 3. AMT (ALTERNATIVE MINIMUM TAX) CALCULATIONS
-- =============================================================================

CREATE TABLE amt_calculations (
    calculation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tax_year INTEGER NOT NULL,
    
    -- Regular tax
    regular_taxable_income DECIMAL(15,2),
    regular_tax DECIMAL(15,2),
    
    -- AMT adjustments
    state_tax_add_back DECIMAL(15,2), -- State taxes not deductible for AMT
    depreciation_adjustment DECIMAL(15,2),
    incentive_stock_options DECIMAL(15,2),
    private_activity_bond_interest DECIMAL(15,2),
    other_adjustments DECIMAL(15,2),
    
    -- AMT calculation
    amt_income DECIMAL(15,2),
    amt_exemption DECIMAL(15,2),
    amt_taxable_income DECIMAL(15,2),
    tentative_minimum_tax DECIMAL(15,2),
    
    -- Final
    amt_owed DECIMAL(15,2), -- Excess of TMT over regular tax
    is_subject_to_amt BOOLEAN DEFAULT FALSE,
    
    -- Planning opportunities
    amt_credit_carryforward DECIMAL(15,2),
    
    calculated_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_amt_client_year (client_id, tax_year)
);

-- =============================================================================
-- 4. NIIT (NET INVESTMENT INCOME TAX) TRACKING
-- =============================================================================

CREATE TABLE niit_calculations (
    calculation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tax_year INTEGER NOT NULL,
    
    -- Net investment income components
    taxable_interest DECIMAL(15,2) DEFAULT 0,
    ordinary_dividends DECIMAL(15,2) DEFAULT 0,
    capital_gains DECIMAL(15,2) DEFAULT 0,
    rental_income DECIMAL(15,2) DEFAULT 0,
    passive_k1_income DECIMAL(15,2) DEFAULT 0,
    
    -- Less: investment expenses
    investment_expenses DECIMAL(15,2) DEFAULT 0,
    
    -- Net investment income
    net_investment_income DECIMAL(15,2),
    
    -- MAGI calculation
    adjusted_gross_income DECIMAL(15,2),
    magi DECIMAL(15,2),
    magi_threshold DECIMAL(15,2), -- $200K single, $250K married
    magi_excess DECIMAL(15,2),
    
    -- NIIT calculation
    niit_base DECIMAL(15,2), -- Lesser of NII or MAGI excess
    niit_owed DECIMAL(15,2), -- 3.8% of base
    
    calculated_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_niit_client_year (client_id, tax_year)
);

-- =============================================================================
-- 5. OPPORTUNITY ZONE INVESTMENTS
-- =============================================================================

CREATE TABLE opportunity_zone_investments (
    oz_investment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    
    -- Investment details
    qof_name TEXT, -- Qualified Opportunity Fund
    investment_date DATE NOT NULL,
    original_investment DECIMAL(15,2) NOT NULL,
    
    -- Gain deferral
    deferred_gain DECIMAL(15,2),
    gain_recognition_date DATE, -- December 31, 2026
    
    -- Basis adjustments
    basis_increase_5yr DECIMAL(15,2), -- 10% basis increase after 5 years
    basis_increase_7yr DECIMAL(15,2), -- Additional 5% after 7 years
    
    -- 10-year exclusion
    ten_year_holding_date DATE,
    eligible_for_exclusion BOOLEAN DEFAULT FALSE,
    
    -- Current value
    current_value DECIMAL(15,2),
    unrealized_gain DECIMAL(15,2),
    
    -- OZ details
    opportunity_zone_tract VARCHAR(20),
    state VARCHAR(2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_oz_client (client_id)
);

-- =============================================================================
-- 6. QBI (QUALIFIED BUSINESS INCOME) DEDUCTION
-- =============================================================================

CREATE TABLE qbi_calculations (
    calculation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tax_year INTEGER NOT NULL,
    
    -- QBI sources
    business_name TEXT,
    qbi_amount DECIMAL(15,2),
    w2_wages DECIMAL(15,2),
    ubia_property DECIMAL(15,2), -- Unadjusted basis of qualified property
    
    -- Business type
    is_sstb BOOLEAN DEFAULT FALSE, -- Specified Service Trade or Business
    
    -- Limitations
    taxable_income DECIMAL(15,2),
    qbi_limit_percentage DECIMAL(5,2) DEFAULT 20.00, -- 20% of QBI
    w2_wage_limit DECIMAL(15,2), -- 50% of W-2 wages
    property_limit DECIMAL(15,2), -- 25% W-2 + 2.5% UBIA
    
    -- Calculated deduction
   qbi_deduction DECIMAL(15,2),
    
    calculated_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_qbi_client_year (client_id, tax_year)
);

-- =============================================================================
-- 7. TAX OPTIMIZATION RECOMMENDATIONS
-- =============================================================================

CREATE TABLE tax_optimization_recommendations (
    recommendation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tax_year INTEGER NOT NULL,
    
    -- Recommendation details
    optimization_type VARCHAR(50), -- 'HARVEST_LOSS', 'DEFER_INCOME', 'ACCELERATE_DEDUCTION', etc.
    title TEXT NOT NULL,
    description TEXT,
    
    -- Potential savings
    estimated_savings DECIMAL(15,2),
    confidence_level VARCHAR(20), -- 'HIGH', 'MEDIUM', 'LOW'
    
    -- Action required
    action_deadline DATE,
    action_description TEXT,
    
    -- Status
    recommendation_status VARCHAR(20) DEFAULT 'PENDING',
    implemented_date DATE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_tax_recs_client (client_id, recommendation_status)
);

-- =============================================================================
-- 8. HELPER FUNCTIONS
-- =============================================================================

-- Calculate state tax allocation
CREATE OR REPLACE FUNCTION calculate_state_tax_allocation(
    p_client_id UUID,
    p_tax_year INTEGER,
    p_state VARCHAR(2)
) RETURNS DECIMAL AS $$
DECLARE
    v_total_income DECIMAL;
    v_state_rate DECIMAL;
    v_state_tax DECIMAL;
BEGIN
    -- Get total state income
    SELECT 
        COALESCE(wages, 0) + COALESCE(business_income, 0) + 
        COALESCE(rental_income, 0) + COALESCE(investment_income, 0) +
        COALESCE(k1_income, 0) - COALESCE(state_deductions, 0)
    INTO v_total_income
    FROM state_tax_allocations
    WHERE client_id = p_client_id
    AND tax_year = p_tax_year
    AND state = p_state;
    
    -- Get state tax rate (simplified - would use brackets in production)
    v_state_rate := CASE p_state
        WHEN 'CA' THEN 0.133 -- California top rate
        WHEN 'NY' THEN 0.109 -- New York top rate
        WHEN 'TX' THEN 0.000 -- Texas (no income tax)
        ELSE 0.05 -- Default estimate
    END;
    
    v_state_tax := v_total_income * v_state_rate;
    
    RETURN v_state_tax;
END;
$$ LANGUAGE plpgsql;

-- Check if subject to AMT
CREATE OR REPLACE FUNCTION check_amt_exposure(p_client_id UUID, p_tax_year INTEGER)
RETURNS BOOLEAN AS $$
DECLARE
    v_state_tax DECIMAL;
    v_iso_exercise DECIMAL;
    v_likely_amt BOOLEAN;
BEGIN
    -- High state taxes and ISO exercises are common AMT triggers
    SELECT COALESCE(SUM(estimated_state_tax), 0)
    INTO v_state_tax
    FROM state_tax_allocations
    WHERE client_id = p_client_id
    AND tax_year = p_tax_year;
    
    -- Check for ISO exercises
    SELECT COALESCE(incentive_stock_options, 0)
    INTO v_iso_exercise
    FROM amt_calculations
    WHERE client_id = p_client_id
    AND tax_year = p_tax_year;
    
    v_likely_amt := (v_state_tax > 50000 OR v_iso_exercise > 100000);
    
    RETURN v_likely_amt;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 9. RLS POLICIES
-- =============================================================================

ALTER TABLE client_tax_profile ENABLE ROW LEVEL SECURITY;
ALTER TABLE state_tax_allocations ENABLE ROW LEVEL SECURITY;
ALTER TABLE tax_optimization_recommendations ENABLE ROW LEVEL SECURITY;

CREATE POLICY tax_profile_tenant_isolation ON client_tax_profile
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 10. COMMENTS
-- =============================================================================

COMMENT ON TABLE client_tax_profile IS 'Client tax situation with filing status, state residency, and tax attributes';
COMMENT ON TABLE state_tax_allocations IS 'Multi-state income allocation for clients with income in multiple states';
COMMENT ON TABLE amt_calculations IS 'Alternative Minimum Tax calculations with adjustments and exemptions';
COMMENT ON TABLE niit_calculations IS '3.8% Net Investment Income Tax for high earners';
COMMENT ON TABLE opportunity_zone_investments IS 'Qualified Opportunity Zone investments with basis tracking and gain deferral';
COMMENT ON TABLE qbi_calculations IS 'Qualified Business Income 20% deduction for pass-through entities';
COMMENT ON FUNCTION calculate_state_tax_allocation IS 'Calculate state tax liability based on income allocation';
COMMENT ON FUNCTION check_amt_exposure IS 'Determine if client is likely subject to Alternative Minimum Tax';
