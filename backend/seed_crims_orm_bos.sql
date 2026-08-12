-- Setup script for CRIMS Front Office ORM Business Objects, Semantic Terms, Graph Edges, Calculated Fields, and Validations in Alpha Postgres Database

BEGIN;

-- 1. Create Front-Office CRIMS ORM Business Objects in catalog_node
INSERT INTO catalog_node (
  id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, properties
) VALUES
  -- Trade Order BO
  (
    'c1111111-1111-4111-8111-111111111101',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '06bb774c-8666-4ab1-84eb-4f4d439ac84c',
    'trade_order',
    'Front-Office Trade Order Execution & Lifecycle Management',
    'bo:trade_order',
    true,
    jsonb_build_object(
      'bo_key', 'trade_order',
      'display_name', 'Trade Order',
      'driver_table_id', '6b267260-a5fa-549b-a80a-7ddb1c712da0',
      'driver_table_name', 'order'
    )
  ),
  -- Trade Execution BO
  (
    'c1111111-1111-4111-8111-111111111102',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '06bb774c-8666-4ab1-84eb-4f4d439ac84c',
    'trade_execution',
    'Executed Fills and Market Matching Details',
    'bo:trade_execution',
    true,
    jsonb_build_object(
      'bo_key', 'trade_execution',
      'display_name', 'Trade Execution Fill',
      'driver_table_id', 'c2df2faf-5cfd-592d-b86d-243417eb4217',
      'driver_table_name', 'execution'
    )
  ),
  -- Portfolio Position BO
  (
    'c1111111-1111-4111-8111-111111111103',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '06bb774c-8666-4ab1-84eb-4f4d439ac84c',
    'portfolio_position',
    'Front Office Portfolio Holding & Position Summary',
    'bo:portfolio_position',
    true,
    jsonb_build_object(
      'bo_key', 'portfolio_position',
      'display_name', 'Portfolio Position',
      'driver_table_id', '9ec690fb-3715-5c5b-af41-eb19794192d1',
      'driver_table_name', 'position'
    )
  ),
  -- Financial Instrument / Security BO
  (
    'c1111111-1111-4111-8111-111111111104',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '06bb774c-8666-4ab1-84eb-4f4d439ac84c',
    'financial_security',
    'Security Master & Financial Instrument Definition',
    'bo:financial_security',
    true,
    jsonb_build_object(
      'bo_key', 'financial_security',
      'display_name', 'Financial Security',
      'driver_table_id', '570cd55c-4c17-5a4a-bd95-7bda4c843672',
      'driver_table_name', 'security'
    )
  ),
  -- Trading Account BO
  (
    'c1111111-1111-4111-8111-111111111105',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '06bb774c-8666-4ab1-84eb-4f4d439ac84c',
    'trading_account',
    'Client Account & Sleeve Portfolio Registry',
    'bo:trading_account',
    true,
    jsonb_build_object(
      'bo_key', 'trading_account',
      'display_name', 'Trading Account',
      'driver_table_id', 'd8efd837-d8d3-5785-a501-eeab856775ad',
      'driver_table_name', 'account'
    )
  ),
  -- Trade Allocation BO
  (
    'c1111111-1111-4111-8111-111111111106',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '06bb774c-8666-4ab1-84eb-4f4d439ac84c',
    'trade_allocation',
    'Post-Trade Block & Account Level Allocations',
    'bo:trade_allocation',
    true,
    jsonb_build_object(
      'bo_key', 'trade_allocation',
      'display_name', 'Trade Allocation',
      'driver_table_id', '32811e9c-50d1-5ef5-ae38-d035815dda95',
      'driver_table_name', 'allocation'
    )
  )
ON CONFLICT (id) DO NOTHING;


