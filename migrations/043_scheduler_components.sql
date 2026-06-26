-- Migration 043: Scheduler Components
-- Settlement calendar, regulatory deadlines, and STP

-- =============================================================================
-- 1. SETTLEMENT CALENDAR
-- =============================================================================

CREATE TYPE settlement_cycle AS ENUM ('T_PLUS_0', 'T_PLUS_1', 'T_PLUS_2', 'T_PLUS_3');

CREATE TABLE custodian_settlement_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    custodian_name VARCHAR(100) NOT NULL,
    
    -- Asset class rules
    equities_settlement settlement_cycle DEFAULT 'T_PLUS_1',
    bonds_settlement settlement_cycle DEFAULT 'T_PLUS_2',
    mutual_funds_settlement settlement_cycle DEFAULT 'T_PLUS_1',
    options_settlement settlement_cycle DEFAULT 'T_PLUS_1',
    fx_spot_settlement settlement_cycle DEFAULT 'T_PLUS_2',
    fx_forward_settlement settlement_cycle DEFAULT 'T_PLUS_2',
    
    -- Special rules
    same_day_settlement_cutoff TIME, -- e.g., '14:00:00' for 2 PM cutoff
    requires_prefunding BOOLEAN DEFAULT FALSE,
    
    -- Calendar
    calendar_code VARCHAR(50), -- References business_calendars
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_settlement_custodian (custodian_name)
);

CREATE TABLE settlement_instructions (
    instruction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trade_id UUID NOT NULL, -- Reference to trade
    client_id UUID NOT NULL REFERENCES clients(client_id),
    
    -- Trade details
    trade_date DATE NOT NULL,
    security_type VARCHAR(50),
    quantity DECIMAL(15,2),
    price DECIMAL(15,6),
    
    -- Settlement calculation
    settlement_cycle settlement_cycle NOT NULL,
    business_days_tosettle INTEGER,
    expected_settlement_date DATE NOT NULL,
    actual_settlement_date DATE,
    
    -- Status
    settlement_status VARCHAR(20) DEFAULT 'PENDING', -- 'PENDING', 'SETTLED', 'FAILED', 'CANCELLED'
    
    -- Fails tracking
    fail_reason TEXT,
    fail_date DATE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_settlement_status (settlement_status, expected_settlement_date),
    INDEX idx_settlement_client (client_id, trade_date DESC)
);

-- =============================================================================
-- 2. REGULATORY DEADLINES
-- =============================================================================

CREATE TYPE deadline_category AS ENUM (
    'FORM_FILING',
    'RMD',
    'ESTIMATED_TAX',
    'CONTRIBUTION_DEADLINE',
    'COMPLIANCE_REPORT',
    'AUDIT_REQUIREMENT'
);

CREATE TABLE regulatory_deadlines (
    deadline_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Deadline details
    deadline_name TEXT NOT NULL,
    deadline_category deadline_category NOT NULL,
    deadline_date DATE NOT NULL,
    
    -- Recurrence
    is_recurring BOOLEAN DEFAULT FALSE,
    recurrence_rule TEXT, -- e.g., 'YEARLY:MONTH=3:DAY=31' for Form ADV
    
    -- Penalty information
    has_penalty BOOLEAN DEFAULT FALSE,
    penalty_rate DECIMAL(5,2), -- Percentage
    penalty_description TEXT,
    
    -- Notification settings
    alert_days_before INTEGER[] DEFAULT ARRAY[30, 14, 7, 1],
    
    -- Status
    completion_required BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_deadlines_date (deadline_date),
    INDEX idx_deadlines_category (deadline_category, deadline_date)
);

CREATE TABLE client_deadline_tracking (
    tracking_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deadline_id UUID NOT NULL REFERENCES regulatory_deadlines(deadline_id),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tax_year INTEGER,
    
    -- Applicability
    applies_to_client BOOLEAN DEFAULT TRUE,
    exemption_reason TEXT,
    
    -- Completion status
    completed BOOLEAN DEFAULT FALSE,
    completed_date DATE,
    completed_by UUID REFERENCES users(user_id),
    
    -- Alerts
    last_alert_sent TIMESTAMPTZ,
    alert_count INTEGER DEFAULT 0,
    
    INDEX idx_client_deadlines (client_id, completed),
    INDEX idx_deadline_tracking (deadline_id, completed)
);

