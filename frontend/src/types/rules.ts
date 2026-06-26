/**
 * Validation Rules and Business Rules types
 */

export interface ValidationRule {
  id: string;
  entity_id: string;
  rule_type: string;
  conditions: any[];
  actions: any[];
  enabled: boolean;
  priority: number;
  description?: string;
  created_at?: string;
  updated_at?: string;
}

export interface Rule {
  id: string;
  name: string;
  description?: string;
  type: 'validation' | 'business' | 'transformation';
  conditions: any[];
  actions: any[];
  enabled: boolean;
  scope?: string;
  metadata?: Record<string, any>;
}

export interface RulePreviewResult {
  rule_id: string;
  status: 'success' | 'error' | 'warning';
  result?: any;
  error?: string;
  warnings?: string[];
}

export interface RuleSimulationResult {
  simulation_id: string;
  rule_id: string;
  input_data: any;
  output_data: any;
  status: 'success' | 'error';
  executed_at: string;
  execution_time_ms?: number;
}

export interface ValidationPreviewResult {
  entity_id: string;
  rule_id: string;
  is_valid: boolean;
  violations: ValidationViolation[];
}

export interface ValidationViolation {
  field: string;
  message: string;
  severity: 'error' | 'warning' | 'info';
}

export interface RuleDiff {
  field: string;
  before: any;
  after: any;
  type: 'added' | 'removed' | 'modified';
}

export interface RuleVersion {
  version: number;
  created_at: string;
  created_by: string;
  changes: RuleDiff[];
}

export interface RuleLineage {
  rule_id: string;
  versions: RuleVersion[];
  current_version: number;
  modified_count: number;
}
