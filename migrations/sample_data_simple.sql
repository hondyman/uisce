-- ============================================================================
-- QUICK START: Load Sample Data for Addepar Platform (Simplified)
-- Database: wealth_app
-- Description: Create sample portfolio with holdings for testing
-- ============================================================================

DO $$
DECLARE
    org_id UUID;
    user_id UUID;
    
    portfolio_id UUID;
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
    -- Get IDs from existing data
    SELECT id INTO org_id FROM organizations LIMIT 1;
    SELECT id INTO user_id FROM users LIMIT 1;
    
    IF org_id IS NULL THEN
        RAISE NOTICE '❌ No organizations found. Please create one first.';
        RETURN;
    END IF;
    
    IF user_id IS NULL THEN
        RAISE NOTICE '❌ No users found. Please create one first.';
        RETURN;
    END IF;
    
    -- ========================================================================
    -- Create Securities as Entities
    -- ========================================================================
    
    -- 1. Apple Stock
    INSERT INTO entities (
        model_type, tenant_id, original_name, display_name,
        ticker, status, ownership_type, source_system
    ) VALUES (
        'STOCK', org_id, 'Apple Inc', 'Apple Inc',
        'AAPL', 'ACTIVE'::entity_status, 'SHARE_BASED'::ownership_type, 'internal'
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
        'MSFT', 'ACTIVE'::entity_status, 'SHARE_BASED'::ownership_type, 'internal'
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
        'SPY', 'ACTIVE'::entity_status, 'SHARE_BASED'::ownership_type, 'internal'
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
    
    -- 4. Bond ETF (AGG)
    INSERT INTO entities (
        model_type, tenant_id, original_name, display_name,
        ticker, status, ownership_type, source_system
    ) VALUES (
        'ETF', org_id, 'iShares Core US Aggregate Bond', 'AGG ETF',
        'AGG', 'ACTIVE'::entity_status, 'SHARE_BASED'::ownership_type, 'internal'
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
        'ACTIVE'::entity_status, 'VALUE_BASED'::ownership_type, 'internal'
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
    
    -- ========================================================================
    -- Create Portfolio Entity
    -- ========================================================================
    
    INSERT INTO entities (
        model_type, tenant_id, original_name, display_name,
        ownership_type, status, source_system, created_by
    ) VALUES (
        'PORTFOLIO', org_id, 'Growth Portfolio 2025', 'Growth Portfolio 2025',
        'VALUE_BASED'::ownership_type, 'ACTIVE'::entity_status, 'internal', user_id
    ) RETURNING id INTO portfolio_id;
    
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
    
    -- ========================================================================
    -- Create Positions (Ownership Relationships)
    -- ========================================================================
    
    -- Position 1: AAPL - 500 shares
    INSERT INTO positions (
        owner_id, owned_id, shares, cost_basis, market_value,
        average_cost_per_unit, average_market_price,
        incepting_date, as_of_date, status, tenant_id, created_by
    ) VALUES (
        portfolio_id, apple_id, 500, 90000, 102000,
        180, 204, '2024-01-15'::date, CURRENT_DATE, 'ACTIVE'::position_status, org_id, user_id
    ) RETURNING id INTO pos1;
    
    -- Position 2: MSFT - 300 shares
    INSERT INTO positions (
        owner_id, owned_id, shares, cost_basis, market_value,
        average_cost_per_unit, average_market_price,
        incepting_date, as_of_date, status, tenant_id, created_by
    ) VALUES (
        portfolio_id, msft_id, 300, 72000, 102000,
        240, 340, '2024-02-20'::date, CURRENT_DATE, 'ACTIVE'::position_status, org_id, user_id
    ) RETURNING id INTO pos2;
    
    -- Position 3: SPY - 1000 shares
    INSERT INTO positions (
        owner_id, owned_id, shares, cost_basis, market_value,
        average_cost_per_unit, average_market_price,
        incepting_date, as_of_date, status, tenant_id, created_by
    ) VALUES (
        portfolio_id, spy_id, 1000, 300000, 330000,
        300, 330, '2023-01-10'::date, CURRENT_DATE, 'ACTIVE'::position_status, org_id, user_id
    ) RETURNING id INTO pos3;
    
    -- Position 4: AGG (Bonds) - 2000 shares
    INSERT INTO positions (
        owner_id, owned_id, shares, cost_basis, market_value,
        average_cost_per_unit, average_market_price,
        incepting_date, as_of_date, status, tenant_id, created_by
    ) VALUES (
        portfolio_id, bond_id, 2000, 150000, 147000,
        75, 73.5, '2023-06-15'::date, CURRENT_DATE, 'ACTIVE'::position_status, org_id, user_id
    ) RETURNING id INTO pos4;
    
    -- Position 5: Cash - $29K
    INSERT INTO positions (
        owner_id, owned_id, units, market_value, cost_basis,
        as_of_date, status, tenant_id, created_by
    ) VALUES (
        portfolio_id, cash_id, 1, 29000, 29000,
        CURRENT_DATE, 'ACTIVE'::position_status, org_id, user_id
    ) RETURNING id INTO pos5;
    
    -- ========================================================================
    -- Update Market Data for Realistic Pricing
    -- ========================================================================
    
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
    
    -- ========================================================================
    -- Record Sample Transactions
    -- ========================================================================
    
    -- Transaction 1: Buy 500 AAPL
    INSERT INTO position_transactions (
        position_id, entity_id, transaction_type,
        trade_date, units, price, amount, fees, net_amount,
        tenant_id, created_by
    ) VALUES (
        pos1, apple_id, 'BUY'::transaction_type,
        '2024-01-15'::date, 500, 180, 90000, 50, 89950,
        org_id, user_id
    );
    
    -- Transaction 2: Dividend on AAPL
    INSERT INTO position_transactions (
        position_id, entity_id, transaction_type,
        trade_date, units, price, amount, fees, net_amount,
        tenant_id, created_by
    ) VALUES (
        pos1, apple_id, 'DIVIDEND'::transaction_type,
        CURRENT_DATE, 500, 0.36, 180, 0, 180,
        org_id, user_id
    );
    
    -- Transaction 3: Buy 300 MSFT
    INSERT INTO position_transactions (
        position_id, entity_id, transaction_type,
        trade_date, units, price, amount, fees, net_amount,
        tenant_id, created_by
    ) VALUES (
        pos2, msft_id, 'BUY'::transaction_type,
        '2024-02-20'::date, 300, 240, 72000, 50, 71950,
        org_id, user_id
    );
    
    RAISE NOTICE '✅ Sample portfolio created successfully!';
    RAISE NOTICE '📊 Portfolio ID: %', portfolio_id;
    RAISE NOTICE '💰 Total Assets: $780,000 (stocks + bonds + cash)';
    RAISE NOTICE '';
    RAISE NOTICE 'Holdings:';
    RAISE NOTICE '  • AAPL: 500 shares @ $204 = $102,000';
    RAISE NOTICE '  • MSFT: 300 shares @ $340 = $102,000';
    RAISE NOTICE '  • SPY: 1,000 shares @ $330 = $330,000';
    RAISE NOTICE '  • AGG: 2,000 shares @ $73.50 = $147,000';
    RAISE NOTICE '  • CASH: $29,000';
    
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE '❌ Error: %', SQLERRM;
    RAISE NOTICE 'DETAIL: %', PG_EXCEPTION_DETAIL;
END $$;

-- ============================================================================
-- VERIFY SAMPLE DATA
-- ============================================================================

-- Show portfolio summary
SELECT 'Portfolio Summary' as section;
SELECT e.display_name, COUNT(p.id) as positions, 
       SUM(COALESCE(p.market_value, p.shares * COALESCE(emd.current_price, p.average_market_price))) as total_value
FROM entities e
LEFT JOIN positions p ON e.id = p.owner_id AND p.is_active = TRUE
LEFT JOIN entity_market_data emd ON p.owned_id = emd.entity_id AND emd.as_of_date = CURRENT_DATE
WHERE e.model_type = 'PORTFOLIO'
GROUP BY e.id, e.display_name;

-- Show holdings from view
SELECT 'Holdings from View' as section;
SELECT holding_name, ticker, shares, current_price, 
       COALESCE(current_market_value, 0) as market_value,
       COALESCE(unrealized_gain_loss, 0) as gain_loss,
       COALESCE(return_pct, 0) as return_pct
FROM v_entity_holdings 
LIMIT 10;

-- Show portfolio performance from function
SELECT 'Portfolio Performance' as section;
SELECT * FROM calculate_portfolio_performance(
    (SELECT id FROM entities WHERE model_type = 'PORTFOLIO' LIMIT 1),
    CURRENT_DATE
);
