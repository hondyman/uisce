-- backend/migrations/026_position_master_bo_semantic.sql
-- Phase: Position Master BO Registration + Semantic Graph Integration

-- ============================================================================
-- PART A: BUSINESS OBJECTS
-- ============================================================================
DO $$
DECLARE
    v_tenant_id       UUID;
    v_pos_bo_id       UUID := '33330001-0000-4000-a000-000000000001';
    v_lot_bo_id       UUID := '33330001-0000-4000-a000-000000000002';
    v_cash_bo_id      UUID := '33330001-0000-4000-a000-000000000003';
    v_snap_bo_id      UUID := '33330001-0000-4000-a000-000000000004';
BEGIN
    SELECT id INTO v_tenant_id FROM public.tenants WHERE id = '00000000-0000-0000-0000-000000000001' LIMIT 1;
    IF v_tenant_id IS NULL THEN
        SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;
    END IF;

    -- ── Position (Root) ────────────────────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_pos_bo_id, v_tenant_id,
        'position', 'Position', 'Position', 'position',
        'Current holdings of a security within a portfolio. The Book of Record.',
        'account_balance_wallet', true, NULL, 'POSITIONS',
        '{"driver_table":"edm.position_master","history_mode":"SNAPSHOT","bi_temporal":true,"cluster_key":["portfolio_id","security_id","position_date"]}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET display_name = EXCLUDED.display_name,
            description  = EXCLUDED.description,
            config       = EXCLUDED.config;

    -- ── PositionLot (Subtype) ────────────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_lot_bo_id, v_tenant_id,
        'position_lot', 'PositionLot', 'Position Lot', 'position_lot',
        'Tax lot tracking for a position — FIFO, LIFO, HIFO, or Specific identification.',
        'layers', true, v_pos_bo_id, 'POSITIONS',
        '{"driver_table":"edm.position_lot_master","parent_key":"position"}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET parent_id    = EXCLUDED.parent_id,
            display_name = EXCLUDED.display_name,
            config       = EXCLUDED.config;

    -- ── CashPosition (Subtype) ───────────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_cash_bo_id, v_tenant_id,
        'cash_position', 'CashPosition', 'Cash Position', 'cash_position',
        'Cash balances by portfolio, currency, and value date.',
        'payments', true, v_pos_bo_id, 'POSITIONS',
        '{"driver_table":"edm.cash_position_master","parent_key":"position"}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET parent_id    = EXCLUDED.parent_id,
            display_name = EXCLUDED.display_name,
            config       = EXCLUDED.config;

    -- ── PositionSnapshot (Subtype) ───────────────────────────────────────
    INSERT INTO public.business_objects (
        id, tenant_id, key, name, display_name, technical_name,
        description, icon, is_core, parent_id, category, config
    ) VALUES (
        v_snap_bo_id, v_tenant_id,
        'position_snapshot', 'PositionSnapshot', 'Position Snapshot', 'position_snapshot',
        'Historical point-in-time snapshots for time-series analysis and performance attribution.',
        'history', true, v_pos_bo_id, 'POSITIONS',
        '{"driver_table":"edm.position_snapshot_master","parent_key":"position","append_only":true}'::jsonb
    ) ON CONFLICT (tenant_id, key) WHERE datasource_id IS NULL DO UPDATE
        SET parent_id    = EXCLUDED.parent_id,
            display_name = EXCLUDED.display_name,
            config       = EXCLUDED.config;

    RAISE NOTICE '✓ Position Master BOs registered: position, position_lot, cash_position, position_snapshot';
END$$;


-- ============================================================================
-- PART B: BO FIELDS
-- ============================================================================
DO $$
DECLARE
    v_tenant_id  UUID;
    v_pos_bo_id  UUID := '33330001-0000-4000-a000-000000000001';
    v_lot_bo_id  UUID := '33330001-0000-4000-a000-000000000002';
    v_cash_bo_id UUID := '33330001-0000-4000-a000-000000000003';
    v_snap_bo_id UUID := '33330001-0000-4000-a000-000000000004';
