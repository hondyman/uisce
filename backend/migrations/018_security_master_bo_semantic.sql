-- backend/migrations/018_security_master_bo_semantic.sql
-- Phase 4 of Security Master: BO Registration + Semantic Graph Integration
-- Revised for Remote Schema (100.84.126.19)

-- ============================================================================
-- PART A: BUSINESS OBJECTS
-- ============================================================================

DO $$
DECLARE
    v_tenant_id  UUID;
    v_security_bo_id   UUID := '11110001-0000-4000-a000-000000000001';
    v_issuer_bo_id     UUID := '11110001-0000-4000-a000-000000000002';
    v_fi_bo_id         UUID := '11110001-0000-4000-a000-000000000010';
    v_eq_bo_id         UUID := '11110001-0000-4000-a000-000000000011';
    v_fund_bo_id       UUID := '11110001-0000-4000-a000-000000000012';
    v_deriv_bo_id      UUID := '11110001-0000-4000-a000-000000000013';
BEGIN
    -- Resolve default tenant (core or first available)
    SELECT id INTO v_tenant_id FROM public.tenants WHERE id = '00000000-0000-0000-0000-000000000001' LIMIT 1;
    IF v_tenant_id IS NULL THEN
        SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;
    END IF;

    -- ── Root: Security ────────────────────────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_security_bo_id, v_tenant_id,
        'security', 'Security', 'Security', 'security',
        'Master record for all financial instruments — equities, bonds, funds, and derivatives',
        'show_chart', true, NULL, 'INVESTMENT',
        '{"driver_table": "edm.security_master", "history_mode": "FULL", "bi_temporal": true}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET display_name = EXCLUDED.display_name,
            description  = EXCLUDED.description,
            config       = EXCLUDED.config;

    -- ── Root: Issuer ──────────────────────────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_issuer_bo_id, v_tenant_id,
        'issuer', 'Issuer', 'Issuer', 'issuer',
        'Legal entity that issues financial instruments (corporation, government, etc.)',
        'account_balance', true, NULL, 'INVESTMENT',
        '{"driver_table": "edm.issuer_master", "history_mode": "FULL"}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET display_name = EXCLUDED.display_name,
            description  = EXCLUDED.description;

    -- ── Subtype: Fixed Income Security ────────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_fi_bo_id, v_tenant_id,
        'fixed_income_security', 'FixedIncomeSecurity', 'Fixed Income Security', 'fixed_income_security',
        'Bond, note, or other debt instrument with fixed or floating coupon',
        'trending_flat', true, v_security_bo_id, 'INVESTMENT',
        '{"driver_table": "edm.fixed_income_attributes", "parent_key": "security"}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET parent_id    = EXCLUDED.parent_id,
            display_name = EXCLUDED.display_name,
            config       = EXCLUDED.config;

    -- ── Subtype: Equity Security ───────────────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_eq_bo_id, v_tenant_id,
        'equity_security', 'EquitySecurity', 'Equity Security', 'equity_security',
        'Common or preferred stock representing ownership in a company',
        'show_chart', true, v_security_bo_id, 'INVESTMENT',
        '{"driver_table": "edm.equity_attributes", "parent_key": "security"}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET parent_id    = EXCLUDED.parent_id,
            display_name = EXCLUDED.display_name,
            config       = EXCLUDED.config;

    RAISE NOTICE '✓ Business Objects registered: security, issuer, + subtypes';
END$$;

-- ============================================================================
-- PART B: BO FIELDS
-- ============================================================================

DO $$
DECLARE
    v_tenant_id      UUID;
    v_security_bo_id UUID := '11110001-0000-4000-a000-000000000001';
    v_fi_bo_id       UUID := '11110001-0000-4000-a000-000000000010';
    v_eq_bo_id       UUID := '11110001-0000-4000-a000-000000000011';
BEGIN
    SELECT id INTO v_tenant_id FROM public.tenants WHERE id = '00000000-0000-0000-0000-000000000001' LIMIT 1;
    IF v_tenant_id IS NULL THEN
        SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;
    END IF;

    -- 1. Root Security Fields
    INSERT INTO public.bo_fields (
        tenant_id, business_object_id, key, field_name, display_label, technical_name,
        field_type, is_core, is_required, display_order, semantic_term_id
    )
    SELECT
        v_tenant_id, v_security_bo_id,
        f.key, f.key, f.label, f.key,
        f.ftype, true, f.req, f.seq,
        (SELECT id FROM edm.semantic_terms WHERE name = f.st_name LIMIT 1)
    FROM (VALUES
        ('security_id',      'Security ID',           'text',    true,  1,  'SecurityID'),
        ('isin',             'ISIN',                  'text',    false, 2,  'ISIN'),
        ('cusip',            'CUSIP',                 'text',    false, 3,  'CUSIP'),
        ('figi',             'FIGI',                  'text',    false, 4,  'FIGI'),
        ('ticker',           'Ticker',                'text',    false, 5,  'Ticker'),
        ('security_name',    'Security Name',         'text',    true,  6,  'SecurityName'),
        ('asset_class',      'Asset Class',           'text',    true,  8,  'AssetClass'),
        ('currency',         'Currency',              'text',    true,  11, 'CurrencyCode'),
        ('confidence_score', 'Confidence Score',      'number',  false, 22, 'GoldCopyConfidenceScore')
    ) AS f(key, label, ftype, req, seq, st_name)
    ON CONFLICT (business_object_id, field_name) DO NOTHING;

    RAISE NOTICE '✓ bo_fields seeded';
END$$;

-- ============================================================================
-- PART C: SEMANTIC GRAPH
-- ============================================================================

DO $$
DECLARE
    v_tenant_id      UUID;
    v_st_type_id     UUID;
    v_bo_type_id     UUID;
    v_phys_type_id   UUID;
    v_datasource_id  UUID;
BEGIN
    SELECT id INTO v_tenant_id FROM public.tenants WHERE id = '00000000-0000-0000-0000-000000000001' LIMIT 1;
    IF v_tenant_id IS NULL THEN SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1; END IF;

    -- Resolve IDs
    SELECT id INTO v_st_type_id   FROM public.catalog_node_type WHERE catalog_type_name = 'semantic_term'   LIMIT 1;
    SELECT id INTO v_bo_type_id   FROM public.catalog_node_type WHERE catalog_type_name = 'business_object' LIMIT 1;
    SELECT id INTO v_phys_type_id FROM public.catalog_node_type WHERE catalog_type_name = 'physical_table'  LIMIT 1;

    -- Safe lookup for datasource
    SELECT id INTO v_datasource_id FROM public.tenant_product_datasource 
    WHERE tenant_instance_id IN (SELECT id FROM public.tenant_instance WHERE tenant_id = v_tenant_id)
    LIMIT 1;

    -- Register BO catalog node
    IF v_bo_type_id IS NOT NULL THEN
        INSERT INTO public.catalog_node (
            node_type_id, node_name, properties, tenant_id, tenant_datasource_id, qualified_path
        ) VALUES (
            v_bo_type_id,
            'security.Security',
            jsonb_build_object('display_name','Security','driver_table','edm.security_master'),
            v_tenant_id,
            v_datasource_id,
            'business_object/Security'
        ) ON CONFLICT DO NOTHING;
    END IF;

    RAISE NOTICE '✓ catalog_node: Security BO registered';
END$$;
