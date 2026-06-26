-- Migration 031: Seed DQ Rules, Survivorship, Calculation Terms, Demo Data
-- Aligns with Whitepaper §7: Rules Engine uses Semantic Terms

-- ============================================
-- DQ RULES (Reference Semantic Terms, Not Columns)
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  -- Cash Ledger Required
  ('rule_cash_ledger_required', 
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'CashLedger_RequiredFields',
   '{
     "dsl": "RULE CashLedger_Required: REQUIRE CashLedger.PortfolioID, CashLedger.Currency, CashLedger.ValueDate, CashLedger.CashEventType, CashLedger.Amount",
     "severity": "ERROR",
     "blocking": true,
     "semantic_terms": ["st_portfolio_id", "st_currency", "st_value_date", "st_cash_event_type", "st_amount"]
   }',
   'rules/cash/ledger/required', '00000000-0000-0000-0000-000000000000'),

  -- Cash Ledger Amount Validity
  ('rule_cash_ledger_amount',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'CashLedger_AmountValidity',
   '{
     "dsl": "RULE CashLedger_AmountValidity: IF CashLedger.Amount = 0 THEN ERROR \"Cash ledger amount cannot be zero\"",
     "severity": "ERROR",
     "semantic_terms": ["st_amount"]
   }',
   'rules/cash/ledger/validation', '00000000-0000-0000-0000-000000000000'),

  -- Cash Ledger Sign Convention
  ('rule_cash_ledger_sign',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'CashLedger_SignConvention',
   '{
     "dsl": "RULE CashLedger_SignConvention: IF CashLedger.CashEventType IN (\"CONTRIBUTION\",\"INCOME\") AND CashLedger.Amount < 0 THEN WARNING \"Expected positive amount for inflow\"",
     "severity": "WARNING",
     "semantic_terms": ["st_cash_event_type", "st_amount"]
   }',
   'rules/cash/ledger/validation', '00000000-0000-0000-0000-000000000000'),

  -- Cash Balance Closing Consistency
  ('rule_cash_balance_closing',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'CashBalance_ClosingConsistency',
   '{
     "dsl": "RULE CashBalance_ClosingConsistency: IF ABS(CashBalance.ClosingBalance - (CashBalance.OpeningBalance + CashBalance.CashInflows - CashBalance.CashOutflows + CashBalance.InterestAccrual + CashBalance.FXEffect)) > 0.01 THEN ERROR \"Closing balance inconsistent\"",
     "severity": "ERROR",
     "tolerance": 0.01,
     "semantic_terms": ["st_closing_balance", "st_opening_balance", "st_cash_inflows", "st_cash_outflows", "st_interest_accrual", "st_fx_effect"]
   }',
   'rules/cash/balance/validation', '00000000-0000-0000-0000-000000000000')
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- SURVIVORSHIP BUNDLE (Gold Copy Engine)
-- Aligns with Position/Transaction Master Patterns
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('bundle_cash_ledger_survivorship',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'rule_bundle'),
   'CashLedger_GoldCopySurvivorship',
   '{
     "cluster_by": ["portfolio_id", "cash_account_id", "currency", "value_date", "amount", "cash_event_type", "external_reference"],
     "priority": {
       "source_system": ["AccountingSystem", "Custodian", "OMS", "TreasurySystem"]
     },
     "field_rules": {
       "amount": {"prefer_source": "AccountingSystem"},
       "value_date": {"prefer_source": "Custodian", "fallback": "AccountingSystem"},
       "status": {"max_status_priority": ["CANCELLED", "PENDING", "POSTED"]}
     }
   }',
   'bundles/cash/ledger/survivorship', '00000000-0000-0000-0000-000000000000')
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- CALCULATION TERMS (WASM/SQL Execution - Whitepaper §7)
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('ct_cash_closing_balance',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'CALCULATION_TERM'),
   'CashClosingBalance',
   '{
     "expression": "opening_balance + cash_inflows - cash_outflows + interest_accrual + fx_effect",
     "depends_on": ["st_opening_balance", "st_cash_inflows", "st_cash_outflows", "st_interest_accrual", "st_fx_effect"],
     "return_type": "numeric",
     "execution_target": "SQL"
   }',
   'calculations/cash/closing', '00000000-0000-0000-0000-000000000000'),

  ('ct_cash_net_flow',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'CALCULATION_TERM'),
   'CashNetFlow',
   '{
     "expression": "cash_inflows - cash_outflows",
     "depends_on": ["st_cash_inflows", "st_cash_outflows"],
     "return_type": "numeric",
     "execution_target": "WASM"
   }',
   'calculations/cash/net_flow', '00000000-0000-0000-0000-000000000000')
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- DEMO DATA
-- ============================================

INSERT INTO edm.cash_ledger (
    cash_ledger_id, portfolio_id, cash_account_id, currency, value_date,
    cash_event_type, amount, amount_sign, transaction_id, status, source_system, external_reference,
    tenant_id
)
VALUES 
  -- Trade Settlement (from Transaction Master)
  (gen_random_uuid(),
   (SELECT id FROM edm.portfolio_master WHERE portfolio_name = 'Demo Growth Fund' LIMIT 1),
   'CUST-USD-001', 'USD', CURRENT_DATE - INTERVAL '3 days',
   'SETTLEMENT', -17540.00, 'NEGATIVE',
   (SELECT transaction_id FROM edm.transaction_master WHERE external_reference = 'DEMO-GS-SELL' LIMIT 1),
   'POSTED', 'Custodian', 'SETTLE-001',
   '00000000-0000-0000-0000-000000000000'),

  -- Dividend Income
  (gen_random_uuid(),
   (SELECT id FROM edm.portfolio_master WHERE portfolio_name = 'Demo Growth Fund' LIMIT 1),
   'CUST-USD-001', 'USD', CURRENT_DATE - INTERVAL '2 days',
   'INCOME', 250.00, 'POSITIVE', NULL,
   'POSTED', 'AccountingSystem', 'DIV-001',
   '00000000-0000-0000-0000-000000000000'),

  -- Management Fee
  (gen_random_uuid(),
   (SELECT id FROM edm.portfolio_master WHERE portfolio_name = 'Demo Growth Fund' LIMIT 1),
   'CUST-USD-001', 'USD', CURRENT_DATE - INTERVAL '1 days',
   'FEE', -50.00, 'NEGATIVE', NULL,
   'POSTED', 'AccountingSystem', 'FEE-001',
   '00000000-0000-0000-0000-000000000000')
ON CONFLICT DO NOTHING;

-- Cash Balance (Roll-Forward Result)
INSERT INTO edm.cash_balance_master (
    cash_balance_id, portfolio_id, cash_account_id, currency, valuation_date,
    opening_balance, cash_inflows, cash_outflows, interest_accrual, fx_effect, closing_balance,
    source_system, is_closed, tenant_id
)
VALUES 
  (gen_random_uuid(),
   (SELECT id FROM edm.portfolio_master WHERE portfolio_name = 'Demo Growth Fund' LIMIT 1),
   'CUST-USD-001', 'USD', CURRENT_DATE,
   125000.00, 250.00, 17590.00, 5.00, 0.00, 107665.00,
   'AccountingSystem', false,
   '00000000-0000-0000-0000-000000000000')
ON CONFLICT DO NOTHING;
