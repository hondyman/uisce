-- Setup script for Northwind Business Objects bindings, semantic terms, edge relationships, calculated fields, and validation rules in Alpha Postgres Database

BEGIN;

-- 1. Update Business Objects properties with driver_table_id and driver_table_name
UPDATE catalog_node
SET properties = properties || jsonb_build_object(
  'driver_table_id', 'e7a5338d-a660-511b-b2d5-af1bfadb5fe0',
  'driver_table_name', 'products'
)
WHERE id = '91c96610-36d1-47e8-b9db-b9bfa68da588'; -- Product Catalog

UPDATE catalog_node
SET properties = properties || jsonb_build_object(
  'driver_table_id', 'a9c328db-2c25-59b7-b308-ea68a6d70e50',
  'driver_table_name', 'categories'
)
WHERE id = '15ff9881-f34f-4e36-a906-ba3a10465dcc'; -- Category Catalog

UPDATE catalog_node
SET properties = properties || jsonb_build_object(
  'driver_table_id', '191ec2ca-63e0-5e9d-851d-79a60a5ce6dc',
  'driver_table_name', 'suppliers'
)
WHERE id = '497b5822-41e5-4f84-a68c-344eda5e1ed3'; -- Vendor Directory

UPDATE catalog_node
SET properties = properties || jsonb_build_object(
  'driver_table_id', '79562d6f-b964-5c91-a944-178d092c9c68',
  'driver_table_name', 'customers'
)
WHERE id = 'ddab2423-7988-4161-add9-48df2da4bd96'; -- Customer Directory

UPDATE catalog_node
SET properties = properties || jsonb_build_object(
  'driver_table_id', 'ddf65296-52f8-521f-94b6-75085de0c5e4',
  'driver_table_name', 'employees'
)
WHERE id = '01a8db46-62d6-47fc-918e-58131f12c1b3'; -- Personnel Register

UPDATE catalog_node
SET properties = properties || jsonb_build_object(
  'driver_table_id', '57f54302-43c0-59e9-bc82-24e13790121f',
  'driver_table_name', 'shippers'
)
WHERE id = '602cc397-6b7c-43f5-a79e-14c3cc67581a'; -- Logistics Carrier

UPDATE catalog_node
SET properties = properties || jsonb_build_object(
  'driver_table_id', 'b4e75141-5869-58ea-99c2-7442b9138012',
  'driver_table_name', 'orders'
)
WHERE id = 'bed7015d-6487-4fee-b93d-79aa4ddb8528'; -- Sales Ledger

UPDATE catalog_node
SET properties = properties || jsonb_build_object(
  'driver_table_id', 'acd07332-1cd2-561d-a32f-81f7191f48be',
  'driver_table_name', 'order_details'
)
WHERE id = '4c8e42b4-8c09-4b61-9cd8-9e64b82aa21d'; -- Order Line Items


