-- Migration 028: Seed DQ Rules, Survivorship, Calculation Terms, Demo Data

-- ============================================
-- DQ RULES (Semantic Terms, Not Columns)
-- Aligns with Whitepaper §7: Rules reference semantic terms
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  -- Tx Required Fields
  ('rule_tx_required', 
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'Transaction_RequiredFields',
   '{
     "dsl": "RULE Tx_Required: REQUIRE Transaction.TransactionID, Transaction.PortfolioID, Transaction.TradeDate, Transaction.TransactionType",
     "severity": "ERROR",
     "blocking": true
   }',
   'rules/transaction/required', '00000000-0000-0000-0000-000000000000'),

  -- Tx Quantity Validity
  ('rule_tx_quantity',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'Transaction_QuantityValidity',
   '{
     "dsl": "RULE Tx_QuantityValidity: IF Transaction.TransactionType IN (\"BUY\",\"SELL\") AND (Transaction.Quantity IS NULL OR Transaction.Quantity = 0) THEN ERROR \"Trade must have non-zero quantity\"",
     "severity": "ERROR",
     "semantic_terms": ["st_tx_quantity", "st_transaction_type"]
   }',
   'rules/transaction/validation', '00000000-0000-0000-0000-000000000000'),

  -- Tx Gross Amount Consistency
  ('rule_tx_amount_consistency',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'Transaction_AmountConsistency',
   '{
     "dsl": "RULE Tx_GrossAmountConsistency: IF Transaction.Quantity IS NOT NULL AND Transaction.Price IS NOT NULL AND ABS(Transaction.GrossAmount - (Transaction.Quantity * Transaction.Price)) > 0.01 THEN WARNING \"Gross amount inconsistent\"",
     "severity": "WARNING",
     "tolerance": 0.01
   }',
   'rules/transaction/calculations', '00000000-0000-0000-0000-000000000000'),

  -- Tx Currency Validity
  ('rule_tx_currency',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'Transaction_CurrencyValidity',
   '{
     "dsl": "RULE Tx_CurrencyValidity: IF NOT InList(Transaction.TransactionCurrency, ISO_CURRENCY_LIST) THEN ERROR \"Invalid currency\"",
     "severity": "ERROR"
   }',
   'rules/transaction/validation', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- SURVIVORSHIP BUNDLE (Gold Copy Engine)
-- Aligns with Position Master Pattern
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('bundle_tx_survivorship',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'rule_bundle'),
   'Transaction_GoldCopySurvivorship',
   '{
     "cluster_by": ["portfolio_id", "security_id", "trade_date", "gross_amount", "transaction_type", "external_reference"],
     "priority": {
       "source_system": ["AccountingSystem", "Custodian", "OMS", "TradingSystem"]
     },
     "field_rules": {
       "quantity": {"prefer_source": "AccountingSystem", "fallback": "Custodian"},
       "price": {"prefer_source": "AccountingSystem", "fallback": "OMS"},
       "gross_amount": {"prefer_source": "AccountingSystem", "fallback": "compute_quantity_price"},
       "net_amount": {"compute": "gross_amount - commission - fees - taxes"},
       "settlement_date": {"prefer_source": "Custodian", "fallback": "AccountingSystem"},
       "status": {"max_status_priority": ["CANCELLED", "PENDING", "SETTLED"]}
     }
   }',
   'bundles/transaction/survivorship', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- CALCULATION TERMS (WASM/SQL Execution)
-- Aligns with Semantic Design §2: Calculation Terms
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('ct_tx_gross_amount',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'CALCULATION_TERM'),
   'TransactionGrossAmount',
   '{
     "expression": "quantity * price",
     "depends_on": ["st_tx_quantity", "st_tx_price"],
     "return_type": "numeric",
     "execution_target": "SQL"
   }',
   'calculations/transaction/gross', '00000000-0000-0000-0000-000000000000'),

  ('ct_tx_net_amount',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'CALCULATION_TERM'),
   'TransactionNetAmount',
   '{
     "expression": "gross_amount - commission - fees - taxes",
     "depends_on": ["st_gross_amount", "st_commission", "st_fees", "st_taxes"],
     "return_type": "numeric",
     "execution_target": "WASM"
   }',
   'calculations/transaction/net', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- DEMO DATA
-- ============================================

INSERT INTO edm.transaction_master (
    transaction_id, portfolio_id, security_id, trade_date, settlement_date,
    transaction_type, quantity, price, gross_amount, commission, net_amount,
    transaction_currency, status, source_system, external_reference,
    tenant_id
)
VALUES 
  -- AAPL Buy
  (gen_random_uuid(),
   (SELECT id FROM edm.portfolio_master WHERE portfolio_name = 'Demo Growth Fund' LIMIT 1),
   (SELECT id FROM edm.security_master WHERE ticker = 'AAPL' LIMIT 1),
   CURRENT_DATE - INTERVAL '5 days', CURRENT_DATE - INTERVAL '3 days',
   'BUY', 100.00, 175.50, 17550.00, 10.00, 17540.00,
   'USD', 'SETTLED', 'Custodian', 'CUST-12345',
   '00000000-0000-0000-0000-000000000000'),

  -- GS Sell
  (gen_random_uuid(),
   (SELECT id FROM edm.portfolio_master WHERE portfolio_name = 'Demo Hedge Fund' LIMIT 1),
   (SELECT id FROM edm.security_master WHERE ticker = 'GS' LIMIT 1),
   CURRENT_DATE - INTERVAL '2 days', CURRENT_DATE + INTERVAL '1 day',
   'SELL', 50.00, 420.00, 21000.00, 15.00, 20985.00,
   'USD', 'PENDING', 'OMS', 'OMS-67890',
   '00000000-0000-0000-0000-000000000000');
