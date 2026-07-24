-- Migration 042: Seed Risk Factors, Scenarios, Demo Data

-- ============================================
-- RISK FACTORS
-- ============================================

INSERT INTO edm.risk_factor (factor_id, factor_code, factor_name, category, factor_type, unit, tenant_id)
VALUES 
  (gen_random_uuid(), 'EQUITY_MKT', 'Equity Market', 'EQUITY', 'SYSTEMATIC', '%', '00000000-0000-0000-0000-000000000000'),
  (gen_random_uuid(), 'VALUE', 'Value Factor', 'EQUITY', 'SYSTEMATIC', '%', '00000000-0000-0000-0000-000000000000'),
  (gen_random_uuid(), 'SIZE', 'Size Factor', 'EQUITY', 'SYSTEMATIC', '%', '00000000-0000-0000-0000-000000000000'),
  (gen_random_uuid(), 'MOMENTUM', 'Momentum Factor', 'EQUITY', 'SYSTEMATIC', '%', '00000000-0000-0000-0000-000000000000'),
  (gen_random_uuid(), 'CREDIT_SPREAD', 'Credit Spread', 'FIXED_INCOME', 'SYSTEMATIC', 'BP', '00000000-0000-0000-0000-000000000000'),
  (gen_random_uuid(), 'DURATION_10Y', '10Y Duration', 'FIXED_INCOME', 'SYSTEMATIC', 'YEARS', '00000000-0000-0000-0000-000000000000'),
  (gen_random_uuid(), 'FX_USD_EUR', 'USD/EUR FX', 'FX', 'SYSTEMATIC', '%', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- STRESS SCENARIOS
-- ============================================

INSERT INTO edm.risk_scenario (scenario_id, scenario_code, scenario_name, description, scenario_type, shocks, status, tenant_id)
VALUES 
  -- 2008 Financial Crisis
  (gen_random_uuid(), 'CRISIS_2008', '2008 Financial Crisis',
   'Historical stress scenario from 2008 financial crisis.',
   'HISTORICAL',
   '{
     "factors": [
       {"factor_id": "EQUITY_MKT", "shock": -0.40},
       {"factor_id": "CREDIT_SPREAD", "shock": 0.05}
     ],
     "yields": [{"tenor": "10Y", "parallel_shift_bps": -150}],
     "fx": [{"pair": "USD/EUR", "shock": 0.15}]
   }'::jsonb,
   'ACTIVE', '00000000-0000-0000-0000-000000000000'),

  -- +200bps Rate Shock
  (gen_random_uuid(), 'RATE_UP_200', '+200bps Rate Shock',
   'Hypothetical parallel shift up 200 basis points.',
   'HYPOTHETICAL',
   '{
     "yields": [{"tenor": "ALL", "parallel_shift_bps": 200}],
     "factors": [{"factor_id": "DURATION_10Y", "shock": -0.15}]
   }'::jsonb,
   'ACTIVE', '00000000-0000-0000-0000-000000000000'),

  -- -20% Equity Market
  (gen_random_uuid(), 'EQUITY_DOWN_20', '-20% Equity Market',
   'Hypothetical equity market decline of 20%.',
   'HYPOTHETICAL',
   '{
     "factors": [{"factor_id": "EQUITY_MKT", "shock": -0.20}]
   }'::jsonb,
   'ACTIVE', '00000000-0000-0000-0000-000000000000'),

  -- COVID-19 March 2020
  (gen_random_uuid(), 'COVID_MAR_2020', 'COVID-19 March 2020',
   'Historical stress from March 2020 market crash.',
   'HISTORICAL',
   '{
     "factors": [
       {"factor_id": "EQUITY_MKT", "shock": -0.34},
       {"factor_id": "CREDIT_SPREAD", "shock": 0.03}
     ],
     "volatility": {"shock_multiplier": 3.5}
   }'::jsonb,
   'ACTIVE', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- CALCULATION TERMS (Risk)
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('ct_portfolio_variance',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'CALCULATION_TERM'),
   'PortfolioVariance',
   '{
     "expression": "F_transpose * covariance_matrix * F",
     "depends_on": ["st_factor_exposure", "st_factor_covariance"],
     "return_type": "numeric",
     "execution_target": "WASM"
   }',
   'calculations/risk/variance', '00000000-0000-0000-0000-000000000000'),

  ('ct_var_95',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'CALCULATION_TERM'),
   'VaR_95',
   '{
     "expression": "1.65 * sqrt(variance) * AUM",
     "depends_on": ["ct_portfolio_variance", "st_total_market_value_base"],
     "return_type": "numeric",
     "execution_target": "SQL"
   }',
   'calculations/risk/var_95', '00000000-0000-0000-0000-000000000000'),

  ('ct_var_99',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'CALCULATION_TERM'),
   'VaR_99',
   '{
     "expression": "2.33 * sqrt(variance) * AUM",
     "depends_on": ["ct_portfolio_variance", "st_total_market_value_base"],
     "return_type": "numeric",
     "execution_target": "SQL"
   }',
   'calculations/risk/var_99', '00000000-0000-0000-0000-000000000000'),

  ('ct_scenario_pnl',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'CALCULATION_TERM'),
   'ScenarioPnL',
   '{
     "expression": "SUM(position_mv * factor_exposure * factor_shock)",
     "depends_on": ["st_market_value_base", "st_factor_exposure"],
     "return_type": "numeric",
     "execution_target": "SQL"
   }',
   'calculations/risk/scenario_pnl', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- DQ RULES (Risk)
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('rule_risk_factor_coverage',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'Risk_FactorCoverage',
   '{
     "dsl": "RULE Risk_FactorCoverage: IF EXISTS Position WHERE NOT EXISTS SecurityFactorExposure WHERE SecurityFactorExposure.SecurityID = Position.SecurityID THEN WARNING \"Missing factor exposures for some holdings\"",
     "severity": "WARNING",
     "semantic_terms": ["st_position_quantity", "st_factor_exposure"]
   }',
   'rules/risk/coverage', '00000000-0000-0000-0000-000000000000'),

  ('rule_risk_var_sanity',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'validation_rule'),
   'Risk_VaR_Sanity',
   '{
     "dsl": "RULE Risk_VaR_Sanity: IF PortfolioRisk.VaR95 < 0 OR PortfolioRisk.VaR99 < 0 THEN ERROR \"VaR cannot be negative\"",
     "severity": "ERROR",
     "semantic_terms": ["st_var_95", "st_var_99"]
   }',
   'rules/risk/validation', '00000000-0000-0000-0000-000000000000');
