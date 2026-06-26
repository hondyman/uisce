-- Investment Entity Types & Hierarchy Rules
-- Adds Addepar-compatible investment entities with hierarchical relationships
-- To be run after your existing entity schema

-- 1. Insert Investment Model Types
INSERT INTO model_types (
  model_type,
  display_name,
  ownership_type,
  description,
  category,
  is_active,
  created_at,
  updated_at
) VALUES
-- Organizational Entities (Top-level)
('household', 'Household', 'Percent-based', 'Primary container for portfolio entities', 'organization', true, NOW(), NOW()),
('person_node', 'Client', 'Percent-based', 'Individual client or person', 'organization', true, NOW(), NOW()),
('prospect', 'Prospect', 'Percent-based', 'Prospective client', 'organization', true, NOW(), NOW()),

-- Partnership & Fund Entities
('managed_partnership', 'Managed Fund', 'Share-based,Value-based', 'Managed fund or partnership', 'fund', true, NOW(), NOW()),
('holding_company', 'Holding Company', 'Percent-based', 'Corporate holding entity', 'fund', true, NOW(), NOW()),
('manager', 'Manager', 'Percent-based', 'Manager entity', 'organization', true, NOW(), NOW()),
('fund', 'Private Fund', 'Value-based', 'Private fund investment vehicle', 'fund', true, NOW(), NOW()),
('trust', 'Trust', 'Percent-based', 'Legal trust entity', 'organization', true, NOW(), NOW()),

-- Container Entities
('vehicle', 'Vehicle', 'Percent-based', 'Investment vehicle container', 'container', true, NOW(), NOW()),
('financial_account', 'Holding Account', 'Percent-based', 'Financial or brokerage account', 'container', true, NOW(), NOW()),
('sleeve', 'Sleeve', 'Percent-based', 'Portfolio sleeve or sub-allocation', 'container', true, NOW(), NOW()),

-- Insurance & Annuity Products
('annuity', 'Annuity', 'Value-based', 'Annuity investment product', 'insurance', true, NOW(), NOW()),

-- Alternative & Collectible Assets
('art', 'Art', 'Share-based,Value-based', 'Art as alternative asset', 'alternative', true, NOW(), NOW()),
('car', 'Car', 'Share-based,Value-based', 'Vehicle as asset', 'alternative', true, NOW(), NOW()),
('collectible', 'Collectible', 'Share-based,Value-based', 'Collectible assets', 'alternative', true, NOW(), NOW()),
('real_estate', 'Real Estate', 'Share-based,Value-based', 'Real estate properties', 'alternative', true, NOW(), NOW()),

-- Fixed Income Securities
('bond', 'Bond', 'Share-based', 'Fixed income bond', 'security', true, NOW(), NOW()),
('cmo', 'CMO', 'Share-based', 'Collateralized mortgage obligation', 'security', true, NOW(), NOW()),
('certificate_of_deposit', 'Certificate of Deposit', 'Share-based', 'CD investment', 'security', true, NOW(), NOW()),

-- Equity & Fund Products
('stock', 'Stock', 'Share-based', 'Common or preferred stock', 'security', true, NOW(), NOW()),
('preferred_stock', 'Preferred Stock', 'Share-based', 'Preferred stock', 'security', true, NOW(), NOW()),
('closed_end_fund', 'Closed End Fund', 'Share-based', 'Closed-end mutual fund', 'security', true, NOW(), NOW()),
('etf', 'ETF', 'Share-based', 'Exchange-traded fund', 'security', true, NOW(), NOW()),
('mutual_fund', 'Mutual Fund', 'Share-based', 'Open-end mutual fund', 'security', true, NOW(), NOW()),
('money_market_fund', 'Money Market Fund', 'Share-based', 'Money market mutual fund', 'security', true, NOW(), NOW()),
('reit', 'REIT', 'Share-based', 'Real estate investment trust', 'security', true, NOW(), NOW()),
('mlp', 'Master Limited Partnership', 'Share-based', 'Master limited partnership', 'security', true, NOW(), NOW()),
('uit', 'UIT', 'Share-based', 'Unit investment trust', 'security', true, NOW(), NOW()),

