-- ============================================================================
-- Addepar 49 Model Types Extended Migration
-- ============================================================================
-- This migration seeds all 49 Addepar business entity types with:
-- - Complete model type definitions
-- - Hierarchical metadata
-- - Suggested attributes per type
-- - Extensibility configuration
--
-- Hierarchical Types:
--   Top-Level: household
--   Containers: person_node, trust, managed_partnership, holding_company, sleeve, financial_account
--   Nested Funds: fund, hedge_fund, private_equity_fund
--   Leaf Assets: All remaining (bond, stock, etf, cash, etc.)
--
-- ============================================================================

DO $$ DECLARE
    v_tenant_id UUID := '00000000-0000-0000-0000-000000000000';
    v_org_id UUID := '00000000-0000-0000-0000-000000000001';
BEGIN

-- ============================================================================
-- STEP 1: Ensure ENUM types exist
-- ============================================================================

IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'ownership_type') THEN
    CREATE TYPE ownership_type AS ENUM (
        'PERCENT_BASED',
        'SHARE_BASED',
        'VALUE_BASED'
    );
END IF;

-- ============================================================================
-- STEP 2: Create hierarchy_rules table (if not exists)
-- ============================================================================

IF NOT EXISTS (SELECT 1 FROM information_schema.tables 
    WHERE table_name = 'entity_hierarchy_rules') THEN
    
    CREATE TABLE entity_hierarchy_rules (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        tenant_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
        parent_model_type VARCHAR(255) NOT NULL,
        child_model_type VARCHAR(255) NOT NULL,
        allowed BOOLEAN DEFAULT true NOT NULL,
        ownership_types VARCHAR(100)[] DEFAULT ARRAY['PERCENT_BASED', 'SHARE_BASED', 'VALUE_BASED'],
        max_children INTEGER,
        min_children INTEGER,
        description TEXT,
        is_exclusive BOOLEAN DEFAULT false, -- If true, child can have only this parent type
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        created_by UUID,
        UNIQUE(tenant_id, parent_model_type, child_model_type)
    );

    CREATE INDEX idx_entity_hierarchy_rules_tenant_id ON entity_hierarchy_rules(tenant_id);
    CREATE INDEX idx_entity_hierarchy_rules_parent ON entity_hierarchy_rules(parent_model_type);
    CREATE INDEX idx_entity_hierarchy_rules_child ON entity_hierarchy_rules(child_model_type);
    
END IF;

-- ============================================================================
-- STEP 3: Create model_type_hierarchy_attributes table (metadata about suggested fields)
-- ============================================================================

IF NOT EXISTS (SELECT 1 FROM information_schema.tables 
    WHERE table_name = 'model_type_hierarchy_attributes') THEN
    
    CREATE TABLE model_type_hierarchy_attributes (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        tenant_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
        model_type VARCHAR(255) NOT NULL,
        attribute_key VARCHAR(255) NOT NULL,
        attribute_type VARCHAR(50), -- 'string', 'date', 'numeric', 'boolean', 'enum', 'object', 'array'
        is_required BOOLEAN DEFAULT false,
        is_searchable BOOLEAN DEFAULT false,
        validation_rule JSONB,
        description TEXT,
        priority INTEGER DEFAULT 0, -- Higher = more important in UI
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(tenant_id, model_type, attribute_key)
    );

    CREATE INDEX idx_model_type_hierarchy_attributes_tenant ON model_type_hierarchy_attributes(tenant_id);
    CREATE INDEX idx_model_type_hierarchy_attributes_type ON model_type_hierarchy_attributes(model_type);
    
END IF;

-- ============================================================================
-- STEP 4: Seed all 49 Addepar model types
-- ============================================================================

-- Delete existing if rerunning (idempotent)
DELETE FROM model_type_definitions 
WHERE tenant_id = v_tenant_id 
  AND model_type IN (
    'household', 'person_node', 'prospect', 'managed_partnership', 'holding_company',
    'manager', 'fund', 'trust', 'vehicle', 'financial_account', 'sleeve', 'annuity',
    'art', 'bond', 'car', 'certificate_of_deposit', 'closed_end_fund', 'cmo',
    'collectible', 'convertible_note', 'generic_asset', 'cash', 'digital_asset',
    'etf', 'etn', 'forward_contract', 'futures_contract', 'hedge_fund',
    'historical_segment', 'loan', 'master_limited_partnership', 'money_market_fund',
    'mutual_fund', 'option', 'preferred_stock', 'private_equity_fund', 'private_investment',
    'promissory_note', 'real_estate', 'reit', 'stock', 'structured_product', 'uit',
    'unknown_security', 'venture_capital', 'warrant'
);

