-- Fix Northwind Semantic Models
-- Seed Northwind Semantic Models for the Northwinds tenant

DO $$
DECLARE
    t_id UUID;
    ds_id UUID;
    model_id UUID;
    dim_id UUID;
    measure_id UUID;
BEGIN

        -- 1. Sales Analytics
        -- Clean up existing attempts to avoid unique constraint violations on (tenant_id, name, version)
        -- caused by changing ID generation strategy
        DELETE FROM semantic_cubes_v2 WHERE tenant_id = t_id AND name = 'core_sales_analytics';

        INSERT INTO semantic_cubes_v2 (
            id, tenant_id, name, display_name, label, description,
            sql, status, is_system, model_type,
            business_object_id,
            config
        )
        VALUES
        (
            -- Deterministic UUID generation based on tenant + name to allow re-runs for multiple tenants
            uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text),
            t_id,
            'core_sales_analytics',
            'Sales Analytics', 'Sales Analytics',
            'Core sales analytics model with revenue, orders, and customer metrics',
            'orders o JOIN order_details od ON o.order_id = od.order_id JOIN products p ON od.product_id = p.product_id JOIN customers c ON o.customer_id = c.customer_id',
            'active', true, 'custom',
            (SELECT id FROM business_objects WHERE key = 'northwind_order' AND tenant_id = t_id LIMIT 1),
            '{"sql_table": "orders o JOIN order_details od ON o.order_id = od.order_id JOIN products p ON od.product_id = p.product_id JOIN customers c ON o.customer_id = c.customer_id"}'::jsonb
        )
        ON CONFLICT (id) DO UPDATE SET
            tenant_id = EXCLUDED.tenant_id,
            business_object_id = EXCLUDED.business_object_id,
            sql = EXCLUDED.sql,
            status = 'active',
            display_name = EXCLUDED.display_name,
            label = EXCLUDED.label,
            model_type = EXCLUDED.model_type,
            is_system = EXCLUDED.is_system;

        -- Dimensions/Measures for Sales Analytics (need to handle IDs carefully or delete/recreate)
        -- For simplicity, we'll DELETE existing dimensions/measures for this cube and re-insert
        -- This avoids complex ON CONFLICT logic with generated UUIDs
        DELETE FROM cube_dimensions_v2 WHERE cube_id = uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text);
        DELETE FROM cube_measures_v2 WHERE cube_id = uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text);

        -- Dimensions
        INSERT INTO cube_dimensions_v2 (id, cube_id, name, label, display_name, sql, type, is_inherited, is_overridden)
        VALUES 
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'order_date', 'Order Date', 'Order Date', 'o.order_date', 'time', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'customer_name', 'Customer Name', 'Customer Name', 'c.company_name', 'string', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'customer_country', 'Customer Country', 'Customer Country', 'c.country', 'string', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'customer_city', 'Customer City', 'Customer City', 'c.city', 'string', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'product_name', 'Product Name', 'Product Name', 'p.product_name', 'string', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'category_id', 'Category ID', 'Category ID', 'p.category_id', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'ship_country', 'Ship Country', 'Ship Country', 'o.ship_country', 'string', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'ship_city', 'Ship City', 'Ship City', 'o.ship_city', 'string', false, false);

        -- Measures
        INSERT INTO cube_measures_v2 (id, cube_id, name, label, display_name, sql, type, is_inherited, is_overridden)
        VALUES 
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'total_revenue', 'Total Revenue', 'Total Revenue', 'SUM(od.unit_price * od.quantity * (1 - od.discount))', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'order_count', 'Order Count', 'Order Count', 'COUNT(DISTINCT o.order_id)', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'items_sold', 'Items Sold', 'Items Sold', 'SUM(od.quantity)', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'avg_order_value', 'Avg Order Value', 'Avg Order Value', 'AVG(od.unit_price * od.quantity * (1 - od.discount))', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'customer_count', 'Customer Count', 'Customer Count', 'COUNT(DISTINCT o.customer_id)', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'total_freight', 'Total Freight', 'Total Freight', 'SUM(o.freight)', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_sales_analytics' || t_id::text), 'discount_amount', 'Discount Amount', 'Discount Amount', 'SUM(od.unit_price * od.quantity * od.discount)', 'number', false, false);


        -- 2. Product Performance
        DELETE FROM semantic_cubes_v2 WHERE tenant_id = t_id AND name = 'core_product_performance';

        INSERT INTO semantic_cubes_v2 (
            id, tenant_id, name, display_name, label, description,
            sql, status, is_system, model_type,
            business_object_id,
            config
        )
        VALUES
        (
            uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text),
            t_id,
            'core_product_performance',
            'Product Performance', 'Product Performance',
            'Product sales performance and inventory analytics',
            'products p JOIN order_details od ON p.product_id = od.product_id JOIN categories c ON p.category_id = c.category_id',
            'active', true, 'custom',
            (SELECT id FROM business_objects WHERE key = 'northwind_product' AND tenant_id = t_id LIMIT 1),
            '{"sql_table": "products p JOIN order_details od ON p.product_id = od.product_id JOIN categories c ON p.category_id = c.category_id"}'::jsonb
        )
        ON CONFLICT (id) DO UPDATE SET
            tenant_id = EXCLUDED.tenant_id,
            business_object_id = EXCLUDED.business_object_id,
            sql = EXCLUDED.sql,
            status = 'active',
            display_name = EXCLUDED.display_name,
            label = EXCLUDED.label,
            model_type = EXCLUDED.model_type,
            is_system = EXCLUDED.is_system;

        DELETE FROM cube_dimensions_v2 WHERE cube_id = uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text);
        DELETE FROM cube_measures_v2 WHERE cube_id = uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text);

        -- Dimensions
        INSERT INTO cube_dimensions_v2 (id, cube_id, name, label, display_name, sql, type, is_inherited, is_overridden)
        VALUES 
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text), 'product_name', 'Product Name', 'Product Name', 'p.product_name', 'string', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text), 'category_name', 'Category', 'Category', 'c.category_name', 'string', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text), 'supplier_id', 'Supplier ID', 'Supplier ID', 'p.supplier_id', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text), 'discontinued', 'Discontinued', 'Discontinued', 'p.discontinued', 'boolean', false, false);

        -- Measures
        INSERT INTO cube_measures_v2 (id, cube_id, name, label, display_name, sql, type, is_inherited, is_overridden)
        VALUES 
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text), 'product_revenue', 'Revenue', 'Revenue', 'SUM(od.unit_price * od.quantity * (1 - od.discount))', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text), 'units_sold', 'Units Sold', 'Units Sold', 'SUM(od.quantity)', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text), 'order_count', 'Orders', 'Orders', 'COUNT(DISTINCT od.order_id)', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text), 'units_in_stock', 'In Stock', 'In Stock', 'MAX(p.units_in_stock)', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_product_performance' || t_id::text), 'reorder_point', 'Reorder Point', 'Reorder Point', 'MAX(p.reorder_level)', 'number', false, false);


        -- 3. Employee Performance
        DELETE FROM semantic_cubes_v2 WHERE tenant_id = t_id AND name = 'core_employee_performance';

        INSERT INTO semantic_cubes_v2 (
            id, tenant_id, name, display_name, label, description,
            sql, status, is_system, model_type,
            business_object_id,
            config
        )
        VALUES
        (
            uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text),
            t_id,
            'core_employee_performance',
            'Employee Performance', 'Employee Performance',
            'Employee sales performance and productivity',
            'employees e JOIN orders o ON e.employee_id = o.employee_id JOIN order_details od ON o.order_id = od.order_id',
            'active', true, 'custom',
            (SELECT id FROM business_objects WHERE key = 'northwind_employee' AND tenant_id = t_id LIMIT 1),
            '{"sql_table": "employees e JOIN orders o ON e.employee_id = o.employee_id JOIN order_details od ON o.order_id = od.order_id"}'::jsonb
        )
        ON CONFLICT (id) DO UPDATE SET
            tenant_id = EXCLUDED.tenant_id,
            business_object_id = EXCLUDED.business_object_id,
            sql = EXCLUDED.sql,
            status = 'active',
            display_name = EXCLUDED.display_name,
            label = EXCLUDED.label,
            model_type = EXCLUDED.model_type,
            is_system = EXCLUDED.is_system;

        DELETE FROM cube_dimensions_v2 WHERE cube_id = uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text);
        DELETE FROM cube_measures_v2 WHERE cube_id = uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text);

        -- Dimensions
        INSERT INTO cube_dimensions_v2 (id, cube_id, name, label, display_name, sql, type, is_inherited, is_overridden)
        VALUES 
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text), 'employee_name', 'Employee Name', 'Employee Name', 'e.first_name || '' '' || e.last_name', 'string', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text), 'title', 'Title', 'Title', 'e.title', 'string', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text), 'hire_date', 'Hire Date', 'Hire Date', 'e.hire_date', 'time', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text), 'order_date', 'Order Date', 'Order Date', 'o.order_date', 'time', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text), 'country', 'Country', 'Country', 'e.country', 'string', false, false);

        -- Measures
        INSERT INTO cube_measures_v2 (id, cube_id, name, label, display_name, sql, type, is_inherited, is_overridden)
        VALUES 
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text), 'sales_revenue', 'Sales Revenue', 'Sales Revenue', 'SUM(od.unit_price * od.quantity * (1 - od.discount))', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text), 'orders_handled', 'Orders Handled', 'Orders Handled', 'COUNT(DISTINCT o.order_id)', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text), 'customers_served', 'Customers Served', 'Customers Served', 'COUNT(DISTINCT o.customer_id)', 'number', false, false),
          (gen_random_uuid(), uuid_generate_v5(uuid_ns_url(), 'core_employee_performance' || t_id::text), 'avg_order_value', 'Avg Order Value', 'Avg Order Value', 'AVG(od.unit_price * od.quantity * (1 - od.discount))', 'number', false, false);

    
END $$;