-- Derivatives & Contracts
('option', 'Option', 'Share-based', 'Options contract', 'derivative', true, NOW(), NOW()),
('futures_contract', 'Futures Contract', 'Share-based', 'Futures contract', 'derivative', true, NOW(), NOW()),
('forward_contract', 'Forward Contract', 'Share-based', 'Forward contract', 'derivative', true, NOW(), NOW()),
('convertible_note', 'Convertible Note', 'Share-based', 'Convertible debt instrument', 'derivative', true, NOW(), NOW()),
('warrant', 'Warrant', 'Share-based', 'Stock warrant', 'derivative', true, NOW(), NOW()),
('etn', 'ETN', 'Share-based', 'Exchange-traded note', 'derivative', true, NOW(), NOW()),

-- Private Investments & Alternative Funds
('private_equity_fund', 'Private Equity Fund', 'Share-based,Value-based', 'Private equity fund investment', 'alternative', true, NOW(), NOW()),
('hedge_fund', 'Hedge Fund', 'Share-based,Value-based', 'Hedge fund investment', 'alternative', true, NOW(), NOW()),
('venture_capital', 'Venture Capital', 'Share-based,Value-based', 'Venture capital investment', 'alternative', true, NOW(), NOW()),
('private_investment', 'Private Investment', 'Share-based,Value-based', 'Direct private investment', 'alternative', true, NOW(), NOW()),

-- Debt & Note Instruments
('promissory_note', 'Promissory Note', 'Share-based,Value-based', 'Promissory note', 'debt', true, NOW(), NOW()),
('loan', 'Loan', 'Value-based', 'Loan asset', 'debt', true, NOW(), NOW()),

-- Digital & Structured Assets
('digital_asset', 'Digital Asset', 'Share-based', 'Cryptocurrency and digital assets', 'digital', true, NOW(), NOW()),
('structured_product', 'Structured Product', 'Share-based', 'Structured investment product', 'structured', true, NOW(), NOW()),

-- Cash & Legacy
('cash', 'Currency', 'Share-based', 'Cash or currency holdings', 'cash', true, NOW(), NOW()),
('historical_segment', 'Historical Segment', 'Value-based', 'Historical data segments', 'legacy', true, NOW(), NOW()),
('unknown_security', 'Unknown Security', 'Share-based', 'Unidentified security', 'legacy', true, NOW(), NOW()),
('generic_asset', 'Custom Asset', 'Any', 'Custom or unspecified asset type', 'custom', true, NOW(), NOW())
ON CONFLICT (model_type) DO NOTHING;

-- 2. Create Hierarchy Rules Table
CREATE TABLE IF NOT EXISTS entity_hierarchy_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    parent_model_type VARCHAR(100) NOT NULL,
    child_model_type VARCHAR(100) NOT NULL,
    allowed BOOLEAN NOT NULL DEFAULT true,
    ownership_types TEXT NOT NULL, -- JSON array: ["PERCENT_BASED", "SHARE_BASED", "VALUE_BASED"]
    max_children INTEGER,
    description TEXT,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_parent_model_type FOREIGN KEY (parent_model_type) REFERENCES model_types(model_type),
    CONSTRAINT fk_child_model_type FOREIGN KEY (child_model_type) REFERENCES model_types(model_type),
    UNIQUE(tenant_id, parent_model_type, child_model_type)
);

CREATE INDEX idx_hierarchy_rules_parent ON entity_hierarchy_rules(parent_model_type);
CREATE INDEX idx_hierarchy_rules_child ON entity_hierarchy_rules(child_model_type);
CREATE INDEX idx_hierarchy_rules_tenant ON entity_hierarchy_rules(tenant_id);