-- Insert Container & Structural Types (Hierarchical Owners)
INSERT INTO model_type_definitions (tenant_id, model_type, display_name, ownership_type, description, is_hierarchical, hierarchy_level, created_by)
VALUES
    (v_tenant_id, 'household', 'Household', 'PERCENT_BASED', 'Represents a household portfolio entity. Top-level container in the ownership hierarchy.', true, 0, v_org_id),
    (v_tenant_id, 'person_node', 'Client', 'PERCENT_BASED', 'Represents an individual client or person. Can own financial accounts, trusts, and investments.', true, 1, v_org_id),
    (v_tenant_id, 'prospect', 'Prospect', 'PERCENT_BASED', 'Represents a prospective client not yet onboarded. Transition to person_node upon activation.', true, 1, v_org_id),
    (v_tenant_id, 'trust', 'Trust', 'PERCENT_BASED', 'Represents a trust entity. Legal container for fiduciary relationships.', true, 1, v_org_id),
    (v_tenant_id, 'managed_partnership', 'Managed fund', 'SHARE_BASED', 'Represents a managed fund or partnership. Fund structure for pooled investments.', true, 1, v_org_id),
    (v_tenant_id, 'holding_company', 'Holding company', 'PERCENT_BASED', 'Represents a holding company entity. Corporate structure for multi-level ownership.', true, 1, v_org_id),
    (v_tenant_id, 'manager', 'Manager', 'PERCENT_BASED', 'Represents a manager entity. Can manage portfolios and investments.', true, 1, v_org_id),
    (v_tenant_id, 'vehicle', 'Vehicle', 'PERCENT_BASED', 'Represents a vehicle entity. General-purpose container.', true, 1, v_org_id),
    (v_tenant_id, 'financial_account', 'Holding Account', 'PERCENT_BASED', 'Represents a financial or holding account. Custodial container for cash and securities.', true, 2, v_org_id),
    (v_tenant_id, 'sleeve', 'Sleeve', 'PERCENT_BASED', 'Represents a sleeve in a portfolio. Sub-portfolio for tactical allocation.', true, 2, v_org_id),
    
    -- Fund Types (Can own other funds - nested funds)
    (v_tenant_id, 'fund', 'Private fund', 'VALUE_BASED', 'Represents a private fund. Base type for pooled investment vehicles.', true, 2, v_org_id),
    (v_tenant_id, 'hedge_fund', 'Hedge fund', 'SHARE_BASED', 'Represents a hedge fund. Available only to firms that started using Addepar on or after September 12, 2025.', true, 2, v_org_id),
    (v_tenant_id, 'private_equity_fund', 'Private equity fund', 'SHARE_BASED', 'Represents a private equity fund. Available only to firms that started using Addepar on or after September 12, 2025.', true, 2, v_org_id)
ON CONFLICT DO NOTHING;

