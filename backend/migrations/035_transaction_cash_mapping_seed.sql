-- Migration 035: Transaction → Cash Ledger Auto-Mapping Rules
-- Per Whitepaper §7: Semantic Execution Fabric

-- ============================================
-- TRANSACTION → CASH LEDGER MAPPING RULES
-- Auto-generates cash ledger entries from settled transactions
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('rule_tx_cash_settlement',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'Transaction_CashSettlementMapping',
   '{
     "dsl": "RULE Tx_CashSettlement: IF Transaction.Status = \"SETTLED\" AND Transaction.TransactionType IN (\"BUY\", \"SELL\") THEN CREATE CashLedgerEntry(amount = Transaction.NetAmount, event_type = \"SETTLEMENT\", value_date = Transaction.SettlementDate)",
     "severity": "INFO",
     "auto_execute": true,
     "mapping_table": "edm.transaction_cash_mapping"
   }',
   'rules/transaction/cash_mapping', '00000000-0000-0000-0000-000000000000') ON CONFLICT DO NOTHING;

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES
  ('rule_tx_cash_commission',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'Transaction_CashCommissionMapping',
   '{
     "dsl": "RULE Tx_CashCommission: IF Transaction.Commission IS NOT NULL AND Transaction.Commission > 0 THEN CREATE CashLedgerEntry(amount = -Transaction.Commission, event_type = \"FEE\", event_subtype = \"COMMISSION\")",
     "severity": "INFO",
     "auto_execute": true,
     "mapping_table": "edm.transaction_cash_mapping"
   }',
   'rules/transaction/cash_mapping', '00000000-0000-0000-0000-000000000000') ON CONFLICT DO NOTHING;

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES
  ('rule_tx_cash_tax',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'Transaction_CashTaxMapping',
   '{
     "dsl": "RULE Tx_CashTax: IF Transaction.Taxes IS NOT NULL AND Transaction.Taxes > 0 THEN CREATE CashLedgerEntry(amount = -Transaction.Taxes, event_type = \"FEE\", event_subtype = \"TAX\")",
     "severity": "INFO",
     "auto_execute": true,
     "mapping_table": "edm.transaction_cash_mapping"
   }',
   'rules/transaction/cash_mapping', '00000000-0000-0000-0000-000000000000') ON CONFLICT DO NOTHING;

-- ============================================
-- DEMO: Transaction → Cash Ledger Mappings
-- ============================================

INSERT INTO edm.transaction_cash_mapping (
    mapping_id, transaction_id, cash_ledger_id, mapping_type, amount, currency, value_date, tenant_id
)
SELECT 
   gen_random_uuid(),
   tm.transaction_id,
   cl.cash_ledger_id,
   'SETTLEMENT', -17540.00, 'USD', CURRENT_DATE - INTERVAL '3 days',
   '00000000-0000-0000-0000-000000000000'
FROM edm.transaction_master tm
CROSS JOIN edm.cash_ledger cl 
WHERE tm.external_reference = 'CUST-12345' AND cl.external_reference = 'SETTLE-001'
ON CONFLICT DO NOTHING;
