-- Seed Northwind Business Objects for Gold Copy Tenant

DO $$
DECLARE
    v_gold_tenant UUID;
BEGIN
    PERFORM set_config('app.goldcopy', (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)::text, true);
    v_gold_tenant := current_setting('app.goldcopy', true)::uuid;

    -- 1. ENTITIES (Level 2)
    INSERT INTO catalog_node (id, tenant_id, name, type, physical_schema, physical_table, description, default_filters)
    VALUES ('ent_customer', v_gold_tenant, 'Customer', 'ENTITY', 'analytics', 'customers', 'Customer master records', '[]'::jsonb)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO catalog_node (id, tenant_id, name, type, physical_schema, physical_table, description, default_filters)
    VALUES ('ent_order', v_gold_tenant, 'Order', 'ENTITY', 'analytics', 'orders', 'Sales header transactions', '[]'::jsonb)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO catalog_node (id, tenant_id, name, type, physical_schema, physical_table, description, default_filters)
    VALUES ('ent_order_detail', v_gold_tenant, 'OrderDetail', 'ENTITY', 'analytics', 'order_details', 'Line-item transaction details', '[]'::jsonb)
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO catalog_node (id, tenant_id, name, type, physical_schema, physical_table, description, default_filters)
    VALUES ('ent_product', v_gold_tenant, 'Product', 'ENTITY', 'analytics', 'products', 'Product catalog and cost data', '[]'::jsonb)
    ON CONFLICT (id) DO NOTHING;

    -- 2. ATTRIBUTES (Level 3: Dimensions & Measures)
    INSERT INTO catalog_node (id, tenant_id, name, type, parent_id, physical_column, format, description)
    VALUES ('attr_cust_country', v_gold_tenant, 'Country', 'ATTRIBUTE', 'ent_customer', 'country', 'string', 'Customer country')
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO catalog_node (id, tenant_id, name, type, parent_id, physical_column, format, description)
    VALUES ('attr_cust_region', v_gold_tenant, 'Region', 'ATTRIBUTE', 'ent_customer', 'region', 'string', 'Customer region')
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO catalog_node (id, tenant_id, name, type, parent_id, physical_column, format, description)
    VALUES ('attr_order_date', v_gold_tenant, 'OrderDate', 'ATTRIBUTE', 'ent_order', 'order_date', 'date', 'Date order was placed')
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO catalog_node (id, tenant_id, name, type, parent_id, physical_column, agg_function, format, description)
    VALUES ('attr_qty', v_gold_tenant, 'Quantity', 'ATTRIBUTE', 'ent_order_detail', 'quantity', 'SUM', 'number', 'Total units sold')
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO catalog_node (id, tenant_id, name, type, parent_id, physical_column, agg_function, format, description)
    VALUES ('attr_gross_revenue', v_gold_tenant, 'GrossRevenue', 'ATTRIBUTE', 'ent_order_detail', 'unit_price * quantity', 'SUM', 'currency', 'Total gross sales revenue')
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO catalog_node (id, tenant_id, name, type, parent_id, physical_column, agg_function, format, description)
    VALUES ('attr_total_cost', v_gold_tenant, 'TotalCost', 'ATTRIBUTE', 'ent_order_detail', 'unit_cost * quantity', 'SUM', 'currency', 'Total product cost of goods sold')
    ON CONFLICT (id) DO NOTHING;

    -- 3. CALCULATED MEASURES (Topological DAG Test)
    INSERT INTO catalog_node (id, tenant_id, name, type, parent_id, expression, format, description)
    VALUES ('attr_gross_profit', v_gold_tenant, 'GrossProfit', 'ATTRIBUTE', 'ent_order_detail', '${OrderDetail.GrossRevenue} - ${OrderDetail.TotalCost}', 'currency', 'Net profit after COGS')
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO catalog_node (id, tenant_id, name, type, parent_id, expression, format, description)
    VALUES ('attr_profit_margin', v_gold_tenant, 'ProfitMargin', 'ATTRIBUTE', 'ent_order_detail', '${OrderDetail.GrossProfit} / NULLIF(${OrderDetail.GrossRevenue}, 0)', 'percentage', 'Profit margin percentage')
    ON CONFLICT (id) DO NOTHING;

    -- 4. EDGES (Join Path Relationships)
    INSERT INTO catalog_edge (tenant_id, source_entity_id, target_entity_id, join_type, join_condition)
    VALUES (v_gold_tenant, 'ent_order', 'ent_customer', 'INNER', 'Order.customer_id = Customer.id')
    ON CONFLICT DO NOTHING;

    INSERT INTO catalog_edge (tenant_id, source_entity_id, target_entity_id, join_type, join_condition)
    VALUES (v_gold_tenant, 'ent_order_detail', 'ent_order', 'INNER', 'OrderDetail.order_id = Order.id')
    ON CONFLICT DO NOTHING;

    INSERT INTO catalog_edge (tenant_id, source_entity_id, target_entity_id, join_type, join_condition)
    VALUES (v_gold_tenant, 'ent_order_detail', 'ent_product', 'INNER', 'OrderDetail.product_id = Product.id')
    ON CONFLICT DO NOTHING;

END $$;
