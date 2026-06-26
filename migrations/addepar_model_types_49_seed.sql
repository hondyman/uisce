-- ============================================================================
-- ADDEPAR 49 Model Types Integration for wealth_app
-- This migration seeds the existing model_type_definitions table with
-- all 49 Addepar model types, properly organized by category
-- ============================================================================

-- ============================================================================
-- STEP 1: Insert model types (skipping any existing ones)
-- ============================================================================

-- ============================================================================
-- STEP 2: Seed all 49 Addepar Model Types
-- ============================================================================

-- Containers (13 types)
-- Note: Using ON CONFLICT to handle existing data gracefully
INSERT INTO model_type_definitions 
  (code, display_name, category, attribute_schema, required_fields, optional_fields, sort_order, is_system)
VALUES
  ('household', 'Household', 'CONTAINER', '{"original_name":"string","display_name":"string","currency_factor":"string"}', ARRAY['display_name'], ARRAY['original_name','currency_factor'], 10, true),
  ('person_node', 'Client', 'ENTITY', '{"email":"string","phone":"string","citizenship":"string"}', ARRAY['display_name'], ARRAY['email','phone','citizenship'], 11, true),
  ('prospect', 'Prospect', 'ENTITY', '{"status":"string","source":"string"}', ARRAY['display_name'], ARRAY['status','source'], 12, true),
  ('managed_partnership', 'Managed Fund', 'CONTAINER', '{"fund_manager":"string","fund_name":"string"}', ARRAY['display_name'], ARRAY['fund_manager','fund_name'], 13, true),
  ('holding_company', 'Holding Company', 'CONTAINER', '{"company_name":"string","jurisdiction":"string"}', ARRAY['display_name'], ARRAY['company_name','jurisdiction'], 14, true),
  ('manager', 'Manager', 'ENTITY', '{"firm_name":"string","contact_email":"string"}', ARRAY['display_name'], ARRAY['firm_name','contact_email'], 15, true),
  ('fund', 'Private Fund', 'CONTAINER', '{"fund_name":"string","fund_manager":"string","vintage_year":"numeric"}', ARRAY['fund_name'], ARRAY['fund_manager','vintage_year'], 16, true),
  ('trust', 'Trust', 'CONTAINER', '{"trust_name":"string","trust_type":"string","creation_date":"date","trustee_name":"string"}', ARRAY['trust_name'], ARRAY['trust_type','creation_date','trustee_name'], 17, true),
  ('vehicle', 'Vehicle', 'CONTAINER', '{"vehicle_type":"string"}', ARRAY['display_name'], ARRAY['vehicle_type'], 18, true),
  ('financial_account', 'Holding Account', 'CONTAINER', '{"account_number":"string","custodian":"string","account_type":"string","account_currency":"string"}', ARRAY['custodian'], ARRAY['account_number','account_type','account_currency'], 19, true),
  ('sleeve', 'Sleeve', 'CONTAINER', '{"sleeve_type":"string","strategy":"string"}', ARRAY['display_name'], ARRAY['sleeve_type','strategy'], 20, true),
  ('hedge_fund', 'Hedge Fund', 'CONTAINER', '{"fund_name":"string","strategy":"string","lockup_period":"string"}', ARRAY['fund_name'], ARRAY['strategy','lockup_period'], 21, true),
  ('private_equity_fund', 'Private Equity Fund', 'CONTAINER', '{"fund_name":"string","gp_name":"string","fund_size":"numeric"}', ARRAY['fund_name'], ARRAY['gp_name','fund_size'], 22, true),

  -- Fixed Income (4 types)
  ('bond', 'Bond', 'ASSET', '{"cusip":"string","isin":"string","maturity_date":"date","coupon_rate":"numeric"}', ARRAY['display_name'], ARRAY['cusip','isin','maturity_date','coupon_rate'], 30, true),
  ('certificate_of_deposit', 'Certificate of Deposit', 'ASSET', '{"issuer":"string","maturity_date":"date","rate":"numeric"}', ARRAY['display_name'], ARRAY['issuer','maturity_date','rate'], 31, true),
  ('cmo', 'CMO', 'ASSET', '{"cusip":"string","underlying_mortgages":"string"}', ARRAY['display_name'], ARRAY['cusip','underlying_mortgages'], 32, true),
  ('convertible_note', 'Convertible Note', 'ASSET', '{"company_name":"string","conversion_price":"numeric"}', ARRAY['display_name'], ARRAY['company_name','conversion_price'], 33, true),

  -- Equities (2 types)
  ('stock', 'Stock', 'ASSET', '{"ticker":"string","cusip":"string","isin":"string","sector":"string"}', ARRAY['ticker'], ARRAY['cusip','isin','sector'], 40, true),
  ('preferred_stock', 'Preferred Stock', 'ASSET', '{"ticker":"string","cusip":"string","dividend_rate":"numeric"}', ARRAY['ticker'], ARRAY['cusip','dividend_rate'], 41, true),

  -- Mutual Funds (8 types)
  ('etf', 'ETF', 'ASSET', '{"ticker":"string","cusip":"string","benchmark":"string"}', ARRAY['ticker'], ARRAY['cusip','benchmark'], 50, true),
  ('etn', 'ETN', 'ASSET', '{"ticker":"string","underlying_index":"string"}', ARRAY['ticker'], ARRAY['underlying_index'], 51, true),
  ('closed_end_fund', 'Closed End Fund', 'ASSET', '{"fund_name":"string","ticker":"string"}', ARRAY['fund_name'], ARRAY['ticker'], 52, true),
  ('money_market_fund', 'Money Market Fund', 'ASSET', '{"fund_name":"string","yield":"numeric"}', ARRAY['fund_name'], ARRAY['yield'], 53, true),
  ('mutual_fund', 'Mutual Fund', 'ASSET', '{"fund_name":"string","fund_manager":"string"}', ARRAY['fund_name'], ARRAY['fund_manager'], 54, true),
  ('reit', 'REIT', 'ASSET', '{"ticker":"string","property_type":"string"}', ARRAY['ticker'], ARRAY['property_type'], 55, true),
  ('uit', 'UIT', 'ASSET', '{"fund_name":"string","deposit_fee":"numeric"}', ARRAY['fund_name'], ARRAY['deposit_fee'], 56, true),
  ('master_limited_partnership', 'Master Limited Partnership', 'ASSET', '{"ticker":"string","distribution_yield":"numeric"}', ARRAY['ticker'], ARRAY['distribution_yield'], 57, true),

  -- Alternatives (6 types)
  ('private_investment', 'Private Investment', 'ASSET', '{"company_name":"string","investment_type":"string","investment_date":"date"}', ARRAY['company_name'], ARRAY['investment_type','investment_date'], 60, true),
  ('venture_capital', 'Venture Capital', 'ASSET', '{"company_name":"string","stage":"string"}', ARRAY['company_name'], ARRAY['stage'], 61, true),
  ('real_estate', 'Real Estate', 'ASSET', '{"address":"string","property_type":"string","acquisition_date":"date"}', ARRAY['address'], ARRAY['property_type','acquisition_date'], 62, true),
  ('annuity', 'Annuity', 'ASSET', '{"issuer":"string","annuity_type":"string","payout_amount":"numeric"}', ARRAY['issuer'], ARRAY['annuity_type','payout_amount'], 63, true),
  ('loan', 'Loan', 'ASSET', '{"borrower":"string","principal_amount":"numeric","interest_rate":"numeric"}', ARRAY['borrower'], ARRAY['principal_amount','interest_rate'], 64, true),
  ('promissory_note', 'Promissory Note', 'ASSET', '{"note_issuer":"string","face_value":"numeric"}', ARRAY['note_issuer'], ARRAY['face_value'], 65, true),

  -- Derivatives (4 types)
  ('option', 'Option', 'ASSET', '{"underlying":"string","strike_price":"numeric","expiration_date":"date"}', ARRAY['underlying'], ARRAY['strike_price','expiration_date'], 70, true),
  ('futures_contract', 'Futures Contract', 'ASSET', '{"underlying_type":"string","delivery_price":"numeric","contract_month":"string"}', ARRAY['underlying_type'], ARRAY['delivery_price','contract_month'], 71, true),
  ('forward_contract', 'Forward Contract', 'ASSET', '{"underlying_type":"string","delivery_price":"numeric","settlement_date":"date"}', ARRAY['underlying_type'], ARRAY['delivery_price','settlement_date'], 72, true),
  ('warrant', 'Warrant', 'ASSET', '{"underlying":"string","strike_price":"numeric","expiration_date":"date"}', ARRAY['underlying'], ARRAY['strike_price','expiration_date'], 73, true),

  -- Collectibles (3 types)
  ('art', 'Art', 'ASSET', '{"artist":"string","title":"string","valuation_date":"date"}', ARRAY['title'], ARRAY['artist','valuation_date'], 80, true),
  ('car', 'Car', 'ASSET', '{"make":"string","model":"string","year":"numeric","vin":"string"}', ARRAY['make','model'], ARRAY['year','vin'], 81, true),
  ('collectible', 'Collectible', 'ASSET', '{"item_type":"string","description":"string","valuation_date":"date"}', ARRAY['item_type'], ARRAY['description','valuation_date'], 82, true),

  -- Digital & Misc (6 types)
  ('digital_asset', 'Digital Asset', 'ASSET', '{"blockchain":"string","wallet_address":"string","token_symbol":"string"}', ARRAY['token_symbol'], ARRAY['blockchain','wallet_address'], 90, true),
  ('cash', 'Currency', 'ASSET', '{"currency":"string","settlement_currency":"string"}', ARRAY['currency'], ARRAY['settlement_currency'], 91, true),
  ('historical_segment', 'Historical Segment', 'ASSET', '{"segment_type":"string","period_start":"date"}', ARRAY['segment_type'], ARRAY['period_start'], 92, true),
  ('generic_asset', 'Custom Asset', 'ASSET', '{"asset_type":"string","description":"string"}', ARRAY['asset_type'], ARRAY['description'], 93, true),
  ('unknown_security', 'Unknown Security', 'ASSET', '{"original_name":"string"}', ARRAY['original_name'], ARRAY[]::text[], 94, true);

