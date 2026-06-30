-- Advanced Fee Billing & Revenue Management Schema
-- Phase 2: Fee Billing System

-- ===========================
-- FEE SCHEDULES TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS fee_schedules (
    schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_name TEXT NOT NULL,
    description TEXT,
    
    -- Fee type classification
    fee_type VARCHAR(50) NOT NULL CHECK (fee_type IN (
        'AUM_TIERED',        -- Tiered AUM-based fees (e.g., 1% on first $1M, 0.75% on next $4M)
        'AUM_FLAT',          -- Flat percentage on all AUM
        'PERFORMANCE',       -- Performance-based only (e.g., 20% over hurdle)
        'SUBSCRIPTION',      -- Fixed subscription fee
        'HYBRID',            -- Combination of AUM and performance
        'RETAINER',          -- Fixed retainer
        'HOURLY'             -- Hourly billing
    )),
    
    -- Tiered AUM structure (JSONB array)
    -- Example: [{"min": 0, "max": 1000000, "rate": 0.01}, {"min": 1000000, "max": 5000000, "rate": 0.0075}]
    tier_structure JSONB,
    
    -- Flat AUM rate (if fee_type = 'AUM_FLAT')
    flat_aum_rate DECIMAL(5,4),
    
    -- Performance fee structure
    performance_hurdle_rate DECIMAL(5,4),     -- e.g., 0.08 = 8% hurdle
    performance_fee_rate DECIMAL(5,4),        -- e.g., 0.20 = 20% of excess returns
    high_water_mark_enabled BOOLEAN DEFAULT TRUE,
    
    -- Billing frequency and timing
    billing_frequency VARCHAR(20) NOT NULL DEFAULT 'QUARTERLY' CHECK (billing_frequency IN (
        'MONTHLY',
        'QUARTERLY',
        'SEMI_ANNUAL',
        'ANNUAL'
    )),
    billing_advance_or_arrears VARCHAR(10) DEFAULT 'ADVANCE' CHECK (billing_advance_or_arrears IN (
        'ADVANCE',   -- Bill at start of period
        'ARREARS'    -- Bill at end of period
    )),
    
    -- Minimum fees
    minimum_quarterly_fee DECIMAL(10,2),
    minimum_annual_fee DECIMAL(10,2),
    
    -- Account-level customizations
    exclude_cash_from_aum BOOLEAN DEFAULT FALSE,
    exclude_alternatives_from_aum BOOLEAN DEFAULT FALSE,
    exclude_held_away_from_aum BOOLEAN DEFAULT TRUE,
    
    -- Billing day preferences
    billing_day_of_month INTEGER CHECK (billing_day_of_month BETWEEN 1 AND 28),
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    is_template BOOLEAN DEFAULT FALSE,  -- Can be used as template for new clients
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID
);

CREATE INDEX IF NOT EXISTS idx_fee_schedules_type ON fee_schedules(fee_type);
CREATE INDEX IF NOT EXISTS idx_fee_schedules_active ON fee_schedules(is_active);
CREATE INDEX IF NOT EXISTS idx_fee_schedules_template ON fee_schedules(is_template) WHERE is_template = TRUE;