-- 2. Create Semantic Terms for CRIMS ORM Entities
INSERT INTO catalog_node (
  id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, properties
) VALUES
  -- Trade Order Terms
  (
    'c1111111-1111-4111-8111-211111111101',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Order Quantity',
    'Target order total quantity requested',
    'semantic:orm:order_quantity',
    true,
    jsonb_build_object(
      'name', 'order_quantity',
      'title', 'Order Quantity',
      'data_type', 'decimal',
      'semantic_type', 'measure',
      'physical_mapping', jsonb_build_object('table', 'order', 'column', 'quantity'),
      'sql', 'SUM(${CUBE}.quantity)'
    )
  ),
  (
    'c1111111-1111-4111-8111-211111111102',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Order Limit Price',
    'Specified execution limit price for trade order',
    'semantic:orm:order_price',
    true,
    jsonb_build_object(
      'name', 'limit_price',
      'title', 'Limit Price',
      'data_type', 'decimal',
      'semantic_type', 'dimension',
      'physical_mapping', jsonb_build_object('table', 'order', 'column', 'price'),
      'sql', '${CUBE}.price'
    )
  ),
  -- Execution Terms
  (
    'c1111111-1111-4111-8111-211111111103',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Executed Price',
    'Fill price recorded on execution match',
    'semantic:orm:execution_price',
    true,
    jsonb_build_object(
      'name', 'exec_price',
      'title', 'Executed Price',
      'data_type', 'decimal',
      'semantic_type', 'dimension',
      'physical_mapping', jsonb_build_object('table', 'execution', 'column', 'price'),
      'sql', '${CUBE}.price'
    )
  ),
  (
    'c1111111-1111-4111-8111-211111111104',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Executed Quantity',
    'Quantity filled during execution event',
    'semantic:orm:execution_quantity',
    true,
    jsonb_build_object(
      'name', 'exec_quantity',
      'title', 'Executed Quantity',
      'data_type', 'decimal',
      'semantic_type', 'measure',
      'physical_mapping', jsonb_build_object('table', 'execution', 'column', 'quantity'),
      'sql', 'SUM(${CUBE}.quantity)'
    )
  ),
  -- Position Terms
  (
    'c1111111-1111-4111-8111-211111111105',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Position Quantity',
    'Current holding position quantity of security',
    'semantic:orm:position_quantity',
    true,
    jsonb_build_object(
      'name', 'position_quantity',
      'title', 'Position Quantity',
      'data_type', 'decimal',
      'semantic_type', 'measure',
      'physical_mapping', jsonb_build_object('table', 'position', 'column', 'quantity'),
      'sql', 'SUM(${CUBE}.quantity)'
    )
  )
ON CONFLICT (id) DO NOTHING;


-- 3. Map Semantic Terms to Table Columns via catalog_edge (MAPS_TO)
INSERT INTO catalog_edge (
  id, tenant_id, edge_type_id, source_node_id, target_node_id, relationship_type, is_active, properties, created_at, updated_at
) VALUES
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', 'c1111111-1111-4111-8111-211111111101', '6b267260-a5fa-549b-a80a-7ddb1c712da0', 'MAPS_TO', true, '{}', NOW(), NOW()),
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', 'c1111111-1111-4111-8111-211111111102', '6b267260-a5fa-549b-a80a-7ddb1c712da0', 'MAPS_TO', true, '{}', NOW(), NOW()),
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', 'c1111111-1111-4111-8111-211111111103', 'c2df2faf-5cfd-592d-b86d-243417eb4217', 'MAPS_TO', true, '{}', NOW(), NOW()),
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', 'c1111111-1111-4111-8111-211111111104', 'c2df2faf-5cfd-592d-b86d-243417eb4217', 'MAPS_TO', true, '{}', NOW(), NOW()),
  (gen_random_uuid(), '99e99e99-99e9-49e9-89e9-99e99e99e999', '87a92dd1-51fc-442c-9a59-1037ccb03d24', 'c1111111-1111-4111-8111-211111111105', '9ec690fb-3715-5c5b-af41-eb19794192d1', 'MAPS_TO', true, '{}', NOW(), NOW())
ON CONFLICT DO NOTHING;


-- 4. Establish Cross-Business Object Relationships for CRIMS ORM
INSERT INTO catalog_edge (
  id, tenant_id, edge_type_id, source_node_id, target_node_id, relationship_type, is_active, properties, created_at, updated_at
) VALUES
  -- order -> security (Trade Order belongs to Financial Security)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    '6b267260-a5fa-549b-a80a-7ddb1c712da0',
    '570cd55c-4c17-5a4a-bd95-7bda4c843672',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'security_id',
      'target_column', 'id',
      'source_bo_id', 'c1111111-1111-4111-8111-111111111101',
      'target_bo_id', 'c1111111-1111-4111-8111-111111111104'
    ),
    NOW(),
    NOW()
  ),
  -- order -> account (Trade Order belongs to Trading Account)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    '6b267260-a5fa-549b-a80a-7ddb1c712da0',
    'd8efd837-d8d3-5785-a501-eeab856775ad',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'account_id',
      'target_column', 'id',
      'source_bo_id', 'c1111111-1111-4111-8111-111111111101',
      'target_bo_id', 'c1111111-1111-4111-8111-111111111105'
    ),
    NOW(),
    NOW()
  ),
  -- execution -> order (Trade Execution belongs to Trade Order)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    'c2df2faf-5cfd-592d-b86d-243417eb4217',
    '6b267260-a5fa-549b-a80a-7ddb1c712da0',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'order_id',
      'target_column', 'id',
      'source_bo_id', 'c1111111-1111-4111-8111-111111111102',
      'target_bo_id', 'c1111111-1111-4111-8111-111111111101'
    ),
    NOW(),
    NOW()
  ),
  -- allocation -> order (Trade Allocation belongs to Trade Order)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    '32811e9c-50d1-5ef5-ae38-d035815dda95',
    '6b267260-a5fa-549b-a80a-7ddb1c712da0',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'order_id',
      'target_column', 'id',
      'source_bo_id', 'c1111111-1111-4111-8111-111111111106',
      'target_bo_id', 'c1111111-1111-4111-8111-111111111101'
    ),
    NOW(),
    NOW()
  ),
  -- position -> account (Portfolio Position belongs to Trading Account)
  (
    gen_random_uuid(),
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'd5fd8908-96ad-4ac5-b2e0-f86bc666f6bd',
    '9ec690fb-3715-5c5b-af41-eb19794192d1',
    'd8efd837-d8d3-5785-a501-eeab856775ad',
    'belongs_to',
    true,
    jsonb_build_object(
      'relationship_type', 'belongs_to',
      'source_column', 'account_id',
      'target_column', 'id',
      'source_bo_id', 'c1111111-1111-4111-8111-111111111103',
      'target_bo_id', 'c1111111-1111-4111-8111-111111111105'
    ),
    NOW(),
    NOW()
  )
ON CONFLICT DO NOTHING;


-- 5. Insert Calculated Fields for CRIMS ORM (Totals, Averages, Sums)
INSERT INTO catalog_node (
  id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, properties
) VALUES
  -- Trade Order Gross Value
  (
    'c1111111-1111-4111-8111-311111111110',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Total Order Consideration Value',
    'Calculated total nominal gross order value (quantity * price)',
    'semantic:orm:total_order_consideration',
    true,
    jsonb_build_object(
      'name', 'total_order_consideration',
      'title', 'Total Order Consideration Value',
      'data_type', 'decimal',
      'semantic_type', 'measure',
      'expression', '${CUBE.quantity} * ${CUBE.price}',
      'is_calculated', true,
      'bo_id', 'c1111111-1111-4111-8111-111111111101',
      'physical_mapping', jsonb_build_object('table', 'order', 'expression', 'quantity * price'),
      'sql', 'SUM(${CUBE.quantity} * ${CUBE.price})'
    )
  ),
  -- Executed Fill Value
  (
    'c1111111-1111-4111-8111-311111111111',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Executed Fill Consideration',
    'Total filled monetary consideration value (execution quantity * execution price)',
    'semantic:orm:executed_fill_consideration',
    true,
    jsonb_build_object(
      'name', 'executed_fill_consideration',
      'title', 'Executed Fill Consideration',
      'data_type', 'decimal',
      'semantic_type', 'measure',
      'expression', '${CUBE.quantity} * ${CUBE.price}',
      'is_calculated', true,
      'bo_id', 'c1111111-1111-4111-8111-111111111102',
      'physical_mapping', jsonb_build_object('table', 'execution', 'expression', 'quantity * price'),
      'sql', 'SUM(${CUBE.quantity} * ${CUBE.price})'
    )
  ),
  -- Average Execution Price
  (
    'c1111111-1111-4111-8111-311111111112',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    '820b942a-9c9e-4abc-acdc-84616db33098',
    'Volume Weighted Average Fill Price',
    'VWAP average fill price across executions',
    'semantic:orm:avg_fill_price',
    true,
    jsonb_build_object(
      'name', 'avg_fill_price',
      'title', 'Average Fill Price',
      'data_type', 'decimal',
      'semantic_type', 'measure',
      'expression', 'AVG(${CUBE.price})',
      'is_calculated', true,
      'bo_id', 'c1111111-1111-4111-8111-111111111102',
      'physical_mapping', jsonb_build_object('table', 'execution', 'expression', 'AVG(price)'),
      'sql', 'AVG(${CUBE.price})'
    )
  )
ON CONFLICT (id) DO NOTHING;


-- 6. Insert Validation Rules into catalog_validation_rules for CRIMS ORM BOs
INSERT INTO catalog_validation_rules (
  id, tenant_id, datasource_id, rule_name, rule_type, description, target_entity, condition_json, severity, is_active, created_at, updated_at
) VALUES
  (
    'b2222222-2222-4222-8222-222222222201',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'e0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e6',
    'Trade Order Quantity Must Be Greater Than Zero',
    'business_logic',
    'Ensures that any front-office trade order specifies a positive quantity',
    'trade_order',
    jsonb_build_object(
      'schema_version', '1',
      'authored_mode', 'designer',
      'payload', jsonb_build_object(
        'field', 'quantity',
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
    'b2222222-2222-4222-8222-222222222202',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'e0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e6',
    'Execution Fill Price Positive Check',
    'business_logic',
    'Verifies that executed match price must be strictly positive',
    'trade_execution',
    jsonb_build_object(
      'schema_version', '1',
      'authored_mode', 'designer',
      'payload', jsonb_build_object(
        'field', 'price',
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
    'b2222222-2222-4222-8222-222222222203',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    'e0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e6',
    'Large Order Size Compliance Alert',
    'business_logic',
    'Generates a compliance warning for orders exceeding block size threshold of 100,000 units',
    'trade_order',
    jsonb_build_object(
      'schema_version', '1',
      'authored_mode', 'designer',
      'payload', jsonb_build_object(
        'field', 'quantity',
        'operator', 'greater_than',
        'value', 100000
      )
    ),
    'warning',
    true,
    NOW(),
    NOW()
  )
ON CONFLICT (id) DO NOTHING;

COMMIT;
