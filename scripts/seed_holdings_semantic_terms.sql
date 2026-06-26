-- Holdings Semantic Terms Seeder
-- This script populates the catalog_node table with semantic terms for the holdings table
-- Following the flow: Database Metadata → Semantic Terms → Business Objects → Cube

-- Ensure semantic_term node type exists
INSERT INTO catalog_node_type (id, catalog_type_name, description)
VALUES ('820b942a-9c9e-4abc-acdc-84616db33098', 'semantic_term', 'Semantic term for governed data access')
ON CONFLICT (id) DO NOTHING;

-- Get or create default tenant
DO $$
DECLARE
    v_tenant_id UUID;
    v_semantic_type_id UUID := '820b942a-9c9e-4abc-acdc-84616db33098';
BEGIN
    -- Find existing tenant or use default
    SELECT id INTO v_tenant_id FROM tenants LIMIT 1;
    IF v_tenant_id IS NULL THEN
        v_tenant_id := '00000000-0000-0000-0000-000000000001';
    END IF;

    -- ========================================================================
    -- PHYSICAL SEMANTIC TERMS (direct column mappings)
    -- ========================================================================

    -- holding.market_value_raw - Raw market_value column
    INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, created_at, updated_at)
    VALUES (
        'a1000001-0000-0000-0000-000000000001',
        v_tenant_id,
        v_semantic_type_id,
        'holding.market_value_raw',
        'holding.market_value_raw',
        jsonb_build_object(
            'type', 'physical',
            'data_type', 'number',
            'display_name', 'Market Value (Raw)',
            'description', 'Raw market_value column; interpretation depends on holding_type',
            'physical_mapping', jsonb_build_object('table', 'holdings', 'column', 'market_value'),
            'tags', jsonb_build_array('holdings', 'physical', 'market_value'),
            'owner', 'data_team',
            'steward', 'governance_team',
            'status', 'published'
        ),
        NOW(), NOW()
    )
    ON CONFLICT (id) DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW();

    -- holding.holding_type - Holding type discriminator
    INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, created_at, updated_at)
    VALUES (
        'a1000002-0000-0000-0000-000000000001',
        v_tenant_id,
        v_semantic_type_id,
        'holding.holding_type',
        'holding.holding_type',
        jsonb_build_object(
            'type', 'physical',
            'data_type', 'string',
            'display_name', 'Holding Type',
            'description', 'Holding type discriminator: SOD (Start of Day), EOD (End of Day), SETTLED',
            'physical_mapping', jsonb_build_object('table', 'holdings', 'column', 'holding_type'),
            'tags', jsonb_build_array('holdings', 'physical', 'dimension'),
            'allowed_values', jsonb_build_array('SOD', 'EOD', 'SETTLED'),
            'owner', 'data_team',
            'steward', 'governance_team',
            'status', 'published'
        ),
        NOW(), NOW()
    )
    ON CONFLICT (id) DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW();

    -- holding.valuation_date
    INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, created_at, updated_at)
    VALUES (
        'a1000003-0000-0000-0000-000000000001',
        v_tenant_id,
        v_semantic_type_id,
        'holding.valuation_date',
        'holding.valuation_date',
        jsonb_build_object(
            'type', 'physical',
            'data_type', 'date',
            'display_name', 'Valuation Date',
            'description', 'Date for which the market value is calculated',
            'physical_mapping', jsonb_build_object('table', 'holdings', 'column', 'valuation_date'),
            'tags', jsonb_build_array('holdings', 'physical', 'time_dimension'),
            'owner', 'data_team',
            'steward', 'governance_team',
            'status', 'published'
        ),
        NOW(), NOW()
    )
    ON CONFLICT (id) DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW();

    -- holding.settlement_date
    INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, created_at, updated_at)
    VALUES (
        'a1000004-0000-0000-0000-000000000001',
        v_tenant_id,
        v_semantic_type_id,
        'holding.settlement_date',
        'holding.settlement_date',
        jsonb_build_object(
            'type', 'physical',
            'data_type', 'date',
            'display_name', 'Settlement Date',
            'description', 'Settlement date for the holding position',
            'physical_mapping', jsonb_build_object('table', 'holdings', 'column', 'settlement_date'),
            'tags', jsonb_build_array('holdings', 'physical', 'time_dimension'),
            'owner', 'data_team',
            'steward', 'governance_team',
            'status', 'published'
        ),
        NOW(), NOW()
    )
    ON CONFLICT (id) DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW();

    -- ========================================================================
    -- CALCULATED SEMANTIC TERMS (filtered access by holding_type)
    -- ========================================================================

    -- holding.market_value_sod - Start of Day market value
    INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, created_at, updated_at)
    VALUES (
        'a2000001-0000-0000-0000-000000000001',
        v_tenant_id,
        v_semantic_type_id,
        'holding.market_value_sod',
        'holding.market_value_sod',
        jsonb_build_object(
            'type', 'calculated',
            'data_type', 'number',
            'display_name', 'Market Value (Start of Day)',
            'description', 'Market value at start of day (SOD)',
            'expression', 'CASE WHEN holding_type = ''SOD'' THEN market_value ELSE NULL END',
            'sql_fragment', 'SELECT market_value FROM holdings WHERE holding_type = ''SOD'' AND {filters}',
            'lineage', jsonb_build_array('holding.market_value_raw', 'holding.holding_type'),
            'tags', jsonb_build_array('holdings', 'calculated', 'market_value', 'sod'),
            'owner', 'quant_team',
            'steward', 'governance_team',
            'status', 'published'
        ),
        NOW(), NOW()
    )
    ON CONFLICT (id) DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW();

    -- holding.market_value_eod - End of Day market value
    INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, created_at, updated_at)
    VALUES (
        'a2000002-0000-0000-0000-000000000001',
        v_tenant_id,
        v_semantic_type_id,
        'holding.market_value_eod',
        'holding.market_value_eod',
        jsonb_build_object(
            'type', 'calculated',
            'data_type', 'number',
            'display_name', 'Market Value (End of Day)',
            'description', 'Market value at end of day (EOD)',
            'expression', 'CASE WHEN holding_type = ''EOD'' THEN market_value ELSE NULL END',
            'sql_fragment', 'SELECT market_value FROM holdings WHERE holding_type = ''EOD'' AND {filters}',
            'lineage', jsonb_build_array('holding.market_value_raw', 'holding.holding_type'),
            'tags', jsonb_build_array('holdings', 'calculated', 'market_value', 'eod'),
            'owner', 'quant_team',
            'steward', 'governance_team',
            'status', 'published'
        ),
        NOW(), NOW()
    )
    ON CONFLICT (id) DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW();

    -- holding.market_value_settled - Settled market value
    INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, created_at, updated_at)
    VALUES (
        'a2000003-0000-0000-0000-000000000001',
        v_tenant_id,
        v_semantic_type_id,
        'holding.market_value_settled',
        'holding.market_value_settled',
        jsonb_build_object(
            'type', 'calculated',
            'data_type', 'number',
            'display_name', 'Market Value (Settled)',
            'description', 'Market value after settlement',
            'expression', 'CASE WHEN holding_type = ''SETTLED'' THEN market_value ELSE NULL END',
            'sql_fragment', 'SELECT market_value FROM holdings WHERE holding_type = ''SETTLED'' AND {filters}',
            'lineage', jsonb_build_array('holding.market_value_raw', 'holding.holding_type', 'holding.settlement_date'),
            'tags', jsonb_build_array('holdings', 'calculated', 'market_value', 'settled'),
            'owner', 'quant_team',
            'steward', 'governance_team',
            'status', 'published'
        ),
        NOW(), NOW()
    )
    ON CONFLICT (id) DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW();

    -- ========================================================================
    -- CANONICAL ACCESSOR with TIE-BREAKER PRECEDENCE
    -- ========================================================================

    -- holding.market_value_resolved - Canonical accessor with precedence: SETTLED > EOD > SOD
    INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, created_at, updated_at)
    VALUES (
        'a3000001-0000-0000-0000-000000000001',
        v_tenant_id,
        v_semantic_type_id,
        'holding.market_value_resolved',
        'holding.market_value_resolved',
        jsonb_build_object(
            'type', 'calculated',
            'data_type', 'number',
            'display_name', 'Market Value (Resolved)',
            'description', 'Canonical market value accessor with tie-breaker precedence: SETTLED > EOD > SOD',
            'expression', 'CASE WHEN holding_type = ''SETTLED'' THEN market_value WHEN holding_type = ''EOD'' THEN market_value WHEN holding_type = ''SOD'' THEN market_value ELSE market_value END',
            'sql_row_mode', 'SELECT id, account_id, security_id, holding_type, market_value, CASE WHEN holding_type = ''SETTLED'' THEN market_value WHEN holding_type = ''EOD'' THEN market_value WHEN holding_type = ''SOD'' THEN market_value END AS market_value_resolved FROM holdings WHERE {filters}',
            'sql_preagg_mode', 'WITH ranked AS (SELECT *, ROW_NUMBER() OVER (PARTITION BY account_id, security_id, valuation_date ORDER BY CASE holding_type WHEN ''SETTLED'' THEN 1 WHEN ''EOD'' THEN 2 WHEN ''SOD'' THEN 3 ELSE 4 END) AS rn FROM holdings WHERE {filters}) SELECT account_id, security_id, SUM(market_value) AS total_market_value FROM ranked WHERE rn = 1 GROUP BY account_id, security_id',
            'tie_breaker', jsonb_build_object(
                'strategy', 'precedence',
                'precedence', jsonb_build_array('SETTLED', 'EOD', 'SOD'),
                'description', 'Prefer SETTLED values when available, then EOD, then SOD',
                'version', 1
            ),
            'lineage', jsonb_build_array('holding.market_value_sod', 'holding.market_value_eod', 'holding.market_value_settled'),
            'tags', jsonb_build_array('holdings', 'calculated', 'market_value', 'canonical', 'tie_breaker'),
            'execution_type', 'plugin',
            'plugin_name', 'HoldingsPlugin',
            'owner', 'quant_team',
            'steward', 'governance_team',
            'status', 'published'
        ),
        NOW(), NOW()
    )
    ON CONFLICT (id) DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW();

    RAISE NOTICE 'Holdings semantic terms seeded successfully';
END $$;
