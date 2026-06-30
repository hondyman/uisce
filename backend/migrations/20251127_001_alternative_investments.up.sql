-- Alternative Investment Management Schema
-- Phase 1: Alternative Investment Platform

-- ===========================
-- CLEANUP (Ensure clean slate)
-- ===========================
DROP TABLE IF EXISTS alt_investment_documents CASCADE;
DROP TABLE IF EXISTS distributions CASCADE;
DROP TABLE IF EXISTS capital_calls CASCADE;
DROP TABLE IF EXISTS alternative_investments CASCADE;

-- ===========================
-- ALTERNATIVE INVESTMENTS TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS alternative_investments (
    investment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    investment_type VARCHAR(50) NOT NULL CHECK (investment_type IN (
        'PRIVATE_EQUITY',
        'VENTURE_CAPITAL', 
        'HEDGE_FUND',
        'REAL_ESTATE',
        'DIRECT_INVESTMENT',
        'INFRASTRUCTURE',
        'PRIVATE_DEBT'
    )),
    fund_name TEXT NOT NULL,
    general_partner TEXT,
    vintage_year INTEGER,
    
    -- Capital commitments and cash flows
    total_commitment_amount DECIMAL(15,2) NOT NULL,
    unfunded_commitment DECIMAL(15,2) DEFAULT 0,
    total_capital_called DECIMAL(15,2) DEFAULT 0,
    total_distributions DECIMAL(15,2) DEFAULT 0,
    
    -- Valuations (often quarterly or annual)
    current_nav DECIMAL(15,2),
    nav_date DATE,
    valuation_source VARCHAR(50) CHECK (valuation_source IN (
        'GP_REPORTED',
        'THIRD_PARTY', 
        'INTERNAL_ESTIMATE'
    )),
    
    -- Performance metrics
    irr_since_inception DECIMAL(5,2), -- Internal Rate of Return
    tvpi DECIMAL(5,2),                -- Total Value to Paid-In
    dpi DECIMAL(5,2),                 -- Distributions to Paid-In
    rvpi DECIMAL(5,2),                -- Residual Value to Paid-In
    moic DECIMAL(5,2),                -- Multiple on Invested Capital
    
    -- Liquidity constraints
    lock_up_end_date DATE,
    redemption_notice_days INTEGER,
    redemption_frequency VARCHAR(50) CHECK (redemption_frequency IN (
        'QUARTERLY',
        'ANNUAL',
        'CLOSED_END',
        'NONE'
    )),
    
    -- Document tracking
    last_capital_call_date DATE,
    last_distribution_date DATE,
    last_k1_received_date DATE,
    
    -- Additional metadata
    metadata JSONB DEFAULT '{}',
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    
    CONSTRAINT valid_commitment CHECK (total_commitment_amount >= 0),
    CONSTRAINT valid_unfunded CHECK (unfunded_commitment >= 0),
    CONSTRAINT valid_nav CHECK (current_nav IS NULL OR current_nav >= 0)
);

CREATE INDEX IF NOT EXISTS idx_alt_inv_client ON alternative_investments(client_id);
CREATE INDEX IF NOT EXISTS idx_alt_inv_type ON alternative_investments(investment_type);
CREATE INDEX IF NOT EXISTS idx_alt_inv_vintage ON alternative_investments(vintage_year);
CREATE INDEX IF NOT EXISTS idx_alt_inv_gp ON alternative_investments(general_partner);

-- ===========================
-- CAPITAL CALLS TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS capital_calls (
    call_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(investment_id) ON DELETE CASCADE,
    
    -- Call details
    notice_date DATE NOT NULL,
    due_date DATE NOT NULL,
    amount_requested DECIMAL(15,2) NOT NULL,
    amount_funded DECIMAL(15,2) DEFAULT 0,
    
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN (
        'PENDING',
        'FUNDED',
        'PARTIALLY_FUNDED',
        'OVERDUE',
        'CANCELLED'
    )),
    
    -- Funding source
    funding_source_account UUID, -- Reference to account that will fund this
    
    -- Liquidity validation
    liquidity_check_passed BOOLEAN,
    liquidity_check_date TIMESTAMPTZ,
    liquidity_shortage_amount DECIMAL(15,2),
    
    -- Notifications
    alert_sent_at TIMESTAMPTZ,
    reminder_sent_at TIMESTAMPTZ,
    
    -- Notes
    advisor_notes TEXT,
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    
    CONSTRAINT valid_call_amount CHECK (amount_requested > 0),
    CONSTRAINT valid_funded_amount CHECK (amount_funded >= 0),
    CONSTRAINT valid_dates CHECK (due_date >= notice_date)
);