-- =============================================================================
-- 3. RMD (REQUIRED MINIMUM DISTRIBUTION) TRACKING
-- =============================================================================

CREATE TABLE rmd_calculations (
    rmd_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    account_id UUID, -- Retirement account
    tax_year INTEGER NOT NULL,
    
    -- Client details
    date_of_birth DATE,
    age_on_dec_31 INTEGER,
    
    -- Account balance
    account_balance_prior_year_end DECIMAL(15,2),
    
    --  Life expectancy factor
    life_expectancy_factor DECIMAL(5,2),
    
    -- RMD calculation
    required_minimum_distribution DECIMAL(15,2),
    
    -- Distribution tracking
    total_distributed_ytd DECIMAL(15,2) DEFAULT 0,
    remaining_required DECIMAL(15,2),
    
    -- Deadline
    rmd_deadline DATE, -- December 31 (or April 1 for first year)
    
    -- Status
    rmd_satisfied BOOLEAN DEFAULT FALSE,
    satisfied_date DATE,
    
    -- Penalty if not taken
    potential_penalty DECIMAL(15,2), -- 50% of shortfall
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_rmd_client_year (client_id, tax_year),
    INDEX idx_rmd_unsatisfied (rmd_satisfied, rmd_deadline) WHERE rmd_satisfied = FALSE
);

-- =============================================================================
-- 4. ESTIMATED TAX PAYMENT TRACKING
-- =============================================================================

