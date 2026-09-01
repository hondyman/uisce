-- 20260930_003_seed_parent_business_objects.up.sql
-- Seeds representative parent business_objects rows for each STI root.
-- Wipes and re-inserts to avoid ON CONFLICT issues with partial unique indexes.
-- driver_table_name = physical schema-qualified table (e.g., 'oms.account').

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

    SELECT id INTO gctd FROM public.tenant_product_datasource
    WHERE tenant_id = gct LIMIT 1;

    -- Wipe all existing BOs for this tenant (fresh seed)
    DELETE FROM bo_fields WHERE business_object_id IN (
        SELECT id FROM business_objects WHERE tenant_id = gct
    );
    DELETE FROM business_objects WHERE tenant_id = gct;

    -- ========================================================================
    -- oms.* parent BOs
    -- ========================================================================

    INSERT INTO business_objects
        (id, tenant_id, key, name, display_name, technical_name,
         tenant_datasource_id, driver_table_name,
         is_core, category,
         created_at, last_modified_at)
    VALUES
        (gen_random_uuid(), gct, 'oms.account', 'account', 'Account', 'oms.account',
         gctd, 'oms.account',
         true, 'Investment & Trading',
         NOW(), NOW()),
        (gen_random_uuid(), gct, 'oms.position', 'position', 'Position', 'oms.position',
         gctd, 'oms.position',
         true, 'Investment & Trading',
         NOW(), NOW()),
        (gen_random_uuid(), gct, 'oms.security', 'security', 'Security', 'oms.security',
         gctd, 'oms.security',
         true, 'Investment & Trading',
         NOW(), NOW()),
        (gen_random_uuid(), gct, 'oms.trade_order', 'trade_order', 'Trade Order', 'oms.trade_order',
         gctd, 'oms.trade_order',
         true, 'Investment & Trading',
         NOW(), NOW());

    -- ========================================================================
    -- altinv.* parent BOs
    -- ========================================================================

    INSERT INTO business_objects
        (id, tenant_id, key, name, display_name, technical_name,
         tenant_datasource_id, driver_table_name,
         is_core, category,
         created_at, last_modified_at)
    VALUES
        (gen_random_uuid(), gct, 'altinv.alternative_investment', 'alternative_investment',
         'Alternative Investment', 'altinv.alternative_investment',
         gctd, 'altinv.alternative_investment',
         true, 'Alternative Investments',
         NOW(), NOW());

    -- ========================================================================
    -- cash_flow.* parent BOs
    -- ========================================================================

    INSERT INTO business_objects
        (id, tenant_id, key, name, display_name, technical_name,
         tenant_datasource_id, driver_table_name,
         is_core, category,
         created_at, last_modified_at)
    VALUES
        (gen_random_uuid(), gct, 'cash_flow.settlement', 'settlement',
         'Settlement', 'cash_flow.settlement',
         gctd, 'cash_flow.settlement',
         true, 'Cash Flow',
         NOW(), NOW());

    -- ========================================================================
    -- master.* parent BOs
    -- ========================================================================

    INSERT INTO business_objects
        (id, tenant_id, key, name, display_name, technical_name,
         tenant_datasource_id, driver_table_name,
         is_core, category,
         created_at, last_modified_at)
    VALUES
        (gen_random_uuid(), gct, 'master.customer', 'customer',
         'Customer', 'master.customer',
         gctd, 'master.customer',
         true, 'Master Data',
         NOW(), NOW()),
        (gen_random_uuid(), gct, 'master.vendor', 'vendor',
         'Vendor', 'master.vendor',
         gctd, 'master.vendor',
         true, 'Master Data',
         NOW(), NOW()),
        (gen_random_uuid(), gct, 'master.personnel', 'personnel',
         'Personnel', 'master.personnel',
         gctd, 'master.personnel',
         true, 'Master Data',
         NOW(), NOW()),
        (gen_random_uuid(), gct, 'master.sales_ledger', 'sales_ledger',
         'Sales Ledger', 'master.sales_ledger',
         gctd, 'master.sales_ledger',
         true, 'Master Data',
         NOW(), NOW());

END $$;

COMMIT;
