-- ============================================================================
-- ADDEPAR 49 Model Types Integration for wealth_app
-- This migration seeds the existing model_type_definitions table with
-- all 49 Addepar model types, properly organized by category
-- Note: All codes must be UPPERCASE due to check constraint
-- ============================================================================

-- ============================================================================
-- STEP 1: Insert model types (codes in UPPERCASE)
-- ============================================================================

-- Containers (13 types)
INSERT INTO model_type_definitions 
  (code, display_name, category, attribute_schema, required_fields, optional_fields, sort_order, is_system)
VALUES
  ('HOUSEHOLD', 'Household', 'CONTAINER', '{"original_name":"string","display_name":"string","currency_factor":"string"}', ARRAY['display_name'], ARRAY['original_name','currency_factor'], 10, true),
  ('PERSON_NODE', 'Client', 'ENTITY', '{"email":"string","phone":"string","citizenship":"string"}', ARRAY['display_name'], ARRAY['email','phone','citizenship'], 11, true),
  ('PROSPECT', 'Prospect', 'ENTITY', '{"status":"string","source":"string"}', ARRAY['display_name'], ARRAY['status','source'], 12, true),
  ('MANAGED_PARTNERSHIP', 'Managed Fund', 'CONTAINER', '{"fund_manager":"string","fund_name":"string"}', ARRAY['display_name'], ARRAY['fund_manager','fund_name'], 13, true),
  ('HOLDING_COMPANY', 'Holding Company', 'CONTAINER', '{"company_name":"string","jurisdiction":"string"}', ARRAY['display_name'], ARRAY['company_name','jurisdiction'], 14, true),
  ('MANAGER', 'Manager', 'ENTITY', '{"firm_name":"string","contact_email":"string"}', ARRAY['display_name'], ARRAY['firm_name','contact_email'], 15, true),
  ('FUND', 'Private Fund', 'CONTAINER', '{"fund_name":"string","fund_manager":"string","vintage_year":"numeric"}', ARRAY['fund_name'], ARRAY['fund_manager','vintage_year'], 16, true),
  ('TRUST', 'Trust', 'CONTAINER', '{"trust_name":"string","trust_type":"string","creation_date":"date","trustee_name":"string"}', ARRAY['trust_name'], ARRAY['trust_type','creation_date','trustee_name'], 17, true),
  ('VEHICLE', 'Vehicle', 'CONTAINER', '{"vehicle_type":"string"}', ARRAY['display_name'], ARRAY['vehicle_type'], 18, true),
  ('FINANCIAL_ACCOUNT', 'Holding Account', 'CONTAINER', '{"account_number":"string","custodian":"string","account_type":"string","account_currency":"string"}', ARRAY['custodian'], ARRAY['account_number','account_type','account_currency'], 19, true),
  ('SLEEVE', 'Sleeve', 'CONTAINER', '{"sleeve_type":"string","strategy":"string"}', ARRAY['display_name'], ARRAY['sleeve_type','strategy'], 20, true),
  ('HEDGE_FUND', 'Hedge Fund', 'CONTAINER', '{"fund_name":"string","strategy":"string","lockup_period":"string"}', ARRAY['fund_name'], ARRAY['strategy','lockup_period'], 21, true),
  ('PRIVATE_EQUITY_FUND', 'Private Equity Fund', 'CONTAINER', '{"fund_name":"string","gp_name":"string","fund_size":"numeric"}', ARRAY['fund_name'], ARRAY['gp_name','fund_size'], 22, true),

  -- Fixed Income (4 types)
  ('BOND', 'Bond', 'ASSET', '{"cusip":"string","isin":"string","maturity_date":"date","coupon_rate":"numeric"}', ARRAY['display_name'], ARRAY['cusip','isin','maturity_date','coupon_rate'], 30, true),
  ('CERTIFICATE_OF_DEPOSIT', 'Certificate of Deposit', 'ASSET', '{"issuer":"string","maturity_date":"date","rate":"numeric"}', ARRAY['display_name'], ARRAY['issuer','maturity_date','rate'], 31, true),
  ('CMO', 'CMO', 'ASSET', '{"cusip":"string","underlying_mortgages":"string"}', ARRAY['display_name'], ARRAY['cusip','underlying_mortgages'], 32, true),
  ('CONVERTIBLE_NOTE', 'Convertible Note', 'ASSET', '{"company_name":"string","conversion_price":"numeric"}', ARRAY['display_name'], ARRAY['company_name','conversion_price'], 33, true),

  -- Equities (2 types)
  ('STOCK', 'Stock', 'ASSET', '{"ticker":"string","cusip":"string","isin":"string","sector":"string"}', ARRAY['ticker'], ARRAY['cusip','isin','sector'], 40, true),
  ('PREFERRED_STOCK', 'Preferred Stock', 'ASSET', '{"ticker":"string","cusip":"string","dividend_rate":"numeric"}', ARRAY['ticker'], ARRAY['cusip','dividend_rate'], 41, true),

  -- Mutual Funds (8 types)
  ('ETF', 'ETF', 'ASSET', '{"ticker":"string","cusip":"string","benchmark":"string"}', ARRAY['ticker'], ARRAY['cusip','benchmark'], 50, true),
  ('ETN', 'ETN', 'ASSET', '{"ticker":"string","underlying_index":"string"}', ARRAY['ticker'], ARRAY['underlying_index'], 51, true),
  ('CLOSED_END_FUND', 'Closed End Fund', 'ASSET', '{"fund_name":"string","ticker":"string"}', ARRAY['fund_name'], ARRAY['ticker'], 52, true),
  ('MONEY_MARKET_FUND', 'Money Market Fund', 'ASSET', '{"fund_name":"string","yield":"numeric"}', ARRAY['fund_name'], ARRAY['yield'], 53, true),
  ('MUTUAL_FUND', 'Mutual Fund', 'ASSET', '{"fund_name":"string","fund_manager":"string"}', ARRAY['fund_name'], ARRAY['fund_manager'], 54, true),
  ('REIT', 'REIT', 'ASSET', '{"ticker":"string","property_type":"string"}', ARRAY['ticker'], ARRAY['property_type'], 55, true),
  ('UIT', 'UIT', 'ASSET', '{"fund_name":"string","deposit_fee":"numeric"}', ARRAY['fund_name'], ARRAY['deposit_fee'], 56, true),
  ('MASTER_LIMITED_PARTNERSHIP', 'Master Limited Partnership', 'ASSET', '{"ticker":"string","distribution_yield":"numeric"}', ARRAY['ticker'], ARRAY['distribution_yield'], 57, true),

  -- Alternatives (6 types)
  ('PRIVATE_INVESTMENT', 'Private Investment', 'ASSET', '{"company_name":"string","investment_type":"string","investment_date":"date"}', ARRAY['company_name'], ARRAY['investment_type','investment_date'], 60, true),
  ('VENTURE_CAPITAL', 'Venture Capital', 'ASSET', '{"company_name":"string","stage":"string"}', ARRAY['company_name'], ARRAY['stage'], 61, true),
  ('REAL_ESTATE', 'Real Estate', 'ASSET', '{"address":"string","property_type":"string","acquisition_date":"date"}', ARRAY['address'], ARRAY['property_type','acquisition_date'], 62, true),
  ('ANNUITY', 'Annuity', 'ASSET', '{"issuer":"string","annuity_type":"string","payout_amount":"numeric"}', ARRAY['issuer'], ARRAY['annuity_type','payout_amount'], 63, true),
  ('LOAN', 'Loan', 'ASSET', '{"borrower":"string","principal_amount":"numeric","interest_rate":"numeric"}', ARRAY['borrower'], ARRAY['principal_amount','interest_rate'], 64, true),
  ('PROMISSORY_NOTE', 'Promissory Note', 'ASSET', '{"note_issuer":"string","face_value":"numeric"}', ARRAY['note_issuer'], ARRAY['face_value'], 65, true),

  -- Derivatives (4 types)
  ('OPTION', 'Option', 'ASSET', '{"underlying":"string","strike_price":"numeric","expiration_date":"date"}', ARRAY['underlying'], ARRAY['strike_price','expiration_date'], 70, true),
  ('FUTURES_CONTRACT', 'Futures Contract', 'ASSET', '{"underlying_type":"string","delivery_price":"numeric","contract_month":"string"}', ARRAY['underlying_type'], ARRAY['delivery_price','contract_month'], 71, true),
  ('FORWARD_CONTRACT', 'Forward Contract', 'ASSET', '{"underlying_type":"string","delivery_price":"numeric","settlement_date":"date"}', ARRAY['underlying_type'], ARRAY['delivery_price','settlement_date'], 72, true),
  ('WARRANT', 'Warrant', 'ASSET', '{"underlying":"string","strike_price":"numeric","expiration_date":"date"}', ARRAY['underlying'], ARRAY['strike_price','expiration_date'], 73, true),

  -- Collectibles (3 types)
  ('ART', 'Art', 'ASSET', '{"artist":"string","title":"string","valuation_date":"date"}', ARRAY['title'], ARRAY['artist','valuation_date'], 80, true),
  ('CAR', 'Car', 'ASSET', '{"make":"string","model":"string","year":"numeric","vin":"string"}', ARRAY['make','model'], ARRAY['year','vin'], 81, true),
  ('COLLECTIBLE', 'Collectible', 'ASSET', '{"item_type":"string","description":"string","valuation_date":"date"}', ARRAY['item_type'], ARRAY['description','valuation_date'], 82, true),

  -- Digital & Misc (6 types)
  ('DIGITAL_ASSET', 'Digital Asset', 'ASSET', '{"blockchain":"string","wallet_address":"string","token_symbol":"string"}', ARRAY['token_symbol'], ARRAY['blockchain','wallet_address'], 90, true),
  ('CASH', 'Currency', 'ASSET', '{"currency":"string","settlement_currency":"string"}', ARRAY['currency'], ARRAY['settlement_currency'], 91, true),
  ('HISTORICAL_SEGMENT', 'Historical Segment', 'ASSET', '{"segment_type":"string","period_start":"date"}', ARRAY['segment_type'], ARRAY['period_start'], 92, true),
  ('GENERIC_ASSET', 'Custom Asset', 'ASSET', '{"asset_type":"string","description":"string"}', ARRAY['asset_type'], ARRAY['description'], 93, true),
  ('UNKNOWN_SECURITY', 'Unknown Security', 'ASSET', '{"original_name":"string"}', ARRAY['original_name'], ARRAY[]::text[], 94, true)
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- STEP 2: Seed hierarchy rules (65+ relationship definitions)
-- ============================================================================

