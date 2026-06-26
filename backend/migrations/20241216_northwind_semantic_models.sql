-- ============================================================================
-- NORTHWIND BUSINESS OBJECTS & SEMANTIC MODELS
-- Complete setup for Northwind sample database
-- ============================================================================

-- Ensure Northwind tables exist (stubbing for migration success if missing)
CREATE TABLE IF NOT EXISTS categories (
    category_id INT PRIMARY KEY,
    category_name VARCHAR(255),
    description TEXT
);

CREATE TABLE IF NOT EXISTS suppliers (
    supplier_id INT PRIMARY KEY,
    company_name VARCHAR(255),
    contact_name VARCHAR(255),
    contact_title VARCHAR(255),
    address VARCHAR(255),
    city VARCHAR(255),
    region VARCHAR(255),
    postal_code VARCHAR(255),
    country VARCHAR(255),
    phone VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS products (
    product_id INT PRIMARY KEY,
    product_name VARCHAR(255),
    supplier_id INT,
    category_id INT,
    quantity_per_unit VARCHAR(255),
    unit_price DECIMAL,
    units_in_stock INT,
    units_on_order INT,
    reorder_level INT,
    discontinued BOOLEAN
);

CREATE TABLE IF NOT EXISTS customers (
    customer_id VARCHAR(255) PRIMARY KEY,
    company_name VARCHAR(255),
    contact_name VARCHAR(255),
    contact_title VARCHAR(255),
    address VARCHAR(255),
    city VARCHAR(255),
    region VARCHAR(255),
    postal_code VARCHAR(255),
    country VARCHAR(255),
    phone VARCHAR(255),
    fax VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS employees (
    employee_id INT PRIMARY KEY,
    last_name VARCHAR(255),
    first_name VARCHAR(255),
    title VARCHAR(255),
    title_of_courtesy VARCHAR(255),
    birth_date DATE,
    hire_date DATE,
    address VARCHAR(255),
    city VARCHAR(255),
    region VARCHAR(255),
    postal_code VARCHAR(255),
    country VARCHAR(255),
    home_phone VARCHAR(255),
    extension VARCHAR(255),
    notes TEXT,
    reports_to INT
);

CREATE TABLE IF NOT EXISTS orders (
    order_id INT PRIMARY KEY,
    customer_id VARCHAR(255),
    employee_id INT,
    order_date DATE,
    required_date DATE,
    shipped_date DATE,
    ship_via INT,
    freight DECIMAL,
    ship_name VARCHAR(255),
    ship_address VARCHAR(255),
    ship_city VARCHAR(255),
    ship_region VARCHAR(255),
    ship_postal_code VARCHAR(255),
    ship_country VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS order_details (
    order_id INT,
    product_id INT,
    unit_price DECIMAL,
    quantity INT,
    discount REAL,
    PRIMARY KEY (order_id, product_id)
);

-- ============================================================================

-- ============================================================================
-- CORE BUSINESS OBJECTS FOR NORTHWIND
-- ============================================================================

-- Customer Business Object
ALTER TABLE public.business_objects ADD COLUMN IF NOT EXISTS config JSONB DEFAULT '{}';

INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, is_core, config)
VALUES 
  (gen_random_uuid(), (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1), 'northwind_customer', 'northwind_customer', 'Customer', 'northwind_customer', 'Northwind customer data', true, 
   '{"fields": [
     {"key": "customer_id", "name": "Customer ID", "type": "string", "is_core": true, "required": true},
     {"key": "company_name", "name": "Company Name", "type": "string", "is_core": true, "required": true},
     {"key": "contact_name", "name": "Contact Name", "type": "string", "is_core": true},
     {"key": "contact_title", "name": "Contact Title", "type": "string", "is_core": true},
     {"key": "address", "name": "Address", "type": "string", "is_core": true},
     {"key": "city", "name": "City", "type": "string", "is_core": true},
     {"key": "region", "name": "Region", "type": "string", "is_core": true},
     {"key": "postal_code", "name": "Postal Code", "type": "string", "is_core": true},
     {"key": "country", "name": "Country", "type": "string", "is_core": true},
     {"key": "phone", "name": "Phone", "type": "string", "is_core": true},
     {"key": "fax", "name": "Fax", "type": "string", "is_core": true}
   ]}'::jsonb)
ON CONFLICT DO NOTHING;

-- Order Business Object
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, is_core, config)
VALUES 
  (gen_random_uuid(), (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1), 'northwind_order', 'northwind_order', 'Order', 'northwind_order', 'Northwind order data', true,
   '{"fields": [
     {"key": "order_id", "name": "Order ID", "type": "integer", "is_core": true, "required": true},
     {"key": "customer_id", "name": "Customer ID", "type": "string", "is_core": true, "required": true},
     {"key": "employee_id", "name": "Employee ID", "type": "integer", "is_core": true},
     {"key": "order_date", "name": "Order Date", "type": "date", "is_core": true},
     {"key": "required_date", "name": "Required Date", "type": "date", "is_core": true},
     {"key": "shipped_date", "name": "Shipped Date", "type": "date", "is_core": true},
     {"key": "ship_via", "name": "Shipper ID", "type": "integer", "is_core": true},
     {"key": "freight", "name": "Freight", "type": "number", "is_core": true},
     {"key": "ship_name", "name": "Ship Name", "type": "string", "is_core": true},
     {"key": "ship_address", "name": "Ship Address", "type": "string", "is_core": true},
     {"key": "ship_city", "name": "Ship City", "type": "string", "is_core": true},
     {"key": "ship_region", "name": "Ship Region", "type": "string", "is_core": true},
     {"key": "ship_postal_code", "name": "Ship Postal Code", "type": "string", "is_core": true},
     {"key": "ship_country", "name": "Ship Country", "type": "string", "is_core": true}
   ]}'::jsonb)