CREATE TABLE estimated_tax_payments (
    payment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tax_year INTEGER NOT NULL,
    quarter INTEGER CHECK (quarter BETWEEN 1 AND 4),
    
    -- Payment details
    estimated_payment_amount DECIMAL(15,2),
    payment_due_date DATE,
    
    -- Actual payment
    actual_payment_amount DECIMAL(15,2),
    payment_date DATE,
    payment_method VARCHAR(50),
    confirmation_number TEXT,
    
    -- Status
    payment_status VARCHAR(20) DEFAULT 'PENDING',
    
    -- Underpayment penalty
    safe_harbor_met BOOLEAN,
    potential_penalty DECIMAL(10,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_est_tax_client (client_id, tax_year, quarter)
);

-- =============================================================================
-- 5. HELPER FUNCTIONS
-- =============================================================================

-- Calculate settlement date
CREATE OR REPLACE FUNCTION calculate_settlement_date(
    p_trade_date DATE,
    p_security_type VARCHAR,
    p_custodian VARCHAR,
    p_calendar_code VARCHAR
) RETURNS DATE AS $$
DECLARE
    v_settlement_cycle settlement_cycle;
    v_business_days INTEGER;
    v_settlement_date DATE;
BEGIN
    -- Get settlement cycle for security type and custodian
    SELECT 
        CASE p_security_type
            WHEN 'EQUITY' THEN equities_settlement
            WHEN 'BOND' THEN bonds_settlement
            WHEN 'MUTUAL_FUND' THEN mutual_funds_settlement
            ELSE 'T_PLUS_2'
        END
    INTO v_settlement_cycle
    FROM custodian_settlement_rules
    WHERE custodian_name = p_custodian;
    
    -- Convert to business days
    v_business_days := CASE v_settlement_cycle
        WHEN 'T_PLUS_0' THEN 0
        WHEN 'T_PLUS_1' THEN 1
        WHEN 'T_PLUS_2' THEN 2
        WHEN 'T_PLUS_3' THEN 3
        ELSE 2
    END;
    
    -- Add business days (using calendar)
    v_settlement_date := p_trade_date + (v_business_days || ' days')::INTERVAL;
    
    -- TODO: Adjust for holidays using business calendar
    -- Would call add_business_days() from calendar module
    
    RETURN v_settlement_date;
END;
$$ LANGUAGE plpgsql;

-- Calculate RMD
CREATE OR REPLACE FUNCTION calculate_rmd(
    p_client_id UUID,
    p_tax_year INTEGER
) RETURNS DECIMAL AS $$
DECLARE
    v_age INTEGER;
    v_balance DECIMAL;
    v_life_expectancy DECIMAL;
    v_rmd DECIMAL;
BEGIN
    -- Get client age on December 31 of prior year
    SELECT EXTRACT(YEAR FROM AGE(
        (p_tax_year - 1 || '-12-31')::DATE,
        date_of_birth
    ))::INTEGER
    INTO v_age
    FROM clients
    WHERE client_id = p_client_id;
    
    -- Get account balance from prior year end
    -- (This is simplified - would sum across retirement accounts)
    v_balance := 1000000; -- Placeholder
    
    -- Get life expectancy factor from IRS Uniform Lifetime Table
    v_life_expectancy := CASE
        WHEN v_age < 72 THEN 27.4
        WHEN v_age = 72 THEN 27.4
        WHEN v_age = 73 THEN 26.5
        WHEN v_age = 74 THEN 25.5
        WHEN v_age = 75 THEN 24.6
        WHEN v_age >= 80 THEN 20.0
        ELSE 25.0
    END;
    
    v_rmd := v_balance / v_life_expectancy;
    
    RETURN v_rmd;
END;
$$ LANGUAGE plpgsql;

-- Get upcoming deadlines
CREATE OR REPLACE FUNCTION get_upcoming_deadlines(p_days_ahead INTEGER DEFAULT 30)
RETURNS TABLE (
    deadline_name TEXT,
    deadline_date DATE,
    deadline_category deadline_category,
    days_until_deadline INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        rd.deadline_name,
        rd.deadline_date,
        rd.deadline_category,
        (rd.deadline_date - CURRENT_DATE)::INTEGER AS days_until_deadline
    FROM regulatory_deadlines rd
    WHERE rd.deadline_date >= CURRENT_DATE
    AND rd.deadline_date <= CURRENT_DATE + (p_days_ahead || ' days')::INTERVAL
    ORDER BY rd.deadline_date;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 6. RLS POLICIES
-- =============================================================================

ALTER TABLE settlement_instructions ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_deadline_tracking ENABLE ROW LEVEL SECURITY;
ALTER TABLE rmd_calculations ENABLE ROW LEVEL SECURITY;

-- =============================================================================
-- 7. SEED DATA
-- =============================================================================

-- Common regulatory deadlines
INSERT INTO regulatory_deadlines (tenant_id, deadline_name, deadline_category, deadline_date, is_recurring, recurrence_rule) VALUES
    (gen_random_uuid(), 'Form ADV Annual Amendment', 'FORM_FILING', '2025-03-31', TRUE, 'YEARLY:MONTH=3:DAY=31'),
    (gen_random_uuid(), 'Q1 Estimated Tax Payment', 'ESTIMATED_TAX', '2025-04-15', TRUE, 'YEARLY:MONTH=4:DAY=15'),
    (gen_random_uuid(), 'Q2 Estimated Tax Payment', 'ESTIMATED_TAX', '2025-06-15', TRUE, 'YEARLY:MONTH=6:DAY=15'),
    (gen_random_uuid(), 'Q3 Estimated Tax Payment', 'ESTIMATED_TAX', '2025-09-15', TRUE, 'YEARLY:MONTH=9:DAY=15'),
    (gen_random_uuid(), 'Q4 Estimated Tax Payment', 'ESTIMATED_TAX', '2026-01-15', TRUE, 'YEARLY:MONTH=1:DAY=15'),
    (gen_random_uuid(), 'RMD Deadline', 'RMD', '2025-12-31', TRUE, 'YEARLY:MONTH=12:DAY=31');

-- =============================================================================
-- 8. COMMENTS
-- =============================================================================

COMMENT ON TABLE custodian_settlement_rules IS 'Settlement cycles and rules by custodian and asset class';
COMMENT ON TABLE regulatory_deadlines IS 'Regulatory filing deadlines with penalty tracking and alerts';
COMMENT ON TABLE rmd_calculations IS 'Required Minimum Distribution calculations for retirement accounts';
COMMENT ON FUNCTION calculate_settlement_date IS 'Calculate settlement date based on trade date, security type, and business calendar';
COMMENT ON FUNCTION calculate_rmd IS 'Calculate Required Minimum Distribution using IRS Uniform Lifetime Table';
