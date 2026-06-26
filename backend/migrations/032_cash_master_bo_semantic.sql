-- Migration 032: Register Business Objects + Semantic Terms + Graph Edges
-- Aligns with Semantic Design §2: BO Fields bind semantic_term_id

-- ============================================
-- SEMANTIC TERMS (24 Terms as catalog_node)
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('st_cash_balance_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'CashBalanceID', '{"data_type":"uuid"}', 'semantic/CashBalanceID', '00000000-0000-0000-0000-000000000000'),
  ('st_cash_ledger_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'CashLedgerID', '{"data_type":"uuid"}', 'semantic/CashLedgerID', '00000000-0000-0000-0000-000000000000'),
  ('st_portfolio_id_cash', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'PortfolioID', '{"data_type":"uuid"}', 'semantic/PortfolioID_Cash', '00000000-0000-0000-0000-000000000000'),
  ('st_cash_account_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'CashAccountID', '{"data_type":"text"}', 'semantic/CashAccountID', '00000000-0000-0000-0000-000000000000'),
  ('st_currency_cash', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'Currency', '{"data_type":"text"}', 'semantic/Currency_Cash', '00000000-0000-0000-0000-000000000000'),
  ('st_valuation_date', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ValuationDate', '{"data_type":"date"}', 'semantic/ValuationDate', '00000000-0000-0000-0000-000000000000'),
  ('st_value_date', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ValueDate', '{"data_type":"date"}', 'semantic/ValueDate', '00000000-0000-0000-0000-000000000000'),
  ('st_booking_date', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'BookingDate', '{"data_type":"date"}', 'semantic/BookingDate', '00000000-0000-0000-0000-000000000000'),
  ('st_opening_balance', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'OpeningBalance', '{"data_type":"numeric"}', 'semantic/OpeningBalance', '00000000-0000-0000-0000-000000000000'),
  ('st_cash_inflows', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'CashInflows', '{"data_type":"numeric"}', 'semantic/CashInflows', '00000000-0000-0000-0000-000000000000'),
  ('st_cash_outflows', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'CashOutflows', '{"data_type":"numeric"}', 'semantic/CashOutflows', '00000000-0000-0000-0000-000000000000'),
  ('st_interest_accrual', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'InterestAccrual', '{"data_type":"numeric"}', 'semantic/InterestAccrual', '00000000-0000-0000-0000-000000000000'),
  ('st_fx_effect', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'FXEffect', '{"data_type":"numeric"}', 'semantic/FXEffect', '00000000-0000-0000-0000-000000000000'),
  ('st_closing_balance', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ClosingBalance', '{"data_type":"numeric"}', 'semantic/ClosingBalance', '00000000-0000-0000-0000-000000000000'),
  ('st_cash_event_type', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'CashEventType', '{"data_type":"text", "allowed_values":["SETTLEMENT","INCOME","FEE","FX","CONTRIBUTION","WITHDRAWAL"]}', 'semantic/CashEventType', '00000000-0000-0000-0000-000000000000'),
  ('st_cash_event_subtype', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'CashEventSubtype', '{"data_type":"text"}', 'semantic/CashEventSubtype', '00000000-0000-0000-0000-000000000000'),
  ('st_amount', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'Amount', '{"data_type":"numeric"}', 'semantic/Amount', '00000000-0000-0000-0000-000000000000'),
  ('st_amount_sign', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'AmountSign', '{"data_type":"text", "allowed_values":["POSITIVE","NEGATIVE"]}', 'semantic/AmountSign', '00000000-0000-0000-0000-000000000000'),
  ('st_transaction_id_cash', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TransactionID', '{"data_type":"uuid"}', 'semantic/TransactionID_Cash', '00000000-0000-0000-0000-000000000000'),
  ('st_security_id_cash', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'SecurityID', '{"data_type":"uuid"}', 'semantic/SecurityID_Cash', '00000000-0000-0000-0000-000000000000'),
  ('st_counterparty_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'CounterpartyID', '{"data_type":"text"}', 'semantic/CounterpartyID', '00000000-0000-0000-0000-000000000000'),
  ('st_status_cash', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'Status', '{"data_type":"text", "allowed_values":["PENDING","POSTED","CANCELLED"]}', 'semantic/Status_Cash', '00000000-0000-0000-0000-000000000000'),
  ('st_source_system_cash', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'SourceSystem', '{"data_type":"text"}', 'semantic/SourceSystem_Cash', '00000000-0000-0000-0000-000000000000'),
  ('st_external_reference_cash', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ExternalReference', '{"data_type":"text"}', 'semantic/ExternalReference_Cash', '00000000-0000-0000-0000-000000000000')
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- BUSINESS OBJECT REGISTRATION
-- ============================================

INSERT INTO public.business_objects (id, name, display_name, key, technical_name, description, category, driver_table_name, is_core, enable_history, history_mode, tenant_id)
VALUES 
  ('bo_cash_balance', 'CashBalance', 'Cash Balance', 'cash_balance', 'cash_balance', 
   'Point-in-time cash balance for a portfolio, currency, and account.', 'Cash',
   'edm.cash_balance_master', true, true, 'SCD2', 
   '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'CashLedgerEntry', 'Cash Ledger Entry', 'cash_ledger_entry', 'cash_ledger_entry',
   'Atomic cash movement affecting cash balances.', 'Cash', 'edm.cash_ledger', true, true, 'SCD2',
   '00000000-0000-0000-0000-000000000000')
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- BO FIELDS WITH SEMANTIC TERM BINDINGS (Semantic Design §6)
-- ============================================

INSERT INTO public.bo_fields (business_object_id, field_name, display_label, field_type, semantic_term_id, is_required, role, tenant_id)
VALUES 
  -- Cash Balance Fields
  ('bo_cash_balance', 'cash_balance_id', 'Cash Balance ID', 'UUID', 'st_cash_balance_id', true, 'IDENTIFIER', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_balance', 'portfolio_id', 'Portfolio', 'REFERENCE', 'st_portfolio_id_cash', true, 'FOREIGN_KEY', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_balance', 'cash_account_id', 'Cash Account', 'TEXT', 'st_cash_account_id', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_balance', 'currency', 'Currency', 'TEXT', 'st_currency_cash', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_balance', 'valuation_date', 'Valuation Date', 'DATE', 'st_valuation_date', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_balance', 'opening_balance', 'Opening Balance', 'NUMERIC', 'st_opening_balance', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_balance', 'cash_inflows', 'Cash Inflows', 'NUMERIC', 'st_cash_inflows', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_balance', 'cash_outflows', 'Cash Outflows', 'NUMERIC', 'st_cash_outflows', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_balance', 'interest_accrual', 'Interest Accrual', 'NUMERIC', 'st_interest_accrual', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_balance', 'fx_effect', 'FX Effect', 'NUMERIC', 'st_fx_effect', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_balance', 'closing_balance', 'Closing Balance', 'NUMERIC', 'st_closing_balance', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_balance', 'source_system', 'Source System', 'TEXT', 'st_source_system_cash', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),

  -- Cash Ledger Fields
  ('bo_cash_ledger', 'cash_ledger_id', 'Cash Ledger ID', 'UUID', 'st_cash_ledger_id', true, 'IDENTIFIER', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'portfolio_id', 'Portfolio', 'REFERENCE', 'st_portfolio_id_cash', true, 'FOREIGN_KEY', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'cash_account_id', 'Cash Account', 'TEXT', 'st_cash_account_id', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'currency', 'Currency', 'TEXT', 'st_currency_cash', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'value_date', 'Value Date', 'DATE', 'st_value_date', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'booking_date', 'Booking Date', 'DATE', 'st_booking_date', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'cash_event_type', 'Cash Event Type', 'TEXT', 'st_cash_event_type', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'cash_event_subtype', 'Subtype', 'TEXT', 'st_cash_event_subtype', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'amount', 'Amount', 'NUMERIC', 'st_amount', true, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'amount_sign', 'Amount Sign', 'TEXT', 'st_amount_sign', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'transaction_id', 'Transaction', 'REFERENCE', 'st_transaction_id_cash', false, 'FOREIGN_KEY', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'security_id', 'Security', 'REFERENCE', 'st_security_id_cash', false, 'FOREIGN_KEY', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'counterparty_id', 'Counterparty', 'TEXT', 'st_counterparty_id', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'status', 'Status', 'TEXT', 'st_status_cash', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'source_system', 'Source System', 'TEXT', 'st_source_system_cash', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_cash_ledger', 'external_reference', 'External Reference', 'TEXT', 'st_external_reference_cash', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000')
ON CONFLICT DO NOTHING;

-- ============================================
-- SEMANTIC GRAPH EDGES (Cross-Domain Wiring)
-- Aligns with Semantic Design §3.4: Edges represent relationships
-- ============================================

INSERT INTO catalog_edge (source_node_id, target_node_id, edge_type_id, properties, tenant_id)
VALUES 
  -- Cash Balance → Portfolio
  ('bo_cash_balance', 'bo_portfolio', 
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'held_in_portfolio'),
   '{"field_mapping": "portfolio_id"}', '00000000-0000-0000-0000-000000000000'),
  
  -- Cash Ledger → Portfolio
  ('bo_cash_ledger', 'bo_portfolio',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'held_in_portfolio'),
   '{"field_mapping": "portfolio_id"}', '00000000-0000-0000-0000-000000000000'),
  
  -- Cash Ledger → Transaction (Settlement Linkage)
  ('bo_cash_ledger', 'bo_transaction',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'settles_from_transaction'),
   '{"field_mapping": "transaction_id", "event_types": ["SETTLEMENT", "FEE", "COMMISSION"]}', '00000000-0000-0000-0000-000000000000'),
  
  -- Cash Ledger → Security (Income Linkage)
  ('bo_cash_ledger', 'bo_security',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'references'),
   '{"field_mapping": "security_id", "event_types": ["DIVIDEND", "INTEREST"]}', '00000000-0000-0000-0000-000000000000'),
  
  -- Cash Balance → FX Rate (Revaluation)
  ('bo_cash_balance', 'bo_fx_rate',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'valued_at_fx'),
   '{"field_mapping": "fx_effect", "fx_pair": "currency_to_base"}', '00000000-0000-0000-0000-000000000000'),
  
  -- Cash Ledger → Cash Balance (Roll-Forward)
  ('bo_cash_ledger', 'bo_cash_balance',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'rolls_up_to'),
   '{"field_mapping": "value_date = valuation_date", "trace_table": "edm.cash_flow_trace"}', '00000000-0000-0000-0000-000000000000')
ON CONFLICT DO NOTHING;