-- 3. Insert Hierarchy Rules (Addepar-compliant structure)
INSERT INTO entity_hierarchy_rules (
    tenant_id,
    parent_model_type,
    child_model_type,
    allowed,
    ownership_types,
    description
) SELECT
    tenant_id,
    parent_model_type,
    child_model_type,
    true,
    ownership_types,
    description
FROM (
    -- Household (primary container)
    SELECT * FROM (VALUES
        ('household', 'person_node', '["PERCENT_BASED"]', 'Household contains clients'),
        ('household', 'trust', '["PERCENT_BASED"]', 'Household can own trusts'),
        ('household', 'managed_partnership', '["PERCENT_BASED"]', 'Household can hold partnerships'),
        ('household', 'holding_company', '["PERCENT_BASED"]', 'Household holds companies'),
        ('household', 'sleeve', '["PERCENT_BASED"]', 'Household contains sleeves'),
        ('household', 'financial_account', '["PERCENT_BASED"]', 'Household has accounts'),
        ('household', 'prospect', '["PERCENT_BASED"]', 'Household can have prospects'),
        
        -- Person/Client (individual owner)
        ('person_node', 'financial_account', '["PERCENT_BASED"]', 'Person owns accounts'),
        ('person_node', 'sleeve', '["PERCENT_BASED"]', 'Person has portfolio sleeves'),
        
        -- Trust (legal entity container)
        ('trust', 'financial_account', '["PERCENT_BASED"]', 'Trust holds accounts'),
        ('trust', 'sleeve', '["PERCENT_BASED"]', 'Trust has sleeves'),
        ('trust', 'real_estate', '["PERCENT_BASED","VALUE_BASED"]', 'Trust owns real estate'),
        ('trust', 'bond', '["SHARE_BASED"]', 'Trust owns bonds'),
        ('trust', 'etf', '["SHARE_BASED"]', 'Trust owns ETFs'),
        
        -- Partnerships & Funds (fund-of-funds structure)
        ('managed_partnership', 'private_equity_fund', '["VALUE_BASED"]', 'Partnership holds PE funds'),
        ('managed_partnership', 'hedge_fund', '["VALUE_BASED"]', 'Partnership holds hedge funds'),
        ('managed_partnership', 'private_investment', '["VALUE_BASED"]', 'Partnership holds investments'),
        
        ('holding_company', 'private_equity_fund', '["VALUE_BASED"]', 'Company holds PE'),
        ('holding_company', 'venture_capital', '["VALUE_BASED"]', 'Company holds VC'),
        ('holding_company', 'private_investment', '["VALUE_BASED"]', 'Company holds investments'),
        
        -- Fund Entities (nested funds)
        ('fund', 'private_equity_fund', '["VALUE_BASED"]', 'Fund holds other funds'),
        ('fund', 'hedge_fund', '["VALUE_BASED"]', 'Fund holds hedge funds'),
        ('fund', 'venture_capital', '["VALUE_BASED"]', 'Fund holds VC'),
        ('fund', 'stock', '["SHARE_BASED"]', 'Fund holds stocks'),
        ('fund', 'bond', '["SHARE_BASED"]', 'Fund holds bonds'),
        
        ('private_equity_fund', 'private_investment', '["VALUE_BASED"]', 'PE fund holds investments'),
        ('private_equity_fund', 'venture_capital', '["VALUE_BASED"]', 'PE fund holds VC'),
        ('hedge_fund', 'stock', '["SHARE_BASED"]', 'Hedge fund holds stocks'),
        ('hedge_fund', 'etf', '["SHARE_BASED"]', 'Hedge fund holds ETFs'),
        ('hedge_fund', 'option', '["SHARE_BASED"]', 'Hedge fund holds options'),
        
        -- Sleeve (portfolio sub-allocation)
        ('sleeve', 'stock', '["SHARE_BASED"]', 'Sleeve contains stocks'),
        ('sleeve', 'bond', '["SHARE_BASED"]', 'Sleeve contains bonds'),
        ('sleeve', 'etf', '["SHARE_BASED"]', 'Sleeve contains ETFs'),
        ('sleeve', 'mutual_fund', '["SHARE_BASED"]', 'Sleeve contains mutual funds'),
        ('sleeve', 'cash', '["SHARE_BASED"]', 'Sleeve contains cash'),
        ('sleeve', 'option', '["SHARE_BASED"]', 'Sleeve contains options'),
        
        -- Financial Account (custodian/brokerage account)
        ('financial_account', 'stock', '["SHARE_BASED"]', 'Account holds stocks'),
        ('financial_account', 'bond', '["SHARE_BASED"]', 'Account holds bonds'),
        ('financial_account', 'etf', '["SHARE_BASED"]', 'Account holds ETFs'),
        ('financial_account', 'mutual_fund', '["SHARE_BASED"]', 'Account holds mutual funds'),
        ('financial_account', 'cash', '["SHARE_BASED"]', 'Account holds cash'),
        ('financial_account', 'money_market_fund', '["SHARE_BASED"]', 'Account holds MMF'),
        ('financial_account', 'certificate_of_deposit', '["SHARE_BASED"]', 'Account holds CDs'),
        
        -- Alternative Assets (can be direct or in sleeve)
        ('sleeve', 'real_estate', '["PERCENT_BASED","VALUE_BASED"]', 'Sleeve contains real estate'),
        ('sleeve', 'art', '["PERCENT_BASED","VALUE_BASED"]', 'Sleeve contains art'),
        ('sleeve', 'collectible', '["PERCENT_BASED","VALUE_BASED"]', 'Sleeve contains collectibles'),
        ('financial_account', 'real_estate', '["PERCENT_BASED","VALUE_BASED"]', 'Account holds real estate'),
        ('financial_account', 'art', '["PERCENT_BASED","VALUE_BASED"]', 'Account holds art'),
        
        -- Derivatives (in sleeves/accounts)
        ('sleeve', 'futures_contract', '["SHARE_BASED"]', 'Sleeve contains futures'),
        ('sleeve', 'forward_contract', '["SHARE_BASED"]', 'Sleeve contains forwards'),
        ('financial_account', 'option', '["SHARE_BASED"]', 'Account holds options'),
        ('financial_account', 'futures_contract', '["SHARE_BASED"]', 'Account holds futures'),
        
        -- Annuities & Insurance
        ('person_node', 'annuity', '["VALUE_BASED"]', 'Person owns annuity'),
        ('sleeve', 'annuity', '["VALUE_BASED"]', 'Sleeve contains annuity'),
        
        -- Digital Assets
        ('sleeve', 'digital_asset', '["SHARE_BASED"]', 'Sleeve contains digital assets'),
        ('financial_account', 'digital_asset', '["SHARE_BASED"]', 'Account holds digital assets')
    ) AS rules(parent_model_type, child_model_type, ownership_types, description),
    (SELECT id FROM tenants LIMIT 1) AS t(tenant_id)
) hierarchy_data
ON CONFLICT (tenant_id, parent_model_type, child_model_type) DO NOTHING;

