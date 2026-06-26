-- Restore Full Product Suite and Connections

-- 1. Restore Additional Products (with ON CONFLICT to avoid duplicates)
INSERT INTO alpha_product (product_name, product_code, description, is_active)
VALUES 
    ('Business Intelligence', 'business_intelligence', 'Business Intelligence & Analytics', true),
    ('Performance', 'performance', 'Performance Management', true),
    ('Risk', 'risk', 'Risk Management', true),
    ('Compliance', 'compliance', 'Regulatory Compliance', true)
ON CONFLICT (product_code) DO NOTHING;

-- Map Products to Uisce Tenant
WITH uisce AS (SELECT id FROM tenants WHERE name = 'uisce'),
     products AS (SELECT id FROM alpha_product WHERE product_code IN ('business_intelligence', 'performance', 'risk', 'compliance')),
     uisce_inst AS (SELECT id FROM tenant_instance WHERE instance_name = 'uisce_primary')
INSERT INTO tenant_product (tenant_id, tenant_instance_id, alpha_product_id, version, is_active)
SELECT 
    u.id, 
    ui.id, 
    p.id, 
    1.0, 
    true
FROM uisce, products p, uisce_inst ui
ON CONFLICT DO NOTHING; -- Avoid duplicates if re-run

-- Map Products to Titan Tenant (Standard Suite)
WITH titan AS (SELECT id FROM tenants WHERE name = 'titan'),
     products AS (SELECT id FROM alpha_product WHERE product_code IN ('risk', 'compliance')),
     titan_inst AS (SELECT id FROM tenant_instance WHERE instance_name = 'titan_primary')
INSERT INTO tenant_product (tenant_id, tenant_instance_id, alpha_product_id, version, is_active)
SELECT 
    t.id, 
    ti.id, 
    p.id, 
    1.0, 
    true
FROM titan t, products p, titan_inst ti
ON CONFLICT DO NOTHING;

-- 2. Restore Default Local Connections (if missing)
-- Assuming a local postgres connection for 'Business Intelligence' or similar

WITH uisce_inst AS (SELECT id, tenant_id FROM tenant_instance WHERE instance_name = 'uisce_primary'),
     pg_ds AS (SELECT id FROM alpha_datasource WHERE datasource_code = 'postgres' LIMIT 1)
INSERT INTO connections (
    tenant_id, 
    tenant_instance_id, 
    name, 
    type, 
    host, 
    port, 
    database, 
    username, 
    password, 
    is_active,
    metadata
)
SELECT 
    ui.tenant_id,
    ui.id,
    'Local Postgres',
    'postgres',
    'postgres',
    5432,
    'alpha', -- Defaulting to alpha db
    'postgres',
    'postgres',
    true,
    '{"auth_type": "basic"}'::jsonb
FROM uisce_inst ui
WHERE NOT EXISTS (SELECT 1 FROM connections WHERE name = 'Local Postgres' AND tenant_id = ui.tenant_id);

-- Link Connection to a Product (e.g., Business Intelligence)
WITH uisce_tp AS (
        SELECT tp.id, tp.tenant_instance_id 
        FROM tenant_product tp 
        JOIN alpha_product ap ON tp.alpha_product_id = ap.id 
        WHERE ap.product_code = 'business_intelligence'
     ),
     conn AS (SELECT id FROM connections WHERE name = 'Local Postgres' LIMIT 1),
     pg_ds AS (SELECT id FROM alpha_datasource WHERE datasource_code = 'postgres' LIMIT 1)
INSERT INTO tenant_product_datasource (
    tenant_product_id,
    tenant_instance_id,
    alpha_tenant_instance_id, -- alpha_datasource_id
    alpha_datasource_id,
    connection_id,
    source_name,
    is_active,
    config
)
SELECT
    tp.id,
    tp.tenant_instance_id,
    ds.id,
    ds.id,
    c.id,
    'Primary Data Warehouse',
    true,
    '{}'::jsonb
FROM uisce_tp tp, conn c, pg_ds ds
WHERE NOT EXISTS (SELECT 1 FROM tenant_product_datasource WHERE connection_id = c.id);
