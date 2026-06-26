-- Migration 033: Direct Indexing Platform
-- Tax-loss harvesting with individual stock tax lot tracking

-- =============================================================================
-- 1. ENUM TYPES
-- =============================================================================

CREATE TYPE tax_lot_method AS ENUM (
    'FIFO',     -- First In First Out
    'LIFO',     -- Last In First Out
    'HIFO',     -- Highest In First Out (maximize tax loss)
    'SPEC_ID'   -- Specific Identification
);

CREATE TYPE direct_index_status AS ENUM (
    'ACTIVE',
    'PENDING',
    'SUSPENDED',
    'CLOSED'
);

CREATE TYPE harvest_status AS ENUM (
    'PENDING',
    'APPROVED',
    'EXECUTED',
    'EXPIRED',
    'DISMISSED'
);

-- =============================================================================
-- 2. DIRECT INDEX ACCOUNTS
-- =============================================================================

CREATE TABLE direct_index_accounts (
    account_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(client_id),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id),
    
    -- Account identification
    account_number VARCHAR(50),
    account_name TEXT NOT NULL,
    custodian VARCHAR(100), -- 'SCHWAB', 'FIDELITY', 'PERSHING'
    
    -- Benchmark configuration
    benchmark_index VARCHAR(50) NOT NULL, -- 'SP500', 'RUSSELL_2000', 'NASDAQ_100'
    tracking_method VARCHAR(50) DEFAULT 'OPTIMIZED', -- 'FULL_REPLICATION', 'OPTIMIZED', 'CUSTOM'
    target_tracking_error_pct DECIMAL(5,2) DEFAULT 0.50, -- Max 0.5% tracking error
    
    -- Customization profile
    customization_profile JSONB NOT NULL DEFAULT '{}',
    /* Example:
    {
        "esg_screening": "MODERATE",
        "exclusions": ["XOM", "CVX"],  // Fossil fuel exclusions
        "tilts": {
            "dividend_yield": 1.2,     // 20% overweight dividends
            "low_volatility": 1.1       // 10% overweight low vol
        },
        "values_alignment": {
            "renewable_energy": true,
            "gender_diversity": true,
            "no_tobacco": true,
            "no_weapons": true
        }
    }
    */
    
    -- Tax settings
    tax_lot_method tax_lot_method DEFAULT 'HIFO',
    harvest_threshold_pct DECIMAL(5,2) DEFAULT 5.00, -- Min 5% loss to harvest
    wash_sale_buffer_days INTEGER DEFAULT 35, -- 30 days + 5 day buffer
    min_harvest_amount DECIMAL(15,2) DEFAULT 500.00, -- Don't harvest < $500
    auto_harvest_enabled BOOLEAN DEFAULT TRUE,
    
    -- Client tax profile
    federal_tax_bracket DECIMAL(5,2), -- e.g., 37.00 for 37%
    state_tax_bracket DECIMAL(5,2),
    ltcg_tax_rate DECIMAL(5,2), -- Long-term capital gains rate
    stcg_tax_rate DECIMAL(5,2), -- Short-term capital gains rate
    
    -- Performance tracking
    total_market_value DECIMAL(15,2) DEFAULT 0,
    total_cost_basis DECIMAL(15,2) DEFAULT 0,
    total_unrealized_gain_loss DECIMAL(15,2) DEFAULT 0,
    
    -- YTD tax metrics
    ytd_tax_loss_harvested DECIMAL(15,2) DEFAULT 0,
    ytd_tax_savings DECIMAL(15,2) DEFAULT 0,
    ytd_realized_gains DECIMAL(15,2) DEFAULT 0,
    ytd_realized_losses DECIMAL(15,2) DEFAULT 0,
    
    -- Benchmark tracking
    ytd_return_pct DECIMAL(8,4),
    ytd_benchmark_return_pct DECIMAL(8,4),
    tracking_error_pct DECIMAL(5,2),
    
    -- Status
    account_status direct_index_status DEFAULT 'ACTIVE',
    inception_date DATE NOT NULL DEFAULT CURRENT_DATE,
    last_rebalance_date DATE,
    next_rebalance_date DATE,
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(user_id),
    
    INDEX idx_di_client (client_id),
    INDEX idx_di_tenant (tenant_id),
    INDEX idx_di_status (account_status) WHERE account_status = 'ACTIVE'
);

