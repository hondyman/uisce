-- ============================================================================
-- QUICK START: Load Sample Data for Addepar Platform
-- Database: wealth_app
-- Description: Create sample portfolio with holdings for testing
-- ============================================================================

-- Get or create tenant (adjust if you have existing tenants)
DO $$
DECLARE
    org_id UUID;
    user_id UUID;
BEGIN
    -- Get existing organization or note the ID
    SELECT id INTO org_id FROM organizations LIMIT 1;
    IF org_id IS NULL THEN
        INSERT INTO organizations (name, email, status) 
        VALUES ('Demo Wealth Firm', 'demo@wealthfirm.com', 'ACTIVE')
        RETURNING id INTO org_id;
    END IF;
    
    -- Get existing user
    SELECT id INTO user_id FROM users LIMIT 1;
    
    -- Create sample CLIENT entity
    INSERT INTO entities (
        model_type, tenant_id, original_name, display_name,
        ownership_type, status, created_by
    ) VALUES (
        'CLIENT', org_id, 'John Doe', 'John Doe',
        'PERCENT_BASED', 'ACTIVE', user_id
    ) ON CONFLICT DO NOTHING;
    
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Sample data setup: %', SQLERRM;
END $$;

-- ============================================================================
-- SCENARIO: Create a $500K Growth Portfolio
-- ============================================================================

-- Step 1: Get or create IDs
DO $$
DECLARE
    client_id UUID;
    portfolio_id UUID;
    account_id UUID;
    org_id UUID;
    user_id UUID;
    
    apple_id UUID;
    msft_id UUID;
    spy_id UUID;
    bond_id UUID;
    cash_id UUID;
    
    pos1 UUID;
    pos2 UUID;
    pos3 UUID;
    pos4 UUID;
    pos5 UUID;
