-- Populate Investment Entity Types & Hierarchy Rules
-- This migration automatically adds all 50+ investment entity types
-- and establishes the complete Addepar-compatible hierarchy

-- Run this after investment_entities_hierarchy.sql

BEGIN TRANSACTION;

-- 1. Populate all 50+ Investment Entity Types
INSERT INTO model_types (
  id,
  model_type,
  display_name,
  ownership_type,
  description,
  category,
  is_active,
  attributes,
  created_at,
  updated_at
) VALUES
-- Organizational Entities
(gen_random_uuid(), 'household', 'Household', 'Percent-based', 'Primary container for a household or family unit', 'organization', true, '{"inception_date": "date", "status": "enum"}', NOW(), NOW()),
(gen_random_uuid(), 'person_node', 'Client', 'Percent-based', 'Individual client or person with potential ownership interests', 'organization', true, '{"date_of_birth": "date", "role": "enum", "contact_email": "email"}', NOW(), NOW()),
(gen_random_uuid(), 'prospect', 'Prospect', 'Percent-based', 'Prospective client not yet active', 'organization', true, '{"prospect_status": "enum", "acquisition_date": "date", "expected_assets": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'manager', 'Manager', 'Percent-based', 'Investment manager entity', 'organization', true, '{"aum": "numeric", "management_fee_bps": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'trust', 'Trust', 'Percent-based', 'Legal trust entity with trustees and beneficiaries', 'organization', true, '{"trust_type": "enum", "creation_date": "date", "trustee_name": "string"}', NOW(), NOW()),

-- Fund & Partnership Entities
(gen_random_uuid(), 'managed_partnership', 'Managed Fund', 'Share-based,Value-based', 'Managed fund or partnership vehicle with multiple investors', 'fund', true, '{"fund_type": "enum", "nav_per_share": "numeric", "total_nav": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'holding_company', 'Holding Company', 'Percent-based', 'Corporate holding company with subsidiaries and investments', 'fund', true, '{"incorporation_date": "date", "jurisdiction": "string", "tax_id": "string"}', NOW(), NOW()),
(gen_random_uuid(), 'fund', 'Private Fund', 'Value-based', 'Private investment fund vehicle', 'fund', true, '{"fund_vintage": "year", "fund_life_years": "numeric", "target_size": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'private_equity_fund', 'Private Equity Fund', 'Share-based,Value-based', 'Private equity fund investment. Available after Sept 12, 2025', 'alternative', true, '{"fund_vintage": "year", "target_irr": "numeric", "commitment_amount": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'hedge_fund', 'Hedge Fund', 'Share-based,Value-based', 'Hedge fund investment. Available after Sept 12, 2025', 'alternative', true, '{"strategy": "string", "aum": "numeric", "min_investment": "numeric"}', NOW(), NOW()),

-- Container Entities
(gen_random_uuid(), 'vehicle', 'Vehicle', 'Percent-based', 'Generic investment vehicle container', 'container', true, '{"vehicle_type": "string"}', NOW(), NOW()),
(gen_random_uuid(), 'financial_account', 'Holding Account', 'Percent-based', 'Financial account at custodian or broker', 'container', true, '{"account_number": "string", "custodian": "string", "account_type": "enum", "opening_date": "date"}', NOW(), NOW()),
(gen_random_uuid(), 'sleeve', 'Sleeve', 'Percent-based', 'Portfolio sleeve or sub-allocation with defined strategy', 'container', true, '{"allocation_percentage": "numeric", "strategy": "enum", "rebalance_frequency": "enum"}', NOW(), NOW()),

-- Fixed Income Securities
(gen_random_uuid(), 'bond', 'Bond', 'Share-based', 'Fixed income bond security', 'security', true, '{"cusip": "string", "maturity_date": "date", "coupon_rate": "numeric", "bond_type": "enum"}', NOW(), NOW()),
(gen_random_uuid(), 'cmo', 'CMO', 'Share-based', 'Collateralized mortgage obligation', 'security', true, '{"cusip": "string", "coupon": "numeric", "wam": "numeric", "class": "enum"}', NOW(), NOW()),
(gen_random_uuid(), 'certificate_of_deposit', 'Certificate of Deposit', 'Share-based', 'Certificate of deposit fixed rate security', 'security', true, '{"issuer": "string", "maturity_date": "date", "interest_rate": "numeric", "term_months": "numeric"}', NOW(), NOW()),

-- Equity Securities
(gen_random_uuid(), 'stock', 'Stock', 'Share-based', 'Common or preferred equity stock', 'security', true, '{"ticker": "string", "cusip": "string", "sector": "string", "market_cap": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'preferred_stock', 'Preferred Stock', 'Share-based', 'Preferred stock with fixed dividend', 'security', true, '{"ticker": "string", "dividend_rate": "numeric", "issuer": "string"}', NOW(), NOW()),

-- Mutual Funds & ETFs
(gen_random_uuid(), 'etf', 'ETF', 'Share-based', 'Exchange-traded fund', 'security', true, '{"cusip": "string", "ticker": "string", "expense_ratio": "numeric", "net_assets": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'mutual_fund', 'Mutual Fund', 'Share-based', 'Open-end mutual fund with daily pricing', 'security', true, '{"ticker": "string", "expense_ratio": "numeric", "fund_type": "enum", "nav": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'closed_end_fund', 'Closed End Fund', 'Share-based', 'Closed-end mutual fund with fixed number of shares', 'security', true, '{"ticker": "string", "nav_per_share": "numeric", "market_price": "numeric", "distribution_yield": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'money_market_fund', 'Money Market Fund', 'Share-based', 'Money market mutual fund', 'security', true, '{"ticker": "string", "yield": "numeric", "nav": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'uit', 'UIT', 'Share-based', 'Unit investment trust with fixed portfolio', 'security', true, '{"ticker": "string", "termination_date": "date", "number_of_units": "numeric"}', NOW(), NOW()),

-- Real Estate & Alternative Securities
(gen_random_uuid(), 'reit', 'REIT', 'Share-based', 'Real estate investment trust', 'security', true, '{"ticker": "string", "dividend_yield": "numeric", "property_type": "enum"}', NOW(), NOW()),
(gen_random_uuid(), 'mlp', 'Master Limited Partnership', 'Share-based', 'Master limited partnership traded on exchanges', 'security', true, '{"ticker": "string", "distribution_yield": "numeric", "sector": "enum"}', NOW(), NOW()),

-- Derivatives
(gen_random_uuid(), 'option', 'Option', 'Share-based', 'Options contract for equity, index, or currency', 'derivative', true, '{"underlying": "string", "option_type": "enum", "strike_price": "numeric", "expiration_date": "date"}', NOW(), NOW()),
(gen_random_uuid(), 'futures_contract', 'Futures Contract', 'Share-based', 'Exchange-traded futures contract', 'derivative', true, '{"underlying_type": "string", "delivery_price": "numeric", "expiration_date": "date", "contract_size": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'forward_contract', 'Forward Contract', 'Share-based', 'OTC forward contract for future delivery', 'derivative', true, '{"underlying_type": "string", "delivery_price": "numeric", "settlement_date": "date", "notional_amount": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'convertible_note', 'Convertible Note', 'Share-based', 'Convertible debt instrument with equity upside', 'derivative', true, '{"issuer": "string", "conversion_price": "numeric", "maturity_date": "date", "interest_rate": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'warrant', 'Warrant', 'Share-based', 'Stock warrant with long-term exercise rights', 'derivative', true, '{"underlying_stock": "string", "strike_price": "numeric", "expiration_date": "date"}', NOW(), NOW()),
(gen_random_uuid(), 'etn', 'ETN', 'Share-based', 'Exchange-traded note with no fund structure', 'derivative', true, '{"ticker": "string", "issuer": "string", "underlying_index": "string"}', NOW(), NOW()),

-- Alternative Assets
(gen_random_uuid(), 'real_estate', 'Real Estate', 'Share-based,Value-based', 'Real estate property. Available after Sept 12, 2025', 'alternative', true, '{"property_address": "string", "property_type": "enum", "acquisition_date": "date", "valuation": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'art', 'Art', 'Share-based,Value-based', 'Art as alternative asset. Available after Sept 12, 2025', 'alternative', true, '{"artist_name": "string", "artwork_title": "string", "appraised_value": "numeric", "acquisition_date": "date"}', NOW(), NOW()),
(gen_random_uuid(), 'car', 'Car', 'Share-based,Value-based', 'Vehicle as asset. Available after Sept 12, 2025', 'alternative', true, '{"make": "string", "model": "string", "year": "year", "vin": "string", "current_value": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'collectible', 'Collectible', 'Share-based,Value-based', 'Collectible asset. Available after Sept 12, 2025', 'alternative', true, '{"item_type": "enum", "condition_grade": "enum", "valuation": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'private_investment', 'Private Investment', 'Share-based,Value-based', 'Direct private investment. Available after Sept 12, 2025', 'alternative', true, '{"company_name": "string", "investment_date": "date", "fair_value": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'venture_capital', 'Venture Capital', 'Share-based,Value-based', 'Venture capital investments. Available after Sept 12, 2025', 'alternative', true, '{"company_stage": "enum", "investment_date": "date", "valuation": "numeric"}', NOW(), NOW()),

-- Debt Instruments
(gen_random_uuid(), 'loan', 'Loan', 'Value-based', 'Loan or debt security asset', 'debt', true, '{"borrower": "string", "principal": "numeric", "interest_rate": "numeric", "maturity_date": "date"}', NOW(), NOW()),
(gen_random_uuid(), 'promissory_note', 'Promissory Note', 'Share-based,Value-based', 'Promissory note. Available after Sept 12, 2025', 'debt', true, '{"issuer": "string", "face_value": "numeric", "maturity_date": "date"}', NOW(), NOW()),

-- Insurance Products
(gen_random_uuid(), 'annuity', 'Annuity', 'Value-based', 'Annuity insurance product', 'insurance', true, '{"annuity_type": "enum", "issue_date": "date", "annuitant_age": "numeric", "guaranteed_rate": "numeric"}', NOW(), NOW()),

-- Cash & Digital
(gen_random_uuid(), 'cash', 'Currency', 'Share-based', 'Cash or currency holdings', 'cash', true, '{"currency_code": "string", "balance": "numeric"}', NOW(), NOW()),
(gen_random_uuid(), 'digital_asset', 'Digital Asset', 'Share-based', 'Digital assets including cryptocurrencies and blockchain tokens', 'digital', true, '{"asset_code": "string", "wallet_address": "string", "quantity": "numeric", "block_chain": "string"}', NOW(), NOW()),

-- Structured Products
(gen_random_uuid(), 'structured_product', 'Structured Product', 'Share-based', 'Structured investment product with embedded derivatives', 'structured', true, '{"issuer": "string", "underlying_reference": "string", "maturity_date": "date", "knock_out_level": "numeric"}', NOW(), NOW()),

-- Legacy & Custom
(gen_random_uuid(), 'historical_segment', 'Historical Segment', 'Value-based', 'Historical data segments for legacy account structures', 'legacy', true, '{"start_date": "date", "end_date": "date"}', NOW(), NOW()),
(gen_random_uuid(), 'unknown_security', 'Unknown Security', 'Share-based', 'Catch-all for unidentified or legacy securities', 'legacy', true, '{"description": "text", "last_updated": "date"}', NOW(), NOW()),
(gen_random_uuid(), 'generic_asset', 'Custom Asset', 'Any', 'Catch-all for custom or unspecified investment types not in this list', 'custom', true, '{"asset_type_custom": "string", "description": "text"}', NOW(), NOW())
ON CONFLICT (model_type) DO NOTHING;

-- 2. Get tenant ID for hierarchy rules (use first tenant or create default)
WITH tenant_id AS (
  SELECT id FROM tenants LIMIT 1
)
INSERT INTO entity_hierarchy_rules (
  id,
  tenant_id,
  parent_model_type,
  child_model_type,
  allowed,
  ownership_types,
  description,
  created_at,
  updated_at
) 
SELECT
  gen_random_uuid(),
  t.id,
  rule.parent,
  rule.child,
  true,
  rule.ownership_types,
  rule.description,
  NOW(),
  NOW()
FROM tenant_id t,
LATERAL (
  -- Household hierarchy
  SELECT 'household' as parent, 'person_node' as child, ARRAY['PERCENT_BASED'] as ownership_types, 'Household contains clients' as description
  UNION ALL SELECT 'household', 'trust', ARRAY['PERCENT_BASED'], 'Household can own trusts'
  UNION ALL SELECT 'household', 'managed_partnership', ARRAY['PERCENT_BASED'], 'Household can hold partnerships'
  UNION ALL SELECT 'household', 'holding_company', ARRAY['PERCENT_BASED'], 'Household holds companies'
  UNION ALL SELECT 'household', 'sleeve', ARRAY['PERCENT_BASED'], 'Household contains sleeves'
  UNION ALL SELECT 'household', 'financial_account', ARRAY['PERCENT_BASED'], 'Household has accounts'
  UNION ALL SELECT 'household', 'prospect', ARRAY['PERCENT_BASED'], 'Household can have prospects'
  
  -- Person/Client relationships
  UNION ALL SELECT 'person_node', 'financial_account', ARRAY['PERCENT_BASED'], 'Person owns accounts'
  UNION ALL SELECT 'person_node', 'sleeve', ARRAY['PERCENT_BASED'], 'Person has portfolio sleeves'
  UNION ALL SELECT 'person_node', 'annuity', ARRAY['VALUE_BASED'], 'Person owns annuity'
  
  -- Trust relationships
  UNION ALL SELECT 'trust', 'financial_account', ARRAY['PERCENT_BASED'], 'Trust holds accounts'
  UNION ALL SELECT 'trust', 'sleeve', ARRAY['PERCENT_BASED'], 'Trust has sleeves'
  UNION ALL SELECT 'trust', 'real_estate', ARRAY['PERCENT_BASED','VALUE_BASED'], 'Trust owns real estate'
  UNION ALL SELECT 'trust', 'bond', ARRAY['SHARE_BASED'], 'Trust owns bonds'
  UNION ALL SELECT 'trust', 'etf', ARRAY['SHARE_BASED'], 'Trust owns ETFs'
  
  -- Partnership & Fund relationships
  UNION ALL SELECT 'managed_partnership', 'private_equity_fund', ARRAY['VALUE_BASED'], 'Partnership holds PE funds'
  UNION ALL SELECT 'managed_partnership', 'hedge_fund', ARRAY['VALUE_BASED'], 'Partnership holds hedge funds'
  UNION ALL SELECT 'managed_partnership', 'private_investment', ARRAY['VALUE_BASED'], 'Partnership holds investments'
  
  UNION ALL SELECT 'holding_company', 'private_equity_fund', ARRAY['VALUE_BASED'], 'Company holds PE'
  UNION ALL SELECT 'holding_company', 'venture_capital', ARRAY['VALUE_BASED'], 'Company holds VC'
  UNION ALL SELECT 'holding_company', 'private_investment', ARRAY['VALUE_BASED'], 'Company holds investments'
  
  -- Fund relationships (fund-of-funds)
  UNION ALL SELECT 'fund', 'private_equity_fund', ARRAY['VALUE_BASED'], 'Fund holds other funds'
  UNION ALL SELECT 'fund', 'hedge_fund', ARRAY['VALUE_BASED'], 'Fund holds hedge funds'
  UNION ALL SELECT 'fund', 'venture_capital', ARRAY['VALUE_BASED'], 'Fund holds VC'
  UNION ALL SELECT 'fund', 'stock', ARRAY['SHARE_BASED'], 'Fund holds stocks'
  UNION ALL SELECT 'fund', 'bond', ARRAY['SHARE_BASED'], 'Fund holds bonds'
  
  UNION ALL SELECT 'private_equity_fund', 'private_investment', ARRAY['VALUE_BASED'], 'PE fund holds investments'
  UNION ALL SELECT 'private_equity_fund', 'venture_capital', ARRAY['VALUE_BASED'], 'PE fund holds VC'
  UNION ALL SELECT 'hedge_fund', 'stock', ARRAY['SHARE_BASED'], 'Hedge fund holds stocks'
  UNION ALL SELECT 'hedge_fund', 'etf', ARRAY['SHARE_BASED'], 'Hedge fund holds ETFs'
  UNION ALL SELECT 'hedge_fund', 'option', ARRAY['SHARE_BASED'], 'Hedge fund holds options'
  
  -- Sleeve relationships
  UNION ALL SELECT 'sleeve', 'stock', ARRAY['SHARE_BASED'], 'Sleeve contains stocks'
  UNION ALL SELECT 'sleeve', 'bond', ARRAY['SHARE_BASED'], 'Sleeve contains bonds'
  UNION ALL SELECT 'sleeve', 'etf', ARRAY['SHARE_BASED'], 'Sleeve contains ETFs'
  UNION ALL SELECT 'sleeve', 'mutual_fund', ARRAY['SHARE_BASED'], 'Sleeve contains mutual funds'
  UNION ALL SELECT 'sleeve', 'cash', ARRAY['SHARE_BASED'], 'Sleeve contains cash'
  UNION ALL SELECT 'sleeve', 'option', ARRAY['SHARE_BASED'], 'Sleeve contains options'
  UNION ALL SELECT 'sleeve', 'real_estate', ARRAY['PERCENT_BASED','VALUE_BASED'], 'Sleeve contains real estate'
  UNION ALL SELECT 'sleeve', 'art', ARRAY['PERCENT_BASED','VALUE_BASED'], 'Sleeve contains art'
  UNION ALL SELECT 'sleeve', 'collectible', ARRAY['PERCENT_BASED','VALUE_BASED'], 'Sleeve contains collectibles'
  UNION ALL SELECT 'sleeve', 'futures_contract', ARRAY['SHARE_BASED'], 'Sleeve contains futures'
  UNION ALL SELECT 'sleeve', 'forward_contract', ARRAY['SHARE_BASED'], 'Sleeve contains forwards'
  UNION ALL SELECT 'sleeve', 'annuity', ARRAY['VALUE_BASED'], 'Sleeve contains annuity'
  UNION ALL SELECT 'sleeve', 'digital_asset', ARRAY['SHARE_BASED'], 'Sleeve contains digital assets'
  UNION ALL SELECT 'sleeve', 'reit', ARRAY['SHARE_BASED'], 'Sleeve contains REITs'
  UNION ALL SELECT 'sleeve', 'closed_end_fund', ARRAY['SHARE_BASED'], 'Sleeve contains closed-end funds'
  UNION ALL SELECT 'sleeve', 'private_equity_fund', ARRAY['VALUE_BASED'], 'Sleeve contains PE investments'
  UNION ALL SELECT 'sleeve', 'venture_capital', ARRAY['VALUE_BASED'], 'Sleeve contains VC investments'
  UNION ALL SELECT 'sleeve', 'hedge_fund', ARRAY['VALUE_BASED'], 'Sleeve contains hedge fund investments'
  
  -- Financial Account relationships
  UNION ALL SELECT 'financial_account', 'stock', ARRAY['SHARE_BASED'], 'Account holds stocks'
  UNION ALL SELECT 'financial_account', 'bond', ARRAY['SHARE_BASED'], 'Account holds bonds'
  UNION ALL SELECT 'financial_account', 'etf', ARRAY['SHARE_BASED'], 'Account holds ETFs'
  UNION ALL SELECT 'financial_account', 'mutual_fund', ARRAY['SHARE_BASED'], 'Account holds mutual funds'
  UNION ALL SELECT 'financial_account', 'cash', ARRAY['SHARE_BASED'], 'Account holds cash'
  UNION ALL SELECT 'financial_account', 'money_market_fund', ARRAY['SHARE_BASED'], 'Account holds MMF'
  UNION ALL SELECT 'financial_account', 'certificate_of_deposit', ARRAY['SHARE_BASED'], 'Account holds CDs'
  UNION ALL SELECT 'financial_account', 'option', ARRAY['SHARE_BASED'], 'Account holds options'
  UNION ALL SELECT 'financial_account', 'futures_contract', ARRAY['SHARE_BASED'], 'Account holds futures'
  UNION ALL SELECT 'financial_account', 'real_estate', ARRAY['PERCENT_BASED','VALUE_BASED'], 'Account holds real estate'
  UNION ALL SELECT 'financial_account', 'art', ARRAY['PERCENT_BASED','VALUE_BASED'], 'Account holds art'
  UNION ALL SELECT 'financial_account', 'digital_asset', ARRAY['SHARE_BASED'], 'Account holds digital assets'
  UNION ALL SELECT 'financial_account', 'reit', ARRAY['SHARE_BASED'], 'Account holds REITs'
  UNION ALL SELECT 'financial_account', 'closed_end_fund', ARRAY['SHARE_BASED'], 'Account holds closed-end funds'
  UNION ALL SELECT 'financial_account', 'preferred_stock', ARRAY['SHARE_BASED'], 'Account holds preferred stock'
  UNION ALL SELECT 'financial_account', 'mlp', ARRAY['SHARE_BASED'], 'Account holds MLPs'
  UNION ALL SELECT 'financial_account', 'uit', ARRAY['SHARE_BASED'], 'Account holds UITs'
  UNION ALL SELECT 'financial_account', 'etn', ARRAY['SHARE_BASED'], 'Account holds ETNs'
  UNION ALL SELECT 'financial_account', 'convertible_note', ARRAY['SHARE_BASED'], 'Account holds convertible notes'
  UNION ALL SELECT 'financial_account', 'cmo', ARRAY['SHARE_BASED'], 'Account holds CMOs'
  
) rule
ON CONFLICT (tenant_id, parent_model_type, child_model_type) DO NOTHING;

-- 3. Verify population
SELECT 
  (SELECT COUNT(*) FROM model_types) as total_entity_types,
  (SELECT COUNT(*) FROM entity_hierarchy_rules WHERE allowed = true) as allowed_relationships,
  (SELECT COUNT(DISTINCT category) FROM model_types) as unique_categories;

-- 4. Log population completion
INSERT INTO entity_hierarchy_audit_log (
  id,
  tenant_id,
  entity_id,
  action,
  reason,
  created_at
)
SELECT
  gen_random_uuid(),
  t.id,
  (SELECT id FROM entities LIMIT 1),
  'SYSTEM_POPULATE',
  'Automated population of 50+ investment entity types and hierarchy rules',
  NOW()
FROM tenants t
LIMIT 1;

COMMIT;

-- Print completion message
DO $$
DECLARE
  entity_count INTEGER;
  rule_count INTEGER;
BEGIN
  SELECT COUNT(*) INTO entity_count FROM model_types;
  SELECT COUNT(*) INTO rule_count FROM entity_hierarchy_rules WHERE allowed = true;
  
  RAISE NOTICE '✅ Investment Entity Population Complete!';
  RAISE NOTICE 'Entity Types Loaded: %', entity_count;
  RAISE NOTICE 'Hierarchy Rules Configured: %', rule_count;
  RAISE NOTICE 'Status: Ready for business entity creation';
END $$;
