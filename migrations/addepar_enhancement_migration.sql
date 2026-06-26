-- ============================================================================
-- MIGRATION: Addepar Competitive Wealth Management - Enhancement
-- Database: wealth_app
-- Author: System Architect
-- Date: October 29, 2025
-- Description: Completes Addepar model integration with computed views,
--              functions, and Hasura-ready configuration
-- ============================================================================

-- ============================================================================
-- PART 1: VERIFY ENUMS EXIST
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE ownership_type AS ENUM ('PERCENT_BASED', 'SHARE_BASED', 'VALUE_BASED');
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE entity_status AS ENUM ('ACTIVE', 'INACTIVE', 'CLOSED', 'PENDING', 'SUSPENDED');
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE entity_category AS ENUM ('ASSET', 'LIABILITY', 'ENTITY', 'CONTAINER');
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE position_status AS ENUM ('ACTIVE', 'CLOSED', 'PENDING', 'TRANSFERRED');
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE transaction_type AS ENUM ('BUY', 'SELL', 'DIVIDEND', 'SPLIT', 'TRANSFER', 'FEE', 'INTEREST', 'TRANSFER_IN', 'TRANSFER_OUT');
EXCEPTION WHEN duplicate_object THEN
    NULL;
END $$;

-- ============================================================================
-- PART 2: ENHANCE model_type_definitions TABLE
-- ============================================================================

-- Add missing columns if they don't exist
ALTER TABLE model_type_definitions 
    ADD COLUMN IF NOT EXISTS category entity_category DEFAULT 'ASSET';

ALTER TABLE model_type_definitions 
    ADD COLUMN IF NOT EXISTS required_fields TEXT[] DEFAULT '{}';

ALTER TABLE model_type_definitions 
    ADD COLUMN IF NOT EXISTS optional_fields TEXT[] DEFAULT '{}';

ALTER TABLE model_type_definitions 
    ADD COLUMN IF NOT EXISTS sort_order INT DEFAULT 100;

ALTER TABLE model_type_definitions 
    ADD COLUMN IF NOT EXISTS is_system BOOLEAN DEFAULT FALSE;

ALTER TABLE model_type_definitions 
    ADD COLUMN IF NOT EXISTS is_custom BOOLEAN DEFAULT FALSE;

-- ============================================================================
-- PART 3: ENHANCE entities TABLE
-- ============================================================================

ALTER TABLE entities 
    ADD COLUMN IF NOT EXISTS legacy_client_id UUID;

ALTER TABLE entities 
    ADD COLUMN IF NOT EXISTS legacy_household_id UUID;

ALTER TABLE entities 
    ADD COLUMN IF NOT EXISTS source_system VARCHAR(50) DEFAULT 'internal';

ALTER TABLE entities 
    ADD COLUMN IF NOT EXISTS external_id VARCHAR(100);

ALTER TABLE entities 
    ADD COLUMN IF NOT EXISTS created_by UUID;

ALTER TABLE entities 
    ADD COLUMN IF NOT EXISTS updated_by UUID;

ALTER TABLE entities 
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Add foreign keys
ALTER TABLE entities 
    ADD CONSTRAINT fk_entities_tenant_id FOREIGN KEY (tenant_id) 
    REFERENCES organizations(id) ON DELETE CASCADE;

ALTER TABLE entities 
    ADD CONSTRAINT fk_entities_model_type FOREIGN KEY (model_type) 
    REFERENCES model_type_definitions(code);

ALTER TABLE entities 
    ADD CONSTRAINT fk_entities_created_by FOREIGN KEY (created_by) 
    REFERENCES users(id);

ALTER TABLE entities 
    ADD CONSTRAINT fk_entities_updated_by FOREIGN KEY (updated_by) 
    REFERENCES users(id);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_entities_model_type ON entities(model_type);
CREATE INDEX IF NOT EXISTS idx_entities_tenant ON entities(tenant_id);
CREATE INDEX IF NOT EXISTS idx_entities_status ON entities(status) WHERE status = 'ACTIVE';
CREATE INDEX IF NOT EXISTS idx_entities_external_id ON entities(external_id) WHERE external_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_entities_ticker ON entities(ticker) WHERE ticker IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_entities_cusip ON entities(cusip) WHERE cusip IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_entities_legacy_client ON entities(legacy_client_id) WHERE legacy_client_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_entities_deleted ON entities(deleted_at) WHERE deleted_at IS NULL;