-- RLS for multi-tenancy
ALTER TABLE direct_index_accounts ENABLE ROW LEVEL SECURITY;

CREATE POLICY di_accounts_tenant_isolation ON direct_index_accounts
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

-- =============================================================================
-- 3. DIRECT INDEX HOLDINGS
-- =============================================================================

CREATE TABLE direct_index_holdings (
    holding_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES direct_index_accounts(account_id) ON DELETE CASCADE,
    
    -- Security details
    ticker VARCHAR(10) NOT NULL,
    cusip VARCHAR(9),
    security_name TEXT,
    sector VARCHAR(50),
    industry VARCHAR(100),
    
    -- Position
    shares_owned DECIMAL(12,4) NOT NULL DEFAULT 0,
    average_cost_basis DECIMAL(15,4),
    current_price DECIMAL(15,4),
    current_market_value DECIMAL(15,2),
    
    -- Weight in portfolio
    portfolio_weight_pct DECIMAL(6,3),
    benchmark_weight_pct DECIMAL(6,3),
    active_weight_pct DECIMAL(6,3) GENERATED ALWAYS AS (
        portfolio_weight_pct - benchmark_weight_pct
    ) STORED,
    
    -- Tax lot detail (JSONB for flexibility)
    tax_lots JSONB NOT NULL DEFAULT '[]',
    /* Example:
    [
        {
            "lot_id": "uuid",
            "acquisition_date": "2024-01-15",
            "shares": 100,
            "cost_basis_per_share": 150.00,
            "total_cost_basis": 15000.00,
            "is_long_term": false,
            "wash_sale_disallowed": 0,
            "acquisition_method": "PURCHASE"
        }
    ]
    */
    
    -- P&L metrics
    unrealized_gain_loss DECIMAL(15,2) GENERATED ALWAYS AS (
        current_market_value - (shares_owned * average_cost_basis)
    ) STORED,
    unrealized_gain_loss_pct DECIMAL(8,4),
    
    -- Tax harvest eligibility
    harvest_eligible BOOLEAN DEFAULT FALSE,
    last_harvest_date DATE,
    days_since_harvest INTEGER,
    estimated_tax_savings DECIMAL(15,2),
    
    -- Dividend metrics
    annual_dividend_per_share DECIMAL(10,4),
    dividend_yield_pct DECIMAL(6,3),
    last_dividend_date DATE,
    
    -- ESG scores (if applicable)
    esg_score DECIMAL(5,2), -- 0-100
    esg_category VARCHAR(10), -- 'AA', 'AAA', etc.
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_price_update TIMESTAMPTZ,
    
    UNIQUE (account_id, ticker),
    INDEX idx_holdings_account (account_id),
    INDEX idx_holdings_ticker (ticker),
    INDEX idx_holdings_harvest (account_id, harvest_eligible) WHERE harvest_eligible = TRUE
);

ALTER TABLE direct_index_holdings ENABLE ROW LEVEL SECURITY;

CREATE POLICY di_holdings_via_account ON direct_index_holdings
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM direct_index_accounts
            WHERE direct_index_accounts.account_id = direct_index_holdings.account_id
            AND direct_index_accounts.tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID
        )
    );

-- =============================================================================
-- 4. TAX-LOSS HARVEST OPPORTUNITIES
-- =============================================================================