ON CONFLICT DO NOTHING;

-- Product Business Object
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, is_core, config)
VALUES 
  (gen_random_uuid(), (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1), 'northwind_product', 'northwind_product', 'Product', 'northwind_product', 'Northwind product catalog', true,
   '{"fields": [
     {"key": "product_id", "name": "Product ID", "type": "integer", "is_core": true, "required": true},
     {"key": "product_name", "name": "Product Name", "type": "string", "is_core": true, "required": true},
     {"key": "supplier_id", "name": "Supplier ID", "type": "integer", "is_core": true},
     {"key": "category_id", "name": "Category ID", "type": "integer", "is_core": true},
     {"key": "quantity_per_unit", "name": "Quantity Per Unit", "type": "string", "is_core": true},
     {"key": "unit_price", "name": "Unit Price", "type": "number", "is_core": true},
     {"key": "units_in_stock", "name": "Units In Stock", "type": "integer", "is_core": true},
     {"key": "units_on_order", "name": "Units On Order", "type": "integer", "is_core": true},
     {"key": "reorder_level", "name": "Reorder Level", "type": "integer", "is_core": true},
     {"key": "discontinued", "name": "Discontinued", "type": "boolean", "is_core": true}
   ]}'::jsonb)
ON CONFLICT DO NOTHING;

