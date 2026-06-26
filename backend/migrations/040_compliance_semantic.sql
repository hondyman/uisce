-- Migration 040: Register Compliance BOs + Semantic Terms + Graph Edges
-- Aligns with Semantic Design §2: BO Fields bind semantic_term_id

-- ============================================
-- SEMANTIC TERMS (Compliance)
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('st_rule_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'RuleID', '{"data_type":"uuid"}', 'semantic/RuleID', '00000000-0000-0000-0000-000000000000'),
  ('st_rule_code', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'RuleCode', '{"data_type":"text"}', 'semantic/RuleCode', '00000000-0000-0000-0000-000000000000'),
  ('st_rule_name', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'RuleName', '{"data_type":"text"}', 'semantic/RuleName', '00000000-0000-0000-0000-000000000000'),
  ('st_rule_expression', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'RuleExpression', '{"data_type":"text"}', 'semantic/RuleExpression', '00000000-0000-0000-0000-000000000000'),
  ('st_rule_severity', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'RuleSeverity', '{"data_type":"text", "allowed_values":["HARD","SOFT","WARNING","ALERT"]}', 'semantic/RuleSeverity', '00000000-0000-0000-0000-000000000000'),
  ('st_effective_from', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'EffectiveFrom', '{"data_type":"date"}', 'semantic/EffectiveFrom', '00000000-0000-0000-0000-000000000000'),
  ('st_evaluation_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'EvaluationID', '{"data_type":"uuid"}', 'semantic/EvaluationID', '00000000-0000-0000-0000-000000000000'),
  ('st_metric_value', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'MetricValue', '{"data_type":"numeric"}', 'semantic/MetricValue', '00000000-0000-0000-0000-000000000000'),
  ('st_threshold_value', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ThresholdValue', '{"data_type":"numeric"}', 'semantic/ThresholdValue', '00000000-0000-0000-0000-000000000000'),
  ('st_evaluation_result', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'EvaluationResult', '{"data_type":"text", "allowed_values":["PASS","FAIL","WARNING"]}', 'semantic/EvaluationResult', '00000000-0000-0000-0000-000000000000'),
  ('st_breach_id', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'BreachID', '{"data_type":"uuid"}', 'semantic/BreachID', '00000000-0000-0000-0000-000000000000'),
  ('st_breach_status', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'BreachStatus', '{"data_type":"text", "allowed_values":["OPEN","ACKNOWLEDGED","RESOLVED","WAIVED"]}', 'semantic/BreachStatus', '00000000-0000-0000-0000-000000000000'),
  ('st_resolution_notes', (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'), 'ResolutionNotes', '{"data_type":"text"}', 'semantic/ResolutionNotes', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- BUSINESS OBJECT REGISTRATION
-- ============================================

INSERT INTO public.business_objects (id, name, display_name, key, technical_name, description, category, driver_table_name, is_core, enable_history, history_mode, tenant_id)
VALUES 
  ('bo_compliance_rule', 'ComplianceRule', 'Compliance Rule', 'compliance_rule', 'compliance_rule', 
   'Defines a compliance constraint evaluated against portfolio exposures.', 'Compliance',
   'edm.compliance_rule', true, true, 'SCD2', 
   '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_evaluation', 'ComplianceEvaluation', 'Compliance Evaluation', 'compliance_evaluation', 'compliance_evaluation',
   'Result of evaluating a compliance rule for a portfolio and date.', 'Compliance', 'edm.compliance_evaluation', true, false, 'NONE',
   '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_breach', 'ComplianceBreach', 'Compliance Breach', 'compliance_breach', 'compliance_breach',
   'Represents a violation of a compliance rule.', 'Compliance', 'edm.compliance_breach', true, false, 'NONE',
   '00000000-0000-0000-0000-000000000000');

-- ============================================
-- BO FIELDS WITH SEMANTIC TERM BINDINGS
-- ============================================

INSERT INTO public.bo_fields (business_object_id, field_name, display_label, field_type, semantic_term_id, is_required, role, tenant_id)
VALUES 
  -- Compliance Rule Fields
  ('bo_compliance_rule', 'rule_id', 'Rule ID', 'UUID', 'st_rule_id', true, 'IDENTIFIER', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_rule', 'rule_code', 'Rule Code', 'TEXT', 'st_rule_code', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_rule', 'rule_name', 'Rule Name', 'TEXT', 'st_rule_name', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_rule', 'expression', 'Rule Expression', 'TEXT', 'st_rule_expression', true, 'METADATA', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_rule', 'severity', 'Severity', 'TEXT', 'st_rule_severity', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_rule', 'effective_from', 'Effective From', 'DATE', 'st_effective_from', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),

  -- Compliance Evaluation Fields
  ('bo_compliance_evaluation', 'evaluation_id', 'Evaluation ID', 'UUID', 'st_evaluation_id', true, 'IDENTIFIER', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_evaluation', 'rule_id', 'Rule', 'REFERENCE', 'st_rule_id', true, 'FOREIGN_KEY', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_evaluation', 'portfolio_id', 'Portfolio', 'REFERENCE', 'st_portfolio_id', true, 'FOREIGN_KEY', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_evaluation', 'valuation_date', 'Valuation Date', 'DATE', 'st_valuation_date', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_evaluation', 'metric_value', 'Metric Value', 'NUMERIC', 'st_metric_value', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_evaluation', 'threshold_value', 'Threshold Value', 'NUMERIC', 'st_threshold_value', false, 'MEASURE', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_evaluation', 'result', 'Result', 'TEXT', 'st_evaluation_result', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),

  -- Compliance Breach Fields
  ('bo_compliance_breach', 'breach_id', 'Breach ID', 'UUID', 'st_breach_id', true, 'IDENTIFIER', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_breach', 'evaluation_id', 'Evaluation', 'REFERENCE', 'st_evaluation_id', true, 'FOREIGN_KEY', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_breach', 'rule_id', 'Rule', 'REFERENCE', 'st_rule_id', true, 'FOREIGN_KEY', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_breach', 'portfolio_id', 'Portfolio', 'REFERENCE', 'st_portfolio_id', true, 'FOREIGN_KEY', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_breach', 'valuation_date', 'Valuation Date', 'DATE', 'st_valuation_date', true, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_breach', 'status', 'Status', 'TEXT', 'st_breach_status', false, 'DIMENSION', '00000000-0000-0000-0000-000000000000'),
  ('bo_compliance_breach', 'resolution_notes', 'Resolution Notes', 'TEXT', 'st_resolution_notes', false, 'METADATA', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- SEMANTIC GRAPH EDGES
-- ============================================

INSERT INTO catalog_edge (source_node_id, target_node_id, edge_type_id, properties, tenant_id)
VALUES 
  -- Compliance Rule → Portfolio
  ('bo_compliance_rule', 'bo_portfolio', 
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'applies_to'),
   '{"field_mapping": "scope_value"}', '00000000-0000-0000-0000-000000000000'),

  -- Compliance Evaluation → Rule
  ('bo_compliance_evaluation', 'bo_compliance_rule',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'evaluates'),
   '{"field_mapping": "rule_id"}', '00000000-0000-0000-0000-000000000000'),

  -- Compliance Evaluation → Portfolio
  ('bo_compliance_evaluation', 'bo_portfolio',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'held_in_portfolio'),
   '{"field_mapping": "portfolio_id"}', '00000000-0000-0000-0000-000000000000'),

  -- Compliance Breach → Evaluation
  ('bo_compliance_breach', 'bo_compliance_evaluation',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'derived_from'),
   '{"field_mapping": "evaluation_id"}', '00000000-0000-0000-0000-000000000000'),

  -- Compliance Rule → Position (for exposure calculation)
  ('bo_compliance_rule', 'bo_position',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'evaluates_exposure'),
   '{"field_mapping": "expression", "depends_on": ["market_value", "quantity"]}', '00000000-0000-0000-0000-000000000000'),

  -- Compliance Rule → Cash Balance
  ('bo_compliance_rule', 'bo_cash_balance',
   (SELECT id FROM catalog_edge_type WHERE edge_type_name = 'evaluates_exposure'),
   '{"field_mapping": "expression", "depends_on": ["closing_balance"]}', '00000000-0000-0000-0000-000000000000');

-- ============================================
-- LINEAGE SUMMARY NODE
-- ============================================

INSERT INTO catalog_node (id, node_type_id, node_name, properties, qualified_path, tenant_id)
VALUES 
  ('lineage_summary_compliance',
   (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'business_term'),
   'ComplianceLineage',
   '{
     "description": "Complete lineage from Rule → Evaluation → Breach with exposure calculations",
     "domains": ["Compliance", "Position", "Cash", "Security", "Portfolio"],
     "edge_count": 6,
     "calculation_terms": ["ct_issuer_exposure", "ct_cash_ratio", "ct_sector_exposure"],
     "trace_tables": ["edm.compliance_lineage", "edm.compliance_evaluation", "edm.compliance_breach"],
     "dq_rules": ["rule_compliance_rule_required", "rule_max_issuer_5pct", "rule_no_tobacco", "rule_min_cash_2pct"]
   }',
   'lineage/compliance', '00000000-0000-0000-0000-000000000000');