BEGIN
    SELECT id INTO v_tenant_id FROM public.tenants WHERE id = '00000000-0000-0000-0000-000000000001' LIMIT 1;
    IF v_tenant_id IS NULL THEN
        SELECT id INTO v_tenant_id FROM public.tenants LIMIT 1;
    END IF;

    -- ── Position fields (18) ──────────────────────────────────────────────
    INSERT INTO public.bo_fields (business_object_id, field_name, field_type, display_label, semantic_term_id, is_required, sort_order)
    VALUES
        (v_pos_bo_id, 'position_id',         'UUID',    'Position ID',               'st_position_id',         true,  1),
        (v_pos_bo_id, 'portfolio_id',        'UUID',    'Portfolio ID',              'st_portfolio_id',        true,  2),
        (v_pos_bo_id, 'security_id',         'UUID',    'Security ID',               'st_security_id',         true,  3),
        (v_pos_bo_id, 'position_date',       'DATE',    'Position Date',             'st_position_date',       true,  4),
        (v_pos_bo_id, 'position_quantity',   'NUMERIC', 'Quantity',                  'st_position_quantity',   true,  5),
        (v_pos_bo_id, 'position_side',       'TEXT',    'Side',                      'st_position_side',       false, 6),
        (v_pos_bo_id, 'market_value_local',  'NUMERIC', 'Market Value (Local)',       'st_market_value_local',  false, 7),
        (v_pos_bo_id, 'market_value_base',   'NUMERIC', 'Market Value (Base)',        'st_market_value_base',   false, 8),
        (v_pos_bo_id, 'cost_basis_local',    'NUMERIC', 'Cost Basis (Local)',         'st_cost_basis_local',    false, 9),
        (v_pos_bo_id, 'unrealized_pl_local', 'NUMERIC', 'Unrealized P&L (Local)',    'st_unrealized_pl_local', false, 10),
        (v_pos_bo_id, 'unrealized_pl_pct',   'NUMERIC', 'Unrealized P&L (%)',        'st_unrealized_pl_pct',   false, 11),
        (v_pos_bo_id, 'position_weight_pct', 'NUMERIC', 'Weight (%)',                'st_position_weight_pct', false, 12),
        (v_pos_bo_id, 'position_currency',   'TEXT',    'Currency',                  'st_position_currency',   true,  13),
        (v_pos_bo_id, 'valuation_fx_rate',   'NUMERIC', 'Valuation FX Rate',         'st_valuation_fx_rate',   false, 14),
        (v_pos_bo_id, 'position_source',     'TEXT',    'Source',                    'st_position_source',     true,  15),
        (v_pos_bo_id, 'position_confidence', 'NUMERIC', 'Confidence Score',          'st_position_confidence', false, 16),
        (v_pos_bo_id, 'is_reconciled',       'BOOLEAN', 'Is Reconciled',             'st_is_reconciled',       false, 17),
        (v_pos_bo_id, 'reconciliation_diff', 'NUMERIC', 'Reconciliation Difference', 'st_reconciliation_diff', false, 18)
    ON CONFLICT (business_object_id, field_name) DO UPDATE
        SET display_label    = EXCLUDED.display_label,
            semantic_term_id = EXCLUDED.semantic_term_id;

    -- ── PositionLot fields (10) ───────────────────────────────────────────
    INSERT INTO public.bo_fields (business_object_id, field_name, field_type, display_label, semantic_term_id, is_required, sort_order)
    VALUES
        (v_lot_bo_id, 'lot_id',           'UUID',    'Lot ID',            'st_lot_id',           true,  1),
        (v_lot_bo_id, 'acquisition_date', 'DATE',    'Acquisition Date',  'st_acquisition_date', true,  2),
        (v_lot_bo_id, 'settlement_date',  'DATE',    'Settlement Date',   'st_settlement_date',  false, 3),
        (v_lot_bo_id, 'lot_quantity',     'NUMERIC', 'Lot Quantity',      'st_lot_quantity',     true,  4),
        (v_lot_bo_id, 'cost_per_unit',    'NUMERIC', 'Cost Per Unit',     'st_cost_per_unit',    true,  5),
        (v_lot_bo_id, 'total_cost_basis', 'NUMERIC', 'Total Cost Basis',  'st_total_cost_basis', true,  6),
        (v_lot_bo_id, 'lot_method',       'TEXT',    'Lot Method',        'st_lot_method',       true,  7),
        (v_lot_bo_id, 'is_closed',        'BOOLEAN', 'Is Closed',         'st_is_lot_closed',    false, 8),
        (v_lot_bo_id, 'closed_date',      'DATE',    'Closed Date',       'st_closed_date',      false, 9),
        (v_lot_bo_id, 'realized_pl',      'NUMERIC', 'Realized P&L',      'st_realized_pl',      false, 10)
    ON CONFLICT (business_object_id, field_name) DO UPDATE
        SET display_label    = EXCLUDED.display_label,
            semantic_term_id = EXCLUDED.semantic_term_id;

    -- ── CashPosition fields (8) ──────────────────────────────────────────
    INSERT INTO public.bo_fields (business_object_id, field_name, field_type, display_label, semantic_term_id, is_required, sort_order)
    VALUES
        (v_cash_bo_id, 'cash_position_id',    'UUID',    'Cash Position ID',     'st_cash_position_id',    true,  1),
        (v_cash_bo_id, 'cash_currency',        'TEXT',    'Currency',             'st_cash_currency',        true,  2),
        (v_cash_bo_id, 'account_id',           'UUID',    'Account ID',           'st_cash_account_id',      false, 3),
        (v_cash_bo_id, 'value_date',           'DATE',    'Value Date',           'st_cash_value_date',      true,  4),
        (v_cash_bo_id, 'balance_amount',       'NUMERIC', 'Balance Amount',       'st_cash_balance_amount',  true,  5),
        (v_cash_bo_id, 'available_balance',    'NUMERIC', 'Available Balance',    'st_available_balance',    false, 6),
        (v_cash_bo_id, 'pending_settlements',  'JSON',    'Pending Settlements',  'st_pending_settlements',  false, 7),
        (v_cash_bo_id, 'interest_accrued',     'NUMERIC', 'Interest Accrued',     'st_interest_accrued',     false, 8)
    ON CONFLICT (business_object_id, field_name) DO UPDATE
        SET display_label    = EXCLUDED.display_label,
            semantic_term_id = EXCLUDED.semantic_term_id;

    -- ── PositionSnapshot fields (5) ──────────────────────────────────────
    INSERT INTO public.bo_fields (business_object_id, field_name, field_type, display_label, semantic_term_id, is_required, sort_order)
    VALUES
        (v_snap_bo_id, 'snapshot_id',           'UUID',    'Snapshot ID',           'st_snapshot_id',           true,  1),
        (v_snap_bo_id, 'snapshot_date',         'DATE',    'Snapshot Date',         'st_snapshot_date',         true,  2),
        (v_snap_bo_id, 'snapshot_quantity',     'NUMERIC', 'Snapshot Quantity',     'st_snapshot_quantity',     false, 3),
        (v_snap_bo_id, 'snapshot_market_value', 'NUMERIC', 'Snapshot Market Value', 'st_snapshot_market_value', false, 4),
        (v_snap_bo_id, 'snapshot_price_used',   'NUMERIC', 'Snapshot Price Used',   'st_snapshot_price_used',   false, 5)
    ON CONFLICT (business_object_id, field_name) DO UPDATE
        SET display_label    = EXCLUDED.display_label,
            semantic_term_id = EXCLUDED.semantic_term_id;

    RAISE NOTICE '✓ Position Master BO fields registered: 18 + 10 + 8 + 5 = 41 total';