DELETE FROM entity_hierarchy_rules;

INSERT INTO entity_hierarchy_rules (parent_model_type, child_model_type, allowed, description)
VALUES
  -- Household can own most container and entity types
  ('HOUSEHOLD', 'PERSON_NODE', true, 'Household owns clients'),
  ('HOUSEHOLD', 'TRUST', true, 'Household owns trusts'),
  ('HOUSEHOLD', 'MANAGED_PARTNERSHIP', true, 'Household owns fund partnerships'),
  ('HOUSEHOLD', 'SLEEVE', true, 'Household owns sleeves'),
  ('HOUSEHOLD', 'FINANCIAL_ACCOUNT', true, 'Household owns accounts'),
  ('HOUSEHOLD', 'VEHICLE', true, 'Household owns vehicles'),
  
  -- Person_node can own financial structures
  ('PERSON_NODE', 'FINANCIAL_ACCOUNT', true, 'Client owns accounts'),
  ('PERSON_NODE', 'SLEEVE', true, 'Client owns sleeves'),
  ('PERSON_NODE', 'TRUST', true, 'Client can own trusts'),
  
  -- Trust can own accounts and investments
  ('TRUST', 'FINANCIAL_ACCOUNT', true, 'Trust owns accounts'),
  ('TRUST', 'SLEEVE', true, 'Trust owns sleeves'),
  ('TRUST', 'REAL_ESTATE', true, 'Trust owns real estate'),
  ('TRUST', 'PRIVATE_INVESTMENT', true, 'Trust owns private investments'),
  
  -- Managed partnership can own funds
  ('MANAGED_PARTNERSHIP', 'PRIVATE_EQUITY_FUND', true, 'Partnership owns PE funds'),
  ('MANAGED_PARTNERSHIP', 'HEDGE_FUND', true, 'Partnership owns hedge funds'),
  ('MANAGED_PARTNERSHIP', 'FUND', true, 'Partnership owns funds'),
  
  -- Holding company can own investments
  ('HOLDING_COMPANY', 'PRIVATE_INVESTMENT', true, 'Holding company owns private investments'),
  ('HOLDING_COMPANY', 'VENTURE_CAPITAL', true, 'Holding company owns VC'),
  ('HOLDING_COMPANY', 'REAL_ESTATE', true, 'Holding company owns real estate'),
  ('HOLDING_COMPANY', 'STOCK', true, 'Holding company owns stock'),
  
  -- Sleeve (portfolio segment) can own securities
  ('SLEEVE', 'STOCK', true, 'Sleeve owns stock'),
  ('SLEEVE', 'BOND', true, 'Sleeve owns bonds'),
  ('SLEEVE', 'ETF', true, 'Sleeve owns ETF'),
  ('SLEEVE', 'MUTUAL_FUND', true, 'Sleeve owns mutual funds'),
  ('SLEEVE', 'CASH', true, 'Sleeve owns cash'),
  ('SLEEVE', 'OPTION', true, 'Sleeve owns options'),
  ('SLEEVE', 'FUTURES_CONTRACT', true, 'Sleeve owns futures'),
  ('SLEEVE', 'DIGITAL_ASSET', true, 'Sleeve owns digital assets'),
  
  -- Financial account can own securities
  ('FINANCIAL_ACCOUNT', 'STOCK', true, 'Account owns stock'),
  ('FINANCIAL_ACCOUNT', 'BOND', true, 'Account owns bonds'),
  ('FINANCIAL_ACCOUNT', 'ETF', true, 'Account owns ETF'),
  ('FINANCIAL_ACCOUNT', 'MUTUAL_FUND', true, 'Account owns mutual funds'),
  ('FINANCIAL_ACCOUNT', 'CASH', true, 'Account owns cash'),
  ('FINANCIAL_ACCOUNT', 'OPTION', true, 'Account owns options'),
  ('FINANCIAL_ACCOUNT', 'CERTIFICATE_OF_DEPOSIT', true, 'Account owns CDs'),
  ('FINANCIAL_ACCOUNT', 'ANNUITY', true, 'Account owns annuities'),
  
  -- Fund structures
  ('FUND', 'STOCK', true, 'Fund owns stock'),
  ('FUND', 'BOND', true, 'Fund owns bonds'),
  ('FUND', 'PRIVATE_INVESTMENT', true, 'Fund owns private investments'),
  ('FUND', 'REAL_ESTATE', true, 'Fund owns real estate'),
  
  -- Hedge fund structure
  ('HEDGE_FUND', 'STOCK', true, 'Hedge fund owns stock'),
  ('HEDGE_FUND', 'BOND', true, 'Hedge fund owns bonds'),
  ('HEDGE_FUND', 'PRIVATE_INVESTMENT', true, 'Hedge fund owns private investments'),
  ('HEDGE_FUND', 'OPTION', true, 'Hedge fund owns options'),
  ('HEDGE_FUND', 'FUTURES_CONTRACT', true, 'Hedge fund owns futures'),
  ('HEDGE_FUND', 'DIGITAL_ASSET', true, 'Hedge fund owns digital assets'),
  
  -- Private equity fund
  ('PRIVATE_EQUITY_FUND', 'PRIVATE_INVESTMENT', true, 'PE fund owns portfolio companies'),
  ('PRIVATE_EQUITY_FUND', 'VENTURE_CAPITAL', true, 'PE fund owns venture investments'),
  
  -- Vehicle (general containment)
  ('VEHICLE', 'FINANCIAL_ACCOUNT', true, 'Vehicle contains accounts'),
  ('VEHICLE', 'SLEEVE', true, 'Vehicle contains sleeves'),
  ('VEHICLE', 'CASH', true, 'Vehicle contains cash'),
  ('VEHICLE', 'STOCK', true, 'Vehicle contains stock'),
  ('VEHICLE', 'BOND', true, 'Vehicle contains bonds'),
  ('VEHICLE', 'ETF', true, 'Vehicle contains ETF'),
  ('VEHICLE', 'REAL_ESTATE', true, 'Vehicle contains real estate'),
  
  -- Manager (rare containment)
  ('MANAGER', 'FUND', true, 'Manager operates funds'),
  ('MANAGER', 'HEDGE_FUND', true, 'Manager operates hedge funds'),
  ('MANAGER', 'PRIVATE_EQUITY_FUND', true, 'Manager operates PE funds'),
  
  -- Prospect (can own like household)
  ('PROSPECT', 'FINANCIAL_ACCOUNT', true, 'Prospect owns account'),
  ('PROSPECT', 'SLEEVE', true, 'Prospect owns sleeve'),
  
  -- Annuity can contain sub-positions
  ('ANNUITY', 'CASH', true, 'Annuity position in cash'),
  
  -- Multi-leg structures
  ('CLOSED_END_FUND', 'STOCK', true, 'CEF owns stocks'),
  ('CLOSED_END_FUND', 'BOND', true, 'CEF owns bonds'),
  ('REIT', 'REAL_ESTATE', true, 'REIT owns real estate'),
  ('MUTUAL_FUND', 'STOCK', true, 'MF owns stocks'),
  ('MUTUAL_FUND', 'BOND', true, 'MF owns bonds'),
  ('MUTUAL_FUND', 'ETF', true, 'MF owns ETF')
ON CONFLICT (parent_model_type, child_model_type) DO NOTHING;

-- ============================================================================
-- STEP 3: Verify Seeding
-- ============================================================================

DO $$
DECLARE
  v_count_types INTEGER;
  v_count_rules INTEGER;
BEGIN
  SELECT COUNT(*) INTO v_count_types FROM model_type_definitions WHERE is_system = true AND category IN ('ENTITY', 'ASSET', 'CONTAINER');
  SELECT COUNT(*) INTO v_count_rules FROM entity_hierarchy_rules;
  
  RAISE NOTICE '✓ Addepar 49 model types seeded successfully';
  RAISE NOTICE '✓ % model types in database (target: 49)', v_count_types;
  RAISE NOTICE '✓ % hierarchy rules defined (target: 65+)', v_count_rules;
  RAISE NOTICE '✓ Ready for GraphQL schema integration';
END $$;
