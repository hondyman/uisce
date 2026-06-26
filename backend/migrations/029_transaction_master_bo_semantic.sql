-- Migration 029: Register Business Objects + Semantic Terms + Graph Edges

-- ============================================
-- SEMANTIC TERMS (26 Terms as catalog_node)
-- Aligns with Semantic Design §4: Semantic Terms are Nodes
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('st_transaction_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TransactionID', '{"data_type":"uuid"}', 'semantic/TransactionID', '00000000-0000-0000-0000-000000000000'),
  ('st_portfolio_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'PortfolioID', '{"data_type":"uuid"}', 'semantic/PortfolioID', '00000000-0000-0000-0000-000000000000'),
  ('st_security_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'SecurityID', '{"data_type":"uuid"}', 'semantic/SecurityID', '00000000-0000-0000-0000-000000000000'),
  ('st_trade_date', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TradeDate', '{"data_type":"date"}', 'semantic/TradeDate', '00000000-0000-0000-0000-000000000000'),
  ('st_settlement_date', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'SettlementDate', '{"data_type":"date"}', 'semantic/SettlementDate', '00000000-0000-0000-0000-000000000000'),
  ('st_booking_date', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'BookingDate', '{"data_type":"date"}', 'semantic/BookingDate', '00000000-0000-0000-0000-000000000000'),
  ('st_transaction_type', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TransactionType', '{"data_type":"text", "allowed_values":["BUY","SELL","SHORT","COVER","DIVIDEND","FEE"]}', 'semantic/TransactionType', '00000000-0000-0000-0000-000000000000'),
  ('st_transaction_subtype', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TransactionSubtype', '{"data_type":"text"}', 'semantic/TransactionSubtype', '00000000-0000-0000-0000-000000000000'),
  ('st_tx_quantity', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TransactionQuantity', '{"data_type":"numeric"}', 'semantic/TransactionQuantity', '00000000-0000-0000-0000-000000000000'),
  ('st_tx_price', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TransactionPrice', '{"data_type":"numeric"}', 'semantic/TransactionPrice', '00000000-0000-0000-0000-000000000000'),
  ('st_gross_amount', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'GrossAmount', '{"data_type":"numeric"}', 'semantic/GrossAmount', '00000000-0000-0000-0000-000000000000'),
  ('st_net_amount', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'NetAmount', '{"data_type":"numeric"}', 'semantic/NetAmount', '00000000-0000-0000-0000-000000000000'),
  ('st_commission', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'Commission', '{"data_type":"numeric"}', 'semantic/Commission', '00000000-0000-0000-0000-000000000000'),
  ('st_fees', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'Fees', '{"data_type":"numeric"}', 'semantic/Fees', '00000000-0000-0000-0000-000000000000'),
  ('st_taxes', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'Taxes', '{"data_type":"numeric"}', 'semantic/Taxes', '00000000-0000-0000-0000-000000000000'),
  ('st_tx_accrued_interest', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TransactionAccruedInterest', '{"data_type":"numeric"}', 'semantic/TransactionAccruedInterest', '00000000-0000-0000-0000-000000000000'),
  ('st_transaction_currency', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TransactionCurrency', '{"data_type":"text"}', 'semantic/TransactionCurrency', '00000000-0000-0000-0000-000000000000'),
  ('st_settlement_currency', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'SettlementCurrency', '{"data_type":"text"}', 'semantic/SettlementCurrency', '00000000-0000-0000-0000-000000000000'),
  ('st_fx_rate', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'FXRate', '{"data_type":"numeric"}', 'semantic/FXRate', '00000000-0000-0000-0000-000000000000'),
  ('st_counterparty_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'CounterpartyID', '{"data_type":"text"}', 'semantic/CounterpartyID', '00000000-0000-0000-0000-000000000000'),
  ('st_broker_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'BrokerID', '{"data_type":"text"}', 'semantic/BrokerID', '00000000-0000-0000-0000-000000000000'),
  ('st_custody_account_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'CustodyAccountID', '{"data_type":"text"}', 'semantic/CustodyAccountID', '00000000-0000-0000-0000-000000000000'),
  ('st_corporate_action_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'CorporateActionID', '{"data_type":"text"}', 'semantic/CorporateActionID', '00000000-0000-0000-0000-000000000000'),
  ('st_tx_status', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TransactionStatus', '{"data_type":"text", "allowed_values":["PENDING","SETTLED","CANCELLED"]}', 'semantic/TransactionStatus', '00000000-0000-0000-0000-000000000000'),
  ('st_source_system', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'SourceSystem', '{"data_type":"text"}', 'semantic/SourceSystem', '00000000-0000-0000-0000-000000000000'),
  ('st_external_reference', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ExternalReference', '{"data_type":"text"}', 'semantic/ExternalReference', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- BUSINESS OBJECT REGISTRATION
-- ============================================

INSERT INTO public.business_objects (
    id, name, display_name, key, technical_name, description, category,
    driver_table_name, is_core, enable_history, history_mode, tenant_id
)
VALUES 
  ('bo_transaction', 'Transaction', 'Transaction', 'transaction', 'transaction', 
   'Atomic economic event affecting positions, cash, and PnL.', 'Transactions',
   'edm.transaction_master', true, true, 'SCD2', 
   '00000000-0000-0000-0000-000000000000');

-- ============================================
-- BO FIELDS WITH SEMANTIC TERM BINDINGS
-- Aligns with Semantic Design §6: BO Fields bind semantic_term_id
-- ============================================

INSERT INTO public.bo_fields (business_object_id, field_name, display_label, field_type, semantic_term_id, is_required, role, tenant_id)
VALUES 
  ('bo_transaction', 'transaction_id', 'Transaction ID', 'UUID', 'st_transaction_id', true, 'IDENTIFIER', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'portfolio_id', 'Portfolio', 'REFERENCE', 'st_portfolio_id', true, 'FOREIGN_KEY', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'security_id', 'Security', 'REFERENCE', 'st_security_id', false, 'FOREIGN_KEY', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'trade_date', 'Trade Date', 'DATE', 'st_trade_date', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'settlement_date', 'Settlement Date', 'DATE', 'st_settlement_date', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'booking_date', 'Booking Date', 'DATE', 'st_booking_date', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'transaction_type', 'Transaction Type', 'TEXT', 'st_transaction_type', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'transaction_subtype', 'Subtype', 'TEXT', 'st_transaction_subtype', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'quantity', 'Quantity', 'NUMERIC', 'st_tx_quantity', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'price', 'Price', 'NUMERIC', 'st_tx_price', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'gross_amount', 'Gross Amount', 'NUMERIC', 'st_gross_amount', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'net_amount', 'Net Amount', 'NUMERIC', 'st_net_amount', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'commission', 'Commission', 'NUMERIC', 'st_commission', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'fees', 'Fees', 'NUMERIC', 'st_fees', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'taxes', 'Taxes', 'NUMERIC', 'st_taxes', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'accrued_interest', 'Accrued Interest', 'NUMERIC', 'st_tx_accrued_interest', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'transaction_currency', 'Transaction Currency', 'TEXT', 'st_transaction_currency', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'settlement_currency', 'Settlement Currency', 'TEXT', 'st_settlement_currency', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'fx_rate', 'FX Rate', 'NUMERIC', 'st_fx_rate', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'counterparty_id', 'Counterparty', 'TEXT', 'st_counterparty_id', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'broker_id', 'Broker', 'TEXT', 'st_broker_id', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'custody_account_id', 'Custody Account', 'TEXT', 'st_custody_account_id', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'corporate_action_id', 'Corporate Action', 'TEXT', 'st_corporate_action_id', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'status', 'Status', 'TEXT', 'st_tx_status', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'source_system', 'Source System', 'TEXT', 'st_source_system', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_transaction', 'external_reference', 'External Reference', 'TEXT', 'st_external_reference', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- SEMANTIC GRAPH EDGES
-- Aligns with Semantic Design §3.4: Edges represent relationships
-- ============================================

INSERT INTO catalog_edge (source_node_id, target_node_id, edge_type_id, properties, tenant_id)
VALUES 
  -- Transaction → Portfolio
  ('bo_transaction', 'bo_portfolio', 
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'held_in_portfolio'),
   '{"field_mapping": "portfolio_id"}', '00000000-0000-0000-0000-000000000000'),
  
  -- Transaction → Security
  ('bo_transaction', 'bo_security',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'references'),
   '{"field_mapping": "security_id"}', '00000000-0000-0000-0000-000000000000'),
  
  -- Transaction → Position (Impact)
  ('bo_transaction', 'bo_position',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'affects_position'),
   '{"field_mapping": "transaction_id", "trace_table": "edm.transaction_flow_trace"}', '00000000-0000-0000-0000-000000000000'),
  
  -- Transaction → Price (Execution)
  ('bo_transaction', 'bo_price',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'uses_price'),
   '{"field_mapping": "price", "join_condition": "trade_date = price_date"}', '00000000-0000-0000-0000-000000000000');