-- 2. Insert missing semantic terms for Product Catalog and other Northwind tables
INSERT INTO catalog_node (
  id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, properties
) VALUES
  -- Product ID
  (
    '6f31e55d-ee7d-590d-832b-e3f06cc2f001',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Product ID',
    'Unique product identifier',
    'semantic:product:product_id',
    true,
    jsonb_build_object(
      'name', 'product_id',
      'title', 'Product ID',
      'data_type', 'integer',
      'semantic_type', 'dimension',
      'primaryKey', true,
      'physical_mapping', jsonb_build_object('table', 'products', 'column', 'product_id'),
      'sql', '${CUBE}.product_id'
    )
  ),
  -- Product Name
  (
    '6f31e55d-ee7d-590d-832b-e3f06cc2f002',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Product Name',
    'Commercial name of the product item',
    'semantic:product:product_name',
    true,
    jsonb_build_object(
      'name', 'product_name',
      'title', 'Product Name',
      'data_type', 'string',
      'semantic_type', 'dimension',
      'physical_mapping', jsonb_build_object('table', 'products', 'column', 'product_name'),
      'sql', '${CUBE}.product_name'
    )
  ),
  -- Quantity Per Unit
  (
    '6f31e55d-ee7d-590d-832b-e3f06cc2f003',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Quantity Per Unit',
    'Packaging quantity specification',
    'semantic:product:quantity_per_unit',
    true,
    jsonb_build_object(
      'name', 'quantity_per_unit',
      'title', 'Quantity Per Unit',
      'data_type', 'string',
      'semantic_type', 'dimension',
      'physical_mapping', jsonb_build_object('table', 'products', 'column', 'quantity_per_unit'),
      'sql', '${CUBE}.quantity_per_unit'
    )
  ),
  -- Discontinued Flag
  (
    '6f31e55d-ee7d-590d-832b-e3f06cc2f004',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Discontinued Status',
    'Flag indicating if the product has been discontinued',
    'semantic:product:discontinued',
    true,
    jsonb_build_object(
      'name', 'discontinued',
      'title', 'Discontinued Status',
      'data_type', 'boolean',
      'semantic_type', 'dimension',
      'physical_mapping', jsonb_build_object('table', 'products', 'column', 'discontinued'),
      'sql', '${CUBE}.discontinued'
    )
  ),
  -- Category ID (FK)
  (
    '6f31e55d-ee7d-590d-832b-e3f06cc2f005',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Category Identifier',
    'Foreign key linking product to product category',
    'semantic:product:category_id',
    true,
    jsonb_build_object(
      'name', 'category_id',
      'title', 'Category Identifier',
      'data_type', 'integer',
      'semantic_type', 'dimension',
      'physical_mapping', jsonb_build_object('table', 'products', 'column', 'category_id'),
      'sql', '${CUBE}.category_id'
    )
  ),
  -- Supplier ID (FK)
  (
    '6f31e55d-ee7d-590d-832b-e3f06cc2f006',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Supplier Identifier',
    'Foreign key linking product to vendor supplier',
    'semantic:product:supplier_id',
    true,
    jsonb_build_object(
      'name', 'supplier_id',
      'title', 'Supplier Identifier',
      'data_type', 'integer',
      'semantic_type', 'dimension',
      'physical_mapping', jsonb_build_object('table', 'products', 'column', 'supplier_id'),
      'sql', '${CUBE}.supplier_id'
    )
  )
ON CONFLICT (id) DO NOTHING;


-- 3. Link semantic terms to physical table columns via catalog_edge (maps_to_column / MAPS_TO)
INSERT INTO catalog_edge (
  id, tenant_id, edge_type_id, source_node_id, target_node_id, relationship_type, is_active, properties, created_at, updated_at
) VALUES
  -- Product Unit Price -> products.unit_price column
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', '6f31e55d-ee7d-590d-832b-e3f06cc2f000', 'a7f4307f-2137-5975-8bec-ab17caf7e398', 'MAPS_TO', true, '{}', NOW(), NOW()),
  -- Units In Stock -> products.units_in_stock column
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', 'f6984ee3-83c9-5233-a97c-f22379d7a777', 'b7e2e988-e6ee-521e-aa6a-94d954b679dc', 'MAPS_TO', true, '{}', NOW(), NOW()),
  -- Reorder Level -> products.reorder_level column
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', 'bcf0d225-6d14-591b-82f5-4c6688190f2a', 'ea1ed577-ca96-5f14-a33c-ff561ce76a65', 'MAPS_TO', true, '{}', NOW(), NOW()),
  -- Product ID -> products.product_id column
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', '6f31e55d-ee7d-590d-832b-e3f06cc2f001', 'd174c3ba-952d-51ee-b344-01598b8a76b7', 'MAPS_TO', true, '{}', NOW(), NOW()),
  -- Product Name -> products.product_name column
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', '6f31e55d-ee7d-590d-832b-e3f06cc2f002', '5a32e15e-9791-5190-88e1-4952e46fdca8', 'MAPS_TO', true, '{}', NOW(), NOW()),
  -- Quantity Per Unit -> products.quantity_per_unit column
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', '6f31e55d-ee7d-590d-832b-e3f06cc2f003', '03f8dea4-130a-5028-9ce9-a9a1a6971c1b', 'MAPS_TO', true, '{}', NOW(), NOW()),
  -- Discontinued -> products.discontinued column
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', '6f31e55d-ee7d-590d-832b-e3f06cc2f004', '8f4c5016-aec8-57e0-a292-e8e3c4e7a579', 'MAPS_TO', true, '{}', NOW(), NOW()),
  -- Category ID -> products.category_id column
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', '6f31e55d-ee7d-590d-832b-e3f06cc2f005', 'b37d2e08-aefc-5f4a-b3ee-60c07a4f6c1f', 'MAPS_TO', true, '{}', NOW(), NOW()),
  -- Supplier ID -> products.supplier_id column
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', '6f31e55d-ee7d-590d-832b-e3f06cc2f006', '8109fe63-67e1-5c29-924e-af23d5ab2f59', 'MAPS_TO', true, '{}', NOW(), NOW())
ON CONFLICT DO NOTHING;


