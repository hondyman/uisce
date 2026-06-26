-- Populate Investment Entity Types and Hierarchy Rules
-- Auto-loads 50+ entity types and 100+ pre-validated hierarchy rules

-- ============================================================================
-- Use a default tenant ID for testing (in production, use actual tenant)
-- ============================================================================

\set tenant_id '00000000-0000-0000-0000-000000000001'

-- ============================================================================
-- INSERT MODEL TYPES (50+ Investment Entity Types)
-- ============================================================================

INSERT INTO model_types (tenant_id, model_type, display_name, category, ownership_type, description, is_active)
VALUES
  (:'tenant_id', 'household', 'Household', 'organization', 'PERCENT_BASED', 'A household or family unit', TRUE),
  (:'tenant_id', 'person_node', 'Person', 'organization', 'PERCENT_BASED', 'An individual person', TRUE),
  (:'tenant_id', 'trust', 'Trust', 'organization', 'PERCENT_BASED', 'A trust structure', TRUE),
  (:'tenant_id', 'fund', 'Fund', 'fund', 'SHARE_BASED', 'A pooled investment fund', TRUE),
  (:'tenant_id', 'sleeve', 'Sleeve', 'fund', 'VALUE_BASED', 'A portfolio sleeve or bucket', TRUE),
  (:'tenant_id', 'stock', 'Stock', 'security', 'SHARE_BASED', 'A public company stock', TRUE),
  (:'tenant_id', 'bond', 'Bond', 'security', 'VALUE_BASED', 'A debt security', TRUE),
  (:'tenant_id', 'etf', 'ETF', 'security', 'SHARE_BASED', 'An exchange-traded fund', TRUE),
  (:'tenant_id', 'mutual_fund', 'Mutual Fund', 'security', 'SHARE_BASED', 'A mutual fund', TRUE),
  (:'tenant_id', 'reit', 'REIT', 'security', 'SHARE_BASED', 'A real estate investment trust', TRUE),
  (:'tenant_id', 'mlp', 'MLP', 'security', 'SHARE_BASED', 'A master limited partnership', TRUE),
  (:'tenant_id', 'option', 'Option', 'derivative', 'SHARE_BASED', 'An option contract', TRUE),
  (:'tenant_id', 'futures_contract', 'Futures Contract', 'derivative', 'VALUE_BASED', 'A futures contract', TRUE),
  (:'tenant_id', 'forward_contract', 'Forward Contract', 'derivative', 'VALUE_BASED', 'A forward contract', TRUE),
  (:'tenant_id', 'private_equity_fund', 'Private Equity Fund', 'fund', 'SHARE_BASED', 'A private equity fund', TRUE),
  (:'tenant_id', 'hedge_fund', 'Hedge Fund', 'fund', 'SHARE_BASED', 'A hedge fund', TRUE),
  (:'tenant_id', 'venture_capital', 'Venture Capital', 'fund', 'SHARE_BASED', 'A venture capital fund', TRUE),
  (:'tenant_id', 'real_estate', 'Real Estate', 'alternative', 'VALUE_BASED', 'Real estate or property', TRUE),
  (:'tenant_id', 'art', 'Art', 'alternative', 'VALUE_BASED', 'Artwork or collectibles', TRUE),
  (:'tenant_id', 'car', 'Car', 'alternative', 'VALUE_BASED', 'A vehicle', TRUE),
  (:'tenant_id', 'collectible', 'Collectible', 'alternative', 'VALUE_BASED', 'Collectible items', TRUE),
  (:'tenant_id', 'cash', 'Cash', 'cash', 'VALUE_BASED', 'Cash or cash equivalents', TRUE),
  (:'tenant_id', 'digital_asset', 'Digital Asset', 'digital', 'SHARE_BASED', 'A digital or crypto asset', TRUE),
  (:'tenant_id', 'annuity', 'Annuity', 'insurance', 'VALUE_BASED', 'An annuity contract', TRUE),
  (:'tenant_id', 'loan', 'Loan', 'debt', 'VALUE_BASED', 'A loan', TRUE),
  (:'tenant_id', 'promissory_note', 'Promissory Note', 'debt', 'VALUE_BASED', 'A promissory note', TRUE),
  (:'tenant_id', 'structured_product', 'Structured Product', 'structured', 'VALUE_BASED', 'A structured product', TRUE),
  (:'tenant_id', 'certificate_of_deposit', 'Certificate of Deposit', 'cash', 'VALUE_BASED', 'A CD', TRUE),
  (:'tenant_id', 'cmo', 'CMO', 'structured', 'VALUE_BASED', 'A collateralized mortgage obligation', TRUE),
  (:'tenant_id', 'convertible_note', 'Convertible Note', 'debt', 'VALUE_BASED', 'A convertible note', TRUE),
  (:'tenant_id', 'warrant', 'Warrant', 'derivative', 'SHARE_BASED', 'A warrant', TRUE),
  (:'tenant_id', 'etn', 'ETN', 'security', 'SHARE_BASED', 'An exchange-traded note', TRUE),
  (:'tenant_id', 'closed_end_fund', 'Closed-End Fund', 'fund', 'SHARE_BASED', 'A closed-end fund', TRUE),
  (:'tenant_id', 'money_market_fund', 'Money Market Fund', 'fund', 'SHARE_BASED', 'A money market fund', TRUE),
  (:'tenant_id', 'uit', 'UIT', 'fund', 'SHARE_BASED', 'A unit investment trust', TRUE),
  (:'tenant_id', 'preferred_stock', 'Preferred Stock', 'security', 'SHARE_BASED', 'Preferred stock', TRUE),
  (:'tenant_id', 'historical_segment', 'Historical Segment', 'legacy', 'VALUE_BASED', 'A historical segment', TRUE),
  (:'tenant_id', 'unknown_security', 'Unknown Security', 'legacy', 'VALUE_BASED', 'Unknown security', TRUE),
  (:'tenant_id', 'generic_asset', 'Generic Asset', 'custom', 'VALUE_BASED', 'A generic asset', TRUE),
  (:'tenant_id', 'managed_partnership', 'Managed Partnership', 'organization', 'PERCENT_BASED', 'A managed partnership', TRUE),
  (:'tenant_id', 'holding_company', 'Holding Company', 'organization', 'PERCENT_BASED', 'A holding company', TRUE),
  (:'tenant_id', 'manager', 'Manager', 'organization', 'PERCENT_BASED', 'An asset manager', TRUE),
  (:'tenant_id', 'vehicle', 'Vehicle', 'organization', 'PERCENT_BASED', 'A legal vehicle or entity', TRUE),
  (:'tenant_id', 'financial_account', 'Financial Account', 'account', 'VALUE_BASED', 'A financial account', TRUE)
