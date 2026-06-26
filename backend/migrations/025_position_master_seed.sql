-- backend/migrations/025_position_master_seed.sql
-- Position Master Seed Data
-- Seeds DQ rules, survivorship strategies, calculation terms, and demo positions

-- ============================================================
-- PART 1: DQ RULES
-- ============================================================
DO $$
BEGIN
    -- Position Required Fields
    INSERT INTO edm.survivorship_rules (id, tenant_id, business_object_id, rule_name, rule_type, field_name, rule_expression, severity, is_active)
    SELECT gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1),
        NULL, name, 'DQ_REQUIRED', field, expr, 'Hard', true
    FROM (VALUES
        ('Position_Required_PortfolioID',   'portfolio_id',       'REQUIRE portfolio_id'),
        ('Position_Required_SecurityID',    'security_id',        'REQUIRE security_id'),
        ('Position_Required_PositionDate',  'position_date',      'REQUIRE position_date'),
        ('Position_Required_Quantity',      'position_quantity',  'REQUIRE position_quantity'),
        ('Position_Required_Currency',      'position_currency',  'REQUIRE position_currency')
    ) AS r(name, field, expr)
    ON CONFLICT DO NOTHING;

    INSERT INTO edm.survivorship_rules (id, tenant_id, business_object_id, rule_name, rule_type, field_name, rule_expression, severity, is_active)
    VALUES
        -- Market value consistency
        (gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1), NULL,
         'Position_MarketValueCalculation', 'DQ_DERIVED', 'market_value_local',
         'market_value_local = position_quantity * price_value WHERE price_date = position_date',
         'Soft', true),
        -- P&L calculation
        (gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1), NULL,
         'Position_PLCalculation', 'DQ_DERIVED', 'unrealized_pl_local',
         'unrealized_pl_local = market_value_local - cost_basis_local',
         'Soft', true),
        -- Reconciliation alert
        (gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1), NULL,
         'Position_ReconciliationAlert', 'DQ_CONSTRAINT', 'reconciliation_diff',
         'ABS(reconciliation_diff) <= 0.01',
         'Soft', true),
        -- Negative cash alert
        (gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1), NULL,
         'Cash_NegativeBalanceAlert', 'DQ_CONSTRAINT', 'balance_amount',
         'balance_amount >= 0 OR currency = ''MARGIN''',
         'Soft', true),
        -- Lot method validity
        (gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1), NULL,
         'Lot_MethodValidity', 'DQ_CONSTRAINT', 'lot_method',
         'lot_method IN (''FIFO'',''LIFO'',''HIFO'',''Specific'',''AverageCost'')',
         'Hard', true),
        -- Lot positive quantity
        (gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1), NULL,
         'Lot_PositiveQuantity', 'DQ_CONSTRAINT', 'lot_quantity',
         'lot_quantity > 0',
         'Hard', true)
    ON CONFLICT DO NOTHING;

    RAISE NOTICE '✓ Position Master DQ rules seeded';
END$$;

-- ============================================================
-- PART 2: CALCULATION TERMS
-- Wire MarketValueLocal, UnrealizedPL, PositionWeight into the semantic graph
-- ============================================================
DO $$
DECLARE
    v_calc_type_id UUID;
    v_st_type_id   UUID;
