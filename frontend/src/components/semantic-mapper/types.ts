export interface DatabaseColumn {
  schema?: string;
  table: string;
  column: string;
  node_id?: string;
  data_type?: string;
  tenant_id?: string;
  tenant_tenant_instance_id?: string;
  [key: string]: any;
}

export interface SemanticTerm {
  node_id?: string;
  term_name: string;
  description?: string;
  [key: string]: any;
}

export interface Mapping {
  id: string;
  database_column: DatabaseColumn;
  semantic_term?: string | null;
  semantic_term_id?: string;
  confidence: number;
  selected?: boolean;
  ignored?: boolean;
  override?: boolean;
  edge_exists?: boolean;
  is_new_term?: boolean;
  is_pending?: boolean; // Added for persisting pending approvals
  match_reason?: string;
  [key: string]: any;
}