-- 4. Establish cross-table relationships via catalog_edge between table driving nodes
INSERT INTO catalog_edge (
  id, tenant_id, edge_type_id, source_node_id, target_node_id, relationship_type, is_active, properties, created_at, updated_at
) VALUES
  -- products (category_id) -> categories (category_id)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    'e7a5338d-a660-511b-b2d5-af1bfadb5fe0',
    'a9c328db-2c25-59b7-b308-ea68a6d70e50',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'category_id',
      'target_column', 'category_id',
      'source_bo_id', '91c96610-36d1-47e8-b9db-b9bfa68da588',
      'target_bo_id', '15ff9881-f34f-4e36-a906-ba3a10465dcc'
    ),
    NOW(),
    NOW()
  ),
  -- products (supplier_id) -> suppliers (supplier_id)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    'e7a5338d-a660-511b-b2d5-af1bfadb5fe0',
    '191ec2ca-63e0-5e9d-851d-79a60a5ce6dc',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'supplier_id',
      'target_column', 'supplier_id',
      'source_bo_id', '91c96610-36d1-47e8-b9db-b9bfa68da588',
      'target_bo_id', '497b5822-41e5-4f84-a68c-344eda5e1ed3'
    ),
    NOW(),
    NOW()
  ),
  -- order_details (product_id) -> products (product_id)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    'acd07332-1cd2-561d-a32f-81f7191f48be',
    'e7a5338d-a660-511b-b2d5-af1bfadb5fe0',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'product_id',
      'target_column', 'product_id',
      'source_bo_id', '4c8e42b4-8c09-4b61-9cd8-9e64b82aa21d',
      'target_bo_id', '91c96610-36d1-47e8-b9db-b9bfa68da588'
    ),
    NOW(),
    NOW()
  ),
  -- order_details (order_id) -> orders (order_id)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    'acd07332-1cd2-561d-a32f-81f7191f48be',
    'b4e75141-5869-58ea-99c2-7442b9138012',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'order_id',
      'target_column', 'order_id',
      'source_bo_id', '4c8e42b4-8c09-4b61-9cd8-9e64b82aa21d',
      'target_bo_id', 'bed7015d-6487-4fee-b93d-79aa4ddb8528'
    ),
    NOW(),
    NOW()
  ),
  -- orders (customer_id) -> customers (customer_id)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    'b4e75141-5869-58ea-99c2-7442b9138012',
    '79562d6f-b964-5c91-a944-178d092c9c68',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'customer_id',
      'target_column', 'customer_id',
      'source_bo_id', 'bed7015d-6487-4fee-b93d-79aa4ddb8528',
      'target_bo_id', 'ddab2423-7988-4161-add9-48df2da4bd96'
    ),
    NOW(),
    NOW()
  ),
  -- orders (employee_id) -> employees (employee_id)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    'b4e75141-5869-58ea-99c2-7442b9138012',
    'ddf65296-52f8-521f-94b6-75085de0c5e4',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'employee_id',
      'target_column', 'employee_id',
      'source_bo_id', 'bed7015d-6487-4fee-b93d-79aa4ddb8528',
      'target_bo_id', '01a8db46-62d6-47fc-918e-58131f12c1b3'
    ),
    NOW(),
    NOW()
  ),
  -- orders (ship_via) -> shippers (shipper_id)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    'b4e75141-5869-58ea-99c2-7442b9138012',
    '57f54302-43c0-59e9-bc82-24e13790121f',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'ship_via',
      'target_column', 'shipper_id',
      'source_bo_id', 'bed7015d-6487-4fee-b93d-79aa4ddb8528',
      'target_bo_id', '602cc397-6b7c-43f5-a79e-14c3cc67581a'
    ),
    NOW(),
    NOW()
  )