-- Insert Asset & Security Types (Leaf Nodes - typically not owners)
INSERT INTO model_type_definitions (tenant_id, model_type, display_name, ownership_type, description, is_hierarchical, hierarchy_level, created_by)
VALUES
    (v_tenant_id, 'annuity', 'Annuity', 'VALUE_BASED', 'Represents an annuity investment. Income product with specific valuation rules.', false, 3, v_org_id),
    (v_tenant_id, 'art', 'Art', 'SHARE_BASED', 'Represents art as an asset. Available only to firms that started using Addepar on or after September 12, 2025.', false, 3, v_org_id),
    (v_tenant_id, 'bond', 'Bond', 'SHARE_BASED', 'Represents a bond investment. Fixed income security with maturity and coupon.', false, 3, v_org_id),
    (v_tenant_id, 'car', 'Car', 'SHARE_BASED', 'Represents a car as an asset. Available only to firms that started using Addepar on or after September 12, 2025.', false, 3, v_org_id),
    (v_tenant_id, 'certificate_of_deposit', 'Certificate of deposit', 'SHARE_BASED', 'Represents a certificate of deposit. Time-bound savings instrument.', false, 3, v_org_id),
    (v_tenant_id, 'closed_end_fund', 'Closed end fund', 'SHARE_BASED', 'Represents a closed-end fund. Fixed number of shares, infrequent trades.', false, 3, v_org_id),
    (v_tenant_id, 'cmo', 'CMO', 'SHARE_BASED', 'Represents a collateralized mortgage obligation. Mortgage-backed security with tranches.', false, 3, v_org_id),
    (v_tenant_id, 'collectible', 'Collectible', 'SHARE_BASED', 'Represents a collectible asset. Available only to firms that started using Addepar on or after September 12, 2025.', false, 3, v_org_id),
    (v_tenant_id, 'convertible_note', 'Convertible note', 'SHARE_BASED', 'Represents a convertible note. Hybrid debt-equity security.', false, 3, v_org_id),
    (v_tenant_id, 'cash', 'Currency', 'SHARE_BASED', 'Represents cash or currency holdings. Base settlement asset.', false, 3, v_org_id),
    (v_tenant_id, 'digital_asset', 'Digital asset', 'SHARE_BASED', 'Represents digital assets like cryptocurrencies. Blockchain-based holdings.', false, 3, v_org_id),
    (v_tenant_id, 'etf', 'ETF', 'SHARE_BASED', 'Represents an exchange-traded fund. Fungible basket of securities.', false, 3, v_org_id),
    (v_tenant_id, 'etn', 'ETN', 'SHARE_BASED', 'Represents an exchange-traded note. Unsecured debt security tracking index.', false, 3, v_org_id),
    (v_tenant_id, 'forward_contract', 'Forward contract', 'SHARE_BASED', 'Represents a forward contract. Custom derivatives with delivery terms.', false, 3, v_org_id),
    (v_tenant_id, 'futures_contract', 'Futures contract', 'SHARE_BASED', 'Represents a futures contract. Standardized exchange-traded derivatives.', false, 3, v_org_id),
    (v_tenant_id, 'historical_segment', 'Historical segment', 'VALUE_BASED', 'Represents historical data segments. Archival records of past positions.', false, 3, v_org_id),
    (v_tenant_id, 'loan', 'Loan', 'VALUE_BASED', 'Represents a loan asset. Receivable or liability instrument.', false, 3, v_org_id),
    (v_tenant_id, 'master_limited_partnership', 'Master limited partnership', 'SHARE_BASED', 'Represents a master limited partnership. Partnership with publicly traded units.', false, 3, v_org_id),
    (v_tenant_id, 'money_market_fund', 'Money market fund', 'SHARE_BASED', 'Represents a money market fund. Short-duration fund vehicle.', false, 3, v_org_id),
    (v_tenant_id, 'mutual_fund', 'Mutual fund', 'SHARE_BASED', 'Represents a mutual fund. Actively or passively managed fund.', false, 3, v_org_id),
    (v_tenant_id, 'option', 'Option', 'SHARE_BASED', 'Represents an option contract. Right to buy or sell at specified price.', false, 3, v_org_id),
    (v_tenant_id, 'preferred_stock', 'Preferred stock', 'SHARE_BASED', 'Represents preferred stock. Senior claim equity security.', false, 3, v_org_id),
    (v_tenant_id, 'private_investment', 'Private investment', 'SHARE_BASED', 'Represents a private investment. Non-publicly traded holding. Available only to firms that started using Addepar on or after September 12, 2025.', false, 3, v_org_id),
    (v_tenant_id, 'promissory_note', 'Promissory note', 'SHARE_BASED', 'Represents a promissory note. IOU or debt instrument. Available only to firms that started using Addepar on or after September 12, 2025.', false, 3, v_org_id),
    (v_tenant_id, 'real_estate', 'Real estate', 'SHARE_BASED', 'Represents real estate assets. Property holdings. Available only to firms that started using Addepar on or after September 12, 2025.', false, 3, v_org_id),
    (v_tenant_id, 'reit', 'REIT', 'SHARE_BASED', 'Represents a real estate investment trust. Traded REIT security.', false, 3, v_org_id),
    (v_tenant_id, 'stock', 'Stock', 'SHARE_BASED', 'Represents common stock. Equity security in a corporation.', false, 3, v_org_id),
    (v_tenant_id, 'structured_product', 'Structured product', 'SHARE_BASED', 'Represents a structured product investment. Customized derivative security.', false, 3, v_org_id),
    (v_tenant_id, 'uit', 'UIT', 'SHARE_BASED', 'Represents a unit investment trust. Fixed portfolio of securities.', false, 3, v_org_id),
    (v_tenant_id, 'unknown_security', 'Unknown security', 'SHARE_BASED', 'Catch-all for unknown securities. Fallback placeholder type.', false, 3, v_org_id),
    (v_tenant_id, 'venture_capital', 'Venture capital', 'SHARE_BASED', 'Represents venture capital investments. Early-stage equity. Available only to firms that started using Addepar on or after September 12, 2025.', false, 3, v_org_id),
    (v_tenant_id, 'warrant', 'Warrant', 'SHARE_BASED', 'Represents a warrant. Long-dated call option on shares.', false, 3, v_org_id),
    (v_tenant_id, 'generic_asset', 'Custom asset', 'PERCENT_BASED', 'Catch-all for custom or unspecified assets. User-extensible fallback.', false, 3, v_org_id)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- STEP 5: Seed hierarchy rules (parent → child allowed relationships)
