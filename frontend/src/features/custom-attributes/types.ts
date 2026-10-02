export interface AttributeDef {
  id: string;
  tenant_id: string;
  core_id?: string | null;
  is_shadow: boolean;
  entity_type: string;
  table_ref: string;
  field_cd: string;
  name: string;
  description?: string;
  data_type: string;
  json_path: string;
  is_required: boolean;
  is_searchable: boolean;
  is_pii: boolean;
  validation_rules?: Record<string, unknown>;
  picklist_values?: unknown[];
  default_value?: string | null;
  display_order: number;
  section?: string;
  is_active: boolean;
  origin?: 'CORE' | 'CUSTOM' | string;
  semantic_term_id?: string | null;
  semantic_term_name?: string;
  created_at?: string;
  updated_at?: string;
}

export interface EligibleEntity {
  entity_type: string;
  table_ref: string;
  display_name: string;
  schema_name: string;
  table_name: string;
  qualified_path: string;
  datasource_id?: string | null;
  field_count: number;
  column_node_id?: string | null;
  table_node_id?: string | null;
}

export interface ColumnMeta {
  key: string;
  label: string;
  type: string;
  source: 'CORE' | 'CUSTOM' | string;
  section?: string;
  field_cd?: string;
}

export interface PreviewResponse {
  columns: ColumnMeta[];
  rows: Record<string, unknown>[];
  total: number;
  showing: number;
}

export interface CreateAttributeInput {
  entity_type: string;
  table_ref: string;
  field_cd?: string;
  name: string;
  description?: string;
  data_type: string;
  json_path?: string;
  is_required?: boolean;
  is_searchable?: boolean;
  is_pii?: boolean;
  validation_rules?: Record<string, unknown>;
  picklist_values?: unknown[];
  default_value?: string | null;
  display_order?: number;
  section?: string;
  semantic_term_id?: string | null;
}

export interface UpdateAttributeInput {
  name?: string;
  description?: string;
  data_type?: string;
  is_required?: boolean;
  is_searchable?: boolean;
  is_pii?: boolean;
  validation_rules?: Record<string, unknown>;
  picklist_values?: unknown[];
  default_value?: string | null;
  display_order?: number;
  section?: string;
  semantic_term_id?: string | null;
  clear_semantic_term?: boolean;
}
