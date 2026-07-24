-- Migration 033: Complete Semantic Graph Across All 6 Domains
-- Aligns with Semantic Design §3: Semantic Graph Architecture
-- Domains: Portfolio, Security, Pricing, Position, Transaction, Cash

-- ============================================
-- CROSS-DOMAIN EDGE TYPES
-- ============================================

INSERT INTO catalog_edge_type (edge_type_name, description, source_node_type, target_node_type, properties)
VALUES 
  ('holds_cash', 'Portfolio holds Cash Balance', 'business_object', 'business_object', '{"field":"portfolio_id"}'),
  ('settles_to_cash', 'Transaction settlement creates Cash Ledger entry', 'business_object', 'business_object', '{"field":"transaction_id"}'),
  ('settles_from_transaction', 'Cash Ledger entry originates from Transaction', 'business_object', 'business_object', '{"field":"transaction_id"}'),
  ('rolls_up_to', 'Cash Ledger entries roll up to Cash Balance', 'business_object', 'business_object', '{"trace_table":"edm.cash_flow_trace"}')
ON CONFLICT (edge_type_name) DO NOTHING;

-- ============================================
-- CROSS-DOMAIN EDGES (Full Graph Wiring)
-- ============================================

INSERT INTO catalog_edge (source_node_id, target_node_id, edge_type_id, properties, tenant_id)
VALUES 
  -- Portfolio → Position
  ('bo_portfolio', 'bo_position', 
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'holds_security'),
   '{"field_mapping": "portfolio_id"}', '00000000-0000-0000-0000-000000000000'),

  -- Portfolio → Transaction
  ('bo_portfolio', 'bo_transaction',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'holds_security'),
   '{"field_mapping": "portfolio_id"}', '00000000-0000-0000-0000-000000000000'),

  -- Portfolio → Cash Balance
  ('bo_portfolio', 'bo_cash_balance',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'holds_cash'),
   '{"field_mapping": "portfolio_id"}', '00000000-0000-0000-0000-000000000000'),

  -- Portfolio → Cash Ledger
  ('bo_portfolio', 'bo_cash_ledger',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'holds_cash'),
   '{"field_mapping": "portfolio_id"}', '00000000-0000-0000-0000-000000000000'),

  -- Security → Position
  ('bo_security', 'bo_position',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'references'),
   '{"field_mapping": "security_id"}', '00000000-0000-0000-0000-000000000000'),

  -- Security → Transaction
  ('bo_security', 'bo_transaction',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'references'),
   '{"field_mapping": "security_id"}', '00000000-0000-0000-0000-000000000000'),

  -- Security → Price
  ('bo_security', 'bo_price',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'has_price'),
   '{"field_mapping": "security_id"}', '00000000-0000-0000-0000-000000000000'),

  -- Security → Cash Ledger (Income)
  ('bo_security', 'bo_cash_ledger',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'references'),
   '{"field_mapping": "security_id", "event_types": ["DIVIDEND", "INTEREST"]}', '00000000-0000-0000-0000-000000000000'),

  -- Position → Price
  ('bo_position', 'bo_price',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'uses_price'),
   '{"field_mapping": "price_id", "join_condition": "price_date = position_date"}', '00000000-0000-0000-0000-000000000000'),

  -- Position → FX Rate
  ('bo_position', 'bo_fx_rate',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'valued_at_fx'),
   '{"field_mapping": "valuation_fx_rate"}', '00000000-0000-0000-0000-000000000000'),

  -- Transaction → Position
  ('bo_transaction', 'bo_position',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'affects_position'),
   '{"field_mapping": "transaction_id", "trace_table": "edm.transaction_flow_trace"}', '00000000-0000-0000-0000-000000000000'),

  -- Transaction → Cash Ledger
  ('bo_transaction', 'bo_cash_ledger',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'settles_to_cash'),
   '{"field_mapping": "transaction_id", "event_types": ["SETTLEMENT", "FEE", "COMMISSION"]}', '00000000-0000-0000-0000-000000000000'),

  -- Cash Ledger → Cash Balance
  ('bo_cash_ledger', 'bo_cash_balance',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'rolls_up_to'),
   '{"field_mapping": "value_date = valuation_date", "trace_table": "edm.cash_flow_trace"}', '00000000-0000-0000-0000-000000000000'),

  -- Cash Balance → FX Rate
  ('bo_cash_balance', 'bo_fx_rate',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'valued_at_fx'),
   '{"field_mapping": "fx_effect"}', '00000000-0000-0000-0000-000000000000'),

  -- Price → FX Rate
  ('bo_price', 'bo_fx_rate',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'valued_at_fx'),
   '{"field_mapping": "fx_rate_to_base"}', '00000000-0000-0000-0000-000000000000')

ON CONFLICT DO NOTHING;

-- ============================================
-- SEMANTIC LINEAGE SUMMARY (for Ops Cockpit - Whitepaper §9)
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('lineage_summary_book_of_record',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'business_term'),
   'BookOfRecordLineage',
   '{
     "description": "Complete lineage from Transaction → Position → Cash → Valuation",
     "domains": ["Portfolio", "Security", "Pricing", "Position", "Transaction", "Cash"],
     "edge_count": 16,
     "calculation_terms": ["ct_market_value_local", "ct_market_value_base", "ct_unrealized_pl", "ct_cash_closing_balance"],
     "trace_tables": ["edm.transaction_flow_trace", "edm.cash_flow_trace", "edm.position_gold_trace", "edm.cash_gold_trace"]
   }',
   'lineage/book_of_record', '00000000-0000-0000-0000-000000000000')
ON CONFLICT (id) DO NOTHING;