-- ============================================================================

DELETE FROM entity_hierarchy_rules WHERE tenant_id = v_tenant_id;

-- Household can own: person_node, trust, managed_partnership, manager, holding_company, sleeve, financial_account, vehicle
INSERT INTO entity_hierarchy_rules (tenant_id, parent_model_type, child_model_type, allowed, ownership_types, description)
VALUES
    (v_tenant_id, 'household', 'person_node', true, ARRAY['PERCENT_BASED'], 'Primary client relationship'),
    (v_tenant_id, 'household', 'trust', true, ARRAY['PERCENT_BASED'], 'Household can create/own trusts'),
    (v_tenant_id, 'household', 'managed_partnership', true, ARRAY['PERCENT_BASED'], 'Fund structure'),
    (v_tenant_id, 'household', 'manager', true, ARRAY['PERCENT_BASED'], 'Manager assignment'),
    (v_tenant_id, 'household', 'holding_company', true, ARRAY['PERCENT_BASED'], 'Corporate structure'),
    (v_tenant_id, 'household', 'sleeve', true, ARRAY['PERCENT_BASED'], 'Tactical sleeve'),
    (v_tenant_id, 'household', 'financial_account', true, ARRAY['PERCENT_BASED'], 'Account relationship'),
    (v_tenant_id, 'household', 'vehicle', true, ARRAY['PERCENT_BASED'], 'General container'),
    
    -- Person_node can own: financial_account, sleeve, trust, vehicle
    (v_tenant_id, 'person_node', 'financial_account', true, ARRAY['PERCENT_BASED'], 'Client bank accounts'),
    (v_tenant_id, 'person_node', 'sleeve', true, ARRAY['PERCENT_BASED'], 'Client sub-portfolio'),
    (v_tenant_id, 'person_node', 'trust', true, ARRAY['PERCENT_BASED'], 'Client trusts'),
    (v_tenant_id, 'person_node', 'vehicle', true, ARRAY['PERCENT_BASED'], 'Client structures'),
    
    -- Trust can own: financial_account, sleeve, vehicle, real_estate, private_investment
    (v_tenant_id, 'trust', 'financial_account', true, ARRAY['PERCENT_BASED'], 'Trust bank accounts'),
    (v_tenant_id, 'trust', 'sleeve', true, ARRAY['PERCENT_BASED'], 'Trust investments'),
    (v_tenant_id, 'trust', 'vehicle', true, ARRAY['PERCENT_BASED'], 'Trust structures'),
    (v_tenant_id, 'trust', 'real_estate', true, ARRAY['VALUE_BASED'], 'Trust property'),
    (v_tenant_id, 'trust', 'private_investment', true, ARRAY['VALUE_BASED'], 'Trust illiquid holdings'),
    
    -- Managed_partnership can own: private_equity_fund, hedge_fund, private_investment
    (v_tenant_id, 'managed_partnership', 'private_equity_fund', true, ARRAY['SHARE_BASED', 'VALUE_BASED'], 'PE fund holding'),
    (v_tenant_id, 'managed_partnership', 'hedge_fund', true, ARRAY['SHARE_BASED', 'VALUE_BASED'], 'HF holding'),
    (v_tenant_id, 'managed_partnership', 'private_investment', true, ARRAY['VALUE_BASED'], 'Direct investment'),
    
    -- Holding_company can own: private_investment, venture_capital, real_estate
    (v_tenant_id, 'holding_company', 'private_investment', true, ARRAY['SHARE_BASED', 'VALUE_BASED'], 'Subsidiary investment'),
    (v_tenant_id, 'holding_company', 'venture_capital', true, ARRAY['SHARE_BASED', 'VALUE_BASED'], 'Venture stake'),
    (v_tenant_id, 'holding_company', 'real_estate', true, ARRAY['VALUE_BASED'], 'Property ownership'),
    
    -- Financial_account can own: cash, bond, etf, stock, mutual_fund, option, real_estate
    (v_tenant_id, 'financial_account', 'cash', true, ARRAY['SHARE_BASED'], 'Account cash'),
    (v_tenant_id, 'financial_account', 'bond', true, ARRAY['SHARE_BASED'], 'Bond holding'),
    (v_tenant_id, 'financial_account', 'etf', true, ARRAY['SHARE_BASED'], 'ETF holding'),
    (v_tenant_id, 'financial_account', 'stock', true, ARRAY['SHARE_BASED'], 'Stock holding'),
    (v_tenant_id, 'financial_account', 'mutual_fund', true, ARRAY['SHARE_BASED'], 'MF holding'),
    (v_tenant_id, 'financial_account', 'option', true, ARRAY['SHARE_BASED'], 'Option position'),
    (v_tenant_id, 'financial_account', 'real_estate', true, ARRAY['VALUE_BASED'], 'Real estate account'),
    
    -- Sleeve can own: cash, bond, etf, stock, mutual_fund, closed_end_fund
    (v_tenant_id, 'sleeve', 'cash', true, ARRAY['PERCENT_BASED'], 'Sleeve allocation'),
    (v_tenant_id, 'sleeve', 'bond', true, ARRAY['PERCENT_BASED'], 'Fixed income sleeve'),
    (v_tenant_id, 'sleeve', 'etf', true, ARRAY['PERCENT_BASED'], 'ETF sleeve'),
    (v_tenant_id, 'sleeve', 'stock', true, ARRAY['PERCENT_BASED'], 'Equity sleeve'),
    (v_tenant_id, 'sleeve', 'mutual_fund', true, ARRAY['PERCENT_BASED'], 'MF sleeve'),
    (v_tenant_id, 'sleeve', 'closed_end_fund', true, ARRAY['PERCENT_BASED'], 'CEF sleeve'),
    
    -- Fund can own: private_investment, venture_capital, real_estate, real_estate_investment
    (v_tenant_id, 'fund', 'private_investment', true, ARRAY['VALUE_BASED'], 'Fund allocation'),
    (v_tenant_id, 'fund', 'venture_capital', true, ARRAY['VALUE_BASED'], 'Fund venture'),
    (v_tenant_id, 'fund', 'real_estate', true, ARRAY['VALUE_BASED'], 'Fund real estate'),
    
    -- Hedge_fund can own same as fund
    (v_tenant_id, 'hedge_fund', 'private_investment', true, ARRAY['VALUE_BASED'], 'HF position'),
    (v_tenant_id, 'hedge_fund', 'stock', true, ARRAY['SHARE_BASED'], 'HF stock position'),
    (v_tenant_id, 'hedge_fund', 'bond', true, ARRAY['SHARE_BASED'], 'HF bond position'),
    
    -- Private_equity_fund can own: private_investment, venture_capital
    (v_tenant_id, 'private_equity_fund', 'private_investment', true, ARRAY['VALUE_BASED'], 'PE portfolio company'),
    (v_tenant_id, 'private_equity_fund', 'venture_capital', true, ARRAY['VALUE_BASED'], 'PE venture'),
    
    -- Vehicle (general) can own most types
    (v_tenant_id, 'vehicle', 'financial_account', true, ARRAY['PERCENT_BASED'], 'Vehicle account'),
    (v_tenant_id, 'vehicle', 'sleeve', true, ARRAY['PERCENT_BASED'], 'Vehicle sleeve'),
    (v_tenant_id, 'vehicle', 'cash', true, ARRAY['PERCENT_BASED'], 'Vehicle cash'),
    (v_tenant_id, 'vehicle', 'stock', true, ARRAY['PERCENT_BASED'], 'Vehicle stock'),
    (v_tenant_id, 'vehicle', 'bond', true, ARRAY['PERCENT_BASED'], 'Vehicle bond'),
    (v_tenant_id, 'vehicle', 'etf', true, ARRAY['PERCENT_BASED'], 'Vehicle etf');

-- ============================================================================
-- STEP 6: Seed model_type_hierarchy_attributes (suggested attributes per type)
-- ============================================================================

DELETE FROM model_type_hierarchy_attributes WHERE tenant_id = v_tenant_id;

-- Universal attributes (all types)
INSERT INTO model_type_hierarchy_attributes 
  (tenant_id, model_type, attribute_key, attribute_type, is_required, is_searchable, priority, description)
SELECT v_tenant_id, mt.model_type, ak, at, req, search, pri, description_val
FROM (VALUES
    ('household', 'original_name', 'string', true, true, 1, 'Household identifier from source system'),
    ('household', 'display_name', 'string', true, true, 2, 'User-friendly household name'),
    ('household', 'currency_factor', 'string', false, false, 3, 'Currency conversion factor'),
    
    ('person_node', 'original_name', 'string', true, true, 1, 'Client name from source'),
    ('person_node', 'display_name', 'string', true, true, 2, 'Display name'),
    ('person_node', 'email', 'string', false, true, 3, 'Contact email'),
    ('person_node', 'phone', 'string', false, false, 4, 'Contact phone'),
    ('person_node', 'citizenship', 'string', false, false, 5, 'Client citizenship'),
    
    ('trust', 'trust_name', 'string', true, true, 1, 'Trust legal name'),
    ('trust', 'trust_type', 'enum', false, false, 2, 'Revocable, Irrevocable, Charitable, etc.'),
    ('trust', 'creation_date', 'date', false, false, 3, 'Trust inception date'),
    ('trust', 'trustee_name', 'string', false, true, 4, 'Primary trustee'),
    
    ('financial_account', 'account_number', 'string', true, false, 1, 'Custodial account ID'),
    ('financial_account', 'custodian', 'string', true, true, 2, 'Bank or broker name'),
    ('financial_account', 'account_type', 'enum', false, false, 3, 'Checking, Savings, Brokerage, Custody'),
    ('financial_account', 'account_currency', 'string', false, false, 4, 'Base currency (USD, EUR, etc.)'),
    
    ('bond', 'cusip', 'string', false, true, 1, 'CUSIP identifier'),
    ('bond', 'isin', 'string', false, true, 2, 'ISIN code'),
    ('bond', 'maturity_date', 'date', false, false, 3, 'Bond maturity'),
    ('bond', 'coupon_rate', 'numeric', false, false, 4, 'Annual coupon percentage'),
    
    ('stock', 'ticker', 'string', true, true, 1, 'Stock ticker symbol'),
    ('stock', 'cusip', 'string', false, true, 2, 'CUSIP identifier'),
    ('stock', 'isin', 'string', false, true, 3, 'ISIN code'),
    ('stock', 'sector', 'string', false, true, 4, 'Industry sector'),
    
    ('etf', 'ticker', 'string', true, true, 1, 'ETF ticker'),
    ('etf', 'cusip', 'string', false, true, 2, 'CUSIP'),
    ('etf', 'benchmark', 'string', false, false, 3, 'Underlying benchmark'),
    
    ('cash', 'currency', 'string', true, true, 1, 'Currency code (USD, EUR, GBP)'),
    ('cash', 'settlement_currency', 'string', false, false, 2, 'Settlement currency if different'),
    
    ('digital_asset', 'blockchain', 'string', true, true, 1, 'Blockchain (Bitcoin, Ethereum, etc.)'),
    ('digital_asset', 'wallet_address', 'string', false, false, 2, 'Blockchain address'),
    ('digital_asset', 'token_symbol', 'string', true, true, 3, 'Token symbol (BTC, ETH, etc.)'),
    
    ('real_estate', 'address', 'string', false, true, 1, 'Property address'),
    ('real_estate', 'property_type', 'enum', false, false, 2, 'Residential, Commercial, Land, etc.'),
    ('real_estate', 'acquisition_date', 'date', false, false, 3, 'Purchase/acquisition date'),
    
    ('private_investment', 'company_name', 'string', true, true, 1, 'Company name'),
    ('private_investment', 'investment_type', 'enum', false, false, 2, 'Common Equity, Preferred, Debt, etc.'),
    ('private_investment', 'investment_date', 'date', false, false, 3, 'Investment date'),
    
    ('fund', 'fund_name', 'string', true, true, 1, 'Fund legal name'),
    ('fund', 'fund_manager', 'string', false, true, 2, 'Fund management company'),
    ('fund', 'vintage_year', 'numeric', false, false, 3, 'Fund vintage year'),
    
    ('private_equity_fund', 'fund_name', 'string', true, true, 1, 'PE fund name'),
    ('private_equity_fund', 'gp_name', 'string', false, true, 2, 'General partner'),
    ('private_equity_fund', 'fund_size', 'numeric', false, false, 3, 'Fund size USD'),
    
    ('hedge_fund', 'fund_name', 'string', true, true, 1, 'Hedge fund name'),
    ('hedge_fund', 'strategy', 'string', false, false, 2, 'Investment strategy'),
    ('hedge_fund', 'lockup_period', 'string', false, false, 3, 'Redemption lockup terms')
) AS u(mt, ak, at, req, search, pri, description_val)
ORDER BY mt, pri;

RAISE NOTICE 'Addepar 49 model types seeded successfully for tenant %', v_tenant_id;

END $$;

-- ============================================================================
-- STEP 7: Create views for hierarchy reporting
-- ============================================================================

DROP VIEW IF EXISTS v_entity_hierarchy_tree CASCADE;
CREATE VIEW v_entity_hierarchy_tree AS
WITH RECURSIVE entity_tree AS (
    -- Base: Root entities (no owners, typically households)
    SELECT 
        e.id,
        e.model_type,
        e.display_name,
        e.tenant_id,
        0 AS depth,
        ARRAY[e.id] AS path,
        NULL::UUID AS parent_id,
        NULL::VARCHAR(255) AS parent_type
    FROM entities e
    LEFT JOIN positions p ON e.id = p.owned_id AND p.incepting_date <= CURRENT_DATE 
        AND (p.closing_date IS NULL OR p.closing_date >= CURRENT_DATE)
    WHERE p.id IS NULL
        AND e.model_type IN ('household', 'prospect', 'manager')
    
    UNION ALL
    
    -- Recursive: Children of root entities
    SELECT 
        e.id,
        e.model_type,
        e.display_name,
        e.tenant_id,
        et.depth + 1,
        et.path || e.id,
        p.owner_id,
        et.model_type
    FROM entities e
    JOIN positions p ON e.id = p.owned_id 
        AND p.incepting_date <= CURRENT_DATE 
        AND (p.closing_date IS NULL OR p.closing_date >= CURRENT_DATE)
    JOIN entity_tree et ON p.owner_id = et.id
    WHERE NOT e.id = ANY(et.path)  -- Prevent cycles
        AND et.depth < 10  -- Limit recursion depth
)
SELECT 
    id,
    model_type,
    display_name,
    tenant_id,
    depth,
    parent_id,
    parent_type,
    (SELECT COUNT(*) FROM positions p2 WHERE p2.owner_id = et.id) AS child_count