-- Employee Business Object
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, is_core, config)
VALUES 
  (gen_random_uuid(), (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1), 'northwind_employee', 'northwind_employee', 'Employee', 'northwind_employee', 'Northwind employee data', true,
   '{"fields": [
     {"key": "employee_id", "name": "Employee ID", "type": "integer", "is_core": true, "required": true},
     {"key": "last_name", "name": "Last Name", "type": "string", "is_core": true, "required": true},
     {"key": "first_name", "name": "First Name", "type": "string", "is_core": true, "required": true},
     {"key": "title", "name": "Title", "type": "string", "is_core": true},
     {"key": "title_of_courtesy", "name": "Title of Courtesy", "type": "string", "is_core": true},
     {"key": "birth_date", "name": "Birth Date", "type": "date", "is_core": true},
     {"key": "hire_date", "name": "Hire Date", "type": "date", "is_core": true},
     {"key": "address", "name": "Address", "type": "string", "is_core": true},
     {"key": "city", "name": "City", "type": "string", "is_core": true},
     {"key": "region", "name": "Region", "type": "string", "is_core": true},
     {"key": "postal_code", "name": "Postal Code", "type": "string", "is_core": true},
     {"key": "country", "name": "Country", "type": "string", "is_core": true},
     {"key": "reports_to", "name": "Reports To", "type": "integer", "is_core": true}
   ]}'::jsonb)
ON CONFLICT DO NOTHING;

-- Category Business Object
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, is_core, config)
VALUES 
  (gen_random_uuid(), (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1), 'northwind_category', 'northwind_category', 'Category', 'northwind_category', 'Product categories', true,
   '{"fields": [
     {"key": "category_id", "name": "Category ID", "type": "integer", "is_core": true, "required": true},
     {"key": "category_name", "name": "Category Name", "type": "string", "is_core": true, "required": true},
     {"key": "description", "name": "Description", "type": "string", "is_core": true}
   ]}'::jsonb)
ON CONFLICT DO NOTHING;

-- Supplier Business Object
INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, is_core, config)
VALUES 
  (gen_random_uuid(), (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1), 'northwind_supplier', 'northwind_supplier', 'Supplier', 'northwind_supplier', 'Product suppliers', true,
   '{"fields": [
     {"key": "supplier_id", "name": "Supplier ID", "type": "integer", "is_core": true, "required": true},
     {"key": "company_name", "name": "Company Name", "type": "string", "is_core": true, "required": true},
     {"key": "contact_name", "name": "Contact Name", "type": "string", "is_core": true},
     {"key": "contact_title", "name": "Contact Title", "type": "string", "is_core": true},
     {"key": "address", "name": "Address", "type": "string", "is_core": true},
     {"key": "city", "name": "City", "type": "string", "is_core": true},
     {"key": "region", "name": "Region", "type": "string", "is_core": true},
     {"key": "postal_code", "name": "Postal Code", "type": "string", "is_core": true},
     {"key": "country", "name": "Country", "type": "string", "is_core": true},
     {"key": "phone", "name": "Phone", "type": "string", "is_core": true}
   ]}'::jsonb)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- CORE SEMANTIC MODELS FOR NORTHWIND (TEMPLATES)
-- ============================================================================

-- Align schema with this migration's expectations (since 000058 was minimal)
ALTER TABLE public.semantic_cubes_v2
  ADD COLUMN IF NOT EXISTS tenant_datasource_id UUID,
  ADD COLUMN IF NOT EXISTS is_system BOOLEAN DEFAULT false,
  ADD COLUMN IF NOT EXISTS model_type TEXT,
  ADD COLUMN IF NOT EXISTS source_cube_id UUID,
  ADD COLUMN IF NOT EXISTS config JSONB DEFAULT '{}';

ALTER TABLE public.semantic_dimensions_v2
  ADD COLUMN IF NOT EXISTS is_inherited BOOLEAN DEFAULT false,
  ADD COLUMN IF NOT EXISTS is_overridden BOOLEAN DEFAULT false;

ALTER TABLE public.semantic_measures_v2
  ADD COLUMN IF NOT EXISTS is_inherited BOOLEAN DEFAULT false,
  ADD COLUMN IF NOT EXISTS is_overridden BOOLEAN DEFAULT false;


