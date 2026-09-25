import apiClient from '../../utils/apiClient';

// Mirrors backend/internal/datapipeline (spec.go and friends).

export type NodeKind =
  | 'file_source' | 'bo_source' | 'validate' | 'rule_check' | 'map'
  | 'bo_sink' | 'staging_sink' | 'file_sink';

export type ColumnType = 'string' | 'int' | 'float' | 'decimal' | 'bool' | 'date' | 'timestamp';

export interface Column { name: string; type: ColumnType; nullable?: boolean; enum?: string[] }
export interface Condition { field: string; operator: string; value?: unknown }
export interface FieldMap { from: string; to: string; transform?: string; lookup?: Record<string, string>; default?: string }

export interface NodeConfigs {
  file_source: { uri: string; format: 'csv' | 'json' | 'parquet'; delimiter?: string; has_header?: boolean; columns?: Column[] };
  bo_source: { bo_key: string; filters?: Condition[]; limit?: number };
  validate: { required?: string[]; unique?: string[] };
  rule_check: { rule_ids: string[]; bo_key?: string /* editor-only: which BO's rules to pick from */ };
  map: { fields: FieldMap[]; keep_unmapped?: boolean };
  bo_sink: { bo_key: string; mode?: 'create' | 'upsert'; key_fields?: string[]; dry_run?: boolean };
  staging_sink: { table: string; source_cd: string; domain: string; run_ref?: string; columns?: Record<string, string> };
  file_sink: { uri: string; format: 'csv' | 'json' | 'parquet'; delimiter?: string };
}

export interface SpecNode<K extends NodeKind = NodeKind> {
  id: string;
  type: K;
  label?: string;
  config: NodeConfigs[K];
  position?: { x: number; y: number };
}
export interface Edge { from: string; to: string }
export interface Spec {
  version: number;
  nodes: SpecNode[];
  edges: Edge[];
  batch_size?: number;
  error_policy?: 'skip_and_log' | 'fail_fast';
}

export interface Definition {
  id: string; name: string; description: string; spec: Spec;
  created_by: string; created_at: string; last_modified_at: string;
}
export interface Issue { node_id?: string; message: string }
export interface NodeType {
  type: NodeKind; label: string; category: 'source' | 'step' | 'destination';
  description: string; available: boolean; unavailable_reason?: string;
}
export interface TargetField { name: string; label?: string; type?: string; required?: boolean }
export interface Suggestion { from: string; to: string; confidence: number; transform?: string; reason: string }
export interface FileEntry { path: string; bytes: number; modified?: number }
export interface StagingTable { table: string; columns: TargetField[] }
export interface NodeStats {
  NodeID: string; Label: string; Type: string; In: number; Out: number;
  Errors: number; Warnings: number; Duration: number; Status: string; Err: string;
}
export interface Row { Num: number; Data: Record<string, unknown> }
export interface PreviewResult {
  summary?: { Nodes: NodeStats[]; RecordsIn: number; RecordsOut: number; Errors: number; samples?: Record<string, Row[]> };
  rejects: { node_id: string; row: number; field?: string; reason: string; kind: 'error' | 'warning' }[];
  error?: string;
}
export interface RunRecord {
  id: string; pipeline_id: string;
  status: 'queued' | 'running' | 'completed' | 'completed_with_errors' | 'failed';
  start_time: string; end_time?: string; records_in: number; records_out: number; errors: number;
  errors_sample: { kind?: string; node_id?: string; row?: number; field?: string; reason?: string; run_error?: string }[];
  steps?: NodeStats[];
}
export interface Schedule { cron: string; timezone?: string; enabled: boolean }
export interface ScheduleView { schedule: Schedule | null; next_runs?: string[] }
export interface BOListItem { id: string; name: string; display_name: string; description?: string }
export interface BOSchemaField { name: string; displayName?: string; type: string; required?: boolean; physicalColumn?: string }
export interface RuleDescriptor { id: string; name: string; description?: string; severity: 'BLOCK' | 'WARN'; is_active: boolean; origin?: string }