CREATE TABLE tax_loss_opportunities (
    opportunity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES direct_index_accounts(account_id),
    holding_id UUID NOT NULL REFERENCES direct_index_holdings(holding_id),
    
    -- Opportunity details
    ticker VARCHAR(10) NOT NULL,
    shares_to_sell DECIMAL(12,4) NOT NULL,
    cost_basis_per_share DECIMAL(15,4),
    current_price DECIMAL(15,4),
    unrealized_loss DECIMAL(15,2) NOT NULL,
    unrealized_loss_pct DECIMAL(8,4),
    
    -- Tax impact
    estimated_tax_savings DECIMAL(15,2),
    tax_rate_used DECIMAL(5,2),
    holding_period_days INTEGER,
    is_long_term BOOLEAN,
    
    -- Replacement strategy
    replacement_ticker VARCHAR(10),
    replacement_name TEXT,
    correlation_with_original DECIMAL(5,4), -- 0.95+ correlation
    replacement_shares DECIMAL(12,4),
    replacement_cost DECIMAL(15,2),
    
    -- Wash sale check
    wash_sale_risk BOOLEAN DEFAULT FALSE,
    wash_sale_window_start DATE,
    wash_sale_window_end DATE,
    
    -- Execution details
    opportunity_status harvest_status DEFAULT 'PENDING',
    detected_at TIMESTAMPTZ DEFAULT NOW(),
    approved_at TIMESTAMPTZ,
    approved_by UUID REFERENCES users(user_id),
    executed_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    dismissal_reason TEXT,
    
    -- Order IDs (if executed)
    sell_order_id UUID,
    buy_order_id UUID,
    
    INDEX idx_opps_account (account_id),
    INDEX idx_opps_status (opportunity_status, detected_at DESC),
    INDEX idx_opps_pending (account_id, opportunity_status) WHERE opportunity_status = 'PENDING'
);

ALTER TABLE tax_loss_opportunities ENABLE ROW LEVEL SECURITY;

CREATE POLICY di_opportunities_via_account ON tax_loss_opportunities
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM direct_index_accounts
            WHERE direct_index_accounts.account_id = tax_loss_opportunities.account_id
            AND direct_index_accounts.tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID
        )
    );

-- =============================================================================
-- 5. WASH SALE TRACKER
-- =============================================================================

CREATE TABLE wash_sale_tracker (
    wash_sale_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES direct_index_accounts(account_id),
    
    -- Security sold at loss
    ticker VARCHAR(10) NOT NULL,
    sale_date DATE NOT NULL,
    shares_sold DECIMAL(12,4) NOT NULL,
    sale_price DECIMAL(15,4),
    realized_loss DECIMAL(15,2) NOT NULL,
    
    -- Wash sale window (30 days before + 30 days after)
    wash_window_start DATE NOT NULL,
    wash_window_end DATE NOT NULL,
    
    -- Violation tracking
    is_violation BOOLEAN DEFAULT FALSE,
    violation_detected_at TIMESTAMPTZ,
    repurchase_date DATE,
    repurchase_shares DECIMAL(12,4),
    disallowed_loss DECIMAL(15,2) DEFAULT 0,
    
    -- Adjusted basis (loss is added to replacement shares)
    adjusted_basis_increase DECIMAL(15,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_wash_account (account_id),
    INDEX idx_wash_ticker_window (ticker, wash_window_end) WHERE wash_window_end >= CURRENT_DATE,
    INDEX idx_wash_violations (account_id, is_violation) WHERE is_violation = TRUE
);

ALTER TABLE wash_sale_tracker ENABLE ROW LEVEL SECURITY;

CREATE POLICY wash_sales_via_account ON wash_sale_tracker
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM direct_index_accounts
            WHERE direct_index_accounts.account_id = wash_sale_tracker.account_id
            AND direct_index_accounts.tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID
        )
    );

-- =============================================================================
-- 6. REBALANCE HISTORY
-- =============================================================================

CREATE TABLE direct_index_rebalances (
    rebalance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES direct_index_accounts(account_id),
    
    -- Rebalance details
    rebalance_date DATE NOT NULL,
    rebalance_type VARCHAR(50), -- 'SCHEDULED', 'THRESHOLD', 'MANUAL', 'HARVEST'
    trigger_reason TEXT,
    
    -- Metrics before/after
    holdings_before INTEGER,
    holdings_after INTEGER,
    tracking_error_before DECIMAL(5,2),
    tracking_error_after DECIMAL(5,2),
    turnover_pct DECIMAL(6,3),
    
    -- Trades executed
    trades_executed JSONB, -- Array of {ticker, action, shares, price}
    total_trades INTEGER,
    
    -- Tax impact
    tax_loss_harvested DECIMAL(15,2) DEFAULT 0,
    tax_savings_realized DECIMAL(15,2) DEFAULT 0,
    
    -- Status
    status VARCHAR(20) DEFAULT 'COMPLETED',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    INDEX idx_rebalances_account (account_id, rebalance_date DESC)
);