-- ===========================
-- CLIENT FEE ASSIGNMENTS TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS client_fee_assignments (
    assignment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    account_id UUID,  -- NULL = applies to all client accounts, otherwise specific account
    schedule_id UUID NOT NULL REFERENCES fee_schedules(schedule_id),
    
    -- Effective date range
    effective_date DATE NOT NULL,
    end_date DATE,  -- NULL = ongoing
    
    -- Negotiated overrides/customizations
    custom_discount_pct DECIMAL(5,2),           -- e.g., 0.10 = 10% discount on all fees
    custom_minimum_fee DECIMAL(10,2),
    custom_tier_structure JSONB,                -- Override standard tier structure
    custom_performance_hurdle DECIMAL(5,4),
    
    -- Billing preferences
    invoice_contact_email TEXT,
    invoice_contact_name TEXT,
    payment_method VARCHAR(50) CHECK (payment_method IN (
        'DEBIT_FROM_ACCOUNT',
        'WIRE',
        'CHECK',
        'ACH',
        'CREDIT_CARD'
    )),
    debit_account_id UUID,  -- Which account to debit if payment_method = 'DEBIT_FROM_ACCOUNT'
    billing_day_of_month INTEGER CHECK (billing_day_of_month BETWEEN 1 AND 28),
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    
    CONSTRAINT valid_date_range CHECK (end_date IS NULL OR end_date >= effective_date),
    CONSTRAINT valid_discount CHECK (custom_discount_pct IS NULL OR (custom_discount_pct >= 0 AND custom_discount_pct <= 100))
);