FROM entity_tree et;

GRANT SELECT ON v_entity_hierarchy_tree TO authenticated;

-- ============================================================================
-- STEP 8: Create function to validate hierarchical positions
-- ============================================================================

DROP FUNCTION IF EXISTS validate_hierarchy_position(UUID, UUID, VARCHAR(255), UUID);
CREATE OR REPLACE FUNCTION validate_hierarchy_position(
    p_owner_id UUID,
    p_owned_id UUID,
    p_tenant_id UUID,
    p_created_by UUID
)
RETURNS TABLE (is_valid BOOLEAN, error_message TEXT) AS $$
DECLARE
    v_owner_type VARCHAR(255);
    v_owned_type VARCHAR(255);
    v_rule_exists BOOLEAN;
    v_current_parent UUID;
    v_exclusive BOOLEAN;
BEGIN
    -- Get entity types
    SELECT model_type INTO v_owner_type FROM entities WHERE id = p_owner_id;
    SELECT model_type INTO v_owned_type FROM entities WHERE id = p_owned_id;
    
    IF v_owner_type IS NULL THEN
        RETURN QUERY SELECT false, 'Owner entity not found'::TEXT;
        RETURN;
    END IF;
    
    IF v_owned_type IS NULL THEN
        RETURN QUERY SELECT false, 'Owned entity not found'::TEXT;
        RETURN;
    END IF;
    
    -- Check hierarchy rule exists and is allowed
    SELECT allowed INTO v_rule_exists 
    FROM entity_hierarchy_rules 
    WHERE parent_model_type = v_owner_type 
        AND child_model_type = v_owned_type 
        AND tenant_id = p_tenant_id;
    
    IF v_rule_exists IS NULL OR NOT v_rule_exists THEN
        RETURN QUERY SELECT false, FORMAT('Hierarchy rule not allowed: %s -> %s', v_owner_type, v_owned_type)::TEXT;
        RETURN;
    END IF;
    
    -- Check if child is marked as exclusive (can have only one parent type)
    SELECT is_exclusive INTO v_exclusive 
    FROM entity_hierarchy_rules 
    WHERE child_model_type = v_owned_type 
        AND tenant_id = p_tenant_id 
        AND is_exclusive = true 
    LIMIT 1;
    
    IF v_exclusive THEN
        SELECT owner_id INTO v_current_parent 
        FROM positions 
        WHERE owned_id = p_owned_id 
            AND incepting_date <= CURRENT_DATE 
            AND (terminating_date IS NULL OR terminating_date >= CURRENT_DATE)
        LIMIT 1;
        
        IF v_current_parent IS NOT NULL AND v_current_parent != p_owner_id THEN
            RETURN QUERY SELECT false, 'Entity already has exclusive parent'::TEXT;
            RETURN;
        END IF;
    END IF;
    
    -- All checks passed
    RETURN QUERY SELECT true, ''::TEXT;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- ============================================================================