BEGIN
    SELECT id INTO v_calc_type_id FROM catalog_node_type WHERE catalog_type_name = 'calculation_term' LIMIT 1;
    SELECT id INTO v_st_type_id   FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;

    IF v_calc_type_id IS NULL THEN
        RAISE NOTICE 'calculation_term node type not found — skipping calculation term seeding';
        RETURN;
    END IF;

    -- Calculation Terms
    INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
    VALUES
        ('ct_market_value_local', v_calc_type_id,
         'MarketValueLocal',
         '{"formula":"quantity * price","dependencies":["st_position_quantity","st_price_value"],"data_type":"numeric"}',
         'calc/MarketValueLocal', '00000000-0000-0000-0000-000000000000'),
        ('ct_market_value_base', v_calc_type_id,
         'MarketValueBase',
         '{"formula":"market_value_local * fx_rate","dependencies":["ct_market_value_local","st_fx_rate"],"data_type":"numeric"}',
         'calc/MarketValueBase', '00000000-0000-0000-0000-000000000000'),
        ('ct_unrealized_pl', v_calc_type_id,
         'UnrealizedPL',
         '{"formula":"market_value_local - cost_basis_local","dependencies":["ct_market_value_local","st_cost_basis_local"],"data_type":"numeric"}',
         'calc/UnrealizedPL', '00000000-0000-0000-0000-000000000000'),
        ('ct_position_weight', v_calc_type_id,
         'PositionWeight',
         '{"formula":"market_value_base / portfolio_nav","dependencies":["ct_market_value_base","ct_nav"],"data_type":"numeric"}',
         'calc/PositionWeight', '00000000-0000-0000-0000-000000000000'),
        ('ct_contribution_to_return', v_calc_type_id,
         'ContributionToReturn',
         '{"formula":"position_weight * security_return","dependencies":["ct_position_weight","st_price_value"],"data_type":"numeric"}',
         'calc/ContributionToReturn', '00000000-0000-0000-0000-000000000000')
    ON CONFLICT (id) DO UPDATE
        SET node_name  = EXCLUDED.node_name,
            properties = EXCLUDED.properties;

    -- Wire calculation dependencies
    INSERT INTO catalog_edge (id, edge_type_id, source_node_id, target_node_id, properties)
    SELECT gen_random_uuid(),
        (SELECT id FROM catalog_edge_type WHERE type_name = 'calc_depends_on_term' LIMIT 1),
        source, target, '{}'::jsonb
    FROM (VALUES
        ('ct_market_value_local', 'st_position_quantity'),
        ('ct_market_value_local', 'st_price_value'),
        ('ct_market_value_base',  'st_fx_rate'),
        ('ct_unrealized_pl',      'st_cost_basis_local'),
        ('ct_position_weight',    'ct_nav')
    ) AS e(source, target)
    WHERE EXISTS (SELECT 1 FROM catalog_node WHERE id = e.source)
      AND EXISTS (SELECT 1 FROM catalog_node WHERE id = e.target)
    ON CONFLICT DO NOTHING;

    RAISE NOTICE '✓ Position Master calculation terms seeded';
END$$;

-- ============================================================
-- PART 3: DEMO POSITIONS
-- Link to existing portfolio and security master data
-- ============================================================
DO $$
DECLARE
    v_tenant_id   UUID;
    v_port_id     UUID;
    v_apple_id    UUID;
    v_gs_id       UUID;
BEGIN
    SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;

    -- Find first portfolio
    SELECT id INTO v_port_id FROM edm.portfolio_master WHERE tenant_id = v_tenant_id LIMIT 1;

    -- Find securities
    SELECT id INTO v_apple_id FROM edm.security_master WHERE ticker = 'AAPL' AND tenant_id = v_tenant_id LIMIT 1;
    SELECT id INTO v_gs_id    FROM edm.security_master WHERE ticker = 'GS'   AND tenant_id = v_tenant_id LIMIT 1;

    IF v_port_id IS NULL THEN
        RAISE NOTICE 'No portfolio found — skipping demo positions';
        RETURN;
    END IF;

    IF v_apple_id IS NOT NULL THEN
        -- Apple position (Custodian source)
        INSERT INTO edm.position_master (
            tenant_id, portfolio_id, security_id, position_date,
            position_quantity, position_side, position_currency,
            market_value_local, market_value_base, cost_basis_local,
            unrealized_pl_local, unrealized_pl_pct, position_weight_pct,
            position_source, position_confidence, is_reconciled, source_systems
        ) VALUES (
            v_tenant_id, v_port_id, v_apple_id, CURRENT_DATE,
            1000.0, 'Long', 'USD',
            182450.00, 182450.00, 150000.00,
            32450.00, 0.2163, 0.0182,
            'Custodian', 98, true,
            '{"Custodian": {"quantity": 1000, "confidence": 98}}'::jsonb
        ) ON CONFLICT (portfolio_id, security_id, position_date, position_source, tenant_id) DO NOTHING;

        -- Accounting system source (lower priority)
        INSERT INTO edm.position_master (
            tenant_id, portfolio_id, security_id, position_date,
            position_quantity, position_side, position_currency,
            market_value_local, market_value_base, cost_basis_local,
            unrealized_pl_local, position_source, position_confidence, is_reconciled, source_systems
        ) VALUES (
            v_tenant_id, v_port_id, v_apple_id, CURRENT_DATE,
            1001.0, 'Long', 'USD',
            182632.45, 182632.45, 150150.00,
            32482.45, 'Accounting', 85, false,
            '{"Accounting": {"quantity": 1001, "confidence": 85}}'::jsonb
        ) ON CONFLICT (portfolio_id, security_id, position_date, position_source, tenant_id) DO NOTHING;

        -- Demo tax lots for Apple position
        INSERT INTO edm.position_lot_master (
            tenant_id, position_id, lot_reference, acquisition_date, settlement_date,
            lot_quantity, cost_per_unit, total_cost_basis, lot_method, is_closed
        )
        SELECT
            v_tenant_id, p.id, 'LOT-AAPL-001', '2023-01-15', '2023-01-17',
            500.0, 135.0, 67500.0, 'FIFO', false
        FROM edm.position_master p
        WHERE p.security_id = v_apple_id AND p.position_source = 'Custodian'
          AND p.tenant_id = v_tenant_id AND p.position_date = CURRENT_DATE
        ON CONFLICT DO NOTHING;

        INSERT INTO edm.position_lot_master (
            tenant_id, position_id, lot_reference, acquisition_date, settlement_date,
            lot_quantity, cost_per_unit, total_cost_basis, lot_method, is_closed
        )
        SELECT
            v_tenant_id, p.id, 'LOT-AAPL-002', '2023-08-22', '2023-08-24',
            500.0, 165.0, 82500.0, 'FIFO', false
        FROM edm.position_master p
        WHERE p.security_id = v_apple_id AND p.position_source = 'Custodian'
          AND p.tenant_id = v_tenant_id AND p.position_date = CURRENT_DATE
        ON CONFLICT DO NOTHING;
    END IF;

    IF v_gs_id IS NOT NULL THEN
        INSERT INTO edm.position_master (
            tenant_id, portfolio_id, security_id, position_date,
            position_quantity, position_side, position_currency,
            market_value_local, market_value_base, cost_basis_local,
            unrealized_pl_local, unrealized_pl_pct, position_weight_pct,
            position_source, position_confidence, is_reconciled, source_systems
        ) VALUES (
            v_tenant_id, v_port_id, v_gs_id, CURRENT_DATE,
            250.0, 'Long', 'USD',
            109780.00, 109780.00, 95000.00,
            14780.00, 0.1556, 0.0110,
            'Custodian', 98, true,
            '{"Custodian": {"quantity": 250, "confidence": 98}}'::jsonb
        ) ON CONFLICT (portfolio_id, security_id, position_date, position_source, tenant_id) DO NOTHING;
    END IF;

    -- Demo USD cash balance
    INSERT INTO edm.cash_position_master (
        tenant_id, portfolio_id, cash_currency, value_date,
        balance_amount, available_balance, interest_accrued, cash_source, source_systems
    ) VALUES (
        v_tenant_id, v_port_id, 'USD', CURRENT_DATE,
        2500000.00, 2450000.00, 1250.00, 'Custodian',
        '{"Custodian": {"balance": 2500000, "confidence": 99}}'::jsonb
    ) ON CONFLICT (portfolio_id, cash_currency, value_date, cash_source, tenant_id) DO NOTHING;

    INSERT INTO edm.cash_position_master (
        tenant_id, portfolio_id, cash_currency, value_date,
        balance_amount, available_balance, interest_accrued, cash_source, source_systems
    ) VALUES (
        v_tenant_id, v_port_id, 'EUR', CURRENT_DATE,
        750000.00, 730000.00, 350.00, 'Custodian',
        '{"Custodian": {"balance": 750000, "confidence": 99}}'::jsonb
    ) ON CONFLICT (portfolio_id, cash_currency, value_date, cash_source, tenant_id) DO NOTHING;

    RAISE NOTICE '✓ Demo positions, lots, and cash balances seeded';
END$$;