-- 4. Create Hierarchy Validation Function
CREATE OR REPLACE FUNCTION validate_entity_hierarchy()
RETURNS TRIGGER AS $$
DECLARE
    v_parent_model_type VARCHAR(100);
    v_child_model_type VARCHAR(100);
    v_allowed BOOLEAN;
    v_parent_tenant_id UUID;
    v_child_tenant_id UUID;
BEGIN
    -- Get model types
    SELECT model_type, tenant_id INTO v_parent_model_type, v_parent_tenant_id
    FROM entities WHERE id = NEW.owner_id;
    
    SELECT model_type, tenant_id INTO v_child_model_type, v_child_tenant_id
    FROM entities WHERE id = NEW.owned_id;
    
    -- Validate tenant consistency
    IF v_parent_tenant_id != v_child_tenant_id THEN
        RAISE EXCEPTION 'Parent and child must belong to same tenant';
    END IF;
    
    -- Check if relationship is allowed
    SELECT allowed INTO v_allowed
    FROM entity_hierarchy_rules
    WHERE tenant_id = v_parent_tenant_id
      AND parent_model_type = v_parent_model_type
      AND child_model_type = v_child_model_type;
    
    IF NOT v_allowed THEN
        RAISE EXCEPTION 'Invalid hierarchy: % cannot own %', v_parent_model_type, v_child_model_type;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 5. Add trigger to positions table (if not already present)