-- Sales Analytics Semantic Model (Core)
INSERT INTO semantic_cubes_v2 (
  id, tenant_id, name, display_name, description, sql, tenant_datasource_id, is_system, status, 
  model_type, source_cube_id, config
)
VALUES (
  'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid,
  (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1),
  'core_sales_analytics',
  'Sales Analytics',
  'Core sales analytics model with revenue, orders, and customer metrics',
  'orders o JOIN order_details od ON o.order_id = od.order_id JOIN products p ON od.product_id = p.product_id JOIN customers c ON o.customer_id = c.customer_id',
  NULL,
  true,
  'active',
  'core',
  NULL,
  '{"sql_table": "orders o JOIN order_details od ON o.order_id = od.order_id JOIN products p ON od.product_id = p.product_id JOIN customers c ON o.customer_id = c.customer_id"}'::jsonb
)
ON CONFLICT DO NOTHING;

-- Insert dimensions for Sales Analytics
INSERT INTO semantic_dimensions_v2 (id, cube_id, name, display_name, sql, type, is_inherited, is_overridden)
VALUES 
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'order_date', 'Order Date', 'o.order_date', 'time', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'customer_name', 'Customer Name', 'c.company_name', 'string', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'customer_country', 'Customer Country', 'c.country', 'string', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'customer_city', 'Customer City', 'c.city', 'string', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'product_name', 'Product Name', 'p.product_name', 'string', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'category_id', 'Category ID', 'p.category_id', 'number', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'ship_country', 'Ship Country', 'o.ship_country', 'string', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'ship_city', 'Ship City', 'o.ship_city', 'string', false, false)
ON CONFLICT DO NOTHING;

