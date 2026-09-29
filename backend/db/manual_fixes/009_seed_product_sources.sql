-- 009_seed_product_sources.sql
-- Seed source_system rows + initial product field/type mappings.
-- Run against crims AFTER 008_staging_product.sql.
--
-- This is the MINIMUM V1 mapping. It will be extended as real vendor
-- files reveal actual columns and taxonomies.
--
-- Schema notes (verified against live crims 2026-09-24):
--   * Mapping FKs reference mdm.source_systems (PLURAL), not mdm.source_system.
--   * Plural codes: BLOOMBERG, FACTSET, REFINITIV (already seeded, tenant 99e9...).
--   * dev tenant 99e99e99-... is where the 3 sources live; seeds use that tenant
--     so lookups + RLS (FORCED) resolve the same rows.
--   * custom_attributes / valid_from have DEFAULTs — omitted here.

\set ON_ERROR_STOP on

BEGIN;

-- Set tenant context for RLS (FORCED on all target tables)
SET LOCAL app.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';

-- ═══════════════════════════════════════════════════════════════════════
-- 1. source_systems rows (idempotent — already present in this env)
-- ═══════════════════════════════════════════════════════════════════════
DO $src$
DECLARE
    c record;
BEGIN
    FOR c IN
        SELECT * FROM (VALUES
            ('BLOOMBERG', 'Bloomberg'),
            ('FACTSET',    'FactSet'),
            ('REFINITIV',  'Refinitiv')
        ) v(code, display_name)
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM mdm.source_systems
            WHERE code = c.code
              AND tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'
        ) THEN
            INSERT INTO mdm.source_systems (tenant_id, code, display_name)
            VALUES ('99e99e99-99e9-49e9-89e9-99e99e99e999', c.code, c.display_name);
        END IF;
    END LOOP;
END
$src$;

-- Abort early if any source is missing (mapping subqueries would yield NULL)
DO $chk$
DECLARE
    missing text;
BEGIN
    SELECT string_agg(v.code, ', ') INTO missing
    FROM (VALUES ('BLOOMBERG'),('FACTSET'),('REFINITIV')) v(code)
    WHERE NOT EXISTS (
        SELECT 1 FROM mdm.source_systems s
        WHERE s.code = v.code
          AND s.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'
    );
    IF missing IS NOT NULL THEN
        RAISE EXCEPTION 'missing source_systems codes: %', missing;
    END IF;
END
$chk$;

-- ═══════════════════════════════════════════════════════════════════════
-- 2. product_type_mapping — vendor taxonomy → internal type_cd
-- ═══════════════════════════════════════════════════════════════════════
-- FactSet fund_type values
INSERT INTO mdm.product_type_mapping
    (mdm_source_system_id, vendor_type_cd, vendor_sub_type_cd,
     internal_type_cd, confidence, is_active, tenant_id)
