-- Phase 2: Seed Trade Business Object Graph + Bindings for gold_copy tenant
-- Creates ORM Trade BOs (Trade Order, Trade Execution Fill, Portfolio Position,
-- Financial Security, Trading Account) with catalog_node entries, semantic terms,
-- business_object_bindings, catalog_edges, and validation rules.

BEGIN;

DO $$
DECLARE
    gold_tid UUID;
    orm_tpd_id UUID;
    orm_ds_id UUID;
    bo_type_id UUID;
    st_type_id UUID;
    calc_type_id UUID;
    belongs_to_edge_type_id UUID;
    maps_to_edge_type_id UUID;
    bo_trade_order_id TEXT := 'bo_trade_order';
    bo_trade_exec_id TEXT := 'bo_trade_exec';
    bo_portfolio_pos_id TEXT := 'bo_portfolio_position';
    bo_fin_sec_id TEXT := 'bo_financial_security';
    bo_trading_acct_id TEXT := 'bo_trading_account';
BEGIN
    -- Resolve gold_copy tenant and ORM Suite datasource
    SELECT id INTO gold_tid FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF gold_tid IS NULL THEN
        RAISE NOTICE 'No gold_copy tenant found, skipping Trade BO graph seed';
        RETURN;
    END IF;

    -- Get the ORM Suite tenant_product_datasource.id for the gold_copy tenant
    SELECT tpd.id INTO orm_tpd_id
    FROM tenant_product_datasource tpd
    JOIN tenant_product tp ON tp.id = tpd.tenant_product_id
    WHERE tp.tenant_id = gold_tid
      AND tpd.source_name = 'orm_suite_primary'
    LIMIT 1;

    IF orm_tpd_id IS NULL THEN
        RAISE NOTICE 'ORM Suite tenant_product_datasource not found for gold_copy. Run Phase 1 migration first.';
        RETURN;
    END IF;

    -- Get alpha_datasource.id for CRIMS ORM
    SELECT id INTO orm_ds_id FROM alpha_datasource WHERE datasource_code = 'crims_orm' LIMIT 1;

    -- Get catalog_node_type IDs
    SELECT id INTO bo_type_id FROM catalog_node_type WHERE catalog_type_name = 'business_object' LIMIT 1;
    SELECT id INTO st_type_id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    SELECT id INTO calc_type_id FROM catalog_node_type WHERE catalog_type_name = 'calculation_term' LIMIT 1;

    -- Get edge type IDs
    SELECT id INTO belongs_to_edge_type_id FROM catalog_edge_type WHERE edge_type_name = 'belongs_to' LIMIT 1;
    IF belongs_to_edge_type_id IS NULL THEN
        SELECT id INTO belongs_to_edge_type_id FROM catalog_edge_types WHERE edge_type_name = 'belongs_to' LIMIT 1;
    END IF;
    SELECT id INTO maps_to_edge_type_id FROM catalog_edge_type WHERE edge_type_name = 'maps_to' LIMIT 1;
    IF maps_to_edge_type_id IS NULL THEN
        SELECT id INTO maps_to_edge_type_id FROM catalog_edge_types WHERE edge_type_name = 'maps_to' LIMIT 1;
    END IF;

    RAISE NOTICE '=== Seeding Trade BO Graph for gold_copy tenant: % ===', gold_tid;
    RAISE NOTICE '  ORM Suite tenant_product_datasource: %', orm_tpd_id;
    RAISE NOTICE '  CRIMS ORM alpha_datasource: %', orm_ds_id;

    -- =============================================
    -- 1. BUSINESS OBJECTS in catalog_node
    -- =============================================

    -- Trade Order BO
    INSERT INTO catalog_node (id, node_type_id, node_name, description, qualified_path, tenant_id, tenant_datasource_id, is_active, properties)
    VALUES (
        bo_trade_order_id, bo_type_id, 'trade_order',
        'Front-Office Trade Order — parent order with full specification and execution state',
        'bo:trade_order', gold_tid, orm_tpd_id::text, true,
        jsonb_build_object(
            'bo_key', 'trade_order', 'display_name', 'Trade Order',
            'driver_table_name', 'oms.orders',
            'category', 'Trading', 'is_core', true
        )
    )
    ON CONFLICT (id) DO UPDATE SET
        description = EXCLUDED.description,
        properties = EXCLUDED.properties,
        tenant_datasource_id = EXCLUDED.tenant_datasource_id;

    -- Trade Execution Fill BO
    INSERT INTO catalog_node (id, node_type_id, node_name, description, qualified_path, tenant_id, tenant_datasource_id, is_active, properties)
    VALUES (
        bo_trade_exec_id, bo_type_id, 'trade_execution',
        'Venue execution fill — immutable record of one market print matched on a slice',
        'bo:trade_execution', gold_tid, orm_tpd_id::text, true,
        jsonb_build_object(
            'bo_key', 'trade_execution', 'display_name', 'Trade Execution Fill',
            'driver_table_name', 'oms.execution',
            'category', 'Trading', 'is_core', true
        )
    )
    ON CONFLICT (id) DO UPDATE SET
        description = EXCLUDED.description,
        properties = EXCLUDED.properties,
        tenant_datasource_id = EXCLUDED.tenant_datasource_id;

    -- Portfolio Position BO
    INSERT INTO catalog_node (id, node_type_id, node_name, description, qualified_path, tenant_id, tenant_datasource_id, is_active, properties)
    VALUES (
        bo_portfolio_pos_id, bo_type_id, 'portfolio_position',
        'Position lot — per-account, per-security holding with cost basis and unrealised PnL',
        'bo:portfolio_position', gold_tid, orm_tpd_id::text, true,
        jsonb_build_object(
            'bo_key', 'portfolio_position', 'display_name', 'Portfolio Position',
            'driver_table_name', 'oms.position_lots',
            'category', 'Portfolio Management', 'is_core', true
        )
    )
    ON CONFLICT (id) DO UPDATE SET
        description = EXCLUDED.description,
        properties = EXCLUDED.properties,
        tenant_datasource_id = EXCLUDED.tenant_datasource_id;

    -- Financial Security BO
    INSERT INTO catalog_node (id, node_type_id, node_name, description, qualified_path, tenant_id, tenant_datasource_id, is_active, properties)
    VALUES (
        bo_fin_sec_id, bo_type_id, 'financial_security',
        'Security master — equities, fixed income, derivatives, and fund instruments',
        'bo:financial_security', gold_tid, orm_tpd_id::text, true,
        jsonb_build_object(
            'bo_key', 'financial_security', 'display_name', 'Financial Security',
            'driver_table_name', 'mds.security_master',
            'category', 'Market Data', 'is_core', true
        )
    )
    ON CONFLICT (id) DO UPDATE SET
        description = EXCLUDED.description,
        properties = EXCLUDED.properties,
        tenant_datasource_id = EXCLUDED.tenant_datasource_id;

    -- Trading Account BO
    INSERT INTO catalog_node (id, node_type_id, node_name, description, qualified_path, tenant_id, tenant_datasource_id, is_active, properties)
    VALUES (
        bo_trading_acct_id, bo_type_id, 'trading_account',
        'Investment account — cash, margin, and segregated account registry',
        'bo:trading_account', gold_tid, orm_tpd_id::text, true,
        jsonb_build_object(
            'bo_key', 'trading_account', 'display_name', 'Trading Account',
            'driver_table_name', 'mds.account',
            'category', 'Account Management', 'is_core', true
        )
    )
    ON CONFLICT (id) DO UPDATE SET
        description = EXCLUDED.description,
        properties = EXCLUDED.properties,
        tenant_datasource_id = EXCLUDED.tenant_datasource_id;

    RAISE NOTICE '  catalog_node BOs created/updated';

    -- =============================================
    -- 2. SEMANTIC TERMS for Trade domain
    -- =============================================

    INSERT INTO catalog_node (id, node_type_id, node_name, description, qualified_path, tenant_id, tenant_datasource_id, is_active, properties)
    VALUES
      -- Order-level semantic terms
      ('st_order_id', st_type_id, 'OrderID',
       'Unique order identifier (OMS-generated UUID)', 'semantic:orm:order_id',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','uuid','semantic_type','dimension','physical_mapping',jsonb_build_object('table','oms.orders','column','id'))),
      ('st_client_order_id', st_type_id, 'ClientOrderID',
       'Firm-side client order reference', 'semantic:orm:client_order_id',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','text','semantic_type','dimension','physical_mapping',jsonb_build_object('table','oms.orders','column','client_order_id'))),
      ('st_order_side', st_type_id, 'OrderSide',
       'Buy or sell side', 'semantic:orm:order_side',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','text','semantic_type','dimension','physical_mapping',jsonb_build_object('table','oms.orders','column','side'))),
      ('st_order_quantity', st_type_id, 'OrderQuantity',
       'Order target quantity', 'semantic:orm:order_quantity',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','measure','physical_mapping',jsonb_build_object('table','oms.orders','column','quantity'))),
      ('st_limit_price', st_type_id, 'LimitPrice',
       'Limit price for the order', 'semantic:orm:limit_price',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','dimension','physical_mapping',jsonb_build_object('table','oms.orders','column','limit_price'))),
      ('st_filled_qty', st_type_id, 'FilledQuantity',
       'Cumulative filled quantity', 'semantic:orm:filled_qty',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','measure','physical_mapping',jsonb_build_object('table','oms.orders','column','filled_qty'))),
      ('st_avg_fill_price', st_type_id, 'AvgFillPrice',
       'Volume-weighted average fill price', 'semantic:orm:avg_fill_price',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','measure','physical_mapping',jsonb_build_object('table','oms.orders','column','avg_fill_price'))),
      ('st_order_status', st_type_id, 'OrderStatus',
       'OMS order status', 'semantic:orm:order_status',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','text','semantic_type','dimension','physical_mapping',jsonb_build_object('table','oms.orders','column','status_id'))),
      -- Execution-level semantic terms
      ('st_exec_id', st_type_id, 'ExecutionID',
       'Unique venue execution identifier', 'semantic:orm:exec_id',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','uuid','semantic_type','dimension','physical_mapping',jsonb_build_object('table','oms.execution','column','id'))),
      ('st_venue_exec_id', st_type_id, 'VenueExecutionID',
       'Venue-side unique print ID', 'semantic:orm:venue_exec_id',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','text','semantic_type','dimension','physical_mapping',jsonb_build_object('table','oms.execution','column','venue_execution_id'))),
      ('st_exec_quantity', st_type_id, 'ExecutedQuantity',
       'Fill quantity', 'semantic:orm:exec_quantity',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','measure','physical_mapping',jsonb_build_object('table','oms.execution','column','qty'))),
      ('st_exec_price', st_type_id, 'ExecutedPrice',
       'Fill price', 'semantic:orm:exec_price',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','dimension','physical_mapping',jsonb_build_object('table','oms.execution','column','price'))),
      ('st_gross_amount', st_type_id, 'GrossAmount',
       'Gross fill monetary value', 'semantic:orm:gross_amount',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','measure','physical_mapping',jsonb_build_object('table','oms.execution','column','gross_amount'))),
      ('st_exec_fee', st_type_id, 'ExecutionFee',
       'Execution fee', 'semantic:orm:exec_fee',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','measure','physical_mapping',jsonb_build_object('table','oms.execution','column','fee'))),
      ('st_net_amount', st_type_id, 'NetAmount',
       'Net fill value (gross - fees)', 'semantic:orm:net_amount',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','measure','physical_mapping',jsonb_build_object('table','oms.execution','column','net_amount'))),
      ('st_executed_at', st_type_id, 'ExecutedAt',
       'Venue execution timestamp', 'semantic:orm:executed_at',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','timestamptz','semantic_type','dimension','physical_mapping',jsonb_build_object('table','oms.execution','column','executed_at'))),
      -- Position-level semantic terms
      ('st_position_qty', st_type_id, 'PositionQuantity',
       'Signed position quantity (+long/-short)', 'semantic:orm:position_qty',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','measure','physical_mapping',jsonb_build_object('table','oms.position_lots','column','qty_signed'))),
      ('st_remaining_qty', st_type_id, 'RemainingQuantity',
       'Unclosed position quantity', 'semantic:orm:remaining_qty',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','measure','physical_mapping',jsonb_build_object('table','oms.position_lots','column','remaining_qty'))),
      ('st_cost_basis', st_type_id, 'CostBasis',
       'Cost basis per unit', 'semantic:orm:cost_basis',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','measure','physical_mapping',jsonb_build_object('table','oms.position_lots','column','cost_basis_per_unit'))),
      ('st_unrealised_pnl', st_type_id, 'UnrealisedPnL',
       'Unrealised profit or loss', 'semantic:orm:unrealised_pnl',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','numeric','semantic_type','measure','physical_mapping',jsonb_build_object('table','oms.position_lots','column','unrealised_pnl'))),
      ('st_open_date', st_type_id, 'OpenDate',
       'Position open date', 'semantic:orm:open_date',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','date','semantic_type','dimension','physical_mapping',jsonb_build_object('table','oms.position_lots','column','open_date'))),
      -- Security-level semantic terms
      ('st_security_id', st_type_id, 'SecurityID',
       'Security master identifier', 'semantic:orm:security_id',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','uuid','semantic_type','dimension','physical_mapping',jsonb_build_object('table','mds.security_master','column','id'))),
      ('st_isin', st_type_id, 'ISIN',
       'International Securities Identification Number', 'semantic:orm:isin',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','text','semantic_type','dimension','physical_mapping',jsonb_build_object('table','mds.security_master','column','isin'))),
      ('st_ticker', st_type_id, 'Ticker',
       'Security ticker symbol', 'semantic:orm:ticker',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','text','semantic_type','dimension','physical_mapping',jsonb_build_object('table','mds.security_master','column','ticker'))),
      ('st_asset_class', st_type_id, 'AssetClass',
       'Asset class (EQUITY, FIXED_INCOME, etc.)', 'semantic:orm:asset_class',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','text','semantic_type','dimension','physical_mapping',jsonb_build_object('table','mds.security_master','column','asset_class'))),
      -- Account-level semantic terms
      ('st_account_id', st_type_id, 'AccountID',
       'Investment account identifier', 'semantic:orm:account_id',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','uuid','semantic_type','dimension','physical_mapping',jsonb_build_object('table','mds.account','column','id'))),
      ('st_account_code', st_type_id, 'AccountCode',
       'Account code/reference', 'semantic:orm:account_code',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','text','semantic_type','dimension','physical_mapping',jsonb_build_object('table','mds.account','column','code'))),
      ('st_account_type', st_type_id, 'AccountType',
       'Account type (CASH, MARGIN, SEGREGATED, HOUSE, FIRM, POOLING)', 'semantic:orm:account_type',
       gold_tid, orm_tpd_id::text, true,
       jsonb_build_object('data_type','text','semantic_type','dimension','physical_mapping',jsonb_build_object('table','mds.account','column','account_type')))
    ON CONFLICT (id) DO UPDATE SET
        properties = EXCLUDED.properties,
        tenant_datasource_id = EXCLUDED.tenant_datasource_id;

    RAISE NOTICE '  Semantic terms created/updated';

    -- =============================================
    -- 3. BUSINESS OBJECT BINDINGS
    -- Maps each BO to its physical ORM table + datasource
    -- =============================================

    -- Trade Order binding
    INSERT INTO business_object_bindings (
        binding_id, tenant_id, bo_id, binding_name, binding_mode, datasource_id,
        physical_table_name, is_primary
    ) VALUES (
        gen_random_uuid(), gold_tid, bo_trade_order_id, 'oms_orders_primary', 'OLTP_CRUD',
        orm_ds_id, 'oms.orders', true
    )
    ON CONFLICT (tenant_id, bo_id, binding_name)
    DO UPDATE SET physical_table_name = EXCLUDED.physical_table_name,
                  datasource_id = EXCLUDED.datasource_id,
                  is_primary = EXCLUDED.is_primary;

    -- Trade Execution Fill binding
    INSERT INTO business_object_bindings (
        binding_id, tenant_id, bo_id, binding_name, binding_mode, datasource_id,
        physical_table_name, is_primary
    ) VALUES (
        gen_random_uuid(), gold_tid, bo_trade_exec_id, 'oms_execution_primary', 'OLTP_CRUD',
        orm_ds_id, 'oms.execution', true
    )
    ON CONFLICT (tenant_id, bo_id, binding_name)
    DO UPDATE SET physical_table_name = EXCLUDED.physical_table_name,
                  datasource_id = EXCLUDED.datasource_id,
                  is_primary = EXCLUDED.is_primary;

    -- Portfolio Position binding
    INSERT INTO business_object_bindings (
        binding_id, tenant_id, bo_id, binding_name, binding_mode, datasource_id,
        physical_table_name, is_primary
    ) VALUES (
        gen_random_uuid(), gold_tid, bo_portfolio_pos_id, 'oms_position_lots_primary', 'OLTP_CRUD',
        orm_ds_id, 'oms.position_lots', true
    )
    ON CONFLICT (tenant_id, bo_id, binding_name)
    DO UPDATE SET physical_table_name = EXCLUDED.physical_table_name,
                  datasource_id = EXCLUDED.datasource_id,
                  is_primary = EXCLUDED.is_primary;

    -- Financial Security binding
    INSERT INTO business_object_bindings (
        binding_id, tenant_id, bo_id, binding_name, binding_mode, datasource_id,
        physical_table_name, is_primary
    ) VALUES (
        gen_random_uuid(), gold_tid, bo_fin_sec_id, 'mds_security_master_primary', 'OLTP_CRUD',
        orm_ds_id, 'mds.security_master', true
    )
    ON CONFLICT (tenant_id, bo_id, binding_name)
    DO UPDATE SET physical_table_name = EXCLUDED.physical_table_name,
                  datasource_id = EXCLUDED.datasource_id,
                  is_primary = EXCLUDED.is_primary;

    -- Trading Account binding
    INSERT INTO business_object_bindings (
        binding_id, tenant_id, bo_id, binding_name, binding_mode, datasource_id,
        physical_table_name, is_primary
    ) VALUES (
        gen_random_uuid(), gold_tid, bo_trading_acct_id, 'mds_account_primary', 'OLTP_CRUD',
        orm_ds_id, 'mds.account', true
    )
    ON CONFLICT (tenant_id, bo_id, binding_name)
    DO UPDATE SET physical_table_name = EXCLUDED.physical_table_name,
                  datasource_id = EXCLUDED.datasource_id,
                  is_primary = EXCLUDED.is_primary;

    RAISE NOTICE '  business_object_bindings created/updated';

    -- =============================================
    -- 4. CATALOG EDGES — belongs_to relationships
    -- =============================================

    -- Order → Security (order belongs to a financial security)
    INSERT INTO catalog_edge (id, tenant_id, edge_type_id, source_node_id, target_node_id, relationship_type, is_active, properties, created_at, updated_at)
    VALUES (
        gen_random_uuid(), gold_tid, belongs_to_edge_type_id,
        bo_trade_order_id, bo_fin_sec_id, 'belongs_to', true,
        jsonb_build_object('source_column','security_id','target_column','id','source_bo_id',bo_trade_order_id,'target_bo_id',bo_fin_sec_id),
        now(), now()
    )
    ON CONFLICT (source_node_id, target_node_id, edge_type_id)
    DO UPDATE SET properties = EXCLUDED.properties;

    -- Order → Account (order belongs to a trading account)
    INSERT INTO catalog_edge (id, tenant_id, edge_type_id, source_node_id, target_node_id, relationship_type, is_active, properties, created_at, updated_at)
    VALUES (
        gen_random_uuid(), gold_tid, belongs_to_edge_type_id,
        bo_trade_order_id, bo_trading_acct_id, 'belongs_to', true,
        jsonb_build_object('source_column','account_id','target_column','id','source_bo_id',bo_trade_order_id,'target_bo_id',bo_trading_acct_id),
        now(), now()
    )
    ON CONFLICT (source_node_id, target_node_id, edge_type_id)
    DO UPDATE SET properties = EXCLUDED.properties;

    -- Execution → Order (execution belongs to an order)
    INSERT INTO catalog_edge (id, tenant_id, edge_type_id, source_node_id, target_node_id, relationship_type, is_active, properties, created_at, updated_at)
    VALUES (
        gen_random_uuid(), gold_tid, belongs_to_edge_type_id,
        bo_trade_exec_id, bo_trade_order_id, 'belongs_to', true,
        jsonb_build_object('source_column','order_id','target_column','id','source_bo_id',bo_trade_exec_id,'target_bo_id',bo_trade_order_id),
        now(), now()
    )
    ON CONFLICT (source_node_id, target_node_id, edge_type_id)
    DO UPDATE SET properties = EXCLUDED.properties;

    -- Position → Account (position belongs to a trading account)
    INSERT INTO catalog_edge (id, tenant_id, edge_type_id, source_node_id, target_node_id, relationship_type, is_active, properties, created_at, updated_at)
    VALUES (
        gen_random_uuid(), gold_tid, belongs_to_edge_type_id,
        bo_portfolio_pos_id, bo_trading_acct_id, 'belongs_to', true,
        jsonb_build_object('source_column','account_id','target_column','id','source_bo_id',bo_portfolio_pos_id,'target_bo_id',bo_trading_acct_id),
        now(), now()
    )
    ON CONFLICT (source_node_id, target_node_id, edge_type_id)
    DO UPDATE SET properties = EXCLUDED.properties;

    -- Position → Security (position is in a financial security)
    INSERT INTO catalog_edge (id, tenant_id, edge_type_id, source_node_id, target_node_id, relationship_type, is_active, properties, created_at, updated_at)
    VALUES (
        gen_random_uuid(), gold_tid, belongs_to_edge_type_id,
        bo_portfolio_pos_id, bo_fin_sec_id, 'belongs_to', true,
        jsonb_build_object('source_column','security_id','target_column','id','source_bo_id',bo_portfolio_pos_id,'target_bo_id',bo_fin_sec_id),
        now(), now()
    )
    ON CONFLICT (source_node_id, target_node_id, edge_type_id)
    DO UPDATE SET properties = EXCLUDED.properties;

    RAISE NOTICE '  catalog_edges (belongs_to) created/updated';

    -- =============================================
    -- 5. VALIDATION RULES for Trade BOs
    -- =============================================

    INSERT INTO catalog_validation_rules (
        id, tenant_id, datasource_id, rule_name, rule_type, description,
        target_entity, condition_json, severity, is_active, created_at, updated_at
    )
    VALUES
      -- Order quantity > 0
      (
        gen_random_uuid(), gold_tid, orm_tpd_id,
        'Trade Order Quantity Must Be Greater Than Zero',
        'business_logic',
        'Any front-office trade order must specify a positive quantity',
        'trade_order',
        jsonb_build_object(
            'schema_version','1','authored_mode','designer',
            'payload',jsonb_build_object('field','quantity','operator','greater_than','value',0)
        ),
        'error', true, now(), now()
      ),
      -- Execution price > 0
      (
        gen_random_uuid(), gold_tid, orm_tpd_id,
        'Execution Fill Price Must Be Positive',
        'business_logic',
        'Venue fill price must be strictly positive',
        'trade_execution',
        jsonb_build_object(
            'schema_version','1','authored_mode','designer',
            'payload',jsonb_build_object('field','price','operator','greater_than','value',0)
        ),
        'error', true, now(), now()
      ),
      -- Large order compliance alert
      (
        gen_random_uuid(), gold_tid, orm_tpd_id,
        'Large Order Size Compliance Alert',
        'business_logic',
        'Triggers a compliance warning for orders exceeding block size threshold of 100,000 units',
        'trade_order',
        jsonb_build_object(
            'schema_version','1','authored_mode','designer',
            'payload',jsonb_build_object('field','quantity','operator','greater_than','value',100000)
        ),
        'warning', true, now(), now()
      )
    ON CONFLICT (id) DO UPDATE SET
        condition_json = EXCLUDED.condition_json,
        severity = EXCLUDED.severity;

    RAISE NOTICE '  catalog_validation_rules created/updated';
    RAISE NOTICE '=== Trade BO Graph Seed Complete ===';

END $$;

COMMIT;