-- ============================================================================
-- PART 4: ENHANCE entity_attributes TABLE
-- ============================================================================

ALTER TABLE entity_attributes 
    ADD COLUMN IF NOT EXISTS created_by UUID;

ALTER TABLE entity_attributes 
    ADD CONSTRAINT fk_entity_attrs_created_by FOREIGN KEY (created_by) 
    REFERENCES users(id);

CREATE INDEX IF NOT EXISTS idx_entity_attrs_entity ON entity_attributes(entity_id);
CREATE INDEX IF NOT EXISTS idx_entity_attrs_valid ON entity_attributes(entity_id, valid_from, valid_to);
CREATE INDEX IF NOT EXISTS idx_entity_attrs_gin ON entity_attributes USING GIN (attributes);

-- ============================================================================
-- PART 5: ENHANCE positions TABLE
-- ============================================================================

ALTER TABLE positions 
    ADD COLUMN IF NOT EXISTS legacy_holding_id UUID;

ALTER TABLE positions 
    ADD COLUMN IF NOT EXISTS position_type VARCHAR(50);

ALTER TABLE positions 
    ADD COLUMN IF NOT EXISTS notes TEXT;

ALTER TABLE positions 
    ADD COLUMN IF NOT EXISTS created_by UUID;

ALTER TABLE positions 
    ADD CONSTRAINT fk_positions_owner_id FOREIGN KEY (owner_id) 
    REFERENCES entities(id) ON DELETE CASCADE;

ALTER TABLE positions 
    ADD CONSTRAINT fk_positions_owned_id FOREIGN KEY (owned_id) 
    REFERENCES entities(id) ON DELETE CASCADE;

ALTER TABLE positions 
    ADD CONSTRAINT fk_positions_tenant_id FOREIGN KEY (tenant_id) 
    REFERENCES organizations(id) ON DELETE CASCADE;

ALTER TABLE positions 
    ADD CONSTRAINT fk_positions_created_by FOREIGN KEY (created_by) 
    REFERENCES users(id);

ALTER TABLE positions 
    ADD CONSTRAINT fk_positions_legacy_holding_id FOREIGN KEY (legacy_holding_id) 
    REFERENCES portfolio_holdings(id);