const base = '/api/data-pipelines';
const json = (body: unknown): RequestInit => ({ body: JSON.stringify(body), headers: { 'Content-Type': 'application/json' } });

export const pipelinesApi = {
  list: () => apiClient<Definition[]>(`${base}/`),
  get: (id: string) => apiClient<Definition>(`${base}/${id}`),
  create: (d: { name: string; description?: string; spec: Spec }) =>
    apiClient<Definition>(`${base}/`, { method: 'POST', ...json(d) }),
  update: (id: string, d: { name: string; description?: string; spec: Spec }) =>
    apiClient<Definition>(`${base}/${id}`, { method: 'PUT', ...json(d) }),
  remove: (id: string) => apiClient<Response>(`${base}/${id}`, { method: 'DELETE' }),
  validate: (spec: Spec) => apiClient<{ valid: boolean; issues: Issue[] }>(`${base}/validate`, { method: 'POST', ...json(spec) }),
  preview: (spec: Spec, rows = 100) => apiClient<PreviewResult>(`${base}/preview`, { method: 'POST', ...json({ spec, rows }) }),
  nodeTypes: () => apiClient<NodeType[]>(`${base}/node-types`),
  suggestMapping: (source: Column[], targets: TargetField[]) =>
    apiClient<Suggestion[]>(`${base}/suggest-mapping`, { method: 'POST', ...json({ source, targets }) }),
  files: (folder = 'uploads') => apiClient<FileEntry[]>(`${base}/files?folder=${encodeURIComponent(folder)}`),
  upload: (file: File) =>
    apiClient<{ uri: string; bytes: number }>(`${base}/files/upload?name=${encodeURIComponent(file.name)}`, {
      method: 'POST', body: file, headers: { 'Content-Type': 'application/octet-stream' },
    }),
  profile: (p: { uri: string; format?: string; delimiter?: string; has_header?: boolean; count_rows?: boolean }) =>
    apiClient<{ format: string; columns: Column[]; sample: Record<string, unknown>[]; row_count?: number }>(
      `${base}/files/profile`, { method: 'POST', ...json(p) }),
  stagingTables: () => apiClient<StagingTable[]>(`${base}/staging-tables`),
  startRun: (id: string) => apiClient<{ run_id: string; status: string }>(`${base}/${id}/runs`, { method: 'POST' }),
  runs: (id: string) => apiClient<RunRecord[]>(`${base}/${id}/runs`),
  run: (runId: string) => apiClient<RunRecord>(`${base}/runs/${runId}`),
  schedule: (id: string) => apiClient<ScheduleView>(`${base}/${id}/schedule`),
  setSchedule: (id: string, s: Schedule) => apiClient<ScheduleView>(`${base}/${id}/schedule`, { method: 'PUT', ...json(s) }),
};

export const platformApi = {
  // /business-objects returns {key, displayName, name, ...}; the pipeline
  // addresses a BO by its key (the /bo/{boKey} routes).
  businessObjects: async (): Promise<BOListItem[]> => {
    const data = await apiClient<unknown>('/api/business-objects');
    const list = (Array.isArray(data) ? data : Object.values((data as object) ?? {})) as
      { id: string; key?: string; name?: string; displayName?: string; description?: string }[];
    return list.filter(b => b.key).map(b => ({
      id: b.id, name: b.key!, display_name: b.displayName || b.name || b.key!, description: b.description,
    }));
  },
  boSchema: (boKey: string) =>
    apiClient<{ fields: BOSchemaField[] }>(`/api/bo/${encodeURIComponent(boKey)}/schema`),
  rules: async (boName: string) => {
    const data = await apiClient<{ validationRules: RuleDescriptor[] }>(
      `/api/validation-rule-nodes?bo_name=${encodeURIComponent(boName)}`);
    return data.validationRules ?? [];
  },
};

/** A BO field as a mapping target: the BO sink writes physical columns. */
export const boTarget = (f: BOSchemaField): TargetField => ({
  name: f.physicalColumn || f.name,
  label: f.displayName || f.name,
  type: f.type,
  required: f.required,
});