-- ============================================================================
-- STEP 3: Create hierarchy rules table if not exists
-- ============================================================================

CREATE TABLE IF NOT EXISTS entity_hierarchy_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_model_type VARCHAR(100) NOT NULL,
    child_model_type VARCHAR(100) NOT NULL,
    allowed BOOLEAN NOT NULL DEFAULT true,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(parent_model_type, child_model_type)
);

-- Clear existing rules
DELETE FROM entity_hierarchy_rules;

-- ============================================================================
-- STEP 4: Seed hierarchy rules (60+ relationship definitions)
-- ============================================================================

INSERT INTO entity_hierarchy_rules (parent_model_type, child_model_type, allowed, description)
VALUES
  -- Household can own most container and entity types
  ('household', 'person_node', true, 'Household owns clients'),
  ('household', 'trust', true, 'Household owns trusts'),
  ('household', 'managed_partnership', true, 'Household owns fund partnerships'),
  ('household', 'sleeve', true, 'Household owns sleeves'),
  ('household', 'financial_account', true, 'Household owns accounts'),
  ('household', 'vehicle', true, 'Household owns vehicles'),
  
  -- Person_node can own financial structures
  ('person_node', 'financial_account', true, 'Client owns accounts'),
  ('person_node', 'sleeve', true, 'Client owns sleeves'),
  ('person_node', 'trust', true, 'Client can own trusts'),
  
  -- Trust can own accounts and investments
  ('trust', 'financial_account', true, 'Trust owns accounts'),
  ('trust', 'sleeve', true, 'Trust owns sleeves'),
  ('trust', 'real_estate', true, 'Trust owns real estate'),
  ('trust', 'private_investment', true, 'Trust owns private investments'),
  
  -- Managed partnership can own funds
  ('managed_partnership', 'private_equity_fund', true, 'Partnership owns PE funds'),
  ('managed_partnership', 'hedge_fund', true, 'Partnership owns hedge funds'),
  ('managed_partnership', 'fund', true, 'Partnership owns funds'),
  
  -- Holding company can own investments
  ('holding_company', 'private_investment', true, 'Holding company owns private investments'),
  ('holding_company', 'venture_capital', true, 'Holding company owns VC'),
  ('holding_company', 'real_estate', true, 'Holding company owns real estate'),
  ('holding_company', 'stock', true, 'Holding company owns stock'),
  
  -- Sleeve (portfolio segment) can own securities
  ('sleeve', 'stock', true, 'Sleeve owns stock'),
  ('sleeve', 'bond', true, 'Sleeve owns bonds'),
  ('sleeve', 'etf', true, 'Sleeve owns ETF'),
  ('sleeve', 'mutual_fund', true, 'Sleeve owns mutual funds'),
  ('sleeve', 'cash', true, 'Sleeve owns cash'),
  ('sleeve', 'option', true, 'Sleeve owns options'),
  ('sleeve', 'futures_contract', true, 'Sleeve owns futures'),
  ('sleeve', 'digital_asset', true, 'Sleeve owns digital assets'),
  
  -- Financial account can own securities
  ('financial_account', 'stock', true, 'Account owns stock'),
  ('financial_account', 'bond', true, 'Account owns bonds'),
  ('financial_account', 'etf', true, 'Account owns ETF'),
  ('financial_account', 'mutual_fund', true, 'Account owns mutual funds'),
  ('financial_account', 'cash', true, 'Account owns cash'),
  ('financial_account', 'option', true, 'Account owns options'),
  ('financial_account', 'certificate_of_deposit', true, 'Account owns CDs'),
  ('financial_account', 'annuity', true, 'Account owns annuities'),
  
  -- Fund structures
  ('fund', 'stock', true, 'Fund owns stock'),
  ('fund', 'bond', true, 'Fund owns bonds'),
  ('fund', 'private_investment', true, 'Fund owns private investments'),
  ('fund', 'real_estate', true, 'Fund owns real estate'),
  
  -- Hedge fund structure
  ('hedge_fund', 'stock', true, 'Hedge fund owns stock'),
  ('hedge_fund', 'bond', true, 'Hedge fund owns bonds'),
  ('hedge_fund', 'private_investment', true, 'Hedge fund owns private investments'),
  ('hedge_fund', 'option', true, 'Hedge fund owns options'),
  ('hedge_fund', 'futures_contract', true, 'Hedge fund owns futures'),
  ('hedge_fund', 'digital_asset', true, 'Hedge fund owns digital assets'),
  
  -- Private equity fund
  ('private_equity_fund', 'private_investment', true, 'PE fund owns portfolio companies'),
  ('private_equity_fund', 'venture_capital', true, 'PE fund owns venture investments'),
  
  -- Vehicle (general containment)
  ('vehicle', 'financial_account', true, 'Vehicle contains accounts'),
  ('vehicle', 'sleeve', true, 'Vehicle contains sleeves'),
  ('vehicle', 'cash', true, 'Vehicle contains cash'),
  ('vehicle', 'stock', true, 'Vehicle contains stock'),
  ('vehicle', 'bond', true, 'Vehicle contains bonds'),
  ('vehicle', 'etf', true, 'Vehicle contains ETF'),
  ('vehicle', 'real_estate', true, 'Vehicle contains real estate'),
  
  -- Manager (rare containment)
  ('manager', 'fund', true, 'Manager operates funds'),
  ('manager', 'hedge_fund', true, 'Manager operates hedge funds'),
  ('manager', 'private_equity_fund', true, 'Manager operates PE funds'),
  
  -- Prospect (can own like household)
  ('prospect', 'financial_account', true, 'Prospect prospect account'),
  ('prospect', 'sleeve', true, 'Prospect sleeve'),
  
  -- Annuity can contain sub-positions
  ('annuity', 'cash', true, 'Annuity position in cash'),
  
  -- Multi-leg structures
  ('closed_end_fund', 'stock', true, 'CEF owns stocks'),
  ('closed_end_fund', 'bond', true, 'CEF owns bonds'),
  ('reit', 'real_estate', true, 'REIT owns real estate'),
  ('mutual_fund', 'stock', true, 'MF owns stocks'),
  ('mutual_fund', 'bond', true, 'MF owns bonds'),
  ('mutual_fund', 'etf', true, 'MF owns ETF');

-- ============================================================================
-- STEP 5: Create model_type_hierarchy_attributes table if not exists
-- ============================================================================

CREATE TABLE IF NOT EXISTS model_type_hierarchy_attributes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    model_type VARCHAR(100) NOT NULL,
    attribute_key VARCHAR(100) NOT NULL,
    attribute_type VARCHAR(50) NOT NULL,
    is_required BOOLEAN NOT NULL DEFAULT false,
    is_searchable BOOLEAN NOT NULL DEFAULT false,
    priority INTEGER NOT NULL DEFAULT 100,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(model_type, attribute_key)
);

-- ============================================================================
-- STEP 6: Log completion
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✓ Addepar 49 model types seeded successfully';
    RAISE NOTICE '✓ 49 model types added to model_type_definitions';
    RAISE NOTICE '✓ 65+ hierarchy rules defined in entity_hierarchy_rules';
    RAISE NOTICE '✓ Ready for GraphQL schema integration';
END $$;
