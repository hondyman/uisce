-- Fix Northwind Business Objects
-- Seed Northwind Business Objects for the Northwinds tenant
DO $$
DECLARE
    t_id UUID;
BEGIN
    -- Get the Northwinds tenant ID
    SELECT id INTO t_id FROM tenants WHERE name = 'Northwinds' LIMIT 1;
    
    IF t_id IS NULL THEN
        RAISE EXCEPTION 'Northwinds tenant not found';
    END IF;

    RAISE NOTICE 'Seeding Northwind Business Objects for tenant: %', t_id;

    -- Customer Business Object
    INSERT INTO business_objects (id, tenant_id, name, display_name, description, key, is_core, config, is_system)
    VALUES 
      (gen_random_uuid(), t_id, 'northwind_customer', 'Customer', 'Northwind customer data', 'northwind_customer', true, 
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
       ]}'::jsonb, false)
    ON CONFLICT (name, tenant_id) DO NOTHING;

    -- Order Business Object
    INSERT INTO business_objects (id, tenant_id, name, display_name, description, key, is_core, config, is_system)
    VALUES 
      (gen_random_uuid(), t_id, 'northwind_order', 'Order', 'Northwind order data', 'northwind_order', true,
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
       ]}'::jsonb, false)
    ON CONFLICT (name, tenant_id) DO NOTHING;

    -- Product Business Object
    INSERT INTO business_objects (id, tenant_id, name, display_name, description, key, is_core, config, is_system)
    VALUES 
      (gen_random_uuid(), t_id, 'northwind_product', 'Product', 'Northwind product catalog', 'northwind_product', true,
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
       ]}'::jsonb, false)
    ON CONFLICT (name, tenant_id) DO NOTHING;

    -- Employee Business Object
    INSERT INTO business_objects (id, tenant_id, name, display_name, description, key, is_core, config, is_system)
    VALUES 
      (gen_random_uuid(), t_id, 'northwind_employee', 'Employee', 'Northwind employee data', 'northwind_employee', true,
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
       ]}'::jsonb, false)
    ON CONFLICT (name, tenant_id) DO NOTHING;

    -- Category Business Object
    INSERT INTO business_objects (id, tenant_id, name, display_name, description, key, is_core, config, is_system)
    VALUES 
      (gen_random_uuid(), t_id, 'northwind_category', 'Category', 'Product categories', 'northwind_category', true,
       '{"fields": [
         {"key": "category_id", "name": "Category ID", "type": "integer", "is_core": true, "required": true},
         {"key": "category_name", "name": "Category Name", "type": "string", "is_core": true, "required": true},
         {"key": "description", "name": "Description", "type": "string", "is_core": true}
       ]}'::jsonb, false)
    ON CONFLICT (name, tenant_id) DO NOTHING;

    -- Supplier Business Object
    INSERT INTO business_objects (id, tenant_id, name, display_name, description, key, is_core, config, is_system)
    VALUES 
      (gen_random_uuid(), t_id, 'northwind_supplier', 'Supplier', 'Product suppliers', 'northwind_supplier', true,
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
       ]}'::jsonb, false)
    ON CONFLICT (name, tenant_id) DO NOTHING;

    RAISE NOTICE 'Seeded 6 Northwind Business Objects for Tenant %', t_id;
    
END $$;
