import apiClient from '../../utils/apiClient';

// Mirrors backend/internal/lakehouse/registry and
// backend/internal/handlers/system_lakehouse_handler.go. All routes are global-admin
// only and name the target tenant in the path; the tenant the user is "operating as"
// is irrelevant to them.

export type LifecycleState = 'unconfigured' | 'provisioning' | 'active' | 'suspended' | 'offboarding' | 'offboarded';

export interface LakehouseConfig {
  tenant_id: string;
  tenant_name?: string;
  tenant_code?: string;
  configured: boolean;
  warehouse_name?: string;
  bucket?: string;
  /** null until set. There is no default and it can only be extended. */
  audit_retention_days: number | null;
  lifecycle_state: LifecycleState;
  provisioned: boolean;
  version?: number;
  updated_at?: string;
}

export interface LakehouseAuditEntry {
  id: number;
  at: string;
  actor_id: string;
  actor_role: string;
  action: string;
  before?: { audit_retention_days?: number } | null;
  after?: { audit_retention_days?: number } | null;
  prev_hash: string;
  hash: string;
}

export interface LakehouseList {
  items: LakehouseConfig[];
  total: number;
  limit: number;
  offset: number;
}

export interface LakehouseAudit {
  entries: LakehouseAuditEntry[];
  chain_intact: boolean;
  first_broken_id?: number;
}

/**
 * apiClient throws `API Error: <status> <text> - <body>`. The body is the API's own
 * `{code, error}`; an admin should read the error, not the envelope.
 */
export function apiMessage(err: unknown): string {
  const raw = err instanceof Error ? err.message : String(err);
  const at = raw.indexOf(' - {');
  if (at >= 0) {
    try {
      const body = JSON.parse(raw.slice(at + 3)) as { error?: string };
      if (body.error) return body.error;
    } catch {
      /* not JSON: fall through to the raw text */
    }
  }
  return raw;
}

async function call<T>(path: string, init?: RequestInit): Promise<T> {
  try {
    return await apiClient<T>(path, init);
  } catch (err) {
    throw new Error(apiMessage(err));
  }
}

const tenantPath = (tenantId: string) => `/system/tenants/${encodeURIComponent(tenantId)}/lakehouse`;

export const systemLakehouseApi = {
  list: (p: { q?: string; limit?: number; offset?: number } = {}) => {
    const qs = new URLSearchParams();
    if (p.q) qs.set('q', p.q);
    if (p.limit) qs.set('limit', String(p.limit));
    if (p.offset) qs.set('offset', String(p.offset));
    return call<LakehouseList>(`/system/lakehouses?${qs}`);
  },
  get: (tenantId: string) => call<LakehouseConfig>(tenantPath(tenantId)),
  setRetention: (tenantId: string, days: number) =>
    call<LakehouseConfig>(tenantPath(tenantId), { method: 'PUT', body: JSON.stringify({ audit_retention_days: days }) }),
  provision: (tenantId: string) =>
    call<{ workflow_id: string; tenant_id: string }>(`${tenantPath(tenantId)}/provision`, { method: 'POST', body: '{}' }),
  audit: (tenantId: string, limit = 100) => call<LakehouseAudit>(`${tenantPath(tenantId)}/audit?limit=${limit}`),
};