-- =============================================================================
-- 7. HELPER FUNCTIONS
-- =============================================================================

-- Calculate tax savings for a harvest opportunity
CREATE OR REPLACE FUNCTION calculate_tax_savings(
    p_unrealized_loss DECIMAL,
    p_federal_bracket DECIMAL,
    p_state_bracket DECIMAL
) RETURNS DECIMAL AS $$
BEGIN
    RETURN ABS(p_unrealized_loss) * ((p_federal_bracket + p_state_bracket) / 100);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Check if position is eligible for harvest
CREATE OR REPLACE FUNCTION is_harvest_eligible(
    p_holding_id UUID
) RETURNS BOOLEAN AS $$
DECLARE
    v_account_id UUID;
    v_threshold DECIMAL;
    v_min_amount DECIMAL;
    v_unrealized_loss DECIMAL;
    v_last_harvest DATE;
    v_buffer_days INTEGER;
BEGIN
    -- Get account settings and holding details
    SELECT 
        h.account_id,
        a.harvest_threshold_pct,
        a.min_harvest_amount,
        h.unrealized_gain_loss,
        h.last_harvest_date,
        a.wash_sale_buffer_days
    INTO 
        v_account_id, v_threshold, v_min_amount, 
        v_unrealized_loss, v_last_harvest, v_buffer_days
    FROM direct_index_holdings h
    JOIN direct_index_accounts a ON h.account_id = a.account_id
    WHERE h.holding_id = p_holding_id;
    
    -- Must have unrealized loss
    IF v_unrealized_loss >= 0 THEN
        RETURN FALSE;
    END IF;
    
    -- Must exceed minimum dollar amount
    IF ABS(v_unrealized_loss) < v_min_amount THEN
        RETURN FALSE;
    END IF;
    
    -- Must exceed threshold percentage
    -- (Check if loss is >= threshold %, e.g., 5%)
    
    -- Must respect wash sale buffer (don't harvest same stock too soon)
    IF v_last_harvest IS NOT NULL THEN
        IF CURRENT_DATE - v_last_harvest < v_buffer_days THEN
            RETURN FALSE;
        END IF;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql STABLE;

-- Update YTD metrics for an account
CREATE OR REPLACE FUNCTION update_ytd_metrics(p_account_id UUID) RETURNS VOID AS $$
BEGIN
    UPDATE direct_index_accounts SET
        total_market_value = (
            SELECT COALESCE(SUM(current_market_value), 0)
            FROM direct_index_holdings
            WHERE account_id = p_account_id
        ),
        total_cost_basis = (
            SELECT COALESCE(SUM(shares_owned * average_cost_basis), 0)
            FROM direct_index_holdings
            WHERE account_id = p_account_id
        ),
        total_unrealized_gain_loss = (
            SELECT COALESCE(SUM(unrealized_gain_loss), 0)
            FROM direct_index_holdings
            WHERE account_id = p_account_id
        ),
        updated_at = NOW()
    WHERE account_id = p_account_id;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 8. TRIGGERS
-- =============================================================================

-- Auto-update account metrics when holdings change
CREATE OR REPLACE FUNCTION trigger_update_account_metrics() RETURNS TRIGGER AS $$
BEGIN
    PERFORM update_ytd_metrics(NEW.account_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER holdings_update_metrics
AFTER INSERT OR UPDATE OR DELETE ON direct_index_holdings
FOR EACH ROW
EXECUTE FUNCTION trigger_update_account_metrics();

-- =============================================================================
-- 9. COMMENTS
-- =============================================================================

COMMENT ON TABLE direct_index_accounts IS 'Client accounts using direct indexing strategy with tax optimization';
COMMENT ON TABLE direct_index_holdings IS 'Individual stock holdings with tax lot tracking for harvest optimization';
COMMENT ON TABLE tax_loss_opportunities IS 'Daily-detected tax-loss harvesting opportunities with replacement securities';
COMMENT ON TABLE wash_sale_tracker IS 'Wash sale violation prevention with 30-day window tracking';
COMMENT ON FUNCTION calculate_tax_savings IS 'Calculate expected tax savings from harvesting a loss';
COMMENT ON FUNCTION is_harvest_eligible IS 'Check if a holding qualifies for tax-loss harvesting';
