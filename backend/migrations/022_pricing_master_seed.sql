-- backend/migrations/022_pricing_master_seed.sql
-- Pricing Master Seed Data
-- Seeds DQ rules, survivorship strategies, and demo pricing data

-- ============================================================
-- PART 1: DQ RULES
-- ============================================================
DO $$
BEGIN
    -- Price DQ Rules
    INSERT INTO edm.survivorship_rules (id, tenant_id, business_object_id, rule_name, rule_type, field_name, rule_expression, severity, is_active)
    SELECT
        gen_random_uuid(),
        (SELECT id FROM public.tenants LIMIT 1),
        NULL, name, 'DQ_REQUIRED', field, expr, 'Hard', true
    FROM (VALUES
        ('Price_Required_SecurityID',    'security_id',    'REQUIRE security_id'),
        ('Price_Required_PriceType',     'price_type',     'REQUIRE price_type'),
        ('Price_Required_PriceValue',    'price_value',    'REQUIRE price_value'),
        ('Price_Required_PriceDate',     'price_date',     'REQUIRE price_date'),
        ('Price_Required_PriceCurrency', 'price_currency', 'REQUIRE price_currency'),
        ('Price_Required_PriceSource',   'price_source',   'REQUIRE price_source')
    ) AS r(name, field, expr)
    ON CONFLICT DO NOTHING;

    INSERT INTO edm.survivorship_rules (id, tenant_id, business_object_id, rule_name, rule_type, field_name, rule_expression, severity, is_active)
    VALUES
        (gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1), NULL,
         'Price_ValueValidity', 'DQ_CONSTRAINT', 'price_value',
         'price_value > 0', 'Hard', true),
        (gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1), NULL,
         'Price_CurrencyValidity', 'DQ_CONSTRAINT', 'price_currency',
         'price_currency IN (''USD'',''EUR'',''GBP'',''JPY'',''CHF'',''CAD'',''AUD'',''HKD'',''SGD'',''CNY'')',
         'Soft', true)
    ON CONFLICT DO NOTHING;

    -- FX DQ Rules
    INSERT INTO edm.survivorship_rules (id, tenant_id, business_object_id, rule_name, rule_type, field_name, rule_expression, severity, is_active)
    SELECT
        gen_random_uuid(),
        (SELECT id FROM public.tenants LIMIT 1),
        NULL, name, 'DQ_REQUIRED', field, expr, 'Hard', true
    FROM (VALUES
        ('FX_Required_BaseCurrency',  'base_currency',  'REQUIRE base_currency'),
        ('FX_Required_QuoteCurrency', 'quote_currency', 'REQUIRE quote_currency'),
        ('FX_Required_FXRate',        'fx_rate',        'REQUIRE fx_rate'),
        ('FX_Required_FXRateDate',    'fx_rate_date',   'REQUIRE fx_rate_date'),
        ('FX_Required_FXSource',      'fx_source',      'REQUIRE fx_source')
    ) AS r(name, field, expr)
    ON CONFLICT DO NOTHING;

    INSERT INTO edm.survivorship_rules (id, tenant_id, business_object_id, rule_name, rule_type, field_name, rule_expression, severity, is_active)
    VALUES
        (gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1), NULL,
         'FX_RateValidity', 'DQ_CONSTRAINT', 'fx_rate',
         'fx_rate > 0', 'Hard', true)
    ON CONFLICT DO NOTHING;

    -- Curve DQ Rules
    INSERT INTO edm.survivorship_rules (id, tenant_id, business_object_id, rule_name, rule_type, field_name, rule_expression, severity, is_active)
    SELECT
        gen_random_uuid(),
        (SELECT id FROM public.tenants LIMIT 1),
        NULL, name, 'DQ_REQUIRED', field, expr, 'Hard', true
    FROM (VALUES
        ('Curve_Required_CurveType',         'curve_type',          'REQUIRE curve_type'),
        ('Curve_Required_CurveCurrency',     'curve_currency',      'REQUIRE curve_currency'),
        ('Curve_Required_CurveAsOfDate',     'curve_as_of_date',    'REQUIRE curve_as_of_date'),
        ('Curve_Required_CurveTenorPoints',  'curve_tenor_points',  'REQUIRE curve_tenor_points')
    ) AS r(name, field, expr)
    ON CONFLICT DO NOTHING;

    INSERT INTO edm.survivorship_rules (id, tenant_id, business_object_id, rule_name, rule_type, field_name, rule_expression, severity, is_active)
    VALUES
        (gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1), NULL,
         'Curve_TenorPointsValidity', 'DQ_CONSTRAINT', 'curve_tenor_points',
         'jsonb_array_length(curve_tenor_points) > 0', 'Hard', true)
    ON CONFLICT DO NOTHING;

    -- VolSurface DQ Rules
    INSERT INTO edm.survivorship_rules (id, tenant_id, business_object_id, rule_name, rule_type, field_name, rule_expression, severity, is_active)
    SELECT
        gen_random_uuid(),
        (SELECT id FROM public.tenants LIMIT 1),
        NULL, name, 'DQ_REQUIRED', field, expr, 'Hard', true
    FROM (VALUES
        ('Vol_Required_UnderlierSecurityID', 'underlier_security_id', 'REQUIRE underlier_security_id'),
        ('Vol_Required_VolSurfaceType',      'vol_surface_type',      'REQUIRE vol_surface_type'),
        ('Vol_Required_VolAsOfDate',         'vol_as_of_date',        'REQUIRE vol_as_of_date'),
        ('Vol_Required_VolGrid',             'vol_grid',              'REQUIRE vol_grid')
    ) AS r(name, field, expr)
    ON CONFLICT DO NOTHING;

    INSERT INTO edm.survivorship_rules (id, tenant_id, business_object_id, rule_name, rule_type, field_name, rule_expression, severity, is_active)
    VALUES
        (gen_random_uuid(), (SELECT id FROM public.tenants LIMIT 1), NULL,
         'Vol_GridValidity', 'DQ_CONSTRAINT', 'vol_grid',
         'vol_grid ? ''strikes'' AND vol_grid ? ''tenors'' AND vol_grid ? ''vols''',
         'Hard', true)
    ON CONFLICT DO NOTHING;

    RAISE NOTICE '✓ Pricing Master DQ rules seeded';
END$$;


-- ============================================================
-- PART 2: DEMO PRICES
-- We use a well-known security_id if available, otherwise skip
-- ============================================================
DO $$
DECLARE
    v_tenant_id  UUID;
    v_apple_id   UUID;
    v_gs_id      UUID;
BEGIN
    SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;

    -- Try to find Apple and Goldman Sachs from security_master
    SELECT id INTO v_apple_id  FROM edm.security_master WHERE ticker = 'AAPL' AND tenant_id = v_tenant_id LIMIT 1;
    SELECT id INTO v_gs_id     FROM edm.security_master WHERE ticker = 'GS'   AND tenant_id = v_tenant_id LIMIT 1;

    -- Demo price for Apple (Bloomberg)
    IF v_apple_id IS NOT NULL THEN
        INSERT INTO edm.price_master (
            tenant_id, security_id, price_type, price_date, price_value,
            price_currency, price_source, price_confidence, source_systems
        ) VALUES (
            v_tenant_id, v_apple_id, 'Close', CURRENT_DATE - INTERVAL '1 day',
            182.45, 'USD', 'Bloomberg', 98,
            '{"Bloomberg": {"value": 182.45, "confidence": 98}}'::jsonb
        ) ON CONFLICT (security_id, price_type, price_date, price_source, tenant_id) DO NOTHING;

        INSERT INTO edm.price_master (
            tenant_id, security_id, price_type, price_date, price_value,
            price_currency, price_source, price_confidence, source_systems
        ) VALUES (
            v_tenant_id, v_apple_id, 'Close', CURRENT_DATE - INTERVAL '1 day',
            182.38, 'USD', 'Refinitiv', 92,
            '{"Refinitiv": {"value": 182.38, "confidence": 92}}'::jsonb
        ) ON CONFLICT (security_id, price_type, price_date, price_source, tenant_id) DO NOTHING;
    END IF;

    -- Demo price for Goldman Sachs
    IF v_gs_id IS NOT NULL THEN
        INSERT INTO edm.price_master (
            tenant_id, security_id, price_type, price_date, price_value,
            price_currency, price_source, price_confidence, source_systems
        ) VALUES (
            v_tenant_id, v_gs_id, 'Close', CURRENT_DATE - INTERVAL '1 day',
            439.12, 'USD', 'Bloomberg', 98,
            '{"Bloomberg": {"value": 439.12, "confidence": 98}}'::jsonb
        ) ON CONFLICT (security_id, price_type, price_date, price_source, tenant_id) DO NOTHING;
    END IF;

    RAISE NOTICE '✓ Demo prices seeded';
END$$;


-- ============================================================
-- PART 3: DEMO FX RATES
-- ============================================================
DO $$
DECLARE
    v_tenant_id UUID;
BEGIN
    SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;

    INSERT INTO edm.fx_rate_master (
        tenant_id, base_currency, quote_currency, fx_rate_date, fx_tenor,
        fx_rate, fx_source, fx_confidence, source_systems
    ) VALUES
        (v_tenant_id, 'EUR', 'USD', CURRENT_DATE, 'Spot', 1.08450, 'Bloomberg', 98,
         '{"Bloomberg": {"value": 1.08450, "confidence": 98}}'::jsonb),
        (v_tenant_id, 'EUR', 'USD', CURRENT_DATE, 'Spot', 1.08440, 'Refinitiv', 94,
         '{"Refinitiv": {"value": 1.08440, "confidence": 94}}'::jsonb),
        (v_tenant_id, 'GBP', 'USD', CURRENT_DATE, 'Spot', 1.26830, 'Bloomberg', 98,
         '{"Bloomberg": {"value": 1.26830, "confidence": 98}}'::jsonb),
        (v_tenant_id, 'USD', 'JPY', CURRENT_DATE, 'Spot', 149.920, 'Bloomberg', 98,
         '{"Bloomberg": {"value": 149.920, "confidence": 98}}'::jsonb),
        (v_tenant_id, 'EUR', 'USD', CURRENT_DATE, '3M',   1.08310, 'Bloomberg', 95,
         '{"Bloomberg": {"value": 1.08310, "confidence": 95}}'::jsonb)
    ON CONFLICT (base_currency, quote_currency, fx_rate_date, fx_tenor, fx_source, tenant_id) DO NOTHING;

    RAISE NOTICE '✓ Demo FX rates seeded';
END$$;


-- ============================================================
-- PART 4: DEMO CURVES
-- ============================================================
DO $$
DECLARE
    v_tenant_id UUID;
BEGIN
    SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;

    INSERT INTO edm.curve_master (
        tenant_id, curve_type, curve_currency, curve_as_of_date, curve_source,
        curve_tenor_points, curve_interpolation, curve_confidence, source_systems
    ) VALUES (
        v_tenant_id, 'SOFR', 'USD', CURRENT_DATE, 'Bloomberg',
        '[
            {"tenor": "ON",  "rate": 0.0530, "discount_factor": 0.9999},
            {"tenor": "1W",  "rate": 0.0531, "discount_factor": 0.9990},
            {"tenor": "1M",  "rate": 0.0533, "discount_factor": 0.9956},
            {"tenor": "3M",  "rate": 0.0540, "discount_factor": 0.9868},
            {"tenor": "6M",  "rate": 0.0545, "discount_factor": 0.9737},
            {"tenor": "12M", "rate": 0.0535, "discount_factor": 0.9493},
            {"tenor": "2Y",  "rate": 0.0498, "discount_factor": 0.9068},
            {"tenor": "5Y",  "rate": 0.0455, "discount_factor": 0.7985},
            {"tenor": "10Y", "rate": 0.0440, "discount_factor": 0.6455}
        ]'::jsonb,
        'CubicSpline', 97,
        '{"Bloomberg": {"confidence": 97}}'::jsonb
    ) ON CONFLICT (curve_type, curve_currency, curve_as_of_date, curve_source, tenant_id) DO NOTHING;

    INSERT INTO edm.curve_master (
        tenant_id, curve_type, curve_currency, curve_as_of_date, curve_source,
        curve_tenor_points, curve_interpolation, curve_confidence, source_systems
    ) VALUES (
        v_tenant_id, 'Treasury', 'USD', CURRENT_DATE, 'Bloomberg',
        '[
            {"tenor": "1M",  "rate": 0.0540},
            {"tenor": "3M",  "rate": 0.0545},
            {"tenor": "6M",  "rate": 0.0544},
            {"tenor": "12M", "rate": 0.0520},
            {"tenor": "2Y",  "rate": 0.0487},
            {"tenor": "5Y",  "rate": 0.0453},
            {"tenor": "10Y", "rate": 0.0438},
            {"tenor": "30Y", "rate": 0.0458}
        ]'::jsonb,
        'Linear', 97,
        '{"Bloomberg": {"confidence": 97}}'::jsonb
    ) ON CONFLICT (curve_type, curve_currency, curve_as_of_date, curve_source, tenant_id) DO NOTHING;

    RAISE NOTICE '✓ Demo curves seeded';
END$$;
