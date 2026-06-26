-- backend/migrations/023_pricing_master_bo_semantic.sql
-- Phase: Pricing Master BO Registration + Semantic Graph Integration

-- ============================================================================
-- PART A: BUSINESS OBJECTS
-- ============================================================================
DO $$
DECLARE
    v_tenant_id       UUID;
    v_price_bo_id     UUID := '22220001-0000-4000-a000-000000000001';
    v_fx_bo_id        UUID := '22220001-0000-4000-a000-000000000002';
    v_curve_bo_id     UUID := '22220001-0000-4000-a000-000000000003';
    v_vol_bo_id       UUID := '22220001-0000-4000-a000-000000000004';
BEGIN
    SELECT id INTO v_tenant_id FROM public.tenants WHERE id = '00000000-0000-0000-0000-000000000001' LIMIT 1;
    IF v_tenant_id IS NULL THEN
        SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;
    END IF;

    -- ── Price ────────────────────────────────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_price_bo_id, v_tenant_id,
        'price', 'Price', 'Price', 'price',
        'Point-in-time gold-copy price for a security from a given source.',
        'attach_money', true, NULL, 'PRICING',
        '{"driver_table": "edm.price_master", "history_mode": "NONE", "cluster_key": ["security_id","price_date","price_type"]}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET display_name = EXCLUDED.display_name,
            description  = EXCLUDED.description,
            config       = EXCLUDED.config;

    -- ── FXRate ──────────────────────────────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_fx_bo_id, v_tenant_id,
        'fx_rate', 'FXRate', 'FX Rate', 'fx_rate',
        'FX spot/forward rate between two currencies.',
        'currency_exchange', true, NULL, 'PRICING',
        '{"driver_table": "edm.fx_rate_master", "history_mode": "NONE", "cluster_key": ["base_currency","quote_currency","fx_rate_date","fx_tenor"]}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET display_name = EXCLUDED.display_name,
            description  = EXCLUDED.description,
            config       = EXCLUDED.config;

    -- ── Curve ────────────────────────────────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_curve_bo_id, v_tenant_id,
        'curve', 'Curve', 'Curve', 'curve',
        'Yield/discount/credit curve for pricing and risk.',
        'timeline', true, NULL, 'PRICING',
        '{"driver_table": "edm.curve_master", "history_mode": "NONE", "cluster_key": ["curve_type","curve_currency","curve_as_of_date"]}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET display_name = EXCLUDED.display_name,
            description  = EXCLUDED.description,
            config       = EXCLUDED.config;

    -- ── VolSurface ───────────────────────────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_vol_bo_id, v_tenant_id,
        'vol_surface', 'VolSurface', 'Vol Surface', 'vol_surface',
        'Volatility surface for options and derivatives.',
        'grid_on', true, NULL, 'PRICING',
        '{"driver_table": "edm.vol_surface_master", "history_mode": "NONE", "cluster_key": ["underlier_security_id","vol_surface_type","vol_as_of_date"]}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET display_name = EXCLUDED.display_name,
            description  = EXCLUDED.description,
            config       = EXCLUDED.config;

    RAISE NOTICE '✓ Pricing Master Business Objects registered: price, fx_rate, curve, vol_surface';
END$$;


-- ============================================================================
-- PART B: BO FIELDS
-- ============================================================================
DO $$
DECLARE
    v_tenant_id   UUID;
    v_price_bo_id UUID := '22220001-0000-4000-a000-000000000001';
    v_fx_bo_id    UUID := '22220001-0000-4000-a000-000000000002';
    v_curve_bo_id UUID := '22220001-0000-4000-a000-000000000003';
    v_vol_bo_id   UUID := '22220001-0000-4000-a000-000000000004';