-- STEP 9: Create trigger to validate positions on insert/update
-- ============================================================================

DROP TRIGGER IF EXISTS trig_validate_position_hierarchy ON positions;
CREATE TRIGGER trig_validate_position_hierarchy
BEFORE INSERT OR UPDATE ON positions
FOR EACH ROW
EXECUTE FUNCTION validate_position_hierarchy();

DROP FUNCTION IF EXISTS validate_position_hierarchy();
CREATE OR REPLACE FUNCTION validate_position_hierarchy()
RETURNS TRIGGER AS $$
DECLARE
    v_validation RECORD;
BEGIN
    -- Call validation function
    SELECT * INTO v_validation 
    FROM validate_hierarchy_position(
        NEW.owner_id,
        NEW.owned_id,
        (SELECT tenant_id FROM entities WHERE id = NEW.owner_id LIMIT 1),
        NEW.created_by
    );
    
    IF NOT v_validation.is_valid THEN
        RAISE EXCEPTION 'Hierarchy validation failed: %', v_validation.error_message;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Summary
-- ============================================================================

-- Verification queries (run manually):
-- SELECT COUNT(*) FROM model_type_definitions WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
-- SELECT COUNT(*) FROM entity_hierarchy_rules WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
-- SELECT COUNT(*) FROM model_type_hierarchy_attributes WHERE tenant_id = '00000000-0000-0000-0000-000000000000';
-- SELECT * FROM entity_hierarchy_rules WHERE parent_model_type = 'household' AND tenant_id = '00000000-0000-0000-0000-000000000000';