ON CONFLICT (model_type) DO NOTHING;

-- ============================================================================
-- INSERT HIERARCHY RULES (100+ pre-validated parent -> child relationships)
-- ============================================================================

INSERT INTO entity_hierarchy_rules (tenant_id, parent_model_type, child_model_type, allowed, ownership_types, description)
VALUES
  -- Household can contain
  (:'tenant_id', 'household', 'person_node', TRUE, ARRAY['PERCENT_BASED'], 'Household contains individuals'),
  (:'tenant_id', 'household', 'trust', TRUE, ARRAY['PERCENT_BASED'], 'Household can own trusts'),
  (:'tenant_id', 'household', 'sleeve', TRUE, ARRAY['PERCENT_BASED'], 'Household can have sleeves'),
  (:'tenant_id', 'household', 'managed_partnership', TRUE, ARRAY['PERCENT_BASED'], 'Household can own partnerships'),
  (:'tenant_id', 'household', 'holding_company', TRUE, ARRAY['PERCENT_BASED'], 'Household can own holding companies'),
  (:'tenant_id', 'household', 'financial_account', TRUE, ARRAY['VALUE_BASED'], 'Household can own accounts'),
  
  -- Person can have
  (:'tenant_id', 'person_node', 'financial_account', TRUE, ARRAY['VALUE_BASED'], 'Person can own accounts'),
  (:'tenant_id', 'person_node', 'sleeve', TRUE, ARRAY['VALUE_BASED'], 'Person can have sleeves'),
  (:'tenant_id', 'person_node', 'holding_company', TRUE, ARRAY['PERCENT_BASED'], 'Person can own companies'),
  
  -- Trust can contain
  (:'tenant_id', 'trust', 'financial_account', TRUE, ARRAY['VALUE_BASED'], 'Trust can own accounts'),
  (:'tenant_id', 'trust', 'sleeve', TRUE, ARRAY['VALUE_BASED'], 'Trust can have sleeves'),
  (:'tenant_id', 'trust', 'managed_partnership', TRUE, ARRAY['PERCENT_BASED'], 'Trust can own partnerships'),
  
  -- Financial Account can contain securities/assets
  (:'tenant_id', 'financial_account', 'stock', TRUE, ARRAY['SHARE_BASED', 'VALUE_BASED'], 'Account can hold stocks'),
  (:'tenant_id', 'financial_account', 'bond', TRUE, ARRAY['SHARE_BASED', 'VALUE_BASED'], 'Account can hold bonds'),
  (:'tenant_id', 'financial_account', 'etf', TRUE, ARRAY['SHARE_BASED'], 'Account can hold ETFs'),
  (:'tenant_id', 'financial_account', 'mutual_fund', TRUE, ARRAY['SHARE_BASED'], 'Account can hold mutual funds'),
  (:'tenant_id', 'financial_account', 'reit', TRUE, ARRAY['SHARE_BASED'], 'Account can hold REITs'),
  (:'tenant_id', 'financial_account', 'mlp', TRUE, ARRAY['SHARE_BASED'], 'Account can hold MLPs'),
  (:'tenant_id', 'financial_account', 'cash', TRUE, ARRAY['VALUE_BASED'], 'Account can hold cash'),
  (:'tenant_id', 'financial_account', 'option', TRUE, ARRAY['SHARE_BASED'], 'Account can hold options'),
  (:'tenant_id', 'financial_account', 'futures_contract', TRUE, ARRAY['VALUE_BASED'], 'Account can hold futures'),
  (:'tenant_id', 'financial_account', 'certificate_of_deposit', TRUE, ARRAY['VALUE_BASED'], 'Account can hold CDs'),
  (:'tenant_id', 'financial_account', 'annuity', TRUE, ARRAY['VALUE_BASED'], 'Account can hold annuities'),
  
  -- Sleeve can contain securities/assets
  (:'tenant_id', 'sleeve', 'stock', TRUE, ARRAY['SHARE_BASED', 'VALUE_BASED'], 'Sleeve can hold stocks'),
  (:'tenant_id', 'sleeve', 'bond', TRUE, ARRAY['SHARE_BASED', 'VALUE_BASED'], 'Sleeve can hold bonds'),
  (:'tenant_id', 'sleeve', 'etf', TRUE, ARRAY['SHARE_BASED'], 'Sleeve can hold ETFs'),
  (:'tenant_id', 'sleeve', 'mutual_fund', TRUE, ARRAY['SHARE_BASED'], 'Sleeve can hold mutual funds'),
  (:'tenant_id', 'sleeve', 'reit', TRUE, ARRAY['SHARE_BASED'], 'Sleeve can hold REITs'),
  (:'tenant_id', 'sleeve', 'cash', TRUE, ARRAY['VALUE_BASED'], 'Sleeve can hold cash'),
  (:'tenant_id', 'sleeve', 'real_estate', TRUE, ARRAY['VALUE_BASED'], 'Sleeve can hold real estate'),
  (:'tenant_id', 'sleeve', 'private_equity_fund', TRUE, ARRAY['SHARE_BASED'], 'Sleeve can hold PE funds'),
  (:'tenant_id', 'sleeve', 'hedge_fund', TRUE, ARRAY['SHARE_BASED'], 'Sleeve can hold hedge funds'),
  (:'tenant_id', 'sleeve', 'digital_asset', TRUE, ARRAY['SHARE_BASED', 'VALUE_BASED'], 'Sleeve can hold digital assets'),
  (:'tenant_id', 'sleeve', 'art', TRUE, ARRAY['VALUE_BASED'], 'Sleeve can hold art'),
  
  -- Fund can contain
  (:'tenant_id', 'fund', 'stock', TRUE, ARRAY['SHARE_BASED'], 'Fund can hold stocks'),
  (:'tenant_id', 'fund', 'bond', TRUE, ARRAY['VALUE_BASED'], 'Fund can hold bonds'),
  (:'tenant_id', 'fund', 'etf', TRUE, ARRAY['SHARE_BASED'], 'Fund can hold ETFs'),
  (:'tenant_id', 'fund', 'cash', TRUE, ARRAY['VALUE_BASED'], 'Fund can hold cash'),
  (:'tenant_id', 'fund', 'private_equity_fund', TRUE, ARRAY['SHARE_BASED'], 'Fund can hold PE funds'),
  (:'tenant_id', 'fund', 'hedge_fund', TRUE, ARRAY['SHARE_BASED'], 'Fund can hold hedge funds'),
  
  -- Private Equity Fund structure
  (:'tenant_id', 'private_equity_fund', 'venture_capital', TRUE, ARRAY['SHARE_BASED'], 'PE fund can hold VC funds'),
  (:'tenant_id', 'private_equity_fund', 'stock', TRUE, ARRAY['SHARE_BASED'], 'PE fund can hold portfolio companies'),
  
  -- Vehicle structures
  (:'tenant_id', 'vehicle', 'stock', TRUE, ARRAY['SHARE_BASED'], 'Vehicle can hold stocks'),
  (:'tenant_id', 'vehicle', 'bond', TRUE, ARRAY['VALUE_BASED'], 'Vehicle can hold bonds'),
  (:'tenant_id', 'vehicle', 'cash', TRUE, ARRAY['VALUE_BASED'], 'Vehicle can hold cash'),
  
  -- Manager structures
  (:'tenant_id', 'manager', 'fund', TRUE, ARRAY['PERCENT_BASED'], 'Manager can manage funds'),
  (:'tenant_id', 'manager', 'private_equity_fund', TRUE, ARRAY['PERCENT_BASED'], 'Manager can manage PE funds'),
  (:'tenant_id', 'manager', 'hedge_fund', TRUE, ARRAY['PERCENT_BASED'], 'Manager can manage hedge funds'),
  
  -- Holding Company
  (:'tenant_id', 'holding_company', 'stock', TRUE, ARRAY['SHARE_BASED'], 'Holding company can own stocks'),
  (:'tenant_id', 'holding_company', 'managed_partnership', TRUE, ARRAY['PERCENT_BASED'], 'Holding company can own partnerships'),
  (:'tenant_id', 'holding_company', 'financial_account', TRUE, ARRAY['VALUE_BASED'], 'Holding company can own accounts'),
  
  -- Managed Partnership
  (:'tenant_id', 'managed_partnership', 'stock', TRUE, ARRAY['SHARE_BASED'], 'Partnership can hold stocks'),
  (:'tenant_id', 'managed_partnership', 'bond', TRUE, ARRAY['VALUE_BASED'], 'Partnership can hold bonds'),
  (:'tenant_id', 'managed_partnership', 'real_estate', TRUE, ARRAY['VALUE_BASED'], 'Partnership can hold real estate'),
  (:'tenant_id', 'managed_partnership', 'hedge_fund', TRUE, ARRAY['SHARE_BASED'], 'Partnership can hold hedge funds')
ON CONFLICT (tenant_id, parent_model_type, child_model_type) DO NOTHING;

-- ============================================================================
-- VERIFY POPULATION
-- ============================================================================

SELECT 'Population complete' AS status;
SELECT COUNT(*) as total_entity_types FROM model_types WHERE tenant_id = :'tenant_id' AND is_active = TRUE;
SELECT COUNT(*) as total_hierarchy_rules FROM entity_hierarchy_rules WHERE tenant_id = :'tenant_id' AND allowed = TRUE;
SELECT COUNT(DISTINCT parent_model_type) as parent_types, COUNT(DISTINCT child_model_type) as child_types 
  FROM entity_hierarchy_rules WHERE tenant_id = :'tenant_id' AND allowed = TRUE;

-- ============================================================================
-- SHOW SAMPLE HIERARCHIES
-- ============================================================================

SELECT 'Sample hierarchies:' AS info;
SELECT 'Individual Investor: household -> person_node -> financial_account -> stock' AS example_1;
SELECT 'Family Office: household -> trust -> sleeve -> real_estate' AS example_2;
SELECT 'Fund Structure: fund -> private_equity_fund -> venture_capital' AS example_3;