END$$;


-- ============================================================================
-- PART C: SEMANTIC TERMS (41 total)
-- ============================================================================
DO $$
DECLARE
    v_st_type_id UUID;
BEGIN
    SELECT id INTO v_st_type_id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    IF v_st_type_id IS NULL THEN
        RAISE NOTICE 'semantic_term node type not found — skipping';
        RETURN;
    END IF;

    INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id) VALUES
        -- Position terms
        ('st_position_id',         v_st_type_id, 'PositionID',             '{"data_type":"uuid"}',    'semantic/PositionID',            '00000000-0000-0000-0000-000000000000'),
        ('st_portfolio_id',        v_st_type_id, 'PortfolioID',            '{"data_type":"uuid"}',    'semantic/PortfolioID',           '00000000-0000-0000-0000-000000000000'),
        ('st_position_date',       v_st_type_id, 'PositionDate',           '{"data_type":"date"}',    'semantic/PositionDate',          '00000000-0000-0000-0000-000000000000'),
        ('st_position_quantity',   v_st_type_id, 'PositionQuantity',       '{"data_type":"numeric"}', 'semantic/PositionQuantity',      '00000000-0000-0000-0000-000000000000'),
        ('st_position_side',       v_st_type_id, 'PositionSide',           '{"data_type":"text"}',    'semantic/PositionSide',          '00000000-0000-0000-0000-000000000000'),
        ('st_market_value_local',  v_st_type_id, 'MarketValueLocal',       '{"data_type":"numeric"}', 'semantic/MarketValueLocal',      '00000000-0000-0000-0000-000000000000'),
        ('st_market_value_base',   v_st_type_id, 'MarketValueBase',        '{"data_type":"numeric"}', 'semantic/MarketValueBase',       '00000000-0000-0000-0000-000000000000'),
        ('st_cost_basis_local',    v_st_type_id, 'CostBasisLocal',         '{"data_type":"numeric"}', 'semantic/CostBasisLocal',        '00000000-0000-0000-0000-000000000000'),
        ('st_unrealized_pl_local', v_st_type_id, 'UnrealizedPLLocal',      '{"data_type":"numeric"}', 'semantic/UnrealizedPLLocal',     '00000000-0000-0000-0000-000000000000'),
        ('st_unrealized_pl_pct',   v_st_type_id, 'UnrealizedPLPct',        '{"data_type":"numeric"}', 'semantic/UnrealizedPLPct',       '00000000-0000-0000-0000-000000000000'),
        ('st_position_weight_pct', v_st_type_id, 'PositionWeightPct',      '{"data_type":"numeric"}', 'semantic/PositionWeightPct',     '00000000-0000-0000-0000-000000000000'),
        ('st_position_currency',   v_st_type_id, 'PositionCurrency',       '{"data_type":"text"}',    'semantic/PositionCurrency',      '00000000-0000-0000-0000-000000000000'),
        ('st_valuation_fx_rate',   v_st_type_id, 'ValuationFXRate',        '{"data_type":"numeric"}', 'semantic/ValuationFXRate',       '00000000-0000-0000-0000-000000000000'),
        ('st_position_source',     v_st_type_id, 'PositionSource',         '{"data_type":"text"}',    'semantic/PositionSource',        '00000000-0000-0000-0000-000000000000'),
        ('st_position_confidence', v_st_type_id, 'PositionConfidenceScore','{"data_type":"numeric"}', 'semantic/PositionConfidenceScore','00000000-0000-0000-0000-000000000000'),
        ('st_is_reconciled',       v_st_type_id, 'IsReconciled',           '{"data_type":"boolean"}', 'semantic/IsReconciled',          '00000000-0000-0000-0000-000000000000'),
        ('st_reconciliation_diff', v_st_type_id, 'ReconciliationDiff',     '{"data_type":"numeric"}', 'semantic/ReconciliationDiff',    '00000000-0000-0000-0000-000000000000'),
        -- Lot terms
        ('st_lot_id',           v_st_type_id, 'LotID',          '{"data_type":"uuid"}',    'semantic/LotID',         '00000000-0000-0000-0000-000000000000'),
        ('st_acquisition_date', v_st_type_id, 'AcquisitionDate','{"data_type":"date"}',    'semantic/AcquisitionDate','00000000-0000-0000-0000-000000000000'),
        ('st_settlement_date',  v_st_type_id, 'SettlementDate', '{"data_type":"date"}',    'semantic/SettlementDate', '00000000-0000-0000-0000-000000000000'),
        ('st_lot_quantity',     v_st_type_id, 'LotQuantity',    '{"data_type":"numeric"}', 'semantic/LotQuantity',   '00000000-0000-0000-0000-000000000000'),
        ('st_cost_per_unit',    v_st_type_id, 'CostPerUnit',    '{"data_type":"numeric"}', 'semantic/CostPerUnit',   '00000000-0000-0000-0000-000000000000'),
        ('st_total_cost_basis', v_st_type_id, 'TotalCostBasis', '{"data_type":"numeric"}', 'semantic/TotalCostBasis','00000000-0000-0000-0000-000000000000'),
        ('st_lot_method',       v_st_type_id, 'LotMethod',      '{"data_type":"text"}',    'semantic/LotMethod',     '00000000-0000-0000-0000-000000000000'),
        ('st_is_lot_closed',    v_st_type_id, 'IsLotClosed',    '{"data_type":"boolean"}', 'semantic/IsLotClosed',   '00000000-0000-0000-0000-000000000000'),
        ('st_closed_date',      v_st_type_id, 'ClosedDate',     '{"data_type":"date"}',    'semantic/ClosedDate',    '00000000-0000-0000-0000-000000000000'),
        ('st_realized_pl',      v_st_type_id, 'RealizedPL',     '{"data_type":"numeric"}', 'semantic/RealizedPL',    '00000000-0000-0000-0000-000000000000'),
        -- Cash terms
        ('st_cash_position_id',   v_st_type_id, 'CashPositionID',     '{"data_type":"uuid"}',    'semantic/CashPositionID',    '00000000-0000-0000-0000-000000000000'),
        ('st_cash_currency',      v_st_type_id, 'CashCurrency',       '{"data_type":"text"}',    'semantic/CashCurrency',      '00000000-0000-0000-0000-000000000000'),
        ('st_cash_account_id',    v_st_type_id, 'CashAccountID',      '{"data_type":"uuid"}',    'semantic/CashAccountID',     '00000000-0000-0000-0000-000000000000'),
        ('st_cash_value_date',    v_st_type_id, 'CashValueDate',      '{"data_type":"date"}',    'semantic/CashValueDate',     '00000000-0000-0000-0000-000000000000'),
        ('st_cash_balance_amount',v_st_type_id, 'CashBalanceAmount',  '{"data_type":"numeric"}', 'semantic/CashBalanceAmount', '00000000-0000-0000-0000-000000000000'),
        ('st_available_balance',  v_st_type_id, 'AvailableBalance',   '{"data_type":"numeric"}', 'semantic/AvailableBalance',  '00000000-0000-0000-0000-000000000000'),
        ('st_pending_settlements',v_st_type_id, 'PendingSettlements', '{"data_type":"json"}',    'semantic/PendingSettlements','00000000-0000-0000-0000-000000000000'),
        ('st_interest_accrued',   v_st_type_id, 'InterestAccrued',    '{"data_type":"numeric"}', 'semantic/InterestAccrued',   '00000000-0000-0000-0000-000000000000'),
        -- Snapshot terms
        ('st_snapshot_id',           v_st_type_id, 'SnapshotID',          '{"data_type":"uuid"}',    'semantic/SnapshotID',          '00000000-0000-0000-0000-000000000000'),
        ('st_snapshot_date',         v_st_type_id, 'SnapshotDate',        '{"data_type":"date"}',    'semantic/SnapshotDate',        '00000000-0000-0000-0000-000000000000'),
        ('st_snapshot_quantity',     v_st_type_id, 'SnapshotQuantity',    '{"data_type":"numeric"}', 'semantic/SnapshotQuantity',    '00000000-0000-0000-0000-000000000000'),
        ('st_snapshot_market_value', v_st_type_id, 'SnapshotMarketValue', '{"data_type":"numeric"}', 'semantic/SnapshotMarketValue', '00000000-0000-0000-0000-000000000000'),
        ('st_snapshot_price_used',   v_st_type_id, 'SnapshotPriceUsed',   '{"data_type":"numeric"}', 'semantic/SnapshotPriceUsed',   '00000000-0000-0000-0000-000000000000')
    ON CONFLICT (id) DO UPDATE
        SET node_name  = EXCLUDED.node_name,
            properties = EXCLUDED.properties;

    RAISE NOTICE '✓ 41 Position Master semantic terms seeded to catalog_node';
END$$;


-- ============================================================================
-- PART D: SEMANTIC EDGE TYPES
-- ============================================================================
DO $$
BEGIN
    INSERT INTO catalog_edge_type (id, type_name, description, source_type_id, target_type_id)
    SELECT
        gen_random_uuid(), e.type_name, e.description,
        (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'business_object' LIMIT 1),
        (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'business_object' LIMIT 1)
    FROM (VALUES
        ('uses_price',         'Links a Position to the Price used for its valuation'),
        ('valued_at_fx',       'Links a Position to the FXRate used for base currency conversion'),
        ('held_in_portfolio',  'Links a Position to its parent Portfolio'),
        ('has_lots',           'Links a Position to its constituent TaxLots'),
        ('has_cash',           'Links a Portfolio to its CashPositions')
    ) AS e(type_name, description)
    ON CONFLICT (type_name) DO NOTHING;

    RAISE NOTICE '✓ Position Master edge types registered: uses_price, valued_at_fx, held_in_portfolio, has_lots, has_cash';
END$$;