BEGIN
    -- Get IDs
    SELECT id INTO org_id FROM organizations LIMIT 1;
    SELECT id INTO user_id FROM users LIMIT 1;
    
    -- Create CLIENT if not exists
    INSERT INTO entities (
        model_type, tenant_id, original_name, display_name,
        status, source_system
    ) VALUES (
        'CLIENT', org_id, 'Sample Portfolio Owner', 'Sample Portfolio Owner',
        'ACTIVE', 'internal'
    ) ON CONFLICT DO NOTHING
    RETURNING id INTO client_id;
    
    IF client_id IS NULL THEN
        SELECT id INTO client_id FROM entities 
        WHERE model_type = 'CLIENT' 
        AND display_name = 'Sample Portfolio Owner' LIMIT 1;
    END IF;
    
    -- Create PORTFOLIO entity
    INSERT INTO entities (
        model_type, tenant_id, original_name, display_name,
        ownership_type, status, source_system, created_by
    ) VALUES (
        'PORTFOLIO', org_id, 'Growth Portfolio 2025', 'Growth Portfolio 2025',
        'VALUE_BASED', 'ACTIVE', 'internal', user_id
    ) RETURNING id INTO portfolio_id;
    
    -- Add portfolio attributes
    INSERT INTO entity_attributes (entity_id, attributes, created_by) VALUES (
        portfolio_id,
        jsonb_build_object(
            'portfolio_name', 'Growth Portfolio 2025',
            'strategy', 'Growth',
            'benchmark_symbol', 'SPY',
            'inception_date', '2025-01-01'::date,
            'target_allocation', jsonb_build_object(
                'US_EQUITIES', 60,
                'INTERNATIONAL', 20,
                'BONDS', 15,
                'CASH', 5
            )
        ),
        user_id
    );
    
    -- Create securities/assets as entities
    -- 1. Apple Stock
    INSERT INTO entities (
        model_type, tenant_id, original_name, display_name,
        ticker, status, ownership_type, source_system
    ) VALUES (
        'STOCK', org_id, 'Apple Inc', 'Apple Inc',
        'AAPL', 'ACTIVE', 'SHARE_BASED', 'internal'
    ) RETURNING id INTO apple_id;
    
    INSERT INTO entity_attributes (entity_id, attributes, created_by) VALUES (
        apple_id,
        jsonb_build_object(
            'sector', 'Technology',
            'industry', 'Consumer Electronics',
            'market_cap', 2800000000000,
            'pe_ratio', 28.5,
            'dividend_yield', 0.4
        ),
        user_id
    );
    
    -- 2. Microsoft Stock
    INSERT INTO entities (
        model_type, tenant_id, original_name, display_name,
        ticker, status, ownership_type, source_system
    ) VALUES (
        'STOCK', org_id, 'Microsoft Corp', 'Microsoft Corp',
        'MSFT', 'ACTIVE', 'SHARE_BASED', 'internal'
    ) RETURNING id INTO msft_id;
    
    INSERT INTO entity_attributes (entity_id, attributes, created_by) VALUES (
        msft_id,
        jsonb_build_object(
            'sector', 'Technology',
            'industry', 'Software',
            'market_cap', 2500000000000,
            'pe_ratio', 32.0,
            'dividend_yield', 0.7
        ),
        user_id
    );
    
    -- 3. SPY ETF
    INSERT INTO entities (
        model_type, tenant_id, original_name, display_name,
        ticker, status, ownership_type, source_system
    ) VALUES (
        'ETF', org_id, 'SPDR S&P 500', 'SPDR S&P 500 ETF',
        'SPY', 'ACTIVE', 'SHARE_BASED', 'internal'
    ) RETURNING id INTO spy_id;
    
    INSERT INTO entity_attributes (entity_id, attributes, created_by) VALUES (
        spy_id,
        jsonb_build_object(
            'expense_ratio', 0.03,
            'aum', 500000000000,
            'inception_date', '1993-01-22'::date,
            'index_tracked', 'S&P 500'
        ),
        user_id
    );
    
    -- 4. Bond ETF
    INSERT INTO entities (
        model_type, tenant_id, original_name, display_name,
        ticker, status, ownership_type, source_system
    ) VALUES (
        'ETF', org_id, 'iShares Core US Aggregate Bond', 'AGG ETF',
        'AGG', 'ACTIVE', 'SHARE_BASED', 'internal'
    ) RETURNING id INTO bond_id;
    
    INSERT INTO entity_attributes (entity_id, attributes, created_by) VALUES (
        bond_id,
        jsonb_build_object(
            'expense_ratio', 0.04,
            'aum', 100000000000,
            'inception_date', '2003-09-22'::date,
            'index_tracked', 'US Aggregate Bond'
        ),
        user_id
    );
    
    -- 5. Cash Account
    INSERT INTO entities (
        model_type, tenant_id, original_name, display_name,
        status, ownership_type, source_system
    ) VALUES (
        'CASH', org_id, 'Money Market Account', 'Cash Position',
        'ACTIVE', 'VALUE_BASED', 'internal'
    ) RETURNING id INTO cash_id;
    
    INSERT INTO entity_attributes (entity_id, attributes, created_by) VALUES (
        cash_id,
        jsonb_build_object(
            'bank', 'Fidelity',
            'account_number', 'XX1234567',
            'interest_rate', 5.25,
            'fdic_insured', true
        ),
        user_id
    );
    
    -- Step 2: Create positions (ownership relationships)
    
    -- Position 1: AAPL - 500 shares @ $180 cost basis
    INSERT INTO positions (
        owner_id, owned_id, shares, cost_basis, market_value,
        average_cost_per_unit, average_market_price,
        incepting_date, as_of_date, status, tenant_id, created_by
    ) VALUES (
        portfolio_id, apple_id, 500, 90000, 102000,
        180, 204, '2024-01-15'::date, CURRENT_DATE, 'ACTIVE', org_id, user_id
    ) RETURNING id INTO pos1;
    
    -- Position 2: MSFT - 300 shares @ $240 cost basis
    INSERT INTO positions (
        owner_id, owned_id, shares, cost_basis, market_value,
        average_cost_per_unit, average_market_price,
        incepting_date, as_of_date, status, tenant_id, created_by
    ) VALUES (
        portfolio_id, msft_id, 300, 72000, 102000,
        240, 340, '2024-02-20'::date, CURRENT_DATE, 'ACTIVE', org_id, user_id
    ) RETURNING id INTO pos2;
    
    -- Position 3: SPY - 1000 shares @ $300 cost basis
    INSERT INTO positions (
        owner_id, owned_id, shares, cost_basis, market_value,
        average_cost_per_unit, average_market_price,
        incepting_date, as_of_date, status, tenant_id, created_by
    ) VALUES (
        portfolio_id, spy_id, 1000, 300000, 330000,
        300, 330, '2023-01-10'::date, CURRENT_DATE, 'ACTIVE', org_id, user_id
    ) RETURNING id INTO pos3;
    
    -- Position 4: AGG (Bonds) - 2000 shares @ $75 cost basis
    INSERT INTO positions (
        owner_id, owned_id, shares, cost_basis, market_value,
        average_cost_per_unit, average_market_price,
        incepting_date, as_of_date, status, tenant_id, created_by
    ) VALUES (
        portfolio_id, bond_id, 2000, 150000, 147000,
        75, 73.5, '2023-06-15'::date, CURRENT_DATE, 'ACTIVE', org_id, user_id
    ) RETURNING id INTO pos4;
    
    -- Position 5: Cash - $29K
    INSERT INTO positions (
        owner_id, owned_id, units, market_value, cost_basis,
        as_of_date, status, tenant_id, created_by
    ) VALUES (
        portfolio_id, cash_id, 1, 29000, 29000,
        CURRENT_DATE, 'ACTIVE', org_id, user_id
    ) RETURNING id INTO pos5;
    
    -- Step 3: Update market data for realistic pricing
    
    INSERT INTO entity_market_data (
        entity_id, current_price, day_change, day_change_pct,
        day_low, day_high, bid_price, ask_price,
        volume, pe_ratio, dividend_yield, as_of_date, as_of_time, source
    ) VALUES
        (apple_id, 204, 2.5, 1.24, 201.5, 205.2, 204.10, 204.25, 45000000, 28.5, 0.004, CURRENT_DATE, CURRENT_TIMESTAMP, 'bloomberg'),
        (msft_id, 340, 1.5, 0.44, 338, 341.5, 339.95, 340.15, 22000000, 32.0, 0.007, CURRENT_DATE, CURRENT_TIMESTAMP, 'bloomberg'),
        (spy_id, 330, 3.2, 0.97, 327, 331, 329.95, 330.05, 80000000, NULL, 0.015, CURRENT_DATE, CURRENT_TIMESTAMP, 'bloomberg'),
        (bond_id, 73.5, 0.1, 0.14, 73.3, 73.8, 73.48, 73.52, 15000000, NULL, 0.042, CURRENT_DATE, CURRENT_TIMESTAMP, 'bloomberg'),
        (cash_id, 1.0, 0, 0, 1.0, 1.0, 1.0, 1.0, NULL, NULL, 0.0525, CURRENT_DATE, CURRENT_TIMESTAMP, 'internal');
    
    -- Step 4: Record sample transactions
    
    -- Transaction 1: Buy 500 AAPL
    INSERT INTO position_transactions (
        position_id, entity_id, transaction_type,
        trade_date, units, price, amount, fees, net_amount,
        tax_lot_id, tenant_id, created_by
    ) VALUES (
        pos1, apple_id, 'BUY',
        '2024-01-15'::date, 500, 180, 90000, 50, 89950,
        NULL, org_id, user_id
    );
    
    -- Transaction 2: Dividend on AAPL
    INSERT INTO position_transactions (
        position_id, entity_id, transaction_type,
        trade_date, units, price, amount, fees, net_amount,
        tenant_id, created_by
    ) VALUES (
        pos1, apple_id, 'DIVIDEND',
        CURRENT_DATE, 500, 0.36, 180, 0, 180,
        org_id, user_id
    );
    
    -- Transaction 3: Buy 300 MSFT
    INSERT INTO position_transactions (
        position_id, entity_id, transaction_type,
        trade_date, units, price, amount, fees, net_amount,
        tenant_id, created_by
    ) VALUES (
        pos2, msft_id, 'BUY',
        '2024-02-20'::date, 300, 240, 72000, 50, 71950,
        NULL, org_id, user_id
    );
    
    RAISE NOTICE '✅ Sample portfolio created successfully!';
    RAISE NOTICE 'Portfolio ID: %', portfolio_id;
    RAISE NOTICE 'Client ID: %', client_id;
    RAISE NOTICE 'Total Assets: $780,000 (500K stocks + 147K bonds + 29K cash)';
    
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE '❌ Error creating sample data: %', SQLERRM;
END $$;

-- ============================================================================
-- VERIFY SAMPLE DATA
-- ============================================================================

-- Check portfolio exists
SELECT e.id, e.display_name, e.model_type, COUNT(p.id) as num_positions,
       SUM(COALESCE(p.market_value, p.shares * emd.current_price)) as total_value
FROM entities e
LEFT JOIN positions p ON e.id = p.owner_id AND p.is_active = TRUE
LEFT JOIN entity_market_data emd ON p.owned_id = emd.entity_id AND emd.as_of_date = CURRENT_DATE
WHERE e.model_type = 'PORTFOLIO'
GROUP BY e.id, e.display_name, e.model_type;

-- Check holdings
SELECT h.* FROM v_entity_holdings h LIMIT 10;

-- Check summary
SELECT s.* FROM v_entity_portfolio_summary s;

-- Check transactions
SELECT pt.id, pt.transaction_type, pt.trade_date, pt.units, pt.price, pt.amount
FROM position_transactions pt
ORDER BY pt.trade_date DESC;
