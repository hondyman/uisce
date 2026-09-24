-- 20260824_008_add_missing_bo_fields_and_seed_data.up.sql
-- Adds missing bo_fields and seeds oms.position, oms.security, cash_flow.settlement.
-- Skips altinv.alternative_investment (schema doesn't match assumptions).

DO $$
DECLARE
    gold_copy_tenant_id UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    dummy_uuid UUID := '00000000-0000-0000-0000-000000000000';

    acc1_id UUID;
    acc2_id UUID;
    acc3_id UUID;
    acc4_id UUID;
    acc5_id UUID;
    acc6_id UUID;
    acc7_id UUID;
    acc8_id UUID;

    settled_long_bo_id UUID;
    equity_bo_id UUID;
    retail_wealth_bo_id UUID;
BEGIN

    -- ============================================================================
    -- PART 1: Add missing bo_fields
    -- ============================================================================

    SELECT id INTO settled_long_bo_id FROM business_objects WHERE tenant_id = gold_copy_tenant_id AND key = 'oms.position/settled_long';
    DELETE FROM bo_fields WHERE business_object_id = settled_long_bo_id;
    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, technical_name, display_name, type, is_core, is_required, is_system, sequence, created_at, last_modified_at)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'custody_account_id', 'custody_account_id', 'custody_account_id', 'Custody Account ID', 'string', false, false, false, 1, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'settled_shares', 'settled_shares', 'settled_shares', 'Settled Shares', 'decimal', false, false, false, 2, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'cost_basis_method', 'cost_basis_method', 'cost_basis_method', 'Cost Basis Method', 'string', false, false, false, 3, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'account_id', 'account_id', 'account_id', 'Account ID', 'string', false, false, false, 4, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'security_id', 'security_id', 'security_id', 'Security ID', 'string', false, false, false, 5, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'quantity', 'quantity', 'quantity', 'Quantity', 'decimal', false, false, false, 6, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'market_value', 'market_value', 'market_value', 'Market Value', 'decimal', false, false, false, 7, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'currency', 'currency', 'currency', 'Currency', 'string', false, false, false, 8, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'subtype_code', 'subtype_code', 'subtype_code', 'Subtype Code', 'string', false, false, false, 9, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'notional_amount', 'notional_amount', 'notional_amount', 'Notional Amount', 'decimal', false, false, false, 10, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'unrealized_pnl', 'unrealized_pnl', 'unrealized_pnl', 'Unrealized PnL', 'decimal', false, false, false, 11, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'held_to_maturity_flag', 'held_to_maturity_flag', 'held_to_maturity_flag', 'Held To Maturity Flag', 'boolean', false, false, false, 12, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'prime_broker_id', 'prime_broker_id', 'prime_broker_id', 'Prime Broker ID', 'string', false, false, false, 13, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'borrow_rate_bps', 'borrow_rate_bps', 'borrow_rate_bps', 'Borrow Rate BPS', 'decimal', false, false, false, 14, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'haircut_pct', 'haircut_pct', 'haircut_pct', 'Haircut PCT', 'decimal', false, false, false, 15, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'expiration_date', 'expiration_date', 'expiration_date', 'Expiration Date', 'date', false, false, false, 16, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, settled_long_bo_id, 'pledged_to_party', 'pledged_to_party', 'pledged_to_party', 'Pledged To Party', 'string', false, false, false, 17, NOW(), NOW());

    SELECT id INTO equity_bo_id FROM business_objects WHERE tenant_id = gold_copy_tenant_id AND key = 'oms.security/equity';
    DELETE FROM bo_fields WHERE business_object_id = equity_bo_id;
    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, technical_name, display_name, type, is_core, is_required, is_system, sequence, created_at, last_modified_at)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, equity_bo_id, 'ticker', 'ticker', 'ticker', 'Ticker', 'string', false, false, false, 1, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, equity_bo_id, 'isin', 'isin', 'isin', 'ISIN', 'string', false, false, false, 2, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, equity_bo_id, 'security_name', 'security_name', 'security_name', 'Security Name', 'string', false, false, false, 3, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, equity_bo_id, 'identifier_type', 'identifier_type', 'identifier_type', 'Identifier Type', 'string', false, false, false, 4, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, equity_bo_id, 'identifier_value', 'identifier_value', 'identifier_value', 'Identifier Value', 'string', false, false, false, 5, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, equity_bo_id, 'subtype_code', 'subtype_code', 'subtype_code', 'Subtype Code', 'string', false, false, false, 6, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, equity_bo_id, 'dividend_currency', 'dividend_currency', 'dividend_currency', 'Dividend Currency', 'string', false, false, false, 7, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, equity_bo_id, 'voting_rights_type', 'voting_rights_type', 'voting_rights_type', 'Voting Rights Type', 'string', false, false, false, 8, NOW(), NOW());

    SELECT id INTO retail_wealth_bo_id FROM business_objects WHERE tenant_id = gold_copy_tenant_id AND key = 'oms.account/retail_wealth';
    DELETE FROM bo_fields WHERE business_object_id = retail_wealth_bo_id;
    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, technical_name, display_name, type, is_core, is_required, is_system, sequence, created_at, last_modified_at)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, retail_wealth_bo_id, 'account_number', 'account_number', 'account_number', 'Account Number', 'string', false, false, false, 1, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, retail_wealth_bo_id, 'account_name', 'account_name', 'account_name', 'Account Name', 'string', false, false, false, 2, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, retail_wealth_bo_id, 'tax_id_type', 'tax_id_type', 'tax_id_type', 'Tax Id Type', 'string', false, false, false, 3, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, retail_wealth_bo_id, 'accredited_investor_status', 'accredited_investor_status', 'accredited_investor_status', 'Accredited Investor Status', 'boolean', false, false, false, 4, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, retail_wealth_bo_id, 'sponsor_firm', 'sponsor_firm', 'sponsor_firm', 'Sponsor Firm', 'string', false, false, false, 5, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, retail_wealth_bo_id, 'fee_schedule_code', 'fee_schedule_code', 'fee_schedule_code', 'Fee Schedule Code', 'string', false, false, false, 6, NOW(), NOW()),
      (gen_random_uuid(), gold_copy_tenant_id, retail_wealth_bo_id, 'subtype_code', 'subtype_code', 'subtype_code', 'Subtype Code', 'string', false, false, false, 7, NOW(), NOW());

    -- ============================================================================
    -- PART 2: Seed oms.position rows (settled_long with notional_amount)
    -- ============================================================================
    DELETE FROM oms.position WHERE tenant_id = gold_copy_tenant_id;

    SELECT id INTO acc1_id FROM oms.account WHERE tenant_id = gold_copy_tenant_id AND account_number = 'ACC-001';
    SELECT id INTO acc2_id FROM oms.account WHERE tenant_id = gold_copy_tenant_id AND account_number = 'ACC-002';
    SELECT id INTO acc3_id FROM oms.account WHERE tenant_id = gold_copy_tenant_id AND account_number = 'ACC-003';
    SELECT id INTO acc4_id FROM oms.account WHERE tenant_id = gold_copy_tenant_id AND account_number = 'ACC-004';
    SELECT id INTO acc5_id FROM oms.account WHERE tenant_id = gold_copy_tenant_id AND account_number = 'ACC-005';
    SELECT id INTO acc6_id FROM oms.account WHERE tenant_id = gold_copy_tenant_id AND account_number = 'ACC-006';
    SELECT id INTO acc7_id FROM oms.account WHERE tenant_id = gold_copy_tenant_id AND account_number = 'ACC-007';
    SELECT id INTO acc8_id FROM oms.account WHERE tenant_id = gold_copy_tenant_id AND account_number = 'ACC-008';

    INSERT INTO oms.position (id, tenant_id, account_id, security_id, subtype_code, quantity, market_value, currency, unrealized_pnl, settled_shares, custody_account_id, cost_basis_method, held_to_maturity_flag, notional_amount, created_at, updated_at, valid_from, valid_to)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, acc1_id, dummy_uuid, 'settled_long', 50000.00000000, 9500000.00, 'USD', 1250000.00, 50000.00000000, dummy_uuid, 'FIFO', false, 9500000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc1_id, dummy_uuid, 'settled_long', 25000.00000000, 12500000.00, 'USD', 850000.00, 25000.00000000, dummy_uuid, 'FIFO', false, 12500000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc2_id, dummy_uuid, 'settled_long', 15000.00000000, 2850000.00, 'USD', 350000.00, 15000.00000000, dummy_uuid, 'FIFO', false, 2850000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc3_id, dummy_uuid, 'settled_long', 8000.00000000, 1280000.00, 'USD', 180000.00, 8000.00000000, dummy_uuid, 'FIFO', false, 1280000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc4_id, dummy_uuid, 'settled_long', 2000000.00000000, 2000000.00, 'USD', 20000.00, 2000000.00000000, dummy_uuid, 'AMORTIZED', true, 2000000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc5_id, dummy_uuid, 'settled_long', 5000000.00000000, 5000000.00, 'USD', 50000.00, 5000000.00000000, dummy_uuid, 'AMORTIZED', true, 5000000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc6_id, dummy_uuid, 'settled_long', 5000.00000000, 2500000.00, 'USD', 125000.00, 5000.00000000, dummy_uuid, 'FIFO', false, 2500000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc7_id, dummy_uuid, 'settled_long', 10000.00000000, 1600000.00, 'USD', 100000.00, 10000.00000000, dummy_uuid, 'FIFO', false, 1600000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc8_id, dummy_uuid, 'settled_long', 30000.00000000, 5700000.00, 'USD', 450000.00, 30000.00000000, dummy_uuid, 'FIFO', false, 5700000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc1_id, dummy_uuid, 'settled_long', 12000.00000000, 1920000.00, 'USD', 220000.00, 12000.00000000, dummy_uuid, 'FIFO', false, 1920000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc2_id, dummy_uuid, 'settled_long', 18000.00000000, 3600000.00, 'USD', 280000.00, 18000.00000000, dummy_uuid, 'FIFO', false, 3600000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc3_id, dummy_uuid, 'settled_long', 22000.00000000, 4400000.00, 'USD', 320000.00, 22000.00000000, dummy_uuid, 'FIFO', false, 4400000.00, NOW(), NOW(), NOW(), NULL);

    -- pledged_collateral
    INSERT INTO oms.position (id, tenant_id, account_id, security_id, subtype_code, quantity, market_value, currency, unrealized_pnl, custody_account_id, prime_broker_id, borrow_rate_bps, pledged_to_party, haircut_pct, created_at, updated_at, valid_from, valid_to)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, acc1_id, dummy_uuid, 'pledged_collateral', 1500000.00000000, 1500000.00, 'USD', 15000.00, dummy_uuid, dummy_uuid, 350.0000, 'Prime Broker A', 0.0500, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, acc4_id, dummy_uuid, 'pledged_collateral', 2000000.00000000, 2000000.00, 'USD', 20000.00, dummy_uuid, dummy_uuid, 325.0000, 'Prime Broker B', 0.0400, NOW(), NOW(), NOW(), NULL);

    -- ============================================================================
    -- PART 3: Seed oms.security rows
    -- ============================================================================
    DELETE FROM oms.security WHERE tenant_id = gold_copy_tenant_id;

    INSERT INTO oms.security (id, tenant_id, security_name, identifier_type, identifier_value, subtype_code, ticker, isin, dividend_currency, created_at, updated_at, valid_from, valid_to)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, 'Apple Inc.', 'CUSIP', '037833100', 'equity', 'AAPL', 'US037833100', 'USD', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Microsoft Corporation', 'CUSIP', '594918104', 'equity', 'MSFT', 'US594918104', 'USD', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Alphabet Inc. Class A', 'CUSIP', '48125L956', 'equity', 'GOOGL', 'US48125L956', 'USD', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Tesla Inc.', 'CUSIP', '88160R101', 'equity', 'TSLA', 'US88160R101', 'USD', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Amazon.com Inc.', 'CUSIP', '023135106', 'equity', 'AMZN', 'US023135106', 'USD', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'SPDR S&P 500 ETF Trust', 'CUSIP', '78409V102', 'equity', 'SPY', 'US78409V102', 'USD', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Meta Platforms Inc.', 'CUSIP', '30303M102', 'equity', 'META', 'US30303M102', 'USD', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'US Treasury 10-Year Bond', 'CUSIP', '91282CHE5', 'sovereign_debt', 'UST10Y', 'US91282CHE52', 'USD', NOW(), NOW(), NOW(), NULL);

    -- ============================================================================
    -- PART 4: Seed cash_flow.settlement rows
    -- ============================================================================
    DELETE FROM cash_flow.settlement WHERE tenant_id = gold_copy_tenant_id;

    INSERT INTO cash_flow.settlement (id, tenant_id, subtype_code, account_id, amount, currency, settlement_date, due_date, settlement_status, created_at, updated_at, valid_from, valid_to)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, 'dividend', acc1_id, 125000.00, 'USD', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days', 'SETTLED', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'dividend', acc2_id, 87500.00, 'USD', NOW() - INTERVAL '12 days', NOW() - INTERVAL '12 days', 'SETTLED', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'dividend', acc6_id, 97500.00, 'USD', NOW() - INTERVAL '8 days', NOW() - INTERVAL '8 days', 'SETTLED', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'dividend', acc3_id, 65000.00, 'USD', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days', 'PENDING', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'dividend', acc7_id, 45000.00, 'USD', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day', 'SETTLED', NOW(), NOW(), NOW(), NULL);

    INSERT INTO cash_flow.settlement (id, tenant_id, subtype_code, account_id, amount, currency, settlement_date, due_date, settlement_status, return_of_capital, carried_interest_retained, created_at, updated_at, valid_from, valid_to)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, 'lp_distribution', acc1_id, 3500000.00, 'USD', NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days', 'SETTLED', 2800000.00, 700000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'lp_distribution', acc4_id, 4200000.00, 'USD', NOW() - INTERVAL '14 days', NOW() - INTERVAL '14 days', 'SETTLED', 3360000.00, 840000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'lp_distribution', acc5_id, 1800000.00, 'USD', NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days', 'SETTLED', 1440000.00, 360000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'lp_distribution', acc7_id, 2500000.00, 'USD', NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days', 'PENDING', 2000000.00, 500000.00, NOW(), NOW(), NOW(), NULL);

    INSERT INTO cash_flow.settlement (id, tenant_id, subtype_code, account_id, amount, currency, settlement_date, due_date, settlement_status, management_fee_portion, created_at, updated_at, valid_from, valid_to)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, 'capital_call', acc1_id, 5000000.00, 'USD', NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days', 'SETTLED', 50000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'capital_call', acc3_id, 3000000.00, 'USD', NOW() - INTERVAL '25 days', NOW() - INTERVAL '25 days', 'SETTLED', 30000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'capital_call', acc7_id, 4000000.00, 'USD', NOW() - INTERVAL '10 days', NOW() - INTERVAL '10 days', 'PENDING', 40000.00, NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'capital_call', acc8_id, 2500000.00, 'USD', NOW() - INTERVAL '15 days', NOW() - INTERVAL '15 days', 'SETTLED', 25000.00, NOW(), NOW(), NOW(), NULL);

    INSERT INTO cash_flow.settlement (id, tenant_id, subtype_code, account_id, amount, currency, settlement_date, due_date, settlement_status, created_at, updated_at, valid_from, valid_to)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, 'expense_fee', acc5_id, 45000.00, 'USD', NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days', 'PENDING', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'expense_fee', acc8_id, 32000.00, 'USD', NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days', 'APPROVED', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'coupon_fixed_income', acc4_id, 250000.00, 'USD', NOW() - INTERVAL '15 days', NOW() - INTERVAL '15 days', 'SETTLED', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'coupon_fixed_income', acc5_id, 180000.00, 'USD', NOW() - INTERVAL '18 days', NOW() - INTERVAL '18 days', 'SETTLED', NOW(), NOW(), NOW(), NULL);

END $$;