-- Insert measures for Sales Analytics
INSERT INTO semantic_measures_v2 (id, cube_id, name, display_name, sql, type, is_inherited, is_overridden)
VALUES 
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'total_revenue', 'Total Revenue', 'SUM(od.unit_price * od.quantity * (1 - od.discount))', 'number', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'order_count', 'Order Count', 'COUNT(DISTINCT o.order_id)', 'number', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'items_sold', 'Items Sold', 'SUM(od.quantity)', 'number', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'avg_order_value', 'Avg Order Value', 'AVG(od.unit_price * od.quantity * (1 - od.discount))', 'number', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'customer_count', 'Customer Count', 'COUNT(DISTINCT o.customer_id)', 'number', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'total_freight', 'Total Freight', 'SUM(o.freight)', 'number', false, false),
  (gen_random_uuid(), 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'::uuid, 'discount_amount', 'Discount Amount', 'SUM(od.unit_price * od.quantity * od.discount)', 'number', false, false)
ON CONFLICT DO NOTHING;

-- Product Performance Semantic Model (Core)
INSERT INTO semantic_cubes_v2 (
  id, tenant_id, name, display_name, description, sql, tenant_datasource_id, is_system, status,
  model_type, source_cube_id, config
)
VALUES (
  'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid,
  (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1),
  'core_product_performance',
  'Product Performance',
  'Product sales performance and inventory analytics',
  'products p JOIN order_details od ON p.product_id = od.product_id JOIN categories c ON p.category_id = c.category_id',
  NULL,
  true,
  'active',
  'core',
  NULL,
  '{"sql_table": "products p JOIN order_details od ON p.product_id = od.product_id JOIN categories c ON p.category_id = c.category_id"}'::jsonb
)
ON CONFLICT DO NOTHING;

-- Insert dimensions for Product Performance
INSERT INTO semantic_dimensions_v2 (id, cube_id, name, display_name, sql, type, is_inherited, is_overridden)
VALUES 
  (gen_random_uuid(), 'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid, 'product_name', 'Product Name', 'p.product_name', 'string', false, false),
  (gen_random_uuid(), 'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid, 'category_name', 'Category', 'c.category_name', 'string', false, false),
  (gen_random_uuid(), 'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid, 'supplier_id', 'Supplier ID', 'p.supplier_id', 'number', false, false),
  (gen_random_uuid(), 'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid, 'discontinued', 'Discontinued', 'p.discontinued', 'boolean', false, false)
ON CONFLICT DO NOTHING;

-- Insert measures for Product Performance
INSERT INTO semantic_measures_v2 (id, cube_id, name, display_name, sql, type, is_inherited, is_overridden)
VALUES 
  (gen_random_uuid(), 'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid, 'product_revenue', 'Revenue', 'SUM(od.unit_price * od.quantity * (1 - od.discount))', 'number', false, false),
  (gen_random_uuid(), 'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid, 'units_sold', 'Units Sold', 'SUM(od.quantity)', 'number', false, false),
  (gen_random_uuid(), 'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid, 'order_count', 'Orders', 'COUNT(DISTINCT od.order_id)', 'number', false, false),
  (gen_random_uuid(), 'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid, 'units_in_stock', 'In Stock', 'MAX(p.units_in_stock)', 'number', false, false),
  (gen_random_uuid(), 'b2c3d4e5-f6a7-8901-bcde-f12345678901'::uuid, 'reorder_point', 'Reorder Point', 'MAX(p.reorder_level)', 'number', false, false)
ON CONFLICT DO NOTHING;

-- Employee Performance Semantic Model (Core)
INSERT INTO semantic_cubes_v2 (
  id, tenant_id, name, display_name, description, sql, tenant_datasource_id, is_system, status,
  model_type, source_cube_id, config
)
VALUES (
  'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid,
  (SELECT id FROM public.tenants WHERE name = 'Default Tenant' LIMIT 1),
  'core_employee_performance',
  'Employee Performance',
  'Employee sales performance and productivity',
  'employees e JOIN orders o ON e.employee_id = o.employee_id JOIN order_details od ON o.order_id = od.order_id',
  NULL,
  true,
  'active',
  'core',
  NULL,
  '{"sql_table": "employees e JOIN orders o ON e.employee_id = o.employee_id JOIN order_details od ON o.order_id = od.order_id"}'::jsonb
)
ON CONFLICT DO NOTHING;

-- Insert dimensions for Employee Performance
INSERT INTO semantic_dimensions_v2 (id, cube_id, name, display_name, sql, type, is_inherited, is_overridden)
VALUES 
  (gen_random_uuid(), 'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid, 'employee_name', 'Employee Name', 'e.first_name || '' '' || e.last_name', 'string', false, false),
  (gen_random_uuid(), 'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid, 'title', 'Title', 'e.title', 'string', false, false),
  (gen_random_uuid(), 'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid, 'hire_date', 'Hire Date', 'e.hire_date', 'time', false, false),
  (gen_random_uuid(), 'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid, 'order_date', 'Order Date', 'o.order_date', 'time', false, false),
  (gen_random_uuid(), 'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid, 'country', 'Country', 'e.country', 'string', false, false)
ON CONFLICT DO NOTHING;

-- Insert measures for Employee Performance
INSERT INTO semantic_measures_v2 (id, cube_id, name, display_name, sql, type, is_inherited, is_overridden)
VALUES 
  (gen_random_uuid(), 'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid, 'sales_revenue', 'Sales Revenue', 'SUM(od.unit_price * od.quantity * (1 - od.discount))', 'number', false, false),
  (gen_random_uuid(), 'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid, 'orders_handled', 'Orders Handled', 'COUNT(DISTINCT o.order_id)', 'number', false, false),
  (gen_random_uuid(), 'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid, 'customers_served', 'Customers Served', 'COUNT(DISTINCT o.customer_id)', 'number', false, false),
  (gen_random_uuid(), 'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid, 'avg_order_value', 'Avg Order Value', 'AVG(od.unit_price * od.quantity * (1 - od.discount))', 'number', false, false)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SEMANTIC VIEWS (Generated from Semantic Models)
-- ============================================================================

-- Sales Analytics View
CREATE OR REPLACE VIEW vw_sales_analytics AS
SELECT 
  o.order_date,
  c.company_name AS customer_name,
  c.country AS customer_country,
  c.city AS customer_city,
  p.product_name,
  p.category_id,
  o.ship_country,
  o.ship_city,
  SUM(od.unit_price * od.quantity * (1 - od.discount)) AS total_revenue,
  COUNT(DISTINCT o.order_id) AS order_count,
  SUM(od.quantity) AS items_sold,
  AVG(od.unit_price * od.quantity * (1 - od.discount)) AS avg_order_value,
  COUNT(DISTINCT o.customer_id) AS customer_count,
  SUM(o.freight) AS total_freight,
  SUM(od.unit_price * od.quantity * od.discount) AS discount_amount
FROM orders o
JOIN order_details od ON o.order_id = od.order_id
JOIN products p ON od.product_id = p.product_id
JOIN customers c ON o.customer_id = c.customer_id
GROUP BY 
  o.order_date, c.company_name, c.country, c.city,
  p.product_name, p.category_id, o.ship_country, o.ship_city;

-- Product Performance View
CREATE OR REPLACE VIEW vw_product_performance AS
SELECT 
  p.product_name,
  cat.category_name,
  p.supplier_id,
  p.discontinued,
  SUM(od.unit_price * od.quantity * (1 - od.discount)) AS revenue,
  SUM(od.quantity) AS units_sold,
  COUNT(DISTINCT od.order_id) AS order_count,
  MAX(p.units_in_stock) AS units_in_stock,
  MAX(p.reorder_level) AS reorder_level
FROM products p
JOIN order_details od ON p.product_id = od.product_id
JOIN categories cat ON p.category_id = cat.category_id
GROUP BY p.product_name, cat.category_name, p.supplier_id, p.discontinued;

-- Employee Performance View
CREATE OR REPLACE VIEW vw_employee_performance AS
SELECT 
  e.first_name || ' ' || e.last_name AS employee_name,
  e.title,
  e.hire_date,
  o.order_date,
  e.country,
  SUM(od.unit_price * od.quantity * (1 - od.discount)) AS sales_revenue,
  COUNT(DISTINCT o.order_id) AS orders_handled,
  COUNT(DISTINCT o.customer_id) AS customers_served,
  AVG(od.unit_price * od.quantity * (1 - od.discount)) AS avg_order_value
FROM employees e
JOIN orders o ON e.employee_id = o.employee_id
JOIN order_details od ON o.order_id = od.order_id
GROUP BY e.first_name, e.last_name, e.title, e.hire_date, o.order_date, e.country;

-- ============================================================================
-- CUSTOMER ANALYTICS VIEW (Additional)
-- ============================================================================
CREATE OR REPLACE VIEW vw_customer_analytics AS
SELECT 
  c.customer_id,
  c.company_name,
  c.contact_name,
  c.country,
  c.city,
  COUNT(DISTINCT o.order_id) AS total_orders,
  SUM(od.unit_price * od.quantity * (1 - od.discount)) AS total_spent,
  AVG(od.unit_price * od.quantity * (1 - od.discount)) AS avg_order_value,
  MIN(o.order_date) AS first_order_date,
  MAX(o.order_date) AS last_order_date,
  COUNT(DISTINCT p.product_id) AS unique_products_ordered
FROM customers c
LEFT JOIN orders o ON c.customer_id = o.customer_id
LEFT JOIN order_details od ON o.order_id = od.order_id
LEFT JOIN products p ON od.product_id = p.product_id
GROUP BY c.customer_id, c.company_name, c.contact_name, c.country, c.city;

COMMENT ON VIEW vw_sales_analytics IS 'Sales analytics aggregated by customer, product, and geography';
COMMENT ON VIEW vw_product_performance IS 'Product performance metrics with inventory status';
COMMENT ON VIEW vw_employee_performance IS 'Employee sales performance and productivity metrics';
COMMENT ON VIEW vw_customer_analytics IS 'Customer analytics with lifetime value metrics';
