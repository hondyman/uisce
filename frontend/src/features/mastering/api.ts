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
  /** RECORD (one golden record per entity) or TIMESERIES (golden prices per date). */
  kind?: 'RECORD' | 'TIMESERIES';
}

export const isSeries = (p?: Profile) => p?.kind === 'TIMESERIES';

export interface Counts {
  records: number; valid: number; invalid: number;
  xref: number; deterministic: number; fuzzy: number; review: number; new: number; conflicts: number;
  published: number; held_for_review: number; unchanged: number; exceptions: number;
  /** Observations that replaced a value their source had already reported (prices). */
  restated?: number;
  /** Later dates re-checked because an earlier golden price changed (prices). */
  rechecked?: number;
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

export interface Candidate {
  source_id: string; source: string; source_key: string; value: unknown; as_of: string; stale?: boolean;
  /** Whether it passed the attribute's selection rule (absent: no selection rule). */
  selected?: boolean;
  note?: string;
}

export interface GoldenDetail {
  id: string;
  code: string;
  name?: string;
  /** The version the fields and decisions are for. */
  selected_version: number;
  versions: { id: string; version: number; status: GoldenStatus; is_current: boolean; attributes: Record<string, unknown>;
    winning_sources: Record<string, string>; dq_score?: number; identity_confidence?: number; knowledge_at: string; published_at?: string }[];
  fields: { name: string; value?: string; source?: string; source_key?: string; confidence?: number }[];
  sources: { source: string; source_key: string; method: string; score?: number; rule?: string; matched_keys: string; status: string; updated_at: string }[];
  identifiers: { type: string; value: string; is_primary: boolean; source?: string }[];
  exceptions: ExceptionRow[];
  decisions: { version: number; field: string; value?: string; source?: string; competing: Candidate[]; reason?: string; rule_id?: string }[];
  /** Attribute -> the business object field (semantic term) mapped to it. */
  terms?: Record<string, string>;
  /** Survivorship rule id -> label. */
  rules?: Record<string, string>;
}

export interface PriceSummary {
  id: string; entity_id: string; code?: string; name?: string;
  price_type: string; date: string; value: number; currency?: string; winner?: string; sources: number;
  variance_pct?: number; change_pct?: number; status: GoldenStatus; version: number; is_current: boolean; is_stale: boolean;
  confidence?: number; updated_at: string;
}

export interface PriceList { date: string; dates: string[]; prices: PriceSummary[] }

/** One source's quote as survivorship saw it (compact provenance). */
export interface PriceCandidate {
  source: string; value: number; currency?: string; as_of?: string; rank?: number; stale?: boolean;
  selected?: boolean; note?: string; excluded?: string; diff_pct?: number;
}

export interface PriceProvenance {
  price_type: string; strategy: string; reason: string; rule_id?: string; ranking: string[]; candidates: PriceCandidate[];
  threshold: { type?: string; warning?: number; error?: number; critical?: number };
  controls?: { control: string; level: string; action: string; pct: number }[] | null;
  prior?: { date: string; value: number }; change_pct?: number; held?: boolean;
  instrument?: { asset_class?: string; sub_asset_class?: string; currency?: string };
}

export interface PriceVersion {
  id: string; version: number; value: number; currency?: string; winner?: string; status: GoldenStatus; is_current: boolean;
  is_stale: boolean; confidence?: number; dq_score?: number; variance_pct?: number; knowledge_at: string; provenance: PriceProvenance;
}

export interface VarianceEvent {
  id: string; source_a?: string; source_b?: string; price_a: number; price_b: number; variance_pct: number; severity: string; status: string;
}

export interface PriceDetail {
  id: string; entity_id: string; code?: string; name?: string; price_type: string; date: string;
  versions: PriceVersion[]; exceptions: ExceptionRow[]; variances: VarianceEvent[];
}

export interface Completeness {
  date: string; expected: number; priced: number; held: number; missing: number; stale: number; raised: number; resolved: number;
  gaps: { entity_id: string; code?: string; name?: string; price_type: string; held: boolean }[];
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
  /** A merge of this pair waiting for approval. */
  merge_request?: {
    id: string; keep: 'a' | 'b'; note?: string; approvals_required: number; approvals: number;
    requested_by_name?: string; mine: boolean; voted: boolean;
  };
}

export type OverrideMode = 'APPROVAL' | 'DIRECT';
export interface Policy {
  entity_cd: string;
  mode: OverrideMode;
  approvals_required: number;
  high_risk_attributes: string[];
  high_risk_approvals: number;
  updated_by?: string;
  updated_at?: string;
  inherited: boolean;
  attributes?: string[];
}

export type OverrideStatus = 'PENDING' | 'APPLIED' | 'REJECTED' | 'WITHDRAWN';
export interface Override {
  /** What the console opens for it: for a price override, the golden price. */
  open_id?: string;
  id: string;
  golden_id: string;
  golden_code?: string;
  golden_name?: string;
  attribute: string;
  action: 'SET' | 'CLEAR';
  value?: unknown;
  previous_value?: unknown;
  reason: string;
  mode: OverrideMode;
  approvals_required: number;
  approvals: number;
  status: OverrideStatus;
  active: boolean;
  requested_by_name?: string;
  requested_at: string;
  applied_at?: string;
  applied_version?: number;
  voters?: string;
  mine: boolean;
  voted: boolean;
}
export interface OverrideRequest { attribute: string; action?: 'SET' | 'CLEAR'; value?: unknown; reason: string }

export interface CandidateDecision { merge: boolean; keep?: 'a' | 'b'; note?: string }
export interface DecisionResult {
  candidate_id: string;
  status: 'APPROVED' | 'REJECTED' | 'PENDING_APPROVAL' | 'MERGE_REJECTED';
  merge_request_id?: string;
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
  getRun: (entity: string, id: string) => apiClient<{ run: Run }>(`${BASE}/${entity}/runs/${id}`),
  /** Starts a run and follows it to the end (runs continue server-side regardless). */
  runToEnd: async (entity: string, r: RunRequest, onProgress?: (run: Run) => void) => {
    let { run } = await masteringApi.run(entity, r);
    const deadline = Date.now() + 2 * 60 * 60 * 1000;
    while (run.status === 'RUNNING' && Date.now() < deadline) {
      onProgress?.(run);
      await new Promise((res) => setTimeout(res, 2000));
      run = (await masteringApi.getRun(entity, run.id)).run;
    }
    return { run };
  },
  golden: (entity: string, f: { q?: string; status?: string }) => apiClient<{ golden: GoldenSummary[] }>(`${BASE}/${entity}/golden${qs(f)}`),
  prices: (entity: string, f: { date?: string; q?: string; status?: string; price_type?: string }) =>
    apiClient<{ prices: PriceList }>(`${BASE}/${entity}/golden${qs(f)}`),
  completeness: (entity: string, date: string) => apiClient<{ completeness: Completeness }>(`${BASE}/${entity}/completeness`, post({ date })),
  priceById: (entity: string, id: string) => apiClient<{ price: PriceDetail }>(`${BASE}/${entity}/golden/${id}`),
  goldenById: (entity: string, id: string, version?: number) =>
    apiClient<{ golden: GoldenDetail }>(`${BASE}/${entity}/golden/${id}${version ? `?version=${version}` : ''}`),
  exceptions: (entity: string, status?: string) => apiClient<{ exceptions: ExceptionRow[] }>(`${BASE}/${entity}/exceptions${qs({ status })}`),
  resolve: (entity: string, id: string, status: 'RESOLVED' | 'WAIVED', note?: string) =>
    apiClient<{ ok: boolean }>(`${BASE}/${entity}/exceptions/${id}/resolve`, post({ status, note })),
  candidates: (entity: string) => apiClient<{ candidates: MatchCandidate[] }>(`${BASE}/${entity}/candidates`),
  policy: (entity: string) => apiClient<{ policy: Policy; can_edit: boolean }>(`${BASE}/${entity}/policy`),
  setPolicy: (entity: string, p: Partial<Policy>) =>
    apiClient<{ policy: Policy; can_edit: boolean }>(`${BASE}/${entity}/policy`, { method: 'PUT', body: JSON.stringify(p), headers: { 'Content-Type': 'application/json' } }),
  overrides: (entity: string, f: { status?: string; golden?: string }) => apiClient<{ overrides: Override[] }>(`${BASE}/${entity}/overrides${qs(f)}`),
  proposeOverride: (entity: string, goldenId: string, o: OverrideRequest) =>
    apiClient<{ override: Override }>(`${BASE}/${entity}/golden/${goldenId}/overrides`, post(o)),
  voteOverride: (entity: string, id: string, approve: boolean, comment?: string) =>
    apiClient<{ override: Override }>(`${BASE}/${entity}/overrides/${id}/${approve ? 'approve' : 'reject'}`, post({ comment })),
  withdrawOverride: (entity: string, id: string) => apiClient<{ override: Override }>(`${BASE}/${entity}/overrides/${id}/withdraw`, { method: 'POST' }),
  voteMerge: (entity: string, id: string, approve: boolean, comment?: string) =>
    apiClient<{ decision: DecisionResult }>(`${BASE}/${entity}/merges/${id}/${approve ? 'approve' : 'reject'}`, post({ comment })),
  withdrawMerge: (entity: string, id: string) => apiClient<{ ok: boolean }>(`${BASE}/${entity}/merges/${id}/withdraw`, { method: 'POST' }),
  decide: (entity: string, id: string, d: CandidateDecision) =>
    apiClient<{ decision: DecisionResult }>(`${BASE}/${entity}/candidates/${id}/decide`, post(d)),
};

export const pct = (v?: number) => (v === undefined || v === null ? '—' : `${Math.round(v * 100)}%`);

export const showValue = (v: unknown) => (v === undefined || v === null || v === '' ? '—' : typeof v === 'object' ? JSON.stringify(v) : String(v));