DROP TRIGGER IF EXISTS check_hierarchy_on_position_insert ON positions;
CREATE TRIGGER check_hierarchy_on_position_insert
    BEFORE INSERT ON positions
    FOR EACH ROW
    EXECUTE FUNCTION validate_entity_hierarchy();

-- 6. Create View: Entity Hierarchy Summary
CREATE OR REPLACE VIEW entity_hierarchy_summary AS
SELECT
    hr.tenant_id,
    hr.parent_model_type,
    hr.child_model_type,
    hr.allowed,
    hr.ownership_types,
    (SELECT COUNT(*) FROM positions p
     JOIN entities ep ON p.owner_id = ep.id
     JOIN entities ec ON p.owned_id = ec.id
     WHERE ep.model_type = hr.parent_model_type
       AND ec.model_type = hr.child_model_type) AS active_relationships,
    hr.description
FROM entity_hierarchy_rules hr;

-- 7. Create View: Organizational Hierarchy (tree view friendly)
CREATE OR REPLACE VIEW entity_hierarchy_tree AS
WITH RECURSIVE hierarchy AS (
    -- Base: Top-level entities (no parent)
    SELECT
        e.id,
        e.tenant_id,
        e.model_type,
        e.display_name,
        NULL::UUID AS parent_id,
        0 AS depth,
        ARRAY[e.id::TEXT] AS path_ids,
        ARRAY[e.display_name] AS path_names
    FROM entities e
    WHERE NOT EXISTS (
        SELECT 1 FROM positions p WHERE p.owned_id = e.id
    )
    
    UNION ALL
    
    -- Recursive: Children
    SELECT
        e.id,
        e.tenant_id,
        e.model_type,
        e.display_name,
        p.owner_id,
        h.depth + 1,
        h.path_ids || e.id::TEXT,
        h.path_names || e.display_name
    FROM entities e
    JOIN positions p ON p.owned_id = e.id
    JOIN hierarchy h ON p.owner_id = h.id
    WHERE h.depth < 10 -- Prevent infinite loops
)
SELECT
    id,
    tenant_id,
    model_type,
    display_name,
    parent_id,
    depth,
    path_ids,
    path_names,
    ARRAY_LENGTH(path_ids, 1) - 1 AS level
FROM hierarchy;

-- 8. Create Audit Log for Hierarchy Changes
CREATE TABLE IF NOT EXISTS entity_hierarchy_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    entity_id UUID NOT NULL,
    position_id UUID,
    action VARCHAR(50), -- 'CREATE', 'UPDATE', 'DELETE', 'VALIDATE_FAIL'
    parent_model_type VARCHAR(100),
    child_model_type VARCHAR(100),
    reason TEXT,
    created_by UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_entity FOREIGN KEY (entity_id) REFERENCES entities(id),
    CONSTRAINT fk_position FOREIGN KEY (position_id) REFERENCES positions(id),
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE INDEX idx_hierarchy_audit_entity ON entity_hierarchy_audit_log(entity_id);
CREATE INDEX idx_hierarchy_audit_tenant ON entity_hierarchy_audit_log(tenant_id);
CREATE INDEX idx_hierarchy_audit_action ON entity_hierarchy_audit_log(action);

GRANT SELECT ON model_types TO PUBLIC;
GRANT SELECT ON entity_hierarchy_rules TO PUBLIC;
GRANT SELECT ON entity_hierarchy_summary TO PUBLIC;
GRANT SELECT ON entity_hierarchy_tree TO PUBLIC;
GRANT SELECT ON entity_hierarchy_audit_log TO PUBLIC;
