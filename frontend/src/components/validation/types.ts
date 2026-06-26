export interface Condition {
  field: string;
  operator: string;
  value: string;
}

export interface ValidationRule {
  id?: string;
  // human-friendly name
  name?: string;
  // old/new forms: support both `name` and `rule_name`
  rule_name?: string;
  // target/entity fields
  entity?: string;
  target_entity?: string;
  target_entities?: string[];
  sub_entity_type?: string;
  rule_type?: string;
  severity: 'error' | 'warning' | 'info';
  description?: string;
  error_message?: string;
  is_active?: boolean;
  is_global?: boolean;
  is_core?: boolean;
  conditions?: Condition[];
  dependent_rule_ids?: string[];
  created_at?: string;
  updated_at?: string;
}

export interface EntityPathSegment {
  entity: string;
  field: string;
  relationship: string;
}

export interface EntityPath {
  segments: EntityPathSegment[];
  displayPath: string;
}

export interface CrossEntityCondition {
  sourcePath: EntityPath;
  operator: string;
  targetPath: EntityPath;
}
