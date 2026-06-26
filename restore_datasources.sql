-- Restore Standard Alpha Datasources

INSERT INTO alpha_datasource (id, display_name, datasource_code, is_active, config)
VALUES
    ('550e8400-e29b-41d4-a716-446655440001', 'Postgres', 'postgres', true, '{}'),
    ('550e8400-e29b-41d4-a716-446655440002', 'SQL Server', 'sql_server', true, '{}'),
    ('550e8400-e29b-41d4-a716-446655440003', 'Oracle', 'oracle', true, '{}'),
    ('550e8400-e29b-41d4-a716-446655440004', 'Snowflake', 'snowflake', true, '{}'),
    ('550e8400-e29b-41d4-a716-446655440005', 'Iceberg', 'iceberg', true, '{}')
ON CONFLICT (id) DO UPDATE SET 
    display_name = EXCLUDED.display_name,
    datasource_code = EXCLUDED.datasource_code;

-- Retry the Tenant Product Datasource Link
INSERT INTO tenant_product_datasource (
    tenant_product_id,
    tenant_instance_id,
    alpha_tenant_instance_id,
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
