-- Tax optimization opportunities
CREATE TABLE IF NOT EXISTS tax_optimization_opportunities (
    opportunity_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    opportunity_type VARCHAR(100) NOT NULL, -- 'TAX_LOSS_HARVEST', 'ROTH_CONVERSION', 'CHARITABLE_DONATION', 'ASSET_LOCATION'
    detected_date DATE DEFAULT CURRENT_DATE,
    
    -- Opportunity details
    estimated_tax_savings DECIMAL(10,2),
    implementation_complexity VARCHAR(20), -- 'LOW', 'MEDIUM', 'HIGH'
    time_sensitivity VARCHAR(50), -- 'BEFORE_YEAR_END', 'BEFORE_RMD_AGE', 'ANYTIME'
    
    -- Actions required
    recommended_actions JSONB,
    
    -- Impact analysis
    positions_affected JSONB, -- Array of ticker symbols and amounts
    projected_bracket_impact DECIMAL(5,2), -- Change in tax bracket %
    
    -- Status
    status VARCHAR(50) DEFAULT 'IDENTIFIED', -- 'IDENTIFIED', 'PRESENTED_TO_CLIENT', 'APPROVED', 'IMPLEMENTED', 'DECLINED'
    advisor_notes TEXT,
    client_response TEXT,
    
    -- Execution tracking
    implemented_date DATE,
    actual_tax_savings DECIMAL(10,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tax_opps_client ON tax_optimization_opportunities(client_id);
CREATE INDEX IF NOT EXISTS idx_tax_opps_status ON tax_optimization_opportunities(status);
CREATE INDEX IF NOT EXISTS idx_tax_opps_type ON tax_optimization_opportunities(opportunity_type);

-- Tax lot tracking for loss harvesting
CREATE TABLE IF NOT EXISTS tax_lots (
    lot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL,
    account_id UUID,
    ticker VARCHAR(10) NOT NULL,
    purchase_date DATE NOT NULL,
    quantity DECIMAL(18,6) NOT NULL,
    cost_basis DECIMAL(15,2) NOT NULL,
    current_value DECIMAL(15,2),
    unrealized_gain_loss DECIMAL(15,2),
    holding_period_days INTEGER,
    
    -- Tax treatment
    is_long_term BOOLEAN DEFAULT FALSE,
    is_wash_sale BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tax_lots_client ON tax_lots(client_id);
CREATE INDEX IF NOT EXISTS idx_tax_lots_ticker ON tax_lots(ticker);
CREATE INDEX IF NOT EXISTS idx_tax_lots_unrealized ON tax_lots(unrealized_gain_loss);

-- Client tax profile
CREATE TABLE IF NOT EXISTS client_tax_profiles (
    profile_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL UNIQUE,
    
    -- Current year estimates
    current_year_income DECIMAL(15,2),
    estimated_tax_bracket DECIMAL(5,2), -- e.g., 0.37 for 37%
    filing_status VARCHAR(50), -- 'SINGLE', 'MARRIED_JOINT', 'MARRIED_SEPARATE', 'HEAD_OF_HOUSEHOLD'
    
    -- Historical data
    average_annual_income DECIMAL(15,2),
    average_tax_bracket DECIMAL(5,2),
    
    -- Planning opportunities
    has_traditional_ira BOOLEAN DEFAULT FALSE,
    has_roth_ira BOOLEAN DEFAULT FALSE,
    charitable_intent BOOLEAN DEFAULT FALSE,
    estate_planning_needed BOOLEAN DEFAULT FALSE,
    
    -- RMD tracking
    age INTEGER,
    rmd_start_year INTEGER,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tax_profiles_client ON client_tax_profiles(client_id);
