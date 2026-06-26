-- Migration 039: Seed DQ Rules, Calculation Terms, Demo Compliance Rules
-- Aligns with Whitepaper §7: Rules reference semantic terms

-- ============================================
-- DQ RULES (Reference Semantic Terms)
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  -- Compliance Rule Required
  ('rule_compliance_rule_required', 
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'ComplianceRule_RequiredFields',
   '{
     "dsl": "RULE ComplianceRule_Required: REQUIRE ComplianceRule.RuleCode, ComplianceRule.RuleName, ComplianceRule.Expression, ComplianceRule.EffectiveFrom",
     "severity": "ERROR",
     "blocking": true,
     "semantic_terms": ["st_rule_code", "st_rule_name", "st_rule_expression", "st_effective_from"]
   }',
   'rules/compliance/rule/required', '00000000-0000-0000-0000-000000000000'),

  -- Max Issuer Exposure 5%
  ('rule_max_issuer_5pct',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'MaxIssuerExposure_5Percent',
   '{
     "dsl": "RULE MaxIssuerExposure_5pct: SCOPE portfolio METRIC = Sum(Position.MarketValue WHERE Security.IssuerID = :issuer) / Sum(Position.MarketValue) CONDITION METRIC <= 0.05 SEVERITY HARD",
     "severity": "ERROR",
     "semantic_terms": ["st_market_value_base", "st_security_id"]
   }',
   'rules/compliance/issuer', '00000000-0000-0000-0000-000000000000'),

  -- No Tobacco
  ('rule_no_tobacco',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'NoTobaccoHoldings',
   '{
     "dsl": "RULE NoTobacco: SCOPE portfolio CONDITION NOT EXISTS Position WHERE Security.Industry = \"Tobacco\" SEVERITY HARD",
     "severity": "ERROR",
     "semantic_terms": ["st_position_quantity", "st_security_id"]
   }',
   'rules/compliance/esg', '00000000-0000-0000-0000-000000000000'),

  -- Min Cash 2%
  ('rule_min_cash_2pct',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'MinCash_2Percent',
   '{
     "dsl": "RULE MinCash_2pct: SCOPE portfolio METRIC = CashBalance.ClosingBalance / (CashBalance.ClosingBalance + Sum(Position.MarketValue)) CONDITION METRIC >= 0.02 SEVERITY SOFT",
     "severity": "WARNING",
     "semantic_terms": ["st_closing_balance", "st_market_value_base"]
   }',
   'rules/compliance/liquidity', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- CALCULATION TERMS (WASM/SQL Execution)
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('ct_issuer_exposure',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'CALCULATION_TERM'),
   'IssuerExposure',
   '{
     "expression": "sum(position_market_value where security_issuer_id = issuer) / sum(position_market_value)",
     "depends_on": ["st_market_value_base", "st_security_id"],
     "return_type": "numeric",
     "execution_target": "SQL"
   }',
   'calculations/compliance/issuer_exposure', '00000000-0000-0000-0000-000000000000'),

  ('ct_cash_ratio',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'CALCULATION_TERM'),
   'CashRatio',
   '{
     "expression": "cash_balance / (cash_balance + total_market_value)",
     "depends_on": ["st_closing_balance", "st_market_value_base"],
     "return_type": "numeric",
     "execution_target": "SQL"
   }',
   'calculations/compliance/cash_ratio', '00000000-0000-0000-0000-000000000000'),

  ('ct_sector_exposure',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'CALCULATION_TERM'),
   'SectorExposure',
   '{
     "expression": "sum(position_market_value where security_sector = sector) / sum(position_market_value)",
     "depends_on": ["st_market_value_base", "st_security_id"],
     "return_type": "numeric",
     "execution_target": "SQL"
   }',
   'calculations/compliance/sector_exposure', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- DEMO COMPLIANCE RULES
-- ============================================

INSERT INTO edm.compliance_rule (
    rule_id, rule_code, rule_name, description, scope_type, scope_value,
    expression, threshold_value, threshold_operator, severity,
    effective_from, effective_to, status, tenant_id
)
VALUES 
  -- Max 5% Issuer Exposure
  (gen_random_uuid(), 'MAX_ISSUER_5PCT', 'Max Issuer Exposure 5%', 
   'No single issuer shall exceed 5% of portfolio market value.',
   'GLOBAL', NULL,
   'SUM(position.market_value) WHERE security.issuer_id = :issuer / SUM(position.market_value) <= 0.05',
   0.05, '<=', 'HARD',
   CURRENT_DATE - INTERVAL '1 year', NULL, 'ACTIVE',
   '00000000-0000-0000-0000-000000000000'),

  -- No Tobacco
  (gen_random_uuid(), 'NO_TOBACCO', 'No Tobacco Holdings',
   'Portfolio shall not hold securities from tobacco industry.',
   'GLOBAL', NULL,
   'NOT EXISTS position WHERE security.industry = "Tobacco"',
   NULL, NULL, 'HARD',
   CURRENT_DATE - INTERVAL '1 year', NULL, 'ACTIVE',
   '00000000-0000-0000-0000-000000000000'),

  -- Min Cash 2%
  (gen_random_uuid(), 'MIN_CASH_2PCT', 'Minimum Cash 2%',
   'Portfolio shall maintain at least 2% cash for liquidity.',
   'GLOBAL', NULL,
   'cash_balance / (cash_balance + total_market_value) >= 0.02',
   0.02, '>=', 'SOFT',
   CURRENT_DATE - INTERVAL '1 year', NULL, 'ACTIVE',
   '00000000-0000-0000-0000-000000000000'),

  -- Max Sector 25%
  (gen_random_uuid(), 'MAX_SECTOR_25PCT', 'Max Sector Exposure 25%',
   'No single sector shall exceed 25% of portfolio.',
   'GLOBAL', NULL,
   'SUM(position.market_value) WHERE security.sector = :sector / SUM(position.market_value) <= 0.25',
   0.25, '<=', 'HARD',
   CURRENT_DATE - INTERVAL '1 year', NULL, 'ACTIVE',
   '00000000-0000-0000-0000-000000000000');

-- Demo Evaluations
INSERT INTO edm.compliance_evaluation (
    evaluation_id, rule_id, portfolio_id, valuation_date,
    metric_value, threshold_value, result, details, evaluation_time_ms, tenant_id
)
VALUES 
  (gen_random_uuid(),
   (SELECT rule_id FROM edm.compliance_rule WHERE rule_code = 'MAX_ISSUER_5PCT' LIMIT 1),
   (SELECT id FROM edm.portfolio_master WHERE portfolio_name = 'Demo Growth Fund' LIMIT 1),
   CURRENT_DATE,
   0.042, 0.05, 'PASS',
   '{"issuer": "Apple Inc", "exposure_pct": 4.2}'::jsonb, 15,
   '00000000-0000-0000-0000-000000000000');
