-- Migration: Extend Fee Billing for Alternative Investments
-- Description: Add support for performance-based fees (2/20 structures, hurdle rates, etc.)
-- Author: Semlayer Platform
-- Date: 2025-11-27

-- ============================================================================
-- EXTEND FEE_SCHEDULES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS fee_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add column to indicate if schedule applies to alternatives
ALTER TABLE fee_schedules 
    ADD COLUMN IF NOT EXISTS applies_to_alternatives BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS alternative_fee_structure_id UUID;

-- ============================================================================
-- ALTERNATIVE FEE STRUCTURES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS alternative_fee_structures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fee_schedule_id UUID NOT NULL,
    
    -- Management fee (on committed capital or NAV)
    management_fee_pct DECIMAL(5,4) NOT NULL CHECK (management_fee_pct >= 0 AND management_fee_pct <= 1),
    management_fee_basis VARCHAR(50) DEFAULT 'COMMITTED_CAPITAL' CHECK (management_fee_basis IN (
        'COMMITTED_CAPITAL',
        'INVESTED_CAPITAL',
        'NAV',
        'DECLINING_BALANCE'
    )),
    
    -- Performance fee (carried interest)
    performance_fee_pct DECIMAL(5,4) DEFAULT 0.20 CHECK (performance_fee_pct >= 0 AND performance_fee_pct <= 1),
    hurdle_rate_pct DECIMAL(5,4) DEFAULT 0.08 CHECK (hurdle_rate_pct >= 0),
    hurdle_type VARCHAR(50) DEFAULT 'PREFERRED_RETURN' CHECK (hurdle_type IN (
        'PREFERRED_RETURN',
        'IRR_BASED',
        'ABSOLUTE'
    )),
    
    -- Fee structure variations
    has_catch_up BOOLEAN DEFAULT TRUE,
    catch_up_rate DECIMAL(5,4), -- e.g., 0.80 = GP gets 80% of profits above hurdle until caught up
    has_high_water_mark BOOLEAN DEFAULT FALSE,
    
    -- Fee calculation frequency
    management_fee_frequency VARCHAR(20) DEFAULT 'QUARTERLY' CHECK (management_fee_frequency IN (
        'MONTHLY',
        'QUARTERLY', 
        'SEMI_ANNUALLY',
        'ANNUALLY'
    )),
    performance_fee_timing VARCHAR(50) DEFAULT 'ON_DISTRIBUTION' CHECK (performance_fee_timing IN (
        'ON_DISTRIBUTION',
        'ANNUALLY',
        'AT_EXIT',
        'QUARTERLY'
    )),
    
    -- Clawback provisions
    has_clawback BOOLEAN DEFAULT TRUE,
    clawback_period_years INTEGER DEFAULT 10,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_alt_fee_structures_schedule ON alternative_fee_structures(fee_schedule_id);

-- ============================================================================
-- PERFORMANCE FEE CALCULATIONS
-- ============================================================================

CREATE TABLE IF NOT EXISTS alternative_performance_fee_calculations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(investment_id) ON DELETE CASCADE,
    fee_structure_id UUID NOT NULL REFERENCES alternative_fee_structures(id),
    
    -- Calculation period
    calculation_date DATE NOT NULL,
    period_start_date DATE NOT NULL,
    period_end_date DATE NOT NULL,
    
    -- Inputs
    total_distributions DECIMAL(15,2) NOT NULL,
    total_called DECIMAL(15,2) NOT NULL,
    current_nav DECIMAL(15,2) NOT NULL,
    total_value DECIMAL(15,2) NOT NULL, -- distributions + NAV
    cumulative_irr DECIMAL(8,5),
    
    -- Hurdle calculation
    hurdle_achieved BOOLEAN DEFAULT FALSE,
    hurdle_amount DECIMAL(15,2),
    excess_over_hurdle DECIMAL(15,2),
    
    -- Performance fee calculation
    performance_fee_before_catch_up DECIMAL(15,2),
    catch_up_amount DECIMAL(15,2),
    performance_fee_after_catch_up DECIMAL(15,2),
    final_performance_fee DECIMAL(15,2),
    
    -- High water mark tracking
    previous_high_water_mark DECIMAL(15,2),
    new_high_water_mark DECIMAL(15,2),
    
    -- Status
    status VARCHAR(50) DEFAULT 'CALCULATED' CHECK (status IN (
        'CALCULATED',
        'APPROVED',
        'PAID',
        'DISPUTED'
    )),
    approved_at TIMESTAMPTZ,
    approved_by UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_perf_fee_calc_investment ON alternative_performance_fee_calculations(investment_id, calculation_date DESC);
CREATE INDEX IF NOT EXISTS idx_perf_fee_calc_status ON alternative_performance_fee_calculations(status);

-- ============================================================================
-- MANAGEMENT FEE CALCULATIONS
-- ============================================================================

CREATE TABLE IF NOT EXISTS alternative_management_fee_calculations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(investment_id) ON DELETE CASCADE,
    fee_structure_id UUID NOT NULL REFERENCES alternative_fee_structures(id),
    
    -- Calculation period
    calculation_date DATE NOT NULL,
    period_start_date DATE NOT NULL,
    period_end_date DATE NOT NULL,
    
    -- Fee calculation inputs
    fee_basis_type VARCHAR(50) NOT NULL,
    committed_capital DECIMAL(15,2),
    invested_capital DECIMAL(15,2),
    average_nav DECIMAL(15,2),
    
    -- Calculation
    fee_basis_amount DECIMAL(15,2) NOT NULL,
    management_fee_rate DECIMAL(5,4) NOT NULL,
    calculated_fee DECIMAL(15,2) NOT NULL,
    
    -- Adjustments
    adjustment_amount DECIMAL(15,2) DEFAULT 0,
    adjustment_reason TEXT,
    final_fee_amount DECIMAL(15,2) NOT NULL,
    
    -- Status
    status VARCHAR(50) DEFAULT 'CALCULATED' CHECK (status IN (
        'CALCULATED',
        'APPROVED',
        'INVOICED',
        'PAID'
    )),
    approved_at TIMESTAMPTZ,
    approved_by UUID,
    invoice_id UUID,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_mgmt_fee_calc_investment ON alternative_management_fee_calculations(investment_id, calculation_date DESC);
CREATE INDEX idx_mgmt_fee_calc_status ON alternative_management_fee_calculations(status);

-- ============================================================================
-- TRIGGER: Update updated_at timestamp
-- ============================================================================

CREATE TRIGGER update_alternative_fee_structures_updated_at
    BEFORE UPDATE ON alternative_fee_structures
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE alternative_fee_structures IS 
'Fee structures for alternative investments supporting:
- 2/20 fee structure (2% management fee, 20% performance fee)
- Preferred return hurdles (typically 8%)
- Catch-up provisions (GP catches up to 20% after LP gets hurdle)
- High water marks (ensure fees only on new profits)
- Clawback provisions (GP returns fees if underperformance)';

COMMENT ON COLUMN alternative_fee_structures.catch_up_rate IS 
'Percentage of profits above hurdle that go to GP during catch-up phase. 
Typically 0.80 (80%) meaning LP gets hurdle + 20% until GP catches up to their 20% share';

COMMENT ON TABLE alternative_performance_fee_calculations IS 
'Tracks performance fee (carried interest) calculations for alternative investments.
Formula: If total_value > total_called * (1 + hurdle_rate), then:
  - LP gets total_called * hurdle_rate first
  - GP catches up via catch_up_rate
  - Then 80/20 split (or per agreement)';