-- Add check constraint
ALTER TABLE positions 
    ADD CONSTRAINT check_positions_owner_owned CHECK (owner_id != owned_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_positions_unique ON positions(owner_id, owned_id, as_of_date) 
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_positions_owner ON positions(owner_id);
CREATE INDEX IF NOT EXISTS idx_positions_owned ON positions(owned_id);
CREATE INDEX IF NOT EXISTS idx_positions_tenant ON positions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_positions_active ON positions(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_positions_status ON positions(status);
CREATE INDEX IF NOT EXISTS idx_positions_as_of_date ON positions(as_of_date DESC);

-- ============================================================================
-- PART 6: ENHANCE position_transactions TABLE
-- ============================================================================

ALTER TABLE position_transactions 
    ADD COLUMN IF NOT EXISTS cost_basis NUMERIC(18, 2);

ALTER TABLE position_transactions 
    ADD COLUMN IF NOT EXISTS created_by UUID;

ALTER TABLE position_transactions 
    ADD CONSTRAINT fk_pos_trans_position_id FOREIGN KEY (position_id) 
    REFERENCES positions(id) ON DELETE CASCADE;

ALTER TABLE position_transactions 
    ADD CONSTRAINT fk_pos_trans_entity_id FOREIGN KEY (entity_id) 
    REFERENCES entities(id) ON DELETE CASCADE;

ALTER TABLE position_transactions 
    ADD CONSTRAINT fk_pos_trans_tenant_id FOREIGN KEY (tenant_id) 
    REFERENCES organizations(id) ON DELETE CASCADE;

ALTER TABLE position_transactions 
    ADD CONSTRAINT fk_pos_trans_created_by FOREIGN KEY (created_by) 
    REFERENCES users(id);

CREATE INDEX IF NOT EXISTS idx_pos_trans_position ON position_transactions(position_id);
CREATE INDEX IF NOT EXISTS idx_pos_trans_entity ON position_transactions(entity_id);
CREATE INDEX IF NOT EXISTS idx_pos_trans_trade_date ON position_transactions(trade_date DESC);
CREATE INDEX IF NOT EXISTS idx_pos_trans_tenant ON position_transactions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_pos_trans_type ON position_transactions(transaction_type);

-- ============================================================================
-- PART 7: ENHANCE entity_market_data TABLE
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_entity_market_data_entity ON entity_market_data(entity_id);
CREATE INDEX IF NOT EXISTS idx_entity_market_data_date ON entity_market_data(as_of_date DESC);
CREATE INDEX IF NOT EXISTS idx_entity_market_data_time ON entity_market_data(as_of_time DESC);

-- ============================================================================
-- PART 8: ENSURE UPDATE TRIGGERS EXIST
-- ============================================================================

-- Create update_updated_at_column function if it doesn't exist
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
DROP TRIGGER IF EXISTS update_entities_updated_at ON entities;
CREATE TRIGGER update_entities_updated_at 
    BEFORE UPDATE ON entities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_entity_attributes_updated_at ON entity_attributes;
CREATE TRIGGER update_entity_attributes_updated_at 
    BEFORE UPDATE ON entity_attributes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_positions_updated_at ON positions;
CREATE TRIGGER update_positions_updated_at 
    BEFORE UPDATE ON positions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_model_type_defs_updated_at ON model_type_definitions;
CREATE TRIGGER update_model_type_defs_updated_at 
    BEFORE UPDATE ON model_type_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- PART 9: SEED/UPDATE STANDARD ADDEPAR MODEL TYPES
-- ============================================================================

-- Delete existing Addepar standard types to ensure fresh seeding
DELETE FROM model_type_definitions WHERE is_system = TRUE AND is_custom = FALSE;

-- Insert standard Addepar model types
INSERT INTO model_type_definitions (
    code, display_name, category, attribute_schema, 
    icon, color, sort_order, is_system, is_custom, is_active
) VALUES
-- Assets - Securities
('BOND', 'Bond', 'ASSET'::entity_category, 
 '{"maturity_date": "date", "coupon_rate": "number", "par_value": "number", "credit_rating": "string", "issuer": "string", "bond_type": "string"}'::jsonb,
 'shield', '#3b82f6', 10, TRUE, FALSE, TRUE),

('STOCK', 'Stock', 'ASSET'::entity_category,
 '{"sector": "string", "industry": "string", "market_cap": "number", "pe_ratio": "number", "dividend_yield": "number"}'::jsonb,
 'trending-up', '#10b981', 20, TRUE, FALSE, TRUE),

('ETF', 'ETF', 'ASSET'::entity_category,
 '{"expense_ratio": "number", "aum": "number", "inception_date": "date", "fund_family": "string", "index_tracked": "string"}'::jsonb,
 'layers', '#8b5cf6', 30, TRUE, FALSE, TRUE),

('MUTUAL_FUND', 'Mutual Fund', 'ASSET'::entity_category,
 '{"expense_ratio": "number", "load": "number", "nav": "number", "fund_family": "string", "investment_style": "string"}'::jsonb,
 'briefcase', '#f59e0b', 40, TRUE, FALSE, TRUE),

('CASH', 'Cash', 'ASSET'::entity_category,
 '{"account_number": "string", "bank": "string", "interest_rate": "number", "fdic_insured": "boolean"}'::jsonb,
 'dollar-sign', '#22c55e', 50, TRUE, FALSE, TRUE),

('REAL_ESTATE', 'Real Estate', 'ASSET'::entity_category,
 '{"address": "string", "property_type": "string", "appraised_value": "number", "purchase_date": "date", "square_feet": "number"}'::jsonb,
 'home', '#ef4444', 60, TRUE, FALSE, TRUE),

('PRIVATE_EQUITY', 'Private Equity', 'ASSET'::entity_category,
 '{"fund_name": "string", "vintage_year": "number", "commitment": "number", "called_capital": "number", "distributed_capital": "number"}'::jsonb,
 'lock', '#6366f1', 70, TRUE, FALSE, TRUE),

('CRYPTOCURRENCY', 'Cryptocurrency', 'ASSET'::entity_category,
 '{"blockchain": "string", "wallet_address": "string", "protocol": "string"}'::jsonb,
 'bitcoin', '#f97316', 75, TRUE, FALSE, TRUE),

('OPTION', 'Option', 'ASSET'::entity_category,
 '{"strike_price": "number", "expiration_date": "date", "option_type": "string", "underlying_symbol": "string"}'::jsonb,
 'zap', '#14b8a6', 80, TRUE, FALSE, TRUE),

-- Entities - Persons & Organizations
('CLIENT', 'Client', 'ENTITY'::entity_category,
 '{"first_name": "string", "last_name": "string", "email": "string", "phone": "string", "date_of_birth": "date", "ssn": "string"}'::jsonb,
 'user', '#06b6d4', 100, TRUE, FALSE, TRUE),

('HOUSEHOLD', 'Household', 'ENTITY'::entity_category,
 '{"household_name": "string", "primary_contact": "string", "address": "string", "tax_filing_status": "string"}'::jsonb,
 'users', '#8b5cf6', 110, TRUE, FALSE, TRUE),

('TRUST', 'Trust', 'ENTITY'::entity_category,
 '{"trust_name": "string", "trustee": "string", "established_date": "date", "trust_type": "string", "tax_id": "string"}'::jsonb,
 'shield-check', '#ec4899', 120, TRUE, FALSE, TRUE),

('INSTITUTION', 'Institution', 'ENTITY'::entity_category,
 '{"institution_name": "string", "institution_type": "string", "tax_id": "string"}'::jsonb,
 'building', '#64748b', 150, TRUE, FALSE, TRUE),

-- Containers
('ACCOUNT', 'Account', 'CONTAINER'::entity_category,
 '{"account_type": "string", "custodian": "string", "account_number": "string", "tax_status": "string"}'::jsonb,
 'credit-card', '#14b8a6', 130, TRUE, FALSE, TRUE),

('PORTFOLIO', 'Portfolio', 'CONTAINER'::entity_category,
 '{"portfolio_name": "string", "strategy": "string", "benchmark": "string", "inception_date": "date"}'::jsonb,
 'pie-chart', '#f97316', 140, TRUE, FALSE, TRUE)

ON CONFLICT (code) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    category = EXCLUDED.category,
    attribute_schema = EXCLUDED.attribute_schema,
    icon = EXCLUDED.icon,
    color = EXCLUDED.color,
    sort_order = EXCLUDED.sort_order,
    is_system = EXCLUDED.is_system,
    updated_at = CURRENT_TIMESTAMP;

-- ============================================================================
-- PART 10: CREATE COMPUTED VIEWS (Hasura-Ready)
-- ============================================================================

-- Portfolio holdings view
CREATE OR REPLACE VIEW v_entity_holdings AS
SELECT 
    p.id AS position_id,
    p.owner_id AS portfolio_entity_id,
    owner.display_name AS portfolio_name,
    owner.model_type AS portfolio_type,
    p.owned_id AS holding_entity_id,
    holding.model_type AS holding_type,
    holding.display_name AS holding_name,
    holding.ticker,
    holding.cusip,
    p.shares,
    p.units,
    p.ownership_percentage,
    p.cost_basis,
    p.average_cost_per_unit,
    emd.current_price,
    CASE 
        WHEN holding.ownership_type = 'SHARE_BASED' 
        THEN p.shares * COALESCE(emd.current_price, p.average_market_price)
        WHEN holding.ownership_type = 'VALUE_BASED'
        THEN p.market_value
        ELSE p.units * COALESCE(emd.current_price, p.average_market_price)
    END AS current_market_value,
    CASE 
        WHEN p.cost_basis IS NOT NULL
        THEN (CASE 
            WHEN holding.ownership_type = 'SHARE_BASED' 
            THEN (p.shares * COALESCE(emd.current_price, p.average_market_price)) - p.cost_basis
            ELSE p.market_value - p.cost_basis
        END)
        ELSE NULL
    END AS unrealized_gain_loss,
    CASE 
        WHEN p.cost_basis IS NOT NULL AND p.cost_basis > 0
        THEN ((CASE 
            WHEN holding.ownership_type = 'SHARE_BASED' 
            THEN (p.shares * COALESCE(emd.current_price, p.average_market_price))
            ELSE p.market_value
        END - p.cost_basis) / p.cost_basis) * 100
        ELSE NULL
    END AS return_pct,
    p.as_of_date,
    p.status,
    p.tenant_id
FROM positions p
JOIN entities owner ON p.owner_id = owner.id
JOIN entities holding ON p.owned_id = holding.id
LEFT JOIN entity_market_data emd ON holding.id = emd.entity_id AND emd.as_of_date = p.as_of_date
WHERE p.is_active = TRUE AND p.status = 'ACTIVE'::position_status;

-- Portfolio summary view
CREATE OR REPLACE VIEW v_entity_portfolio_summary AS
SELECT 
    portfolio_entity_id,
    portfolio_name,
    portfolio_type,
    tenant_id,
    COUNT(*) AS total_positions,
    SUM(current_market_value) AS total_market_value,
    SUM(cost_basis) AS total_cost_basis,
    SUM(unrealized_gain_loss) AS total_unrealized_gain_loss,
    CASE 
        WHEN SUM(cost_basis) > 0
        THEN (SUM(unrealized_gain_loss) / SUM(cost_basis)) * 100
        ELSE NULL
    END AS portfolio_return_pct,
    as_of_date
FROM v_entity_holdings
GROUP BY portfolio_entity_id, portfolio_name, portfolio_type, tenant_id, as_of_date;

-- Entity positions hierarchy
CREATE OR REPLACE VIEW v_entity_positions_hierarchy AS
SELECT 
    p.id AS position_id,
    p.owner_id,
    owner.model_type AS owner_type,
    owner.display_name AS owner_name,
    p.owned_id,
    owned.model_type AS owned_type,
    owned.display_name AS owned_name,
    owned.ticker,
    p.shares,
    p.units,
    p.ownership_percentage,
    p.market_value,
    p.cost_basis,
    p.status,
    p.as_of_date,
    p.incepting_date,
    p.closing_date,
    p.tenant_id,
    CASE 
        WHEN p.shares IS NOT NULL THEN 'SHARE_BASED'
        WHEN p.ownership_percentage IS NOT NULL THEN 'PERCENT_BASED'
        ELSE 'VALUE_BASED'
    END AS ownership_mode
FROM positions p
JOIN entities owner ON p.owner_id = owner.id
JOIN entities owned ON p.owned_id = owned.id
WHERE p.is_active = TRUE;

-- ============================================================================
-- PART 11: CREATE HELPER FUNCTIONS
-- ============================================================================

-- Function to get entity market value
CREATE OR REPLACE FUNCTION get_entity_market_value(entity_id UUID, as_of_date DATE DEFAULT CURRENT_DATE)
RETURNS NUMERIC(18, 2) AS $$
DECLARE
    market_value NUMERIC(18, 2);
BEGIN
    SELECT COALESCE(emd.current_price, 0) * 
           COALESCE((SELECT shares FROM positions WHERE owned_id = $1 AND as_of_date = $2), 1)
    INTO market_value
    FROM entity_market_data emd
    WHERE emd.entity_id = $1 AND emd.as_of_date = $2;
    
    RETURN COALESCE(market_value, 0);
END;
$$ LANGUAGE plpgsql;

-- Function to calculate portfolio performance
CREATE OR REPLACE FUNCTION calculate_portfolio_performance(
    portfolio_id UUID,
    as_of_date DATE DEFAULT CURRENT_DATE
)
RETURNS TABLE(
    total_value NUMERIC(18, 2),
    total_cost_basis NUMERIC(18, 2),
    unrealized_gain NUMERIC(18, 2),
    total_return_pct NUMERIC(10, 4)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        SUM(current_market_value)::NUMERIC(18, 2),
        SUM(cost_basis)::NUMERIC(18, 2),
        SUM(unrealized_gain_loss)::NUMERIC(18, 2),
        CASE 
            WHEN SUM(cost_basis) > 0 THEN (SUM(unrealized_gain_loss) / SUM(cost_basis) * 100)::NUMERIC(10, 4)
            ELSE NULL
        END
    FROM v_entity_holdings
    WHERE portfolio_entity_id = $1 AND as_of_date = $2;
END;
$$ LANGUAGE plpgsql;

-- Function to find or create entity by identifier
CREATE OR REPLACE FUNCTION find_or_create_entity(
    p_ticker VARCHAR(20),
    p_cusip VARCHAR(9),
    p_model_type VARCHAR(50),
    p_tenant_id UUID,
    p_display_name TEXT DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_entity_id UUID;
BEGIN
    -- Try to find existing entity
    SELECT id INTO v_entity_id
    FROM entities
    WHERE 
        model_type = p_model_type 
        AND tenant_id = p_tenant_id
        AND (ticker = p_ticker OR cusip = p_cusip)
        AND status = 'ACTIVE'::entity_status
    LIMIT 1;
    
    -- If not found, create new entity
    IF v_entity_id IS NULL THEN
        INSERT INTO entities (
            model_type,
            tenant_id,
            original_name,
            display_name,
            ticker,
            cusip,
            ownership_type,
            status
        ) VALUES (
            p_model_type,
            p_tenant_id,
            COALESCE(p_display_name, COALESCE(p_ticker, p_cusip, 'Unknown')),
            COALESCE(p_display_name, COALESCE(p_ticker, p_cusip, 'Unknown')),
            p_ticker,
            p_cusip,
            'SHARE_BASED'::ownership_type,
            'ACTIVE'::entity_status
        ) RETURNING id INTO v_entity_id;
    END IF;
    
    RETURN v_entity_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- PART 12: ENABLE ROW-LEVEL SECURITY (Hasura Compatible)
-- ============================================================================

ALTER TABLE entities ENABLE ROW LEVEL SECURITY;
ALTER TABLE positions ENABLE ROW LEVEL SECURITY;
ALTER TABLE position_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE entity_attributes ENABLE ROW LEVEL SECURITY;
ALTER TABLE entity_market_data ENABLE ROW LEVEL SECURITY;

-- Create RLS policies
DROP POLICY IF EXISTS entities_tenant_isolation ON entities;
CREATE POLICY entities_tenant_isolation ON entities
    FOR SELECT
    USING (
        tenant_id = current_setting('hasura.user.x-hasura-tenant-id', TRUE)::UUID
        OR current_setting('hasura.user.x-hasura-admin-secret', TRUE) IS NOT NULL
    );

DROP POLICY IF EXISTS positions_tenant_isolation ON positions;
CREATE POLICY positions_tenant_isolation ON positions
    FOR SELECT
    USING (
        tenant_id = current_setting('hasura.user.x-hasura-tenant-id', TRUE)::UUID
        OR current_setting('hasura.user.x-hasura-admin-secret', TRUE) IS NOT NULL
    );

DROP POLICY IF EXISTS position_transactions_tenant_isolation ON position_transactions;
CREATE POLICY position_transactions_tenant_isolation ON position_transactions
    FOR SELECT
    USING (
        tenant_id = current_setting('hasura.user.x-hasura-tenant-id', TRUE)::UUID
        OR current_setting('hasura.user.x-hasura-admin-secret', TRUE) IS NOT NULL
    );

-- ============================================================================
-- PART 13: DATA MIGRATION HELPERS (Optional - Run Separately)
-- ============================================================================

-- Function to migrate securities to entities
CREATE OR REPLACE FUNCTION migrate_securities_to_entities()
RETURNS TABLE(migrated_count INT, total_securities INT) AS $$
DECLARE
    v_migrated_count INT := 0;
    v_total_count INT;
BEGIN
    SELECT COUNT(*) INTO v_total_count FROM securities;
    
    INSERT INTO entities (
        model_type,
        tenant_id,
        original_name,
        display_name,
        ticker,
        cusip,
        isin,
        currency_factor,
        ownership_type,
        status,
        legacy_security_id,
        source_system,
        created_at,
        updated_at
    )
    SELECT
        CASE s.security_type
            WHEN 'EQUITY' THEN 'STOCK'
            WHEN 'FIXED_INCOME' THEN 'BOND'
            WHEN 'ETF' THEN 'ETF'
            WHEN 'MUTUAL_FUND' THEN 'MUTUAL_FUND'
            WHEN 'CASH' THEN 'CASH'
            ELSE 'STOCK'
        END,
        org.id,
        s.security_name,
        s.security_name,
        s.symbol,
        s.cusip,
        s.isin,
        s.currency,
        'SHARE_BASED'::ownership_type,
        CASE WHEN s.is_active THEN 'ACTIVE'::entity_status ELSE 'INACTIVE'::entity_status END,
        s.id,
        COALESCE(s.data_source, 'internal'),
        s.created_at,
        s.updated_at
    FROM securities s
    CROSS JOIN (SELECT id FROM organizations LIMIT 1) org
    WHERE s.id NOT IN (
        SELECT legacy_security_id FROM entities WHERE legacy_security_id IS NOT NULL
    )
    ON CONFLICT DO NOTHING;
    
    GET DIAGNOSTICS v_migrated_count = ROW_COUNT;
    
    RETURN QUERY SELECT v_migrated_count, v_total_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- PART 14: COMMENTS FOR DOCUMENTATION
-- ============================================================================

COMMENT ON TABLE entities IS 'Polymorphic entity table supporting Addepar model types (BOND, STOCK, CLIENT, HOUSEHOLD, etc.)';
COMMENT ON TABLE positions IS 'Ownership graph representing owner → owned relationships with multi-tenant isolation';
COMMENT ON TABLE position_transactions IS 'Transaction flows (buy/sell/transfer) for positions tracking cost basis and tax lots';
COMMENT ON TABLE model_type_definitions IS 'Addepar-compatible model type definitions with JSON schema validation';
COMMENT ON TABLE entity_attributes IS 'JSONB storage for type-specific attributes with versioning support';
COMMENT ON TABLE entity_market_data IS 'Real-time market data for entities with historical tracking';

COMMENT ON COLUMN entities.model_type IS 'Addepar model_type discriminator (BOND, STOCK, CLIENT, HOUSEHOLD, etc.)';
COMMENT ON COLUMN entities.ownership_type IS 'How ownership is measured: PERCENT_BASED, SHARE_BASED, or VALUE_BASED';
COMMENT ON COLUMN positions.owner_id IS 'Entity that owns this position (e.g., Portfolio, Account, Household)';
COMMENT ON COLUMN positions.owned_id IS 'Entity that is owned (e.g., Stock, Bond, ETF, Cash Account)';

COMMENT ON VIEW v_entity_holdings IS 'Portfolio holdings with real-time market values and unrealized gains/losses (Hasura-compatible)';
COMMENT ON VIEW v_entity_portfolio_summary IS 'Portfolio summary aggregations by entity, optimized for dashboards';
COMMENT ON VIEW v_entity_positions_hierarchy IS 'Complete position ownership hierarchy with all metadata';

COMMENT ON FUNCTION get_entity_market_value(UUID, DATE) IS 'Calculate current market value of entity for given date';
COMMENT ON FUNCTION calculate_portfolio_performance(UUID, DATE) IS 'Calculate total return, gain/loss, and performance metrics for portfolio';
COMMENT ON FUNCTION find_or_create_entity(VARCHAR, VARCHAR, VARCHAR, UUID, TEXT) IS 'Find existing entity or create new one by ticker/CUSIP';

-- ============================================================================
-- PART 15: VERIFICATION QUERIES
-- ============================================================================

-- Check model types
SELECT COUNT(*) as addepar_model_types FROM model_type_definitions WHERE is_system = TRUE;

-- Check entities
SELECT model_type, COUNT(*) as count, COUNT(DISTINCT tenant_id) as tenants
FROM entities
GROUP BY model_type
ORDER BY count DESC;

-- Check positions
SELECT 
    COUNT(*) as total_positions,
    COUNT(DISTINCT owner_id) as unique_owners,
    COUNT(DISTINCT owned_id) as unique_holdings,
    SUM(market_value) as total_market_value
FROM positions
WHERE is_active = TRUE;

-- ============================================================================
-- MIGRATION COMPLETE
-- ============================================================================
-- Run the following to test:
-- SELECT * FROM v_entity_holdings LIMIT 10;
-- SELECT * FROM v_entity_portfolio_summary LIMIT 10;
-- SELECT * FROM calculate_portfolio_performance('your-portfolio-id'::uuid);