CREATE INDEX IF NOT EXISTS idx_capital_calls_investment ON capital_calls(investment_id);
CREATE INDEX IF NOT EXISTS idx_capital_calls_status ON capital_calls(status);
CREATE INDEX IF NOT EXISTS idx_capital_calls_due_date ON capital_calls(due_date);
CREATE INDEX IF NOT EXISTS idx_capital_calls_overdue ON capital_calls(due_date) WHERE status = 'PENDING' OR status = 'PARTIALLY_FUNDED';

-- ===========================
-- DISTRIBUTIONS TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS distributions (
    distribution_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(investment_id) ON DELETE CASCADE,
    
    -- Distribution details
    distribution_date DATE NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    distribution_type VARCHAR(50) NOT NULL CHECK (distribution_type IN (
        'INCOME',
        'RETURN_OF_CAPITAL',
        'CAPITAL_GAIN',
        'RECALLABLE'
    )),
    
    -- Reinvestment
    reinvested BOOLEAN DEFAULT FALSE,
    reinvestment_date DATE,
    reinvestment_account UUID,
    
    -- Tax implications
    tax_year INTEGER,
    taxable_amount DECIMAL(15,2),
    
    -- Notes
    advisor_notes TEXT,
    
    -- Audit fields
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    
    CONSTRAINT valid_distribution_amount CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_distributions_investment ON distributions(investment_id);
CREATE INDEX IF NOT EXISTS idx_distributions_date ON distributions(distribution_date);
CREATE INDEX IF NOT EXISTS idx_distributions_tax_year ON distributions(tax_year);
CREATE INDEX IF NOT EXISTS idx_distributions_type ON distributions(distribution_type);

-- ===========================
-- ALTERNATIVE INVESTMENT DOCUMENTS TABLE
-- ===========================
CREATE TABLE IF NOT EXISTS alt_investment_documents (
    document_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    investment_id UUID NOT NULL REFERENCES alternative_investments(investment_id) ON DELETE CASCADE,
    
    -- Document metadata
    document_type VARCHAR(50) NOT NULL CHECK (document_type IN (
        'K1',
        'CAPITAL_STATEMENT',
        'QUARTERLY_REPORT',
        'ANNUAL_REPORT',
        'SUBSCRIPTION_AGREEMENT',
        'OPERATING_AGREEMENT',
        'SIDE_LETTER',
        'OTHER'
    )),
    document_date DATE,
    tax_year INTEGER,
    
    -- File storage
    file_url TEXT NOT NULL,
    file_name TEXT,
    file_size_bytes INTEGER,
    mime_type VARCHAR(100),
    
    -- AI extraction
    extracted_data JSONB DEFAULT '{}',
    extraction_status VARCHAR(50) CHECK (extraction_status IN (
        'PENDING',
        'IN_PROGRESS',
        'COMPLETED',
        'FAILED',
        'MANUAL_REVIEW_REQUIRED'
    )),
    extraction_confidence DECIMAL(3,2), -- 0.0 to 1.0
    
    -- Processing
    processed_at TIMESTAMPTZ,
    processed_by VARCHAR(100), -- 'GEMINI_AI', 'MANUAL', etc.
    
    -- Audit fields
    uploaded_at TIMESTAMPTZ DEFAULT NOW(),
    uploaded_by UUID,
    
    CONSTRAINT valid_confidence CHECK (extraction_confidence IS NULL OR (extraction_confidence >= 0 AND extraction_confidence <= 1))
);

CREATE INDEX IF NOT EXISTS idx_alt_docs_investment ON alt_investment_documents(investment_id);
CREATE INDEX IF NOT EXISTS idx_alt_docs_type ON alt_investment_documents(document_type);
CREATE INDEX IF NOT EXISTS idx_alt_docs_tax_year ON alt_investment_documents(tax_year);
CREATE INDEX IF NOT EXISTS idx_alt_docs_status ON alt_investment_documents(extraction_status);

-- ===========================
-- UPDATE TRIGGER FOR ALTERNATIVE INVESTMENTS
-- ===========================
CREATE OR REPLACE FUNCTION update_alternative_investment_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS alternative_investments_updated_at ON alternative_investments;
CREATE TRIGGER alternative_investments_updated_at
    BEFORE UPDATE ON alternative_investments
    FOR EACH ROW
    EXECUTE FUNCTION update_alternative_investment_timestamp();

DROP TRIGGER IF EXISTS capital_calls_updated_at ON capital_calls;
CREATE TRIGGER capital_calls_updated_at
    BEFORE UPDATE ON capital_calls
    FOR EACH ROW
    EXECUTE FUNCTION update_alternative_investment_timestamp();

DROP TRIGGER IF EXISTS distributions_updated_at ON distributions;
CREATE TRIGGER distributions_updated_at
    BEFORE UPDATE ON distributions
    FOR EACH ROW
    EXECUTE FUNCTION update_alternative_investment_timestamp();

-- ===========================
-- VIEWS
-- ===========================

-- View for investment performance summary
CREATE OR REPLACE VIEW alt_investment_performance AS
SELECT 
    ai.investment_id,
    ai.client_id,
    ai.fund_name,
    ai.investment_type,
    ai.vintage_year,
    ai.total_commitment_amount,
    ai.unfunded_commitment,
    ai.total_capital_called,
    ai.total_distributions,
    ai.current_nav,
    ai.nav_date,
    
    -- Performance metrics
    ai.irr_since_inception,
    ai.tvpi,
    ai.dpi,
    ai.rvpi,
    ai.moic,
    
    -- Calculated fields
    (ai.total_capital_called - ai.total_distributions) AS net_cash_flow,
    ((ai.current_nav + ai.total_distributions) / NULLIF(ai.total_capital_called, 0)) AS total_value_multiple,
    (ai.unfunded_commitment / NULLIF(ai.total_commitment_amount, 0) * 100) AS pct_unfunded
FROM alternative_investments ai;

-- View for upcoming capital calls
CREATE OR REPLACE VIEW upcoming_capital_calls AS
SELECT 
    cc.call_id,
    cc.investment_id,
    ai.client_id,
    ai.fund_name,
    cc.notice_date,
    cc.due_date,
    cc.amount_requested,
    cc.amount_funded,
    cc.status,
    cc.liquidity_check_passed,
    cc.funding_source_account,
    (cc.due_date - CURRENT_DATE) AS days_until_due
FROM capital_calls cc
JOIN alternative_investments ai ON cc.investment_id = ai.investment_id
WHERE cc.status IN ('PENDING', 'PARTIALLY_FUNDED')
ORDER BY cc.due_date;

-- View for client alternative investment allocation
CREATE OR REPLACE VIEW client_alt_allocation AS
WITH type_stats AS (
    SELECT
        client_id,
        investment_type,
        COUNT(*) as type_count,
        SUM(current_nav) as type_nav
    FROM alternative_investments
    GROUP BY client_id, investment_type
),
overall_stats AS (
    SELECT
        client_id,
        COUNT(*) AS total_investments,
        SUM(total_commitment_amount) AS total_committed,
        SUM(unfunded_commitment) AS total_unfunded,
        SUM(total_capital_called) AS total_called,
        SUM(current_nav) AS total_current_value,
        SUM(total_distributions) AS total_distributions_received,
        AVG(irr_since_inception) AS avg_irr,
        AVG(tvpi) AS avg_tvpi
    FROM alternative_investments
    GROUP BY client_id
)
SELECT
    o.client_id,
    o.total_investments,
    o.total_committed,
    o.total_unfunded,
    o.total_called,
    o.total_current_value,
    o.total_distributions_received,
    o.avg_irr,
    o.avg_tvpi,
    (
        SELECT JSONB_OBJECT_AGG(
            t.investment_type,
            JSONB_BUILD_OBJECT(
                'count', t.type_count,
                'total_nav', t.type_nav
            )
        )
        FROM type_stats t
        WHERE t.client_id = o.client_id
    ) AS allocation_by_type
FROM overall_stats o;

COMMENT ON TABLE alternative_investments IS 'Tracks alternative investments including private equity, venture capital, hedge funds, real estate, and direct investments';
COMMENT ON TABLE capital_calls IS 'Tracks capital call notices and funding status with liquidity validation';
COMMENT ON TABLE distributions IS 'Tracks distributions from alternative investments with tax categorization';
COMMENT ON TABLE alt_investment_documents IS 'Stores alternative investment documents with AI-powered data extraction status';
