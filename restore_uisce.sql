-- Restore Uisce (Gold Copy) Tenant and Standard Products

WITH uisce_tenant AS (
    INSERT INTO tenants (name, display_name, description, gold_copy, is_active)
    VALUES ('uisce', 'Uisce', 'Gold Copy Tenant', true, true)
    RETURNING id
),
titan_tenant AS (
    INSERT INTO tenants (name, display_name, description, gold_copy, is_active)
    VALUES ('titan', 'Titan', 'Standard Tenant', false, true)
    RETURNING id
),
prod_retail AS (
    INSERT INTO alpha_product (product_name, description, is_active)
    VALUES ('Retail Banking', 'Retail Banking Product', true)
    RETURNING id
),
prod_commercial AS (
    INSERT INTO alpha_product (product_name, description, is_active)
    VALUES ('Commercial Banking', 'Commercial Banking Product', true)
    RETURNING id
),
prod_wealth AS (
    INSERT INTO alpha_product (product_name, description, is_active)
    VALUES ('Wealth Management', 'Wealth Management Product', true)
    RETURNING id
),
uisce_instance AS (
    INSERT INTO tenant_instance (tenant_id, instance_name, display_name, is_active)
    SELECT id, 'uisce_primary', 'Uisce Primary', true FROM uisce_tenant
    RETURNING id, tenant_id
),
titan_instance AS (
    INSERT INTO tenant_instance (tenant_id, instance_name, display_name, is_active)
    SELECT id, 'titan_primary', 'Titan Primary', true FROM titan_tenant
    RETURNING id, tenant_id
)
-- Link Uisce to all products
INSERT INTO tenant_product (tenant_id, tenant_instance_id, alpha_product_id, version, is_active)
SELECT 
    ui.tenant_id, 
    ui.id, 
    p.id, 
    1.0, 
    true
FROM uisce_instance ui, prod_retail p
UNION ALL
SELECT 
    ui.tenant_id, 
    ui.id, 
    p.id, 
    1.0, 
    true
FROM uisce_instance ui, prod_commercial p
UNION ALL
SELECT 
    ui.tenant_id, 
    ui.id, 
    p.id, 
    1.0, 
    true
FROM uisce_instance ui, prod_wealth p
UNION ALL
-- Link Titan to Retail Banking only
SELECT 
    ti.tenant_id, 
    ti.id, 
    p.id, 
    1.0, 
    true
FROM titan_instance ti, prod_retail p;
