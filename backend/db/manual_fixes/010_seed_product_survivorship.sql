-- 010_seed_product_survivorship.sql
-- Seed survivorship rules + source priority for Product.
-- Run against crims AFTER 009_seed_product_sources.sql.
--
-- Schema notes (verified against live crims 2026-09-24):
--   * mdm.survivorship_rule: UNIQUE (tenant_id, entity_type, attribute_name);
--     has staleness_max_age_sec (integer), custom_attributes DEFAULT '{}'.
--   * mdm.product_source_priority FK source_system_id → mdm.source_systems(id).
--   * priority_vendors jsonb values use source_systems.code
--     (BLOOMBERG / FACTSET / REFINITIV) so the engine can join them.
--   * Tenant 99e99e99-... matches the seeded source_systems rows (see 009).

\set ON_ERROR_STOP on
BEGIN;
SET LOCAL app.current_tenant = '99e99e99-99e9-49e9-89e9-99e99e99e999';

-- ═══════════════════════════════════════════════════════════════════════
-- 1. survivorship_rule — per field strategy
-- ═══════════════════════════════════════════════════════════════════════
INSERT INTO mdm.survivorship_rule
    (entity_type, attribute_name, strategy, priority_vendors,
     anomaly_tolerance_pct, staleness_max_age_sec, is_active, tenant_id)
VALUES
    ('PRODUCT', 'name',                'SOURCE_PRIORITY',
        '["REFINITIV","BLOOMBERG","FACTSET"]'::jsonb, 10.00, 86400, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999'),
    ('PRODUCT', 'legal_name',          'SOURCE_PRIORITY',
        '["BLOOMBERG","REFINITIV","FACTSET"]'::jsonb, 10.00, 86400, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999'),
    ('PRODUCT', 'short_name',          'SOURCE_PRIORITY',
        '["REFINITIV","BLOOMBERG","FACTSET"]'::jsonb, 10.00, 86400, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999'),
    ('PRODUCT', 'inception_date',      'PROVIDER_AUTHORITATIVE',
        '["BLOOMBERG"]'::jsonb, 0.00, 86400, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999'),
    ('PRODUCT', 'base_currency',       'SOURCE_PRIORITY',
        '["BLOOMBERG","REFINITIV","FACTSET"]'::jsonb, 0.00, 86400, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999'),
    ('PRODUCT', 'domicile',            'SOURCE_PRIORITY',
        '["BLOOMBERG","REFINITIV","FACTSET"]'::jsonb, 0.00, 86400, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999'),
    ('PRODUCT', 'manager_name',        'MOST_RECENT',
        '[]'::jsonb, 0.00, 604800, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999'),
    ('PRODUCT', 'product_type_cd',     'SOURCE_PRIORITY',
        '["BLOOMBERG","REFINITIV","FACTSET"]'::jsonb, 0.00, 86400, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999'),
    ('PRODUCT', 'product_category_cd', 'SOURCE_PRIORITY',
        '["REFINITIV","BLOOMBERG","FACTSET"]'::jsonb, 0.00, 86400, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999'),
    ('PRODUCT', 'benchmark_name',      'SOURCE_PRIORITY',
        '["FACTSET","BLOOMBERG","REFINITIV"]'::jsonb, 0.00, 86400, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999'),
    ('PRODUCT', 'aum',                 'MOST_RECENT',
        '[]'::jsonb, 25.00, 172800, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999'),
    ('PRODUCT', 'is_active',           'SOURCE_PRIORITY',
        '["BLOOMBERG","REFINITIV","FACTSET"]'::jsonb, 0.00, 86400, true,
        '99e99e99-99e9-49e9-89e9-99e99e99e999')
ON CONFLICT (tenant_id, entity_type, attribute_name) DO NOTHING;

-- ═══════════════════════════════════════════════════════════════════════
-- 2. source_priority — canonical source per field group
-- ═══════════════════════════════════════════════════════════════════════
-- Table: mdm.product_source_priority (FK → mdm.source_systems.id)
INSERT INTO mdm.product_source_priority
    (product_type_cd, asset_class_cd, field_group,
     source_system_id, priority, is_active, tenant_id)
SELECT
    NULL, NULL, v.field_group,
    (SELECT id FROM mdm.source_systems
     WHERE code=v.source_cd
       AND tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999'),
    v.priority, true,
    '99e99e99-99e9-49e9-89e9-99e99e99e999'
FROM (VALUES
    -- Identity fields
    ('IDENTITY', 'REFINITIV', 10),
    ('IDENTITY', 'BLOOMBERG', 20),
    ('IDENTITY', 'FACTSET',   30),

    -- Naming
    ('NAME',     'REFINITIV', 10),
    ('NAME',     'BLOOMBERG', 20),
    ('NAME',     'FACTSET',   30),

    -- Classification
    ('CLASSIFICATION', 'BLOOMBERG', 10),
    ('CLASSIFICATION', 'REFINITIV', 20),
    ('CLASSIFICATION', 'FACTSET',   30),

    -- Dates
    ('DATES',    'BLOOMBERG', 10),
    ('DATES',    'REFINITIV', 20),
    ('DATES',    'FACTSET',   30),

    -- Fees / economics
    ('FEES',     'FACTSET',   10),
    ('FEES',     'REFINITIV', 20),
    ('FEES',     'BLOOMBERG', 30),

    -- AUM / size
    ('AUM',      'BLOOMBERG', 10),
    ('AUM',      'REFINITIV', 20),
    ('AUM',      'FACTSET',   30),

    -- Benchmark
    ('BENCHMARK', 'FACTSET',   10),
    ('BENCHMARK', 'BLOOMBERG', 20),
    ('BENCHMARK', 'REFINITIV', 30)
) AS v(field_group, source_cd, priority)
ON CONFLICT DO NOTHING;

-- ═══════════════════════════════════════════════════════════════════════
-- Verification (strict — handoff target: survivorship rules 12; priorities 21)
-- ═══════════════════════════════════════════════════════════════════════
DO $verify$
DECLARE
    surv_count int;
    prio_count int;
    null_src int;
BEGIN
    SELECT count(*) INTO surv_count
    FROM mdm.survivorship_rule
    WHERE entity_type = 'PRODUCT'
      AND tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999';

    SELECT count(*) INTO prio_count
    FROM mdm.product_source_priority
    WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999';

    SELECT count(*) INTO null_src
    FROM mdm.product_source_priority
    WHERE source_system_id IS NULL;

    RAISE NOTICE '010_seed_product_survivorship: rules=% priorities=%',
        surv_count, prio_count;

    IF surv_count <> 12 THEN
        RAISE EXCEPTION 'expected 12 survivorship rules, got %', surv_count;
    END IF;
    IF prio_count <> 21 THEN
        RAISE EXCEPTION 'expected 21 product_source_priority rows, got %', prio_count;
    END IF;
    IF null_src > 0 THEN
        RAISE EXCEPTION '% product_source_priority rows have NULL source_system_id (lookup failed)', null_src;
    END IF;
END
$verify$;

COMMIT;
