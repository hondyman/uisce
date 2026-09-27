import apiClient from '../../utils/apiClient';

// Types mirror backend/internal/mastering (engine.go, store.go).

export interface Profile {
  id: string;
  entity_cd: string;
  display_name: string;
  bo_key: string;
  anchor_table: string;
  inherited: boolean;
  is_active: boolean;
}

export interface Counts {
  records: number; valid: number; invalid: number;
  xref: number; deterministic: number; fuzzy: number; review: number; new: number; conflicts: number;
  published: number; held_for_review: number; unchanged: number; exceptions: number;
}

export type RunStatus = 'RUNNING' | 'COMPLETED' | 'PARTIAL' | 'FAILED';

export interface Run {
  id: string;
  entity_cd: string;
  load_run_id?: string;
  idempotency_key: string;
  trigger: 'manual' | 'schedule' | 'external';
  status: RunStatus;
  stage: string;
  counts: Partial<Counts>;
  error_code?: string;
  error_detail?: string;
  started_by?: string;
  started_at: string;
  finished_at?: string;
  replayed?: boolean;
}

export interface Issue { code: string; severity: 'ERROR' | 'WARNING'; attribute?: string; message: string }
export interface Preview { counts: Counts; exceptions: Issue[]; unmastered_fields?: string[] }

export type GoldenStatus = 'PUBLISHED' | 'REVIEW' | 'SUPERSEDED' | 'DRAFT' | 'RETRACTED';

export interface GoldenSummary {
  id: string;
  code: string;
  name?: string;
  version: number;
  status: GoldenStatus;
  is_current: boolean;
  dq_score?: number;
  identity_confidence?: number;
  sources: number;
  winning_sources: Record<string, string>;
  updated_at: string;
}

export interface Candidate { source_id: string; source: string; source_key: string; value: unknown; as_of: string; stale?: boolean }

export interface GoldenDetail {
  id: string;
  code: string;
  versions: { id: string; version: number; status: GoldenStatus; is_current: boolean; attributes: Record<string, unknown>;
    winning_sources: Record<string, string>; dq_score?: number; identity_confidence?: number; knowledge_at: string; published_at?: string }[];
  fields: { name: string; value?: string; source?: string; source_key?: string; confidence?: number }[];
  sources: { source: string; source_key: string; method: string; score?: number; rule?: string; matched_keys: string; status: string; updated_at: string }[];
  identifiers: { type: string; value: string; is_primary: boolean; source?: string }[];
  exceptions: ExceptionRow[];
  decisions: { version: number; field: string; value?: string; source?: string; competing: Candidate[]; reason?: string }[];
}

export interface ExceptionRow {
  id: string;
  golden_id?: string;
  source_key?: string;
  type: string;
  severity: 'ERROR' | 'WARNING' | 'INFO';
  description: string;
  source?: string;
  status: 'OPEN' | 'IN_REVIEW' | 'RESOLVED' | 'WAIVED';
  detected_at: string;
}

export interface MatchCandidate {
  id: string;
  a: string; a_code?: string; a_name?: string;
  b: string; b_code?: string; b_name?: string;
  score: number;
  rule?: string;
  matched_keys: string[];
  status: string;
}

export interface CandidateDecision { merge: boolean; keep?: 'a' | 'b'; note?: string }
export interface DecisionResult {
  candidate_id: string;
  status: 'APPROVED' | 'REJECTED';
  survivor_id?: string;
  merged_id?: string;
  moved: { sources: number; identifiers: number };
  published: boolean;
}

export interface Load {
  id: string;
  source: string;
  run_ref?: string;
  file_name?: string;
  status?: string;
  received_rows?: number;
  started_at?: string;
  mastering_run_id?: string;
  mastering_status?: RunStatus;
}

export interface RunRequest { staging_table: string; load_run_id: string; idempotency_key?: string; dry_run?: boolean }

const BASE = '/api/mastering';
const post = (body: unknown) => ({ method: 'POST', body: JSON.stringify(body), headers: { 'Content-Type': 'application/json' } });
const qs = (p: Record<string, string | number | undefined>) => {
  const s = new URLSearchParams();
  Object.entries(p).forEach(([k, v]) => { if (v !== undefined && v !== '') s.set(k, String(v)); });
  const out = s.toString();
  return out ? `?${out}` : '';
};

export const masteringApi = {
  profiles: () => apiClient<{ profiles: Profile[] }>(`${BASE}/profiles`),
  runs: (entity: string) => apiClient<{ runs: Run[] }>(`${BASE}/${entity}/runs`),
  loads: (entity: string) => apiClient<{ loads: Load[] }>(`${BASE}/${entity}/loads`),
  preview: (entity: string, r: RunRequest) => apiClient<{ preview: Preview }>(`${BASE}/${entity}/runs`, post({ ...r, dry_run: true })),
  run: (entity: string, r: RunRequest) => apiClient<{ run: Run }>(`${BASE}/${entity}/runs`, post(r)),
  golden: (entity: string, f: { q?: string; status?: string }) => apiClient<{ golden: GoldenSummary[] }>(`${BASE}/${entity}/golden${qs(f)}`),
  goldenById: (entity: string, id: string) => apiClient<{ golden: GoldenDetail }>(`${BASE}/${entity}/golden/${id}`),
  exceptions: (entity: string, status?: string) => apiClient<{ exceptions: ExceptionRow[] }>(`${BASE}/${entity}/exceptions${qs({ status })}`),
  resolve: (entity: string, id: string, status: 'RESOLVED' | 'WAIVED', note?: string) =>
    apiClient<{ ok: boolean }>(`${BASE}/${entity}/exceptions/${id}/resolve`, post({ status, note })),
  candidates: (entity: string) => apiClient<{ candidates: MatchCandidate[] }>(`${BASE}/${entity}/candidates`),
  decide: (entity: string, id: string, d: CandidateDecision) =>
    apiClient<{ decision: DecisionResult }>(`${BASE}/${entity}/candidates/${id}/decide`, post(d)),
};

export const pct = (v?: number) => (v === undefined || v === null ? '—' : `${Math.round(v * 100)}%`);