BEGIN
    SELECT id INTO v_tenant_id FROM public.tenants WHERE id = '00000000-0000-0000-0000-000000000001' LIMIT 1;
    IF v_tenant_id IS NULL THEN
        SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;
    END IF;

    -- ── Price fields ──────────────────────────────────────────────────────────
    INSERT INTO public.bo_fields (business_object_id, field_name, field_type, display_label, semantic_term_id, is_required, sort_order)
    VALUES
        (v_price_bo_id, 'price_id',           'UUID',      'Price ID',               'st_price_id',           true,  1),
        (v_price_bo_id, 'security_id',        'UUID',      'Security ID',            'st_security_id',        true,  2),
        (v_price_bo_id, 'price_type',         'TEXT',      'Price Type',             'st_price_type',         true,  3),
        (v_price_bo_id, 'price_value',        'NUMERIC',   'Price',                  'st_price_value',        true,  4),
        (v_price_bo_id, 'price_date',         'DATE',      'Price Date',             'st_price_date',         true,  5),
        (v_price_bo_id, 'price_time',         'TIMESTAMP', 'Price Time',             'st_price_time',         false, 6),
        (v_price_bo_id, 'price_currency',     'TEXT',      'Price Currency',         'st_price_currency',     true,  7),
        (v_price_bo_id, 'fx_rate_to_base',    'NUMERIC',   'FX Rate To Base',        'st_fx_rate_to_base',    false, 8),
        (v_price_bo_id, 'price_source',       'TEXT',      'Price Source',           'st_price_source',       true,  9),
        (v_price_bo_id, 'price_confidence',   'NUMERIC',   'Price Confidence Score', 'st_price_confidence',   false, 10),
        (v_price_bo_id, 'is_composite_price', 'BOOLEAN',   'Is Composite Price',     'st_is_composite_price', false, 11),
        (v_price_bo_id, 'composite_method',   'TEXT',      'Composite Method',       'st_composite_method',   false, 12),
        (v_price_bo_id, 'is_stale_price',     'BOOLEAN',   'Is Stale Price',         'st_is_stale_price',     false, 13),
        (v_price_bo_id, 'stale_reason',       'TEXT',      'Stale Reason',           'st_stale_reason',       false, 14)
    ON CONFLICT (business_object_id, field_name) DO UPDATE
        SET display_label    = EXCLUDED.display_label,
            semantic_term_id = EXCLUDED.semantic_term_id;

    -- ── FXRate fields ─────────────────────────────────────────────────────────
    INSERT INTO public.bo_fields (business_object_id, field_name, field_type, display_label, semantic_term_id, is_required, sort_order)
    VALUES
        (v_fx_bo_id, 'fx_id',           'UUID',    'FX ID',                'st_fx_id',           true,  1),
        (v_fx_bo_id, 'base_currency',   'TEXT',    'Base Currency',        'st_base_currency',   true,  2),
        (v_fx_bo_id, 'quote_currency',  'TEXT',    'Quote Currency',       'st_quote_currency',  true,  3),
        (v_fx_bo_id, 'fx_rate',         'NUMERIC', 'FX Rate',              'st_fx_rate',         true,  4),
        (v_fx_bo_id, 'fx_rate_date',    'DATE',    'FX Rate Date',         'st_fx_rate_date',    true,  5),
        (v_fx_bo_id, 'fx_source',       'TEXT',    'FX Source',            'st_fx_source',       true,  6),
        (v_fx_bo_id, 'fx_forward_points','NUMERIC','FX Forward Points',    'st_fx_forward_points',false,7),
        (v_fx_bo_id, 'fx_tenor',        'TEXT',    'FX Tenor',             'st_fx_tenor',        false, 8),
        (v_fx_bo_id, 'fx_confidence',   'NUMERIC', 'FX Confidence Score',  'st_fx_confidence',   false, 9)
    ON CONFLICT (business_object_id, field_name) DO UPDATE
        SET display_label    = EXCLUDED.display_label,
            semantic_term_id = EXCLUDED.semantic_term_id;

    -- ── Curve fields ──────────────────────────────────────────────────────────
    INSERT INTO public.bo_fields (business_object_id, field_name, field_type, display_label, semantic_term_id, is_required, sort_order)
    VALUES
        (v_curve_bo_id, 'curve_id',            'UUID',    'Curve ID',                       'st_curve_id',            true,  1),
        (v_curve_bo_id, 'curve_type',          'TEXT',    'Curve Type',                     'st_curve_type',          true,  2),
        (v_curve_bo_id, 'curve_currency',      'TEXT',    'Curve Currency',                 'st_curve_currency',      true,  3),
        (v_curve_bo_id, 'curve_as_of_date',    'DATE',    'Curve As Of Date',               'st_curve_as_of_date',    true,  4),
        (v_curve_bo_id, 'curve_tenor_points',  'JSON',    'Curve Tenor Points',             'st_curve_tenor_points',  true,  5),
        (v_curve_bo_id, 'curve_interpolation', 'TEXT',    'Curve Interpolation Method',     'st_curve_interpolation', false, 6),
        (v_curve_bo_id, 'curve_extrapolation', 'TEXT',    'Curve Extrapolation Method',     'st_curve_extrapolation', false, 7),
        (v_curve_bo_id, 'curve_confidence',    'NUMERIC', 'Curve Confidence Score',         'st_curve_confidence',    false, 8)
    ON CONFLICT (business_object_id, field_name) DO UPDATE
        SET display_label    = EXCLUDED.display_label,
            semantic_term_id = EXCLUDED.semantic_term_id;

    -- ── VolSurface fields ─────────────────────────────────────────────────────
    INSERT INTO public.bo_fields (business_object_id, field_name, field_type, display_label, semantic_term_id, is_required, sort_order)
    VALUES
        (v_vol_bo_id, 'vol_surface_id',       'UUID',    'Vol Surface ID',             'st_vol_surface_id',       true,  1),
        (v_vol_bo_id, 'underlier_security_id','UUID',    'Underlier Security ID',      'st_underlier_security_id',true,  2),
        (v_vol_bo_id, 'vol_surface_type',     'TEXT',    'Vol Surface Type',           'st_vol_surface_type',     true,  3),
        (v_vol_bo_id, 'vol_as_of_date',       'DATE',    'Vol As Of Date',             'st_vol_as_of_date',       true,  4),
        (v_vol_bo_id, 'vol_grid',             'JSON',    'Vol Grid',                   'st_vol_grid',             true,  5),
        (v_vol_bo_id, 'vol_interpolation',    'TEXT',    'Vol Interpolation Method',   'st_vol_interpolation',    false, 6),
        (v_vol_bo_id, 'vol_extrapolation',    'TEXT',    'Vol Extrapolation Method',   'st_vol_extrapolation',    false, 7),
        (v_vol_bo_id, 'vol_confidence',       'NUMERIC', 'Vol Confidence Score',       'st_vol_confidence',       false, 8)
    ON CONFLICT (business_object_id, field_name) DO UPDATE
        SET display_label    = EXCLUDED.display_label,
            semantic_term_id = EXCLUDED.semantic_term_id;

    RAISE NOTICE '✓ Pricing Master BO fields registered: % price + % fx + % curve + % vol',
        (SELECT COUNT(*) FROM public.bo_fields WHERE business_object_id = v_price_bo_id),
        (SELECT COUNT(*) FROM public.bo_fields WHERE business_object_id = v_fx_bo_id),
        (SELECT COUNT(*) FROM public.bo_fields WHERE business_object_id = v_curve_bo_id),
        (SELECT COUNT(*) FROM public.bo_fields WHERE business_object_id = v_vol_bo_id);
