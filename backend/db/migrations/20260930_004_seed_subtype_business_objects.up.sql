-- 20260930_004_seed_subtype_business_objects.up.sql
-- Seeds child business_objects rows for every STI subtype in oms.subtype_registry.
-- Each child BO: key = '<parent_key>/<subtype_code>', parent_id = parent BO id.
-- Wipes and re-inserts to avoid partial unique index / ON CONFLICT issues.

BEGIN;

DO $$
DECLARE
    gct UUID;
    gctd UUID;
BEGIN
    SELECT id INTO gct FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF gct IS NULL THEN
        gct := '00000000-0000-0000-0000-000000000001'::UUID;
    END IF;

    SELECT id INTO gctd FROM public.tenant_product_datasource WHERE tenant_id = gct LIMIT 1;

    -- Wipe child BOs for this tenant (they will be re-seeded)
    DELETE FROM bo_fields WHERE business_object_id IN (
        SELECT id FROM business_objects WHERE tenant_id = gct AND parent_id IS NOT NULL
    );
    DELETE FROM business_objects WHERE tenant_id = gct AND parent_id IS NOT NULL;

    -- ========================================================================
    -- Seed all subtypes from oms.subtype_registry
    -- ========================================================================

    INSERT INTO business_objects
        (id, tenant_id, key, name, display_name, technical_name,
         driver_table_id, driver_table_name,
         parent_id,
         tenant_datasource_id,
         is_core, datasource_id,
         created_at, last_modified_at)
    WITH parent_bo AS (
        SELECT id, key, driver_table_id, driver_table_name
        FROM business_objects
        WHERE tenant_id = gct AND parent_id IS NULL
    )
    SELECT
        gen_random_uuid(),
        gct,
        CASE
            WHEN sr.root_object IN ('account','position','security','trade_order')
                THEN 'oms.' || sr.root_object || '/' || sr.subtype_code
            WHEN sr.root_object = 'alternative_investment'
                THEN 'altinv.alternative_investment/' || sr.subtype_code
            WHEN sr.root_object = 'settlement'
                THEN 'cash_flow.settlement/' || sr.subtype_code
            WHEN sr.root_object IN ('customer','vendor','personnel','sales_ledger')
                THEN 'master.' || sr.root_object || '/' || sr.subtype_code
            ELSE sr.root_object || '/' || sr.subtype_code
        END AS key,

        sr.subtype_code AS name,
        sr.display_name,
        CASE
            WHEN sr.root_object IN ('account','position','security','trade_order')
                THEN 'oms.' || sr.root_object
            WHEN sr.root_object = 'alternative_investment'
                THEN 'altinv.alternative_investment'
            WHEN sr.root_object = 'settlement'
                THEN 'cash_flow.settlement'
            WHEN sr.root_object IN ('customer','vendor','personnel','sales_ledger')
                THEN 'master.' || sr.root_object
            ELSE sr.root_object
        END AS technical_name,

        pbo.driver_table_id,
        pbo.driver_table_name,
        pbo.id AS parent_id,

        gctd AS tenant_datasource_id,
        true,
        NULL::uuid AS datasource_id,
        NOW(),
        NOW()
    FROM oms.subtype_registry sr
    JOIN parent_bo pbo
      ON pbo.key = CASE
            WHEN sr.root_object IN ('account','position','security','trade_order')
                THEN 'oms.' || sr.root_object
            WHEN sr.root_object = 'alternative_investment'
                THEN 'altinv.alternative_investment'
            WHEN sr.root_object = 'settlement'
                THEN 'cash_flow.settlement'
            WHEN sr.root_object IN ('customer','vendor','personnel','sales_ledger')
                THEN 'master.' || sr.root_object
            ELSE sr.root_object
        END
    WHERE sr.tenant_id = gct
      AND sr.is_active = true;

END $$;

COMMIT;
