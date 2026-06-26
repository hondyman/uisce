import JSONValue from '../../../types/json';

export interface FieldChange {
  path: string;
  before: JSONValue;
  after: JSONValue;
  rule_id?: string;
  provenance?: string;
}

export interface ModelDiff {
  model: string;
  change_type: 'added' | 'removed' | 'modified';
  field_changes: FieldChange[];
}

export interface DiffSummary {
  added: number;
  removed: number;
  modified: number;
}

export interface CompareResponse {
  diff_id: string;
  diff_summary: DiffSummary;
  diff_details: ModelDiff[];
  groups?: Record<string, RuleGroup>;
}

export interface RuleGroup {
  rule_id: string;
  changes: FieldChange[];
}

export interface SemanticElement {
  id: string;
  name: string;
  title: string;
  sourceTable: string;
  sourceColumn: string;
  type: string;
  sql: string;
  public: boolean;
  description: string;
  meta?: Record<string, JSONValue>;
  relationship?: string; // For joins
  primary_key?: boolean; // For dimensions
}

export interface SemanticModelPart {
  dimensions: SemanticElement[];
  measures: SemanticElement[];
  joins: SemanticElement[];
  overrides?: {
    dimensions?: Record<string, Partial<SemanticElement>>;
    measures?: Record<string, Partial<SemanticElement>>;
  };
  // Optional cube-level options to pass through to YAML
  options?: CubeOptions;
}

export interface SemanticModelConfig {
  core: SemanticModelPart;
  custom: SemanticModelPart;
}

// Optional cube-level options to be passed through to YAML when present
export interface CubeOptions {
  title?: string;
  description?: string;
  public?: boolean;
  meta?: Record<string, JSONValue>;
  // refresh_key supports a simple SQL string or a structured object
  refresh_key?: string | { sql?: string; every?: string } | Record<string, JSONValue>;
  access_policy?: Record<string, JSONValue>;
  segments?: Array<Record<string, JSONValue>>;
  hierarchies?: Array<Record<string, JSONValue>>;
  sql_alias?: string;
  data_source?: string;
  governance?: {
    pkFields?: string[]; // ordered list for composite PKs
    pkOrigin?: { sourceSystem?: string; columns?: string[] };
    steward?: string;
    pii?: boolean;
    lineage?: string;
    audit_fields?: string[];
    tenantField?: string; // common tenant field to inject into joins
  };
  pre_aggregations?: PreAggregationConfig[];
  extends?: string | string[];
}

export interface ColumnInfo {
  column_name: string;
  data_type: string;
}

export interface TableSchema {
  table_name: string;
  columns: ColumnInfo[];
}

// Mock data for DATABASE_SCHEMA since it's used in ExplorerTab
// It's initialized as empty to avoid providing stale or incorrect mock data.
export const DATABASE_SCHEMA: TableSchema[] = [];

// Types for pre-aggregation admin UI (DB-driven)
export interface ScheduledPreAggregation {
  cube: string;
  name: string;
  scheduled?: string;
  storage?: string;
  last_run?: string;
  refresh_key?: string | null;
  cron_entry_id?: number;
  next_run?: string;
}

export interface JobRun {
  id: number;
  job_id: string;
  started_at: string;
  finished_at?: string;
  success: boolean;
  message?: string;
}

// Lightweight typed shape for a parsed pre-aggregation YAML/config
export interface PreAggregationConfig {
  name?: string;
  type?: string;
  dimensions?: string[];
  measures?: string[];
  storage?: string;
  [k: string]: unknown;
}

export interface PreAggSuggestion {
  id: string;
  model_name: string;
  hit_count: number;
  suggested: PreAggregationConfig;
}

export interface PreAggLog {
  id: string;
  executed_at: string;
  model_name: string;
  measures: string[];
  dimensions: string[];
  time_dimension: string;
  hit_preaggregation: boolean;
  preaggregation_name?: string | null;
}