-- Migration: Create Metadata-First Rebalancing Platform Tables
-- Version: 20241126_001
-- Description: Implements the Rule Definition Language (RDL) schema and supporting tables



-- ============================================================================
-- RULE DEFINITIONS TABLE (The "Secret Sauce")
-- ============================================================================
-- This table stores all business rules as metadata, enabling "Config over Code"

CREATE TABLE IF NOT EXISTS rule_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rule_id VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    name VARCHAR(200),
    description TEXT,
    jurisdiction VARCHAR(10),  -- ISO 3166-1 alpha-2 (US, GB, DE, etc.)
    
    -- Core rule configuration (JSON for flexibility)
    parameters JSONB NOT NULL DEFAULT '{}',
    
    -- CEL expression for rule evaluation
    expression TEXT NOT NULL,
    
    -- Optional scoring formula for prioritization
    scoring_formula TEXT,
    
    -- Rule lifecycle
    active BOOLEAN DEFAULT true,
    effective_from DATE,
    effective_to DATE,
    
    -- Extended configurations
    wash_sale_config JSONB,
    substitute_asset_rules JSONB,
    schedule JSONB,
    notifications JSONB,
    
    -- Audit trail
    audit JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT rule_definitions_tenant_id_fk FOREIGN KEY (tenant_id) 
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT rule_definitions_unique_version 
        UNIQUE (tenant_id, rule_id, version),
    CONSTRAINT rule_definitions_type_check 
        CHECK (type IN ('tax_loss_harvesting', 'wash_sale', 'tax_constraint', 
                       'esg_restriction', 'drift_trigger', 'cash_flow', 
                       'cppi_floor', 'sector_limit', 'concentration_limit',
                       'custom'))
);

-- Partitioning by tenant_id for performance isolation
CREATE INDEX idx_rule_definitions_tenant ON rule_definitions(tenant_id);
CREATE INDEX idx_rule_definitions_type ON rule_definitions(tenant_id, type);
CREATE INDEX idx_rule_definitions_active ON rule_definitions(tenant_id, active) 
    WHERE active = true;
CREATE INDEX idx_rule_definitions_jurisdiction ON rule_definitions(tenant_id, jurisdiction);

-- ============================================================================
-- GLOBAL CALENDARS TABLE
-- ============================================================================
-- Metadata-driven business day calendars per region

CREATE TABLE IF NOT EXISTS global_calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    calendar_code VARCHAR(50) NOT NULL,  -- e.g., 'US_NYSE', 'UK_LSE', 'IN_BSE'
    region VARCHAR(10) NOT NULL,         -- ISO 3166-1 alpha-2
    year INT NOT NULL,
    
    -- Holiday data
    holidays JSONB NOT NULL DEFAULT '[]',  -- Array of dates with names
    
    -- Trading hours (per day of week)
    trading_hours JSONB DEFAULT '{
        "monday": {"open": "09:30", "close": "16:00"},
        "tuesday": {"open": "09:30", "close": "16:00"},
        "wednesday": {"open": "09:30", "close": "16:00"},
        "thursday": {"open": "09:30", "close": "16:00"},
        "friday": {"open": "09:30", "close": "16:00"}
    }',
    
    -- Special sessions (early close, late open)
    special_sessions JSONB DEFAULT '[]',
    
    -- Timezone
    timezone VARCHAR(50) NOT NULL DEFAULT 'America/New_York',
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT global_calendars_tenant_id_fk FOREIGN KEY (tenant_id) 
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT global_calendars_unique 
        UNIQUE (tenant_id, calendar_code, year)
);

CREATE INDEX idx_global_calendars_lookup ON global_calendars(tenant_id, calendar_code, year);

-- ============================================================================
-- TRIGGER DEFINITIONS TABLE
-- ============================================================================
-- Event-driven trigger configurations

CREATE TABLE IF NOT EXISTS trigger_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    trigger_id VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    
    -- CEL condition for trigger activation
    condition TEXT NOT NULL,
    
    -- Trigger-specific parameters
    parameters JSONB NOT NULL DEFAULT '{}',
    
    -- What to execute when triggered
    workflow_ref VARCHAR(200),  -- Temporal workflow name
    activity_ref VARCHAR(200),  -- Or direct activity
    webhook_url TEXT,           -- Or webhook
    
    -- Execution settings
    priority INT DEFAULT 0,
    max_concurrent INT DEFAULT 1,
    retry_policy JSONB DEFAULT '{"max_attempts": 3, "initial_interval": "1m"}',
    
    -- Lifecycle
    active BOOLEAN DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT trigger_definitions_tenant_id_fk FOREIGN KEY (tenant_id) 
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT trigger_definitions_unique 
        UNIQUE (tenant_id, trigger_id),
    CONSTRAINT trigger_definitions_type_check 
        CHECK (type IN ('TIME', 'DRIFT', 'CASH_FLOW', 'MARKET', 'RISK', 'TLH', 
                       'PRICE_MOVEMENT', 'REBALANCE_DUE', 'CUSTOM'))
);

CREATE INDEX idx_trigger_definitions_tenant ON trigger_definitions(tenant_id);
CREATE INDEX idx_trigger_definitions_active ON trigger_definitions(tenant_id, active) 
    WHERE active = true;
CREATE INDEX idx_trigger_definitions_type ON trigger_definitions(tenant_id, type);

-- ============================================================================
-- CPPI FLOOR CONFIGURATIONS TABLE
-- ============================================================================
-- Personalized floor protection for HNW clients

CREATE TABLE IF NOT EXISTS cppi_floors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_id UUID NOT NULL,
    
    -- Floor configuration
    floor_value_usd DECIMAL(18,2) NOT NULL,
    floor_type VARCHAR(20) DEFAULT 'ABSOLUTE',  -- ABSOLUTE, PERCENTAGE, INFLATION_ADJUSTED
    floor_percentage DECIMAL(5,2),              -- If type = PERCENTAGE
    
    -- CPPI parameters
    multiplier DECIMAL(5,2) DEFAULT 3.0,        -- Typical range: 2-5
    cushion_calculation VARCHAR(50) DEFAULT 'NAV_MINUS_FLOOR',
    
    -- Risk-free asset allocation
    risk_free_asset VARCHAR(50) DEFAULT 'TREASURY',
    risk_free_ticker VARCHAR(20),
    
    -- Rebalancing rules
    rebalance_threshold_pct DECIMAL(5,2) DEFAULT 5.0,
    min_rebalance_interval_days INT DEFAULT 1,
    last_rebalance_date DATE,
    last_rebalance_reason TEXT,
    
    -- Emergency settings
    floor_breach_action VARCHAR(50) DEFAULT 'LIQUIDATE_TO_FLOOR',
    notification_threshold_pct DECIMAL(5,2) DEFAULT 10.0,  -- Warn at 10% above floor
    
    -- Client context
    purpose TEXT,  -- e.g., "Daughter's college fund", "Retirement minimum"
    target_date DATE,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT cppi_floors_tenant_id_fk FOREIGN KEY (tenant_id) 
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT cppi_floors_unique 
        UNIQUE (tenant_id, portfolio_id),
    CONSTRAINT cppi_floors_type_check 
        CHECK (floor_type IN ('ABSOLUTE', 'PERCENTAGE', 'INFLATION_ADJUSTED'))
);

CREATE INDEX idx_cppi_floors_tenant ON cppi_floors(tenant_id);
CREATE INDEX idx_cppi_floors_portfolio ON cppi_floors(tenant_id, portfolio_id);

-- ============================================================================
-- RULE EVALUATION HISTORY TABLE
-- ============================================================================
-- Audit trail of all rule evaluations

CREATE TABLE IF NOT EXISTS rule_evaluation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rule_definition_id UUID NOT NULL,
    
    -- Context
    portfolio_id UUID,
    account_id UUID,
    household_id UUID,
    
    -- Evaluation details
    evaluated_at TIMESTAMPTZ DEFAULT NOW(),
    input_data JSONB NOT NULL,
    result BOOLEAN NOT NULL,
    score DECIMAL(10,4),
    
    -- If triggered, what happened
    action_taken VARCHAR(100),
    workflow_run_id VARCHAR(200),
    
    -- Performance
    evaluation_time_ms INT,
    
    -- Constraints
    CONSTRAINT rule_evaluation_history_tenant_id_fk FOREIGN KEY (tenant_id) 
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT rule_evaluation_history_rule_fk FOREIGN KEY (rule_definition_id) 
        REFERENCES rule_definitions(id) ON DELETE CASCADE
);

-- Partitioned by time for efficient querying
CREATE INDEX idx_rule_evaluation_tenant_time ON rule_evaluation_history(tenant_id, evaluated_at DESC);
CREATE INDEX idx_rule_evaluation_rule ON rule_evaluation_history(rule_definition_id, evaluated_at DESC);

-- ============================================================================
-- SUBSTITUTE ASSET MAPPINGS TABLE
-- ============================================================================
-- Pre-approved substitute securities for TLH

CREATE TABLE IF NOT EXISTS substitute_asset_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Original security
    original_ticker VARCHAR(20) NOT NULL,
    original_cusip VARCHAR(9),
    original_isin VARCHAR(12),
    
    -- Substitute security
    substitute_ticker VARCHAR(20) NOT NULL,
    substitute_cusip VARCHAR(9),
    substitute_isin VARCHAR(12),
    
    -- Matching quality
    correlation DECIMAL(5,4),
    tracking_error DECIMAL(5,4),
    factor_similarity DECIMAL(5,4),
    
    -- Categorization
    asset_class VARCHAR(50),
    sector VARCHAR(50),
    market_cap_tier VARCHAR(20),
    
    -- Priority for selection
    priority INT DEFAULT 0,
    
    -- Compliance
    wash_sale_safe BOOLEAN DEFAULT true,
    esg_score_substitute INT,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT substitute_asset_mappings_tenant_id_fk FOREIGN KEY (tenant_id) 
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT substitute_asset_mappings_unique 
        UNIQUE (tenant_id, original_ticker, substitute_ticker)
);

CREATE INDEX idx_substitute_assets_lookup ON substitute_asset_mappings(tenant_id, original_ticker);

-- ============================================================================
-- DRIFT SNAPSHOTS TABLE
-- ============================================================================
-- Historical drift tracking for analysis and prediction

CREATE TABLE IF NOT EXISTS drift_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    portfolio_id UUID NOT NULL,
    
    -- Snapshot time
    snapshot_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Drift metrics
    total_drift_pct DECIMAL(8,4) NOT NULL,
    tracking_error_pct DECIMAL(8,4),
    max_asset_drift_pct DECIMAL(8,4),
    
    -- Breakdown by asset class
    drift_by_asset_class JSONB,
    
    -- Top contributors
    top_overweight JSONB,
    top_underweight JSONB,
    
    -- Market context
    market_volatility DECIMAL(8,4),
    vix_level DECIMAL(8,2),
    
    -- Action taken (if any)
    triggered_rebalance BOOLEAN DEFAULT false,
    rebalance_id UUID,
    
    -- Constraints
    CONSTRAINT drift_snapshots_tenant_id_fk FOREIGN KEY (tenant_id) 
        REFERENCES tenants(id) ON DELETE CASCADE
);

-- Time-series index for efficient queries
CREATE INDEX idx_drift_snapshots_time ON drift_snapshots(tenant_id, portfolio_id, snapshot_at DESC);

-- ============================================================================
-- FUNCTIONS & TRIGGERS
-- ============================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply to all tables
CREATE TRIGGER update_rule_definitions_updated_at
    BEFORE UPDATE ON rule_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_global_calendars_updated_at
    BEFORE UPDATE ON global_calendars
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_trigger_definitions_updated_at
    BEFORE UPDATE ON trigger_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cppi_floors_updated_at
    BEFORE UPDATE ON cppi_floors
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- SEED DATA: US NYSE Calendar 2024
-- ============================================================================

INSERT INTO global_calendars (tenant_id, calendar_code, region, year, holidays, timezone)
SELECT 
    t.id,
    'US_NYSE',
    'US',
    2024,
    '[
        {"date": "2024-01-01", "name": "New Year''s Day"},
        {"date": "2024-01-15", "name": "Martin Luther King Jr. Day"},
        {"date": "2024-02-19", "name": "Presidents'' Day"},
        {"date": "2024-03-29", "name": "Good Friday"},
        {"date": "2024-05-27", "name": "Memorial Day"},
        {"date": "2024-06-19", "name": "Juneteenth"},
        {"date": "2024-07-04", "name": "Independence Day"},
        {"date": "2024-09-02", "name": "Labor Day"},
        {"date": "2024-11-28", "name": "Thanksgiving Day"},
        {"date": "2024-12-25", "name": "Christmas Day"}
    ]'::jsonb,
    'America/New_York'
FROM tenants t
WHERE NOT EXISTS (
    SELECT 1 FROM global_calendars gc 
    WHERE gc.tenant_id = t.id AND gc.calendar_code = 'US_NYSE' AND gc.year = 2024
);

-- ============================================================================
-- SEED DATA: Sample TLH Rule Definition
-- ============================================================================

INSERT INTO rule_definitions (
    tenant_id, 
    rule_id, 
    type, 
    version, 
    name, 
    description, 
    jurisdiction,
    parameters,
    expression,
    scoring_formula,
    wash_sale_config,
    substitute_asset_rules,
    active
)
SELECT 
    t.id,
    'TLH_STANDARD_US_V1',
    'tax_loss_harvesting',
    '1.0.0',
    'Standard US Tax-Loss Harvesting',
    'Identifies tax-loss harvesting opportunities for US taxable accounts with wash sale protection',
    'US',
    '{
        "min_loss_percentage": 10,
        "min_loss_amount_usd": 1000,
        "holding_period_days": 0,
        "long_term_threshold_days": 366,
        "annual_loss_limit_usd": 3000,
        "carryforward_enabled": true,
        "estimated_tax_rate": 0.35,
        "transaction_cost_threshold_usd": 50
    }'::jsonb,
    'input.unrealized_loss_pct >= params.min_loss_percentage && input.unrealized_loss_usd >= params.min_loss_amount_usd && input.account_type == ''TAXABLE'' && !isInWashSaleWindow(input.household_id, input.ticker)',
    '(input.unrealized_loss_usd * params.estimated_tax_rate) - (input.estimated_transaction_cost * 2)',
    '{
        "enabled": true,
        "window_days_before": 30,
        "window_days_after": 30,
        "check_household": true,
        "check_ira": true
    }'::jsonb,
    '{
        "enabled": true,
        "min_correlation": 0.90,
        "sector_match_required": true,
        "wait_period_for_repurchase_days": 31
    }'::jsonb,
    true
FROM tenants t
WHERE NOT EXISTS (
    SELECT 1 FROM rule_definitions rd 
    WHERE rd.tenant_id = t.id AND rd.rule_id = 'TLH_STANDARD_US_V1'
)
LIMIT 1;  -- Only insert for one tenant as seed

-- ============================================================================
-- SEED DATA: Common Substitute Asset Mappings
-- ============================================================================

INSERT INTO substitute_asset_mappings (tenant_id, original_ticker, substitute_ticker, correlation, asset_class, sector, priority, wash_sale_safe)
SELECT 
    t.id,
    original_ticker,
    substitute_ticker,
    correlation,
    asset_class,
    sector,
    priority,
    true
FROM tenants t
CROSS JOIN (VALUES
    ('SPY', 'IVV', 0.9998, 'Equity', 'Large Cap Blend', 1),
    ('SPY', 'VOO', 0.9997, 'Equity', 'Large Cap Blend', 2),
    ('IVV', 'SPY', 0.9998, 'Equity', 'Large Cap Blend', 1),
    ('IVV', 'VOO', 0.9996, 'Equity', 'Large Cap Blend', 2),
    ('VOO', 'SPY', 0.9997, 'Equity', 'Large Cap Blend', 1),
    ('VOO', 'IVV', 0.9996, 'Equity', 'Large Cap Blend', 2),
    ('QQQ', 'QQQM', 0.9999, 'Equity', 'Large Cap Growth', 1),
    ('QQQ', 'ONEQ', 0.9850, 'Equity', 'Large Cap Growth', 2),
    ('VTI', 'ITOT', 0.9995, 'Equity', 'Total Market', 1),
    ('VTI', 'SPTM', 0.9990, 'Equity', 'Total Market', 2),
    ('AGG', 'BND', 0.9980, 'Fixed Income', 'Aggregate Bond', 1),
    ('AGG', 'SCHZ', 0.9970, 'Fixed Income', 'Aggregate Bond', 2),
    ('BND', 'AGG', 0.9980, 'Fixed Income', 'Aggregate Bond', 1),
    ('BND', 'SCHZ', 0.9975, 'Fixed Income', 'Aggregate Bond', 2),
    ('VEA', 'IEFA', 0.9985, 'Equity', 'International Developed', 1),
    ('VEA', 'EFA', 0.9900, 'Equity', 'International Developed', 2),
    ('VWO', 'IEMG', 0.9970, 'Equity', 'Emerging Markets', 1),
    ('VWO', 'EEM', 0.9850, 'Equity', 'Emerging Markets', 2)
) AS subs(original_ticker, substitute_ticker, correlation, asset_class, sector, priority)
WHERE NOT EXISTS (
    SELECT 1 FROM substitute_asset_mappings sam 
    WHERE sam.tenant_id = t.id 
    AND sam.original_ticker = subs.original_ticker 
    AND sam.substitute_ticker = subs.substitute_ticker
)
LIMIT 100;  -- Limit seed data



-- ============================================================================
-- ROLLBACK SCRIPT (for reference)
-- ============================================================================
-- DROP TABLE IF EXISTS drift_snapshots CASCADE;
-- DROP TABLE IF EXISTS substitute_asset_mappings CASCADE;
-- DROP TABLE IF EXISTS rule_evaluation_history CASCADE;
-- DROP TABLE IF EXISTS cppi_floors CASCADE;
-- DROP TABLE IF EXISTS trigger_definitions CASCADE;
-- DROP TABLE IF EXISTS global_calendars CASCADE;
-- DROP TABLE IF EXISTS rule_definitions CASCADE;
-- DROP FUNCTION IF EXISTS update_updated_at_column();