ON CONFLICT DO NOTHING;


-- 5. Insert Calculated Fields into catalog_node for Product Catalog BO
INSERT INTO catalog_node (
  id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, properties
) VALUES
  (
    '6f31e55d-ee7d-590d-832b-e3f06cc2f010',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Total Stock Valuation',
    'Calculated total monetary valuation of product items in inventory (unit_price * units_in_stock)',
    'semantic:product:total_stock_valuation',
    true,
    jsonb_build_object(
      'name', 'total_stock_valuation',
      'title', 'Total Stock Valuation',
      'data_type', 'decimal',
      'semantic_type', 'measure',
      'expression', '${CUBE.unit_price} * ${CUBE.units_in_stock}',
      'is_calculated', true,
      'bo_id', '91c96610-36d1-47e8-b9db-b9bfa68da588',
      'physical_mapping', jsonb_build_object('table', 'products', 'expression', 'unit_price * units_in_stock'),
      'sql', 'SUM(${CUBE.unit_price} * ${CUBE.units_in_stock})'
    )
  ),
  (
    '6f31e55d-ee7d-590d-832b-e3f06cc2f011',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Reorder Threshold Shortfall',
    'Calculated deficit between reorder level and current stock',
    'semantic:product:reorder_shortfall',
    true,
    jsonb_build_object(
      'name', 'reorder_shortfall',
      'title', 'Reorder Shortfall',
      'data_type', 'integer',
      'semantic_type', 'measure',
      'expression', 'GREATEST(0, ${CUBE.reorder_level} - ${CUBE.units_in_stock})',
      'is_calculated', true,
      'bo_id', '91c96610-36d1-47e8-b9db-b9bfa68da588',
      'physical_mapping', jsonb_build_object('table', 'products', 'expression', 'GREATEST(0, reorder_level - units_in_stock)'),
      'sql', 'GREATEST(0, ${CUBE.reorder_level} - ${CUBE.units_in_stock})'
    )
  )
ON CONFLICT (id) DO NOTHING;


-- 6. Insert Validation Rules into catalog_validation_rules for Product Catalog BO
INSERT INTO catalog_validation_rules (
  id, tenant_id, datasource_id, rule_name, rule_type, description, target_entity, condition_json, severity, is_active, created_at, updated_at
) VALUES
  (
    'b1111111-1111-4111-8111-111111111101',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'c0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e5',
    'Unit Price Must Be Positive',
    'business_logic',
    'Ensures that the unit price of any active product is greater than 0',
    'product',
    jsonb_build_object(
      'schema_version', '1',
      'authored_mode', 'designer',
      'payload', jsonb_build_object(
        'field', 'unit_price',
        'operator', 'greater_than',
        'value', 0
      )
    ),
    'error',
    true,
    NOW(),
    NOW()
  ),
  (
    'b1111111-1111-4111-8111-111111111102',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'c0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e5',
    'Units In Stock Non-Negative',
    'business_logic',
    'Verifies that inventory stock level cannot fall below zero',
    'product',
    jsonb_build_object(
      'schema_version', '1',
      'authored_mode', 'designer',
      'payload', jsonb_build_object(
        'field', 'units_in_stock',
        'operator', 'greater_than_or_equal',
        'value', 0
      )
    ),
    'error',
    true,
    NOW(),
    NOW()
  ),
  (
    'b1111111-1111-4111-8111-111111111103',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'c0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e5',
    'Reorder Warning Alert',
    'business_logic',
    'Triggers an alert when units in stock fall at or below reorder level for active non-discontinued items',
    'product',
    jsonb_build_object(
      'schema_version', '1',
      'authored_mode', 'designer',
      'payload', jsonb_build_object(
        'field', 'units_in_stock',
        'operator', 'less_than_or_equal',
        'target_field', 'reorder_level'
      )
    ),
    'warning',
    true,
    NOW(),
    NOW()
  )
ON CONFLICT (id) DO NOTHING;

COMMIT;
