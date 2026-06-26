// Semantic Playground Types

export interface SemanticField {
  field_id: string;
  name: string;
  display_name: string;
  semantic_term: string;
  subtype?: string;
  aliases?: string[];
  physical: PhysicalMapping;
  description?: string;
}

export interface PhysicalMapping {
  datasource_id: string;
  table: string;
  column: string;
}

export interface SemanticRelationship {
  target_bo_id: string;
  join_type: "INNER" | "LEFT" | "RIGHT" | "FULL";
  source_column: string;
  target_column: string;
  target_table: string;
}

export interface DiscriminatorMetadata {
  column_name: string;
  subtypes: SubtypeDefinition[];
}

export interface SubtypeDefinition {
  id: string;
  label: string;
  discriminator_value: string;
  fields?: string[];
  required_fields?: string[];
}

export interface SemanticBundle {
  business_object_id: string;
  business_object_name: string;
  datasource_id: string;
  driving_table: string;
  version: string;
  discriminator?: DiscriminatorMetadata;
  fields: SemanticField[];
  relationships?: SemanticRelationship[];
  created_at?: string;
  updated_at?: string;
}

export interface FilterCondition {
  field: string;
  op: "=" | "!=" | ">" | "<" | ">=" | "<=" | "in" | "not_in" | "like";
  value: any;
}

export interface OrderByClause {
  field: string;
  direction: "asc" | "desc";
}

export interface SemanticQuery {
  datasource: string;
  version: string;
  select: string[];
  filters?: FilterCondition[];
  order_by?: OrderByClause[];
  limit?: number;
  offset?: number;
  aggregations?: Record<string, any>;
}

export interface PlannerRequest {
  datasource: string;
  version: string;
  prompt: string;
  mode: "exploratory" | "strict" | "CRUD";
}

export interface PlannerResponse {
  semantic_query: SemanticQuery;
  explanation?: string;
  confidence?: number;
  warnings?: string[];
}

export interface ExecutorRequest {
  datasource: string;
  version: string;
  semantic_query: SemanticQuery;
}

export interface ExecutorResponse {
  generated_sql: string;
  semantic_sql?: SemanticQuery;
  execution_plan?: string;
  warnings?: string[];
}

export interface QueryExecutionRequest {
  sql: string;
  limit?: number;
  timeout?: number;
}

export interface QueryExecutionResponse {
  rows: Record<string, any>[];
  row_count: number;
  execution_time_ms: number;
  columns: string[];
}

export interface Datasource {
  id: string;
  name: string;
  type: string;
  description?: string;
}

export interface BundleVersion {
  version: string;
  created_at: string;
  created_by: string;
  change_type?: string;
}

export interface LineageNode {
  field_id: string;
  field_name: string;
  physical_column: string;
  physical_table: string;
  subtype?: string;
  version: string;
  aliases: string[];
}

export interface PlaygroundState {
  datasource: string | null;
  version: string | null;
  nlPrompt: string;
  mode: "exploratory" | "strict" | "CRUD";
  semanticQuery: SemanticQuery | null;
  generatedSQL: string | null;
  results: QueryExecutionResponse | null;
  loading: {
    planner: boolean;
    executor: boolean;
    runner: boolean;
  };
  errors: {
    planner?: string;
    executor?: string;
    runner?: string;
  };
}
