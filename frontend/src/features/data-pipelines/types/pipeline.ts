export type PipelineMode = 'business_object' | 'catalog_graph' | 'hybrid' | 'external';

export type TileCategory = 'source' | 'transform' | 'validator' | 'loader' | 'sink' | 'graph_synthesizer';

export interface PipelineNodeData {
  label: string;
  category: TileCategory;
  subType: string;
  icon?: string;
  description?: string;
  badge?: string;
  config: Record<string, any>;
  metrics?: {
    status?: 'pending' | 'running' | 'completed' | 'failed';
    recordsIn?: number;
    recordsOut?: number;
    recordsError?: number;
    rowsPerSec?: number;
    durationMs?: number;
    errorMessage?: string;
  };
}

export interface PipelineDefinition {
  id: string;
  tenant_id?: string;
  name: string;
  description: string;
  mode: PipelineMode;
  target_entity: string;
  dag_json: {
    nodes: any[];
    edges: any[];
  };
  concurrency: number;
  batch_size: number;
  error_policy: 'fail_fast' | 'skip_and_log' | 'dead_letter';
  is_active: boolean;
  created_at?: string;
  last_modified_at?: string;
}

export interface StepTelemetry {
  node_id: string;
  node_label: string;
  node_type: string;
  records_in: number;
  records_out: number;
  records_error: number;
  bytes_processed: number;
  duration: number; // nanoseconds or duration
  rows_per_sec: number;
  status: 'pending' | 'running' | 'completed' | 'failed';
  error_message?: string;
}

export interface PipelineExecutionRun {
  run_id: string;
  pipeline_id: string;
  tenant_id: string;
  status: 'queued' | 'running' | 'completed' | 'failed' | 'simulated';
  start_time: string;
  end_time?: string;
  total_records_in: number;
  total_records_out: number;
  total_errors: number;
  peak_throughput_rows_sec: number;
  step_telemetry: Record<string, StepTelemetry>;
  error_details?: string[];
  sample_output?: Record<string, any>[];
}

export interface BOSchemaResponse {
  tables: {
    table: string;
    domain: string;
    label: string;
    subtypes: string[];
  }[];
  registry: {
    id: string;
    tenant_id: string;
    root_object: string;
    subtype_code: string;
    subtype_name: string;
    field_allowlist: string[];
  }[];
}

export interface CatalogSchemaResponse {
  is_gold_copy: boolean;
  node_types: {
    type: string;
    label: string;
    icon: string;
  }[];
  edge_types: {
    predicate: string;
    label: string;
  }[];
}