END$$;


-- ============================================================================
-- PART C: SEMANTIC GRAPH TERMS
-- ============================================================================
DO $$
DECLARE
    v_st_type_id UUID;
BEGIN
    SELECT id INTO v_st_type_id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    IF v_st_type_id IS NULL THEN
        RAISE NOTICE 'semantic_term node type not found — skipping term seeding';
        RETURN;
    END IF;

    INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id) VALUES
        -- Price terms
        ('st_price_id',           v_st_type_id, 'PriceID',              '{"data_type":"uuid"}',      'semantic/PriceID',              '00000000-0000-0000-0000-000000000000'),
        ('st_security_id',        v_st_type_id, 'SecurityID',           '{"data_type":"uuid"}',      'semantic/SecurityID',           '00000000-0000-0000-0000-000000000000'),
        ('st_price_type',         v_st_type_id, 'PriceType',            '{"data_type":"text"}',      'semantic/PriceType',            '00000000-0000-0000-0000-000000000000'),
        ('st_price_value',        v_st_type_id, 'PriceValue',           '{"data_type":"numeric"}',   'semantic/PriceValue',           '00000000-0000-0000-0000-000000000000'),
        ('st_price_date',         v_st_type_id, 'PriceDate',            '{"data_type":"date"}',      'semantic/PriceDate',            '00000000-0000-0000-0000-000000000000'),
        ('st_price_time',         v_st_type_id, 'PriceTime',            '{"data_type":"timestamp"}', 'semantic/PriceTime',            '00000000-0000-0000-0000-000000000000'),
        ('st_price_currency',     v_st_type_id, 'PriceCurrency',        '{"data_type":"text"}',      'semantic/PriceCurrency',        '00000000-0000-0000-0000-000000000000'),
        ('st_fx_rate_to_base',    v_st_type_id, 'FXRateToBase',         '{"data_type":"numeric"}',   'semantic/FXRateToBase',         '00000000-0000-0000-0000-000000000000'),
        ('st_price_source',       v_st_type_id, 'PriceSource',          '{"data_type":"text"}',      'semantic/PriceSource',          '00000000-0000-0000-0000-000000000000'),
        ('st_price_confidence',   v_st_type_id, 'PriceConfidenceScore', '{"data_type":"numeric"}',   'semantic/PriceConfidenceScore', '00000000-0000-0000-0000-000000000000'),
        ('st_is_composite_price', v_st_type_id, 'IsCompositePrice',     '{"data_type":"boolean"}',   'semantic/IsCompositePrice',     '00000000-0000-0000-0000-000000000000'),
        ('st_composite_method',   v_st_type_id, 'CompositeMethod',      '{"data_type":"text"}',      'semantic/CompositeMethod',      '00000000-0000-0000-0000-000000000000'),
        ('st_is_stale_price',     v_st_type_id, 'IsStalePrice',         '{"data_type":"boolean"}',   'semantic/IsStalePrice',         '00000000-0000-0000-0000-000000000000'),
        ('st_stale_reason',       v_st_type_id, 'StaleReason',          '{"data_type":"text"}',      'semantic/StaleReason',          '00000000-0000-0000-0000-000000000000'),
        -- FX terms
        ('st_fx_id',              v_st_type_id, 'FXID',                 '{"data_type":"uuid"}',      'semantic/FXID',                 '00000000-0000-0000-0000-000000000000'),
        ('st_base_currency',      v_st_type_id, 'BaseCurrency',         '{"data_type":"text"}',      'semantic/BaseCurrency',         '00000000-0000-0000-0000-000000000000'),
        ('st_quote_currency',     v_st_type_id, 'QuoteCurrency',        '{"data_type":"text"}',      'semantic/QuoteCurrency',        '00000000-0000-0000-0000-000000000000'),
        ('st_fx_rate',            v_st_type_id, 'FXRate',               '{"data_type":"numeric"}',   'semantic/FXRate',               '00000000-0000-0000-0000-000000000000'),
        ('st_fx_rate_date',       v_st_type_id, 'FXRateDate',           '{"data_type":"date"}',      'semantic/FXRateDate',           '00000000-0000-0000-0000-000000000000'),
        ('st_fx_source',          v_st_type_id, 'FXSource',             '{"data_type":"text"}',      'semantic/FXSource',             '00000000-0000-0000-0000-000000000000'),
        ('st_fx_forward_points',  v_st_type_id, 'FXForwardPoints',      '{"data_type":"numeric"}',   'semantic/FXForwardPoints',      '00000000-0000-0000-0000-000000000000'),
        ('st_fx_tenor',           v_st_type_id, 'FXTenor',              '{"data_type":"text"}',      'semantic/FXTenor',              '00000000-0000-0000-0000-000000000000'),
        ('st_fx_confidence',      v_st_type_id, 'FXConfidenceScore',    '{"data_type":"numeric"}',   'semantic/FXConfidenceScore',    '00000000-0000-0000-0000-000000000000'),
        -- Curve terms
        ('st_curve_id',           v_st_type_id, 'CurveID',              '{"data_type":"uuid"}',      'semantic/CurveID',              '00000000-0000-0000-0000-000000000000'),
        ('st_curve_type',         v_st_type_id, 'CurveType',            '{"data_type":"text"}',      'semantic/CurveType',            '00000000-0000-0000-0000-000000000000'),
        ('st_curve_currency',     v_st_type_id, 'CurveCurrency',        '{"data_type":"text"}',      'semantic/CurveCurrency',        '00000000-0000-0000-0000-000000000000'),
        ('st_curve_as_of_date',   v_st_type_id, 'CurveAsOfDate',        '{"data_type":"date"}',      'semantic/CurveAsOfDate',        '00000000-0000-0000-0000-000000000000'),
        ('st_curve_tenor_points', v_st_type_id, 'CurveTenorPoints',     '{"data_type":"json"}',      'semantic/CurveTenorPoints',     '00000000-0000-0000-0000-000000000000'),
        ('st_curve_interpolation',v_st_type_id, 'CurveInterpolationMethod','{"data_type":"text"}',   'semantic/CurveInterpolationMethod','00000000-0000-0000-0000-000000000000'),
        ('st_curve_extrapolation',v_st_type_id, 'CurveExtrapolationMethod','{"data_type":"text"}',   'semantic/CurveExtrapolationMethod','00000000-0000-0000-0000-000000000000'),
        ('st_curve_confidence',   v_st_type_id, 'CurveConfidenceScore', '{"data_type":"numeric"}',   'semantic/CurveConfidenceScore', '00000000-0000-0000-0000-000000000000'),
        -- VolSurface terms
        ('st_vol_surface_id',         v_st_type_id, 'VolSurfaceID',           '{"data_type":"uuid"}',    'semantic/VolSurfaceID',           '00000000-0000-0000-0000-000000000000'),
        ('st_underlier_security_id',  v_st_type_id, 'UnderlierSecurityID',    '{"data_type":"uuid"}',    'semantic/UnderlierSecurityID',    '00000000-0000-0000-0000-000000000000'),
        ('st_vol_surface_type',       v_st_type_id, 'VolSurfaceType',         '{"data_type":"text"}',    'semantic/VolSurfaceType',         '00000000-0000-0000-0000-000000000000'),
        ('st_vol_as_of_date',         v_st_type_id, 'VolAsOfDate',            '{"data_type":"date"}',    'semantic/VolAsOfDate',            '00000000-0000-0000-0000-000000000000'),
        ('st_vol_grid',               v_st_type_id, 'VolGrid',                '{"data_type":"json"}',    'semantic/VolGrid',                '00000000-0000-0000-0000-000000000000'),
        ('st_vol_interpolation',      v_st_type_id, 'VolInterpolationMethod', '{"data_type":"text"}',    'semantic/VolInterpolationMethod', '00000000-0000-0000-0000-000000000000'),
        ('st_vol_extrapolation',      v_st_type_id, 'VolExtrapolationMethod', '{"data_type":"text"}',    'semantic/VolExtrapolationMethod', '00000000-0000-0000-0000-000000000000'),
        ('st_vol_confidence',         v_st_type_id, 'VolConfidenceScore',     '{"data_type":"numeric"}', 'semantic/VolConfidenceScore',     '00000000-0000-0000-0000-000000000000')
    ON CONFLICT (id) DO UPDATE
        SET node_name  = EXCLUDED.node_name,
            properties = EXCLUDED.properties;

    RAISE NOTICE '✓ 39 Pricing Master semantic terms seeded to catalog_node';
END$$;


-- ============================================================================
-- PART D: SEMANTIC EDGE TYPES (priced_by, uses_curve, uses_vol_surface)
-- ============================================================================
DO $$
BEGIN
    INSERT INTO catalog_edge_type (id, type_name, description, source_type_id, target_type_id)
    SELECT
        gen_random_uuid(), e.type_name, e.description,
        (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'business_object' LIMIT 1),
        (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'business_object' LIMIT 1)
    FROM (VALUES
        ('priced_by',       'Links a Security to the Price gold copy that prices it'),
        ('uses_curve',      'Links a Security or Position to a Curve for risk calculations'),
        ('uses_vol_surface','Links a Derivative Security to its VolSurface for pricing')
    ) AS e(type_name, description)
    ON CONFLICT (type_name) DO NOTHING;

    RAISE NOTICE '✓ Semantic edge types registered: priced_by, uses_curve, uses_vol_surface';
END$$;
