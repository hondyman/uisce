-- Migration: Register Iceberg Cold-Tier Wide Table Target for Denormalized Reporting
-- Date: 2026-07-28
-- Description: Registers iceberg.analytics.sales_ledger_flat target nodes and maps semantic terms to OLAP context.

BEGIN;

PERFORM set_config('app.goldcopy', (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)::text, true);

-- ============================================================================
-- 1. REGISTER THE ICEBERG OLAP PHYSICAL TARGET TABLE
-- ============================================================================
INSERT INTO public.catalog_node (
    id,
    tenant_id,
    node_type_id,
    name,
    description,
    properties,
    created_at,
    updated_at
) VALUES (
    'a1b2c3d4-0001-4000-8000-000000000001',
    current_setting('app.goldcopy', true)::uuid,
    COALESCE(
        (SELECT id FROM public.catalog_node_type WHERE catalog_type_name = 'physical_table' LIMIT 1),
        '820b942a-9c9e-4abc-acdc-84616db33098'::uuid
    ),
    'iceberg.analytics.sales_ledger_flat',
    'Denormalized wide table for Sales Ledger reporting in Apache Iceberg',
    '{"engine": "trino_iceberg", "schema": "analytics", "table": "sales_ledger_flat", "partition_keys": ["tenant_id", "order_year_month"]}'::jsonb,
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW();

-- ============================================================================
-- 2. REGISTER THE ICEBERG FLATTENED COLUMNS
-- ============================================================================
INSERT INTO public.catalog_node (
    id,
    tenant_id,
    node_type_id,
    name,
    description,
    properties,
    created_at,
    updated_at
) VALUES 
(
    'a1b2c3d4-0002-4000-8000-000000000002',
    current_setting('app.goldcopy', true)::uuid,
    COALESCE(
        (SELECT id FROM public.catalog_node_type WHERE catalog_type_name = 'physical_column' LIMIT 1),
        '820b942a-9c9e-4abc-acdc-84616db33098'::uuid
    ),
    'sales_ledger_flat.order_id',
    'Order ID column in flattened Iceberg reporting table',
    '{"column_name": "order_id", "data_type": "UUID"}'::jsonb,
    NOW(),
    NOW()
),
(
    'a1b2c3d4-0003-4000-8000-000000000003',
    current_setting('app.goldcopy', true)::uuid,
    COALESCE(
        (SELECT id FROM public.catalog_node_type WHERE catalog_type_name = 'physical_column' LIMIT 1),
        '820b942a-9c9e-4abc-acdc-84616db33098'::uuid
    ),
    'sales_ledger_flat.customer_company_name',
    'Customer company name in flattened Iceberg reporting table',
    '{"column_name": "customer_company_name", "data_type": "VARCHAR"}'::jsonb,
    NOW(),
    NOW()
),
(
    'a1b2c3d4-0004-4000-8000-000000000004',
    current_setting('app.goldcopy', true)::uuid,
    COALESCE(
        (SELECT id FROM public.catalog_node_type WHERE catalog_type_name = 'physical_column' LIMIT 1),
        '820b942a-9c9e-4abc-acdc-84616db33098'::uuid
    ),
    'sales_ledger_flat.line_total_revenue',
    'Line total revenue in flattened Iceberg reporting table',
    '{"column_name": "line_total_revenue", "data_type": "NUMERIC"}'::jsonb,
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW();

-- ============================================================================
-- 3. MAP SEMANTIC TERMS TO FLATTENED ICEBERG COLD TARGET (OLAP BINDING)
-- ============================================================================
INSERT INTO public.catalog_edge (
    id,
    tenant_id,
    edge_type_id,
    source_node_id,
    target_node_id,
    properties,
    created_at,
    updated_at
) VALUES 
(
    'e1f2g3h4-0001-4000-8000-000000000001',
    current_setting('app.goldcopy', true)::uuid,
    COALESCE(
        (SELECT id FROM public.catalog_edge_type WHERE catalog_edge_type_name = 'MAPS_TO' LIMIT 1),
        '710b942a-9c9e-4abc-acdc-84616db33099'::uuid
    ),
    COALESCE(
        (SELECT id FROM public.catalog_node WHERE name = 'term.order_id' AND tenant_id = current_setting('app.goldcopy', true)::uuid LIMIT 1),
        'a1b2c3d4-0002-4000-8000-000000000002'::uuid
    ),
    'a1b2c3d4-0002-4000-8000-000000000002',
    '{"binding_context": "OLAP", "tier": "COLD_REPORTING"}'::jsonb,
    NOW(),
    NOW()
),
(
    'e1f2g3h4-0002-4000-8000-000000000002',
    current_setting('app.goldcopy', true)::uuid,
    COALESCE(
        (SELECT id FROM public.catalog_edge_type WHERE catalog_edge_type_name = 'MAPS_TO' LIMIT 1),
        '710b942a-9c9e-4abc-acdc-84616db33099'::uuid
    ),
    COALESCE(
        (SELECT id FROM public.catalog_node WHERE name = 'term.customer_company_name' AND tenant_id = current_setting('app.goldcopy', true)::uuid LIMIT 1),
        'a1b2c3d4-0003-4000-8000-000000000003'::uuid
    ),
    'a1b2c3d4-0003-4000-8000-000000000003',
    '{"binding_context": "OLAP", "tier": "COLD_REPORTING"}'::jsonb,
    NOW(),
    NOW()
) ON CONFLICT (id) DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW();

COMMIT;