CREATE INDEX IF NOT EXISTS idx_client_fee_assignments_client ON client_fee_assignments(client_id);
CREATE INDEX IF NOT EXISTS idx_client_fee_assignments_account ON client_fee_assignments(account_id);
CREATE INDEX IF NOT EXISTS idx_client_fee_assignments_schedule ON client_fee_assignments(schedule_id);
CREATE INDEX IF NOT EXISTS idx_client_fee_assignments_active ON client_fee_assignments(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_client_fee_assignments_effective ON client_fee_assignments(effective_date, end_date);

-- ===========================
-- FEE CALCULATIONS TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS fee_calculations (
    calculation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    assignment_id UUID REFERENCES client_fee_assignments(assignment_id),
    
    -- Billing period
    billing_period_start DATE NOT NULL,
    billing_period_end DATE NOT NULL,
    billing_frequency VARCHAR(20),
    
    -- AUM-based fee calculation
    average_daily_balance DECIMAL(15,2),       -- Average AUM for period
    beginning_balance DECIMAL(15,2),
    ending_balance DECIMAL(15,2),
    aum_based_fee DECIMAL(12,2) DEFAULT 0,
    aum_calculation_method VARCHAR(50) DEFAULT 'AVERAGE_DAILY' CHECK (aum_calculation_method IN (
        'AVERAGE_DAILY',
        'BEGINNING_BALANCE',
        'ENDING_BALANCE',
        'AVERAGE_BEGINNING_ENDING'
    )),
    
    -- Performance fee calculation
    portfolio_return_pct DECIMAL(7,4),         -- e.g., 0.1520 = 15.20%
    hurdle_return_pct DECIMAL(7,4),
    excess_return DECIMAL(15,2),               -- Dollar amount of excess return
    performance_fee DECIMAL(12,2) DEFAULT 0,
    high_water_mark DECIMAL(15,2),             -- Previous peak value
    current_high_water_mark DECIMAL(15,2),     -- Updated after this period
    
    -- Additional fees
    planning_fee DECIMAL(12,2) DEFAULT 0,
    hourly_fees DECIMAL(12,2) DEFAULT 0,
    other_fees DECIMAL(12,2) DEFAULT 0,
    
    -- Adjustments
    prior_period_adjustment DECIMAL(12,2) DEFAULT 0,  -- For advance billing corrections
    discount_amount DECIMAL(12,2) DEFAULT 0,
    minimum_fee_adjustment DECIMAL(12,2) DEFAULT 0,
    
    -- Total fee breakdown
    gross_fee DECIMAL(12,2),
    net_fee DECIMAL(12,2),                     -- After discounts and adjustments
    
    -- Tax
    taxable_amount DECIMAL(12,2),
    
    -- Status and workflow
    calculation_status VARCHAR(50) NOT NULL DEFAULT 'DRAFT' CHECK (calculation_status IN (
        'DRAFT',
        'PENDING_REVIEW',
        'APPROVED',
        'INVOICED',
        'PAID',
        'PARTIALLY_PAID',
        'WRITTEN_OFF'
    )),
    requires_manual_review BOOLEAN DEFAULT FALSE,
    review_notes TEXT,
    
    -- Approval tracking
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    
    -- Invoice reference
    invoice_id UUID,
    invoice_number TEXT,
    invoice_sent_at TIMESTAMPTZ,
    
    -- Payment tracking
    payment_received_at TIMESTAMPTZ,
    payment_amount DECIMAL(12,2),
    payment_method VARCHAR(50),
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    
    CONSTRAINT valid_billing_period CHECK (billing_period_end >= billing_period_start),
    CONSTRAINT valid_fees CHECK (net_fee >= 0)
);

CREATE INDEX IF NOT EXISTS idx_fee_calc_client ON fee_calculations(client_id);
CREATE INDEX IF NOT EXISTS idx_fee_calc_period ON fee_calculations(billing_period_start, billing_period_end);
CREATE INDEX IF NOT EXISTS idx_fee_calc_status ON fee_calculations(calculation_status);
CREATE INDEX IF NOT EXISTS idx_fee_calc_pending_review ON fee_calculations(calculation_status) 
    WHERE calculation_status = 'PENDING_REVIEW' OR requires_manual_review = TRUE;
CREATE INDEX IF NOT EXISTS idx_fee_calc_invoice ON fee_calculations(invoice_id) WHERE invoice_id IS NOT NULL;

-- ===========================
-- REVENUE RECOGNITION SCHEDULE TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS revenue_recognition_schedule (
    schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calculation_id UUID NOT NULL REFERENCES fee_calculations(calculation_id) ON DELETE CASCADE,
    
    -- Recognition details
    recognition_date DATE NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    
    -- Status
    recognized BOOLEAN DEFAULT FALSE,
    recognized_at TIMESTAMPTZ,
    
    -- Accounting system integration
    journal_entry_id UUID,
    ledger_account VARCHAR(100),
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT valid_recognition_amount CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_revenue_recognition_calc ON revenue_recognition_schedule(calculation_id);
CREATE INDEX IF NOT EXISTS idx_revenue_recognition_date ON revenue_recognition_schedule(recognition_date);
CREATE INDEX IF NOT EXISTS idx_revenue_recognition_pending ON revenue_recognition_schedule(recognized) WHERE recognized = FALSE;

-- ===========================
-- HIGH WATER MARKS TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS high_water_marks (
    hwm_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    account_id UUID,  -- NULL = portfolio level
    
    -- High water mark tracking
    current_high_water_mark DECIMAL(15,2) NOT NULL,
    previous_high_water_mark DECIMAL(15,2),
    hwm_date DATE NOT NULL,
    
    -- Reset information
    last_reset_date DATE,
    reset_reason TEXT,
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_hwm_client_account ON high_water_marks(client_id, COALESCE(account_id, '00000000-0000-0000-0000-000000000000'::UUID));
CREATE INDEX IF NOT EXISTS idx_hwm_client ON high_water_marks(client_id);

-- ===========================
-- UPDATE TRIGGERS
-- ===========================
CREATE OR REPLACE FUNCTION update_fee_billing_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS fee_schedules_updated_at ON fee_schedules;
CREATE TRIGGER fee_schedules_updated_at
    BEFORE UPDATE ON fee_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_fee_billing_timestamp();

DROP TRIGGER IF EXISTS client_fee_assignments_updated_at ON client_fee_assignments;
CREATE TRIGGER client_fee_assignments_updated_at
    BEFORE UPDATE ON client_fee_assignments
    FOR EACH ROW
    EXECUTE FUNCTION update_fee_billing_timestamp();

DROP TRIGGER IF EXISTS fee_calculations_updated_at ON fee_calculations;
CREATE TRIGGER fee_calculations_updated_at
    BEFORE UPDATE ON fee_calculations
    FOR EACH ROW
    EXECUTE FUNCTION update_fee_billing_timestamp();

DROP TRIGGER IF EXISTS high_water_marks_updated_at ON high_water_marks;
CREATE TRIGGER high_water_marks_updated_at
    BEFORE UPDATE ON high_water_marks
    FOR EACH ROW
    EXECUTE FUNCTION update_fee_billing_timestamp();

-- ===========================
-- VIEWS
-- ===========================

-- View for monthly revenue summary
CREATE OR REPLACE VIEW monthly_revenue_summary AS
SELECT 
    DATE_TRUNC('month', fc.billing_period_end) AS month,
    COUNT(DISTINCT fc.client_id) AS clients_billed,
    COUNT(*) AS total_invoices,
    SUM(fc.aum_based_fee) AS total_aum_fees,
    SUM(fc.performance_fee) AS total_performance_fees,
    SUM(fc.planning_fee + fc.hourly_fees + fc.other_fees) AS total_other_fees,
    SUM(fc.net_fee) AS total_revenue,
    SUM(fc.payment_amount) AS total_collected,
    SUM(CASE WHEN fc.calculation_status = 'PAID' THEN 1 ELSE 0 END) AS invoices_paid,
    SUM(CASE WHEN fc.calculation_status IN ('INVOICED', 'PARTIALLY_PAID') THEN fc.net_fee - COALESCE(fc.payment_amount, 0) ELSE 0 END) AS outstanding_ar
FROM fee_calculations fc
WHERE fc.calculation_status != 'DRAFT'
GROUP BY DATE_TRUNC('month', fc.billing_period_end)
ORDER BY month DESC;

-- View for client billing summary
CREATE OR REPLACE VIEW client_billing_summary AS
SELECT 
    fc.client_id,
    COUNT(*) AS total_invoices,
    SUM(fc.net_fee) AS lifetime_fees_billed,
    SUM(fc.payment_amount) AS lifetime_fees_paid,
    AVG(fc.net_fee) AS average_invoice_amount,
    SUM(CASE WHEN fc.calculation_status IN ('INVOICED', 'PARTIALLY_PAID') THEN fc.net_fee - COALESCE(fc.payment_amount, 0) ELSE 0 END) AS current_balance_due,
    MAX(fc.billing_period_end) AS last_billing_date,
    AVG(CASE WHEN fc.payment_received_at IS NOT NULL 
        THEN EXTRACT(DAY FROM fc.payment_received_at - fc.invoice_sent_at) 
        ELSE NULL END) AS avg_days_to_payment
FROM fee_calculations fc
WHERE fc.calculation_status != 'DRAFT'
GROUP BY fc.client_id;

-- View for pending approvals
CREATE OR REPLACE VIEW fee_calc_pending_approval AS
SELECT 
    fc.calculation_id,
    fc.client_id,
    fc.billing_period_start,
    fc.billing_period_end,
    fc.net_fee,
    fc.requires_manual_review,
    fc.review_notes,
    fc.created_at,
    cfa.schedule_id,
    fs.schedule_name
FROM fee_calculations fc
LEFT JOIN client_fee_assignments cfa ON fc.assignment_id = cfa.assignment_id
LEFT JOIN fee_schedules fs ON cfa.schedule_id = fs.schedule_id
WHERE fc.calculation_status = 'PENDING_REVIEW'
   OR fc.requires_manual_review = TRUE
ORDER BY fc.created_at;

COMMENT ON TABLE fee_schedules IS 'Fee schedule templates with tiered AUM, performance fees, and billing configurations';
COMMENT ON TABLE client_fee_assignments IS 'Client-specific fee schedule assignments with custom overrides';
COMMENT ON TABLE fee_calculations IS 'Individual fee calculations with full audit trail and approval workflow';
COMMENT ON TABLE revenue_recognition_schedule IS 'Revenue recognition schedules for accrual accounting';
COMMENT ON TABLE high_water_marks IS 'High water mark tracking for performance fee calculations';
