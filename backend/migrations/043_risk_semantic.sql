-- Migration 043: Register Risk BOs + Semantic Terms + Graph Edges

DO $$
DECLARE
    v_gold_tenant UUID;
BEGIN
    PERFORM set_config('app.goldcopy', (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)::text, true);
    v_gold_tenant := current_setting('app.goldcopy', true)::uuid;

    -- ============================================
    -- SEMANTIC TERMS (Risk)
    -- ============================================

    INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
    VALUES
      ('st_factor_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'FactorID', '{"data_type":"uuid"}', 'semantic/FactorID', v_gold_tenant),
      ('st_factor_name', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'FactorName', '{"data_type":"text"}', 'semantic/FactorName', v_gold_tenant),
      ('st_factor_category', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'FactorCategory', '{"data_type":"text"}', 'semantic/FactorCategory', v_gold_tenant),
      ('st_factor_exposure', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'FactorExposure', '{"data_type":"numeric"}', 'semantic/FactorExposure', v_gold_tenant),
      ('st_as_of_date', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'AsOfDate', '{"data_type":"date"}', 'semantic/AsOfDate', v_gold_tenant),
      ('st_portfolio_risk_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'PortfolioRiskID', '{"data_type":"uuid"}', 'semantic/PortfolioRiskID', v_gold_tenant),
      ('st_total_volatility', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TotalVolatility', '{"data_type":"numeric"}', 'semantic/TotalVolatility', v_gold_tenant),
      ('st_tracking_error', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'TrackingError', '{"data_type":"numeric"}', 'semantic/TrackingError', v_gold_tenant),
      ('st_var_95', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'VaR_95', '{"data_type":"numeric"}', 'semantic/VaR_95', v_gold_tenant),
      ('st_var_99', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'VaR_99', '{"data_type":"numeric"}', 'semantic/VaR_99', v_gold_tenant),
      ('st_es_97_5', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ExpectedShortfall_97_5', '{"data_type":"numeric"}', 'semantic/ExpectedShortfall_97_5', v_gold_tenant),
      ('st_scenario_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ScenarioID', '{"data_type":"uuid"}', 'semantic/ScenarioID', v_gold_tenant),
      ('st_scenario_name', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ScenarioName', '{"data_type":"text"}', 'semantic/ScenarioName', v_gold_tenant),
      ('st_scenario_pnl', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ScenarioPnL', '{"data_type":"numeric"}', 'semantic/ScenarioPnL', v_gold_tenant),
      ('st_scenario_result_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ScenarioResultID', '{"data_type":"uuid"}', 'semantic/ScenarioResultID', v_gold_tenant);

    -- ============================================
    -- BUSINESS OBJECT REGISTRATION
    -- ============================================

    INSERT INTO public.business_objects (id, name, display_name, key, technical_name, description, category, driver_table_name, is_core, enable_history, history_mode, tenant_id)
    VALUES
      ('bo_risk_factor', 'RiskFactor', 'Risk Factor', 'risk_factor', 'risk_factor',
       'Defines a systematic risk factor used in the factor model.', 'Risk',
       'edm.risk_factor', true, true, 'SCD2',
       v_gold_tenant),
      ('bo_security_factor_exposure', 'SecurityFactorExposure', 'Security Factor Exposure', 'security_factor_exposure', 'security_factor_exposure',
       'Exposure of a security to a given risk factor.', 'Risk', 'edm.security_factor_exposure', true, true, 'SCD2',
       v_gold_tenant),
      ('bo_portfolio_risk', 'PortfolioRisk', 'Portfolio Risk', 'portfolio_risk', 'portfolio_risk',
       'Risk measures for a portfolio at a given date.', 'Risk', 'edm.portfolio_risk', true, false, 'NONE',
       v_gold_tenant),
      ('bo_risk_scenario', 'RiskScenario', 'Risk Scenario', 'risk_scenario', 'risk_scenario',
       'Defines a stress scenario as shocks to factors or market variables.', 'Risk', 'edm.risk_scenario', true, true, 'SCD2',
       v_gold_tenant),
      ('bo_risk_scenario_result', 'RiskScenarioResult', 'Risk Scenario Result', 'risk_scenario_result', 'risk_scenario_result',
       'Result of applying a risk scenario to a portfolio.', 'Risk', 'edm.risk_scenario_result', true, false, 'NONE',
       v_gold_tenant);

    -- ============================================
    -- BO FIELDS WITH SEMANTIC TERM BINDINGS (abbreviated for brevity)
    -- ============================================

    INSERT INTO public.bo_fields (business_object_id, field_name, display_label, field_type, semantic_term_id, is_required, role, tenant_id)
    VALUES
      -- Risk Factor Fields
      ('bo_risk_factor', 'factor_id', 'Factor ID', 'UUID', 'st_factor_id', true, 'IDENTIFIER', v_gold_tenant),
      ('bo_risk_factor', 'factor_code', 'Factor Code', 'TEXT', 'st_rule_code', true, 'DIMENSION', v_gold_tenant),
      ('bo_risk_factor', 'factor_name', 'Factor Name', 'TEXT', 'st_factor_name', true, 'DIMENSION', v_gold_tenant),
      ('bo_risk_factor', 'category', 'Category', 'TEXT', 'st_factor_category', false, 'DIMENSION', v_gold_tenant),

      -- Security Factor Exposure Fields
      ('bo_security_factor_exposure', 'security_id', 'Security', 'REFERENCE', 'st_security_id', true, 'FOREIGN_KEY', v_gold_tenant),
      ('bo_security_factor_exposure', 'factor_id', 'Factor', 'REFERENCE', 'st_factor_id', true, 'FOREIGN_KEY', v_gold_tenant),
      ('bo_security_factor_exposure', 'exposure', 'Exposure', 'NUMERIC', 'st_factor_exposure', true, 'MEASURE', v_gold_tenant),
      ('bo_security_factor_exposure', 'as_of_date', 'As Of Date', 'DATE', 'st_as_of_date', true, 'DIMENSION', v_gold_tenant),

      -- Portfolio Risk Fields
      ('bo_portfolio_risk', 'portfolio_risk_id', 'Portfolio Risk ID', 'UUID', 'st_portfolio_risk_id', true, 'IDENTIFIER', v_gold_tenant),
      ('bo_portfolio_risk', 'portfolio_id', 'Portfolio', 'REFERENCE', 'st_portfolio_id', true, 'FOREIGN_KEY', v_gold_tenant),
      ('bo_portfolio_risk', 'valuation_date', 'Valuation Date', 'DATE', 'st_valuation_date', true, 'DIMENSION', v_gold_tenant),
      ('bo_portfolio_risk', 'total_volatility', 'Total Volatility', 'NUMERIC', 'st_total_volatility', false, 'MEASURE', v_gold_tenant),
      ('bo_portfolio_risk', 'var_95', 'VaR 95', 'NUMERIC', 'st_var_95', false, 'MEASURE', v_gold_tenant),
      ('bo_portfolio_risk', 'var_99', 'VaR 99', 'NUMERIC', 'st_var_99', false, 'MEASURE', v_gold_tenant),

      -- Risk Scenario Fields
      ('bo_risk_scenario', 'scenario_id', 'Scenario ID', 'UUID', 'st_scenario_id', true, 'IDENTIFIER', v_gold_tenant),
      ('bo_risk_scenario', 'scenario_code', 'Scenario Code', 'TEXT', 'st_rule_code', true, 'DIMENSION', v_gold_tenant),
      ('bo_risk_scenario', 'scenario_name', 'Scenario Name', 'TEXT', 'st_scenario_name', true, 'DIMENSION', v_gold_tenant),

      -- Scenario Result Fields
      ('bo_risk_scenario_result', 'scenario_result_id', 'Scenario Result ID', 'UUID', 'st_scenario_result_id', true, 'IDENTIFIER', v_gold_tenant),
      ('bo_risk_scenario_result', 'scenario_id', 'Scenario', 'REFERENCE', 'st_scenario_id', true, 'FOREIGN_KEY', v_gold_tenant),
      ('bo_risk_scenario_result', 'portfolio_id', 'Portfolio', 'REFERENCE', 'st_portfolio_id', true, 'FOREIGN_KEY', v_gold_tenant),
      ('bo_risk_scenario_result', 'pnl', 'Scenario P&L', 'NUMERIC', 'st_scenario_pnl', false, 'MEASURE', v_gold_tenant);

    -- ============================================
    -- SEMANTIC GRAPH EDGES (Risk)
    -- ============================================

    INSERT INTO catalog_edge (source_node_id, target_node_id, edge_type_id, properties, tenant_id)
    VALUES
      -- Risk Factor → Security Factor Exposure
      ('bo_risk_factor', 'bo_security_factor_exposure',
       (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'measured_by'),
       '{"field_mapping": "factor_id"}', v_gold_tenant),

      -- Security Factor Exposure → Security
      ('bo_security_factor_exposure', 'bo_security',
       (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'references'),
       '{"field_mapping": "security_id"}', v_gold_tenant),

      -- Portfolio Risk → Portfolio
      ('bo_portfolio_risk', 'bo_portfolio',
       (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'held_in_portfolio'),
       '{"field_mapping": "portfolio_id"}', v_gold_tenant),

      -- Portfolio Risk → Position (for calculation)
      ('bo_portfolio_risk', 'bo_position',
       (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'calculates_from'),
       '{"field_mapping": "portfolio_id, valuation_date"}', v_gold_tenant),

      -- Risk Scenario → Scenario Result
      ('bo_risk_scenario', 'bo_risk_scenario_result',
       (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'produces'),
       '{"field_mapping": "scenario_id"}', v_gold_tenant),

      -- Scenario Result → Portfolio
      ('bo_risk_scenario_result', 'bo_portfolio',
       (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'held_in_portfolio'),
       '{"field_mapping": "portfolio_id"}', v_gold_tenant);

    -- ============================================
    -- LINEAGE SUMMARY NODE
    -- ============================================

    INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
    VALUES
      ('lineage_summary_risk',
       (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'business_term'),
       'RiskLineage',
       '{
         "description": "Complete lineage from Risk Factor → Security Exposure → Portfolio Risk → Scenario Result",
         "domains": ["Risk", "Security", "Position", "Portfolio", "Pricing"],
         "edge_count": 6,
         "calculation_terms": ["ct_portfolio_variance", "ct_var_95", "ct_var_99", "ct_scenario_pnl"],
         "trace_tables": ["edm.portfolio_risk", "edm.risk_scenario_result", "edm.security_factor_exposure"],
         "dq_rules": ["rule_risk_factor_coverage", "rule_risk_var_sanity"]
       }',
       'lineage/risk', v_gold_tenant);

END $$;
