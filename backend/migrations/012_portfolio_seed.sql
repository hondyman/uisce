-- backend/migrations/012_portfolio_seed.sql
-- Portfolio Master Gold Copy — Seed Data
-- Gold Copy Tenant: 99e99e99-99e9-49e9-89e9-99e99e99e999

-- ============================================================
-- 1. SOURCE REGISTRY — External vendor catalog (core / gold copy)
-- ============================================================
DO $$
DECLARE
    v_gold_tenant UUID  := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    v_system_user UUID  := '99e99e99-99e9-49e9-89e9-99e99e99e999';
BEGIN

    -- Bloomberg
    INSERT INTO edm.source_registry (
        source_name, source_code, source_type, endpoint_url, is_active,
        priority_score, confidence_base,
        account_types, asset_classes, regions,
        tenant_id, created_by
    ) VALUES (
        'Bloomberg', 'BBG', 'API', 'https://api.bloomberg.com/', true,
        5, 95,
        ARRAY['retail','institutional','private_wealth','private_markets'],
        ARRAY['EQUITY','FIXED_INCOME','ALTERNATIVES'],
        ARRAY['GLOBAL'],
        v_gold_tenant, v_system_user
    ) ON CONFLICT (source_name, tenant_id) DO NOTHING;

    -- Refinitiv (LSEG)
    INSERT INTO edm.source_registry (
        source_name, source_code, source_type, endpoint_url, is_active,
        priority_score, confidence_base,
        account_types, asset_classes, regions,
        tenant_id, created_by
    ) VALUES (
        'Refinitiv', 'RFN', 'API', 'https://api.refinitiv.com/', true,
        4, 92,
        ARRAY['retail','institutional','private_wealth'],
        ARRAY['EQUITY','FIXED_INCOME'],
        ARRAY['GLOBAL','EMEA','APAC'],
        v_gold_tenant, v_system_user
    ) ON CONFLICT (source_name, tenant_id) DO NOTHING;

    -- S&P Global
    INSERT INTO edm.source_registry (
        source_name, source_code, source_type, endpoint_url, is_active,
        priority_score, confidence_base,
        account_types, asset_classes, regions,
        tenant_id, created_by
    ) VALUES (
        'S&P', 'SP', 'API', 'https://api.spglobal.com/', true,
        3, 88,
        ARRAY['institutional','private_wealth'],
        ARRAY['EQUITY'],
        ARRAY['GLOBAL','NAM'],
        v_gold_tenant, v_system_user
    ) ON CONFLICT (source_name, tenant_id) DO NOTHING;

    -- FactSet
    INSERT INTO edm.source_registry (
        source_name, source_code, source_type, endpoint_url, is_active,
        priority_score, confidence_base,
        account_types, asset_classes, regions,
        tenant_id, created_by
    ) VALUES (
        'FactSet', 'FS', 'API', 'https://api.factset.com/', true,
        3, 85,
        ARRAY['institutional'],
        ARRAY['EQUITY','FIXED_INCOME','ALTERNATIVES'],
        ARRAY['GLOBAL'],
        v_gold_tenant, v_system_user
    ) ON CONFLICT (source_name, tenant_id) DO NOTHING;

END $$;

-- ============================================================
-- 2. CORE SOURCE PREFERENCES — Portfolio / by account_type
--    These are the gold copy defaults (core_id IS NULL).
--    Tenants may override by inserting rows with non-null core_id.
-- ============================================================
DO $$
DECLARE
    v_gold_tenant UUID  := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    v_system_user UUID  := '99e99e99-99e9-49e9-89e9-99e99e99e999';

    -- Helper: insert a preference if none exists for the combination
    PROCEDURE upsert_pref(
        p_account_type  TEXT,
        p_semantic_term TEXT,
        p_priority      INT,
        p_source_system TEXT,
        p_confidence    INT
    ) AS $$
    BEGIN
        INSERT INTO edm.source_preferences (
            tenant_id, business_object, semantic_term,
            region, account_type,
            priority, source_system, confidence,
            status, version, valid_from,
            created_by, updated_by
        ) VALUES (
            v_gold_tenant, 'Portfolio', p_semantic_term,
            'GLOBAL', p_account_type,
            p_priority, p_source_system, p_confidence,
            'production', 1, NOW(),
            v_system_user, v_system_user
        ) ON CONFLICT DO NOTHING;
    END;
BEGIN

    -- ---- RETAIL ----
    -- Price
    CALL upsert_pref('retail', 'Price', 1, 'Bloomberg', 95);
    CALL upsert_pref('retail', 'Price', 2, 'Refinitiv',  92);
    -- Quantity
    CALL upsert_pref('retail', 'Quantity', 1, 'Bloomberg', 95);
    CALL upsert_pref('retail', 'Quantity', 2, 'Refinitiv',  92);

    -- ---- INSTITUTIONAL ----
    -- Price
    CALL upsert_pref('institutional', 'Price', 1, 'Bloomberg', 95);
    CALL upsert_pref('institutional', 'Price', 2, 'FactSet',   85);
    CALL upsert_pref('institutional', 'Price', 3, 'S&P',       88);
    -- Quantity
    CALL upsert_pref('institutional', 'Quantity', 1, 'FactSet',   85);
    CALL upsert_pref('institutional', 'Quantity', 2, 'Bloomberg', 95);

    -- ---- PRIVATE WEALTH ----
    -- Price
    CALL upsert_pref('private_wealth', 'Price', 1, 'Bloomberg', 95);
    CALL upsert_pref('private_wealth', 'Price', 2, 'Refinitiv',  92);
    CALL upsert_pref('private_wealth', 'Price', 3, 'S&P',        88);
    -- Quantity
    CALL upsert_pref('private_wealth', 'Quantity', 1, 'Bloomberg', 95);
    CALL upsert_pref('private_wealth', 'Quantity', 2, 'Refinitiv',  92);

    -- ---- PRIVATE MARKETS ----
    -- Price (only Bloomberg covers private markets broadly)
    CALL upsert_pref('private_markets', 'Price',    1, 'Bloomberg', 95);
    CALL upsert_pref('private_markets', 'Quantity', 1, 'Bloomberg', 95);

END $$;