SELECT
    (SELECT id FROM mdm.source_systems
     WHERE code='FACTSET' AND tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999'),
    v.vendor_type, NULL, v.internal_type, 100.00, true,
    '99e99e99-99e9-49e9-89e9-99e99e99e999'
FROM (VALUES
    ('Mutual Fund',             'MUTUAL_FUND'),
    ('Exchange Traded Fund',    'ETF'),
    ('Closed-End Fund',         'CEF'),
    ('Unit Investment Trust',   'UNIT_TRUST'),
    ('Collective Investment Trust', 'CIT'),
    ('Separately Managed Account',  'SMA'),
    ('Hedge Fund',              'HEDGE_FUND'),
    ('Private Equity Fund',     'PE_FUND')
) AS v(vendor_type, internal_type)
ON CONFLICT (tenant_id, mdm_source_system_id, vendor_type_cd,
             COALESCE(vendor_sub_type_cd, ''::varchar)) DO NOTHING;

-- Bloomberg fund_type values
INSERT INTO mdm.product_type_mapping
    (mdm_source_system_id, vendor_type_cd, vendor_sub_type_cd,
     internal_type_cd, confidence, is_active, tenant_id)
SELECT
    (SELECT id FROM mdm.source_systems
     WHERE code='BLOOMBERG' AND tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999'),
    v.vendor_type, NULL, v.internal_type, 100.00, true,
    '99e99e99-99e9-49e9-89e9-99e99e99e999'
FROM (VALUES
    ('Mutual Fund',         'MUTUAL_FUND'),
    ('ETF',                 'ETF'),
    ('Closed End Fund',     'CEF'),
    ('Unit Trust',          'UNIT_TRUST'),
    ('Hedge Fund',          'HEDGE_FUND'),
    ('Variable Annuity',    'VARIABLE_ANNUITY')
) AS v(vendor_type, internal_type)
ON CONFLICT (tenant_id, mdm_source_system_id, vendor_type_cd,
             COALESCE(vendor_sub_type_cd, ''::varchar)) DO NOTHING;

-- Refinitiv lipper_global_class values (top-level)
INSERT INTO mdm.product_type_mapping
    (mdm_source_system_id, vendor_type_cd, vendor_sub_type_cd,
     internal_type_cd, confidence, is_active, tenant_id)
SELECT
    (SELECT id FROM mdm.source_systems
     WHERE code='REFINITIV' AND tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999'),
    v.vendor_type, NULL, v.internal_type, 100.00, true,
    '99e99e99-99e9-49e9-89e9-99e99e99e999'
FROM (VALUES
    ('Equity Fund',         'MUTUAL_FUND'),
    ('Bond Fund',           'MUTUAL_FUND'),
    ('Money Market Fund',   'MONEY_MARKET_FUND'),
    ('Mixed Asset Fund',    'MUTUAL_FUND'),
    ('Alternative Fund',    'HEDGE_FUND'),
    ('ETF',                 'ETF')
) AS v(vendor_type, internal_type)
ON CONFLICT (tenant_id, mdm_source_system_id, vendor_type_cd,
             COALESCE(vendor_sub_type_cd, ''::varchar)) DO NOTHING;

-- ═══════════════════════════════════════════════════════════════════════
-- 3. product_field_mapping — vendor column → canonical field
-- ═══════════════════════════════════════════════════════════════════════
-- FactSet field mappings
INSERT INTO mdm.product_field_mapping
    (mdm_source_system_id, product_type_cd, vendor_field,
     internal_table, internal_field, transform_expression,
     is_required, is_active, tenant_id)
SELECT
    (SELECT id FROM mdm.source_systems
     WHERE code='FACTSET' AND tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999'),
    NULL, v.vendor_field, v.internal_table, v.internal_field,
    v.transform, v.is_required, true,
    '99e99e99-99e9-49e9-89e9-99e99e99e999'
FROM (VALUES
    ('fsym_id',         'mdm.product','product_cd',     NULL::text,           false),
    ('isin',            'mdm.product','_payload.isin',  NULL,                 false),
    ('cusip',           'mdm.product','_payload.cusip', NULL,                 false),
    ('sedol',           'mdm.product','_payload.sedol', NULL,                 false),
    ('ticker',          'mdm.product','_payload.ticker',NULL,                 false),
    ('fund_name',       'mdm.product','name',           NULL,                 true),
    ('fund_name_short', 'mdm.product','short_name',     NULL,                 false),
    ('fund_type',       'mdm.product','product_type_cd','__type_map__',       true),
    ('fund_category',   'mdm.product','product_category_cd', NULL,            false),
    ('domicile_country','mdm.product','domicile',       'UPPER(TRIM(%s))',    false),
    ('base_currency',   'mdm.product','base_currency',  'UPPER(TRIM(%s))',    false),
    ('inception_date',  'mdm.product','inception_date', NULL,                 false),
    ('fund_manager',    'mdm.product','manager_name',   NULL,                 false),
    ('benchmark_name',  'mdm.product','_payload.benchmark_name', NULL,        false),
    ('is_active',       'mdm.product','is_active',      NULL,                 false),
    ('aum',             'mdm.product','_payload.aum',   NULL,                 false),
    ('aum_currency',    'mdm.product','aum_currency',   'UPPER(TRIM(%s))',    false)
) AS v(vendor_field, internal_table, internal_field, transform, is_required)
ON CONFLICT DO NOTHING;

-- Bloomberg field mappings
INSERT INTO mdm.product_field_mapping
    (mdm_source_system_id, product_type_cd, vendor_field,
     internal_table, internal_field, transform_expression,
     is_required, is_active, tenant_id)
SELECT
    (SELECT id FROM mdm.source_systems
     WHERE code='BLOOMBERG' AND tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999'),
    NULL, v.vendor_field, v.internal_table, v.internal_field,
    v.transform, v.is_required, true,
    '99e99e99-99e9-49e9-89e9-99e99e99e999'
FROM (VALUES
    ('bbg_id',          'mdm.product','_payload.bbg_id', NULL::text,          false),
    ('bbg_ticker',      'mdm.product','_payload.ticker', '__parse_bbg_ticker__', false),
    ('isin',            'mdm.product','_payload.isin',   NULL,                false),
    ('cusip',           'mdm.product','_payload.cusip',  NULL,                false),
    ('long_name',       'mdm.product','name',            NULL,                true),
    ('short_name',      'mdm.product','short_name',      NULL,                false),
    ('fund_type',       'mdm.product','product_type_cd', '__type_map__',      true),
    ('country_domicile','mdm.product','domicile',        'UPPER(TRIM(%s))',   false),
    ('crncy',           'mdm.product','base_currency',   'UPPER(TRIM(%s))',   false),
    ('inception_dt',    'mdm.product','inception_date',  NULL,                false),
    ('fund_manager',    'mdm.product','manager_name',    NULL,                false),
    ('benchmark',       'mdm.product','_payload.benchmark_name', NULL,        false),
    ('total_assets',    'mdm.product','_payload.aum',    NULL,                false)
) AS v(vendor_field, internal_table, internal_field, transform, is_required)
ON CONFLICT DO NOTHING;

-- Refinitiv field mappings
INSERT INTO mdm.product_field_mapping
    (mdm_source_system_id, product_type_cd, vendor_field,
     internal_table, internal_field, transform_expression,
     is_required, is_active, tenant_id)
SELECT
    (SELECT id FROM mdm.source_systems
     WHERE code='REFINITIV' AND tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999'),
    NULL, v.vendor_field, v.internal_table, v.internal_field,
    v.transform, v.is_required, true,
    '99e99e99-99e9-49e9-89e9-99e99e99e999'
FROM (VALUES
    ('ric',                 'mdm.product','_payload.ric',      NULL::text,    false),
    ('isin',                'mdm.product','_payload.isin',     NULL,          false),
    ('cusip',               'mdm.product','_payload.cusip',    NULL,          false),
    ('sedol',               'mdm.product','_payload.sedol',    NULL,          false),
    ('lipper_id',           'mdm.product','_payload.lipper_id',NULL,          false),
    ('fund_legal_name',     'mdm.product','legal_name',        NULL,          true),
    ('fund_display_name',   'mdm.product','name',              NULL,          true),
    ('lipper_global_class', 'mdm.product','product_type_cd',   '__type_map__',true),
    ('lipper_class',        'mdm.product','product_category_cd',NULL,         false),
    ('domicile',            'mdm.product','domicile',          'UPPER(TRIM(%s))', false),
    ('currency',            'mdm.product','base_currency',     'UPPER(TRIM(%s))', false),
    ('launch_date',         'mdm.product','inception_date',    NULL,          false),
    ('manager_name',        'mdm.product','manager_name',      NULL,          false),
    ('total_net_assets',    'mdm.product','_payload.aum',      NULL,          false)
) AS v(vendor_field, internal_table, internal_field, transform, is_required)
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════════════════════
-- Verification (strict — matches handoff targets: sources 3, types 20, fields 44)
-- ═══════════════════════════════════════════════════════════════════════
DO $verify$
DECLARE
    src_count int;
    type_map_count int;
    field_map_count int;
BEGIN
    SELECT count(*) INTO src_count
    FROM mdm.source_systems
    WHERE code IN ('BLOOMBERG','FACTSET','REFINITIV')
      AND tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999';

    SELECT count(*) INTO type_map_count
    FROM mdm.product_type_mapping;

    SELECT count(*) INTO field_map_count
    FROM mdm.product_field_mapping;

    RAISE NOTICE '009_seed_product_sources: sources=% types=% fields=%',
        src_count, type_map_count, field_map_count;

    IF src_count <> 3 THEN
        RAISE EXCEPTION 'expected 3 source_systems rows, got %', src_count;
    END IF;
    IF type_map_count <> 20 THEN
        RAISE EXCEPTION 'expected 20 product_type_mapping rows, got %', type_map_count;
    END IF;
    IF field_map_count <> 44 THEN
        RAISE EXCEPTION 'expected 44 product_field_mapping rows, got %', field_map_count;
    END IF;
END
$verify$;

COMMIT;
