-- Migration 036: Complete Cash Domain Semantic Graph
-- Per Semantic Design §3.4: Edges represent relationships
-- Per Whitepaper §9: Lineage & Traceability

-- ============================================
-- ADDITIONAL EDGE TYPES FOR CASH DOMAIN
-- ============================================

INSERT INTO catalog_edge_type (edge_type_name, description, source_node_type, target_node_type, properties)
VALUES 
  ('settles_to_cash', 'Transaction settlement creates Cash Ledger entry', 'business_object', 'business_object', '{"field":"transaction_id"}'),
  ('settles_from_transaction', 'Cash Ledger entry originates from Transaction', 'business_object', 'business_object', '{"field":"transaction_id"}'),
  ('rolls_up_to', 'Cash Ledger entries roll up to Cash Balance', 'business_object', 'business_object', '{"trace_table":"edm.cash_flow_trace"}'),
  ('maps_to_cash', 'Transaction maps to Cash Ledger via mapping table', 'business_object', 'business_object', '{"mapping_table":"edm.transaction_cash_mapping"}')
ON CONFLICT (edge_type_name) DO NOTHING;

-- ============================================
-- CROSS-DOMAIN EDGES (Transaction → Cash → Balance)
-- ============================================

-- Transaction → Cash Ledger (Settlement)
INSERT INTO catalog_edge (source_node_id, target_node_id, edge_type_id, properties, tenant_id)
SELECT 'bo_transaction', 'bo_cash_ledger', id, '{"field_mapping": "transaction_id", "mapping_table": "edm.transaction_cash_mapping"}', '00000000-0000-0000-0000-000000000000'
FROM catalog_edge_type WHERE edge_type_name = 'settles_to_cash'
ON CONFLICT DO NOTHING;

-- Cash Ledger → Cash Balance (Roll-Forward)
INSERT INTO catalog_edge (source_node_id, target_node_id, edge_type_id, properties, tenant_id)
SELECT 'bo_cash_ledger', 'bo_cash_balance', id, '{"field_mapping": "value_date = valuation_date", "trace_table": "edm.cash_flow_trace"}', '00000000-0000-0000-0000-000000000000'
FROM catalog_edge_type WHERE edge_type_name = 'rolls_up_to'
ON CONFLICT DO NOTHING;

-- Cash Balance → Portfolio
INSERT INTO catalog_edge (source_node_id, target_node_id, edge_type_id, properties, tenant_id)
SELECT 'bo_cash_balance', 'bo_portfolio', id, '{"field_mapping": "portfolio_id"}', '00000000-0000-0000-0000-000000000000'
FROM catalog_edge_type WHERE edge_type_name = 'held_in_portfolio'
ON CONFLICT DO NOTHING;

-- Cash Ledger → Portfolio
INSERT INTO catalog_edge (source_node_id, target_node_id, edge_type_id, properties, tenant_id)
SELECT 'bo_cash_ledger', 'bo_portfolio', id, '{"field_mapping": "portfolio_id"}', '00000000-0000-0000-0000-000000000000'
FROM catalog_edge_type WHERE edge_type_name = 'held_in_portfolio'
ON CONFLICT DO NOTHING;

-- ============================================
-- LINEAGE SUMMARY NODE (Whitepaper §9)
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('lineage_summary_cash_domain',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'business_term'),
   'CashDomainLineage',
   '{
     "description": "Complete lineage from Transaction → Cash Ledger → Cash Balance",
     "domains": ["Transaction", "Cash Ledger", "Cash Balance"],
     "edge_count": 4,
     "calculation_terms": ["ct_cash_closing_balance", "ct_cash_net_flow", "ct_cash_balance_delta"],
     "trace_tables": ["edm.cash_flow_trace", "edm.cash_gold_trace", "edm.transaction_cash_mapping"],
     "roll_forward_formula": "closing = opening + inflows - outflows + interest + fx"
   }',
   'lineage/cash_domain', '00000000-0000-0000-0000-000000000000')
ON CONFLICT DO NOTHING;
