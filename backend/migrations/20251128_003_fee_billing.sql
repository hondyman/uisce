-- Fee structure templates
CREATE TABLE IF NOT EXISTS fee_schedules (
    schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_name TEXT NOT NULL,
    fee_type VARCHAR(50) NOT NULL, -- 'AUM_TIERED', 'FLAT_ANNUAL', 'PERFORMANCE', 'SUBSCRIPTION', 'HYBRID'
    
    -- Tiered AUM example: 1% on first $1M, 0.75% on next $4M, 0.5% above $5M
    tier_structure JSONB, -- [{"min": 0, "max": 1000000, "rate": 0.01}, ...]
    
    -- Performance fee structure (e.g., 20% over 8% hurdle with high water mark)
    performance_hurdle_rate DECIMAL(5,4),
    performance_fee_rate DECIMAL(5,4),
    high_water_mark_enabled BOOLEAN DEFAULT FALSE,
    
    -- Billing frequency and timing
    billing_frequency VARCHAR(20) NOT NULL, -- 'MONTHLY', 'QUARTERLY', 'ANNUAL'
    billing_advance_or_arrears VARCHAR(10) NOT NULL, -- 'ADVANCE', 'ARREARS'
    
    -- Minimum fees
    minimum_quarterly_fee DECIMAL(10,2),
    minimum_annual_fee DECIMAL(10,2),
    
    -- Account-level customizations
    exclude_cash_from_aum BOOLEAN DEFAULT FALSE,
    exclude_alternatives_from_aum BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Client fee assignments
CREATE TABLE IF NOT EXISTS client_fee_assignments (
    assignment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    account_id UUID, -- NULL = applies to all accounts
    schedule_id UUID REFERENCES fee_schedules(schedule_id),
    effective_date DATE NOT NULL,
    end_date DATE, -- NULL = ongoing
    
    -- Negotiated overrides
    custom_discount_pct DECIMAL(5,2), -- e.g., 0.10 = 10% discount
    custom_minimum_fee DECIMAL(10,2),
    
    -- Billing preferences
    invoice_contact_email TEXT,
    payment_method VARCHAR(50), -- 'DEBIT_FROM_ACCOUNT', 'WIRE', 'CHECK', 'ACH'
    billing_day_of_month INTEGER,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fee_assignments_client ON client_fee_assignments(client_id);

-- Fee calculations (audit trail)
CREATE TABLE IF NOT EXISTS fee_calculations (
    calculation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    billing_period_start DATE NOT NULL,
    billing_period_end DATE NOT NULL,
    
    -- AUM-based fees
    average_daily_balance DECIMAL(15,2),
    aum_based_fee DECIMAL(12,2),
    
    -- Performance fees
    portfolio_return_pct DECIMAL(5,2),
    hurdle_return_pct DECIMAL(5,2),
    excess_return DECIMAL(12,2),
    performance_fee DECIMAL(12,2),
    
    -- Adjustments
    prior_period_adjustment DECIMAL(12,2), -- For advance billing corrections
    discount_amount DECIMAL(12,2),
    minimum_fee_adjustment DECIMAL(12,2),
    
    -- Total
    total_fee DECIMAL(12,2),
    
    -- Status
    calculation_status VARCHAR(50), -- 'DRAFT', 'APPROVED', 'INVOICED', 'PAID'
    approved_by UUID, -- REFERENCES users(id)
    approved_at TIMESTAMPTZ,
    
    -- Invoice reference
    invoice_id UUID,
    invoice_sent_at TIMESTAMPTZ,
    payment_received_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fee_calcs_client ON fee_calculations(client_id);
CREATE INDEX IF NOT EXISTS idx_fee_calcs_period ON fee_calculations(billing_period_end);

-- Revenue recognition (for accrual accounting)
CREATE TABLE IF NOT EXISTS revenue_recognition_schedule (
    schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calculation_id UUID REFERENCES fee_calculations(calculation_id),
    recognition_date DATE NOT NULL,
    amount DECIMAL(12,2),
    recognized BOOLEAN DEFAULT FALSE,
    journal_entry_id UUID, -- Link to accounting system
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rev_rec_date ON revenue_recognition_schedule(recognition_date);
