-- Restore Relationships and Connections V2

-- 1. Map Products to Uisce Tenant provided aliases are correct
-- Uisce gets Business Intelligence, Performance, Risk, Compliance
INSERT INTO tenant_product (tenant_id, tenant_instance_id, alpha_product_id, version, is_active)
SELECT 
    t.id, 
    ti.id, 
    p.id, 
    1.0, 
    true
FROM tenants t
JOIN tenant_instance ti ON t.id = ti.tenant_id
JOIN alpha_product p ON p.product_code IN ('business_intelligence', 'performance', 'risk', 'compliance')
WHERE t.name = 'uisce' AND ti.instance_name = 'uisce_primary'
ON CONFLICT DO NOTHING;

-- 2. Map Products to Titan Tenant
-- Titan gets Risk, Compliance
INSERT INTO tenant_product (tenant_id, tenant_instance_id, alpha_product_id, version, is_active)
SELECT 
    t.id, 
    ti.id, 
    p.id, 
    1.0, 
    true
FROM tenants t
JOIN tenant_instance ti ON t.id = ti.tenant_id
JOIN alpha_product p ON p.product_code IN ('risk', 'compliance')
WHERE t.name = 'titan' AND ti.instance_name = 'titan_primary'
ON CONFLICT DO NOTHING;

-- 3. Restore Default Local Connections (Removed tenant_instance_id)
INSERT INTO connections (
    tenant_id, 
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
    t.id,
    'Local Postgres',
    'postgres',
    'postgres',
    5432,
    'alpha',
    'postgres',
    'postgres',
    true,
    '{"auth_type": "basic"}'::jsonb
FROM tenants t
WHERE t.name = 'uisce'
AND NOT EXISTS (SELECT 1 FROM connections WHERE name = 'Local Postgres' AND tenant_id = t.id);

-- 4. Link Connection to a Product (Business Intelligence) via tenant_product_datasource
-- We need to link the NEW connection to the NEW tenant_product entry for BI
INSERT INTO tenant_product_datasource (
    tenant_product_id,
    tenant_instance_id,
    alpha_tenant_instance_id, -- REFERENCES alpha_datasource(id)
    alpha_datasource_id,
    connection_id,
    source_name,
    is_active,
    config
)
SELECT
    tp.id,
    tp.tenant_instance_id,
    ds.id, -- alpha_datasource_id
    ds.id, -- alpha_datasource_id
    c.id, -- connection_id
    'Primary Data Warehouse',
    true,
    '{}'::jsonb
FROM 
    tenant_product tp
    JOIN alpha_product ap ON tp.alpha_product_id = ap.id
    JOIN tenants t ON tp.tenant_id = t.id
    JOIN connections c ON c.tenant_id = t.id
    CROSS JOIN alpha_datasource ds
WHERE 
    t.name = 'uisce' 
    AND ap.product_code = 'business_intelligence'
    AND c.name = 'Local Postgres'
    AND ds.datasource_code = 'postgres'
    AND NOT EXISTS (
        SELECT 1 FROM tenant_product_datasource 
        WHERE tenant_product_id = tp.id AND connection_id = c.id
    );
