import apiClient from '../../utils/apiClient';

// Types mirror backend/internal/stagingbind (store.go, editor.go).

/** A staging table's binding to a business object: field -> staging column. */
export interface Binding {
  id: string;
  tenant_id: string;
  bo_key: string;
  bo_name: string;
  staging_table: string;
  fields: Record<string, string>;
  version: number;
  /** core: inherited from the gold copy (read-only here); tenant: this tenant's own. */
  origin: 'core' | 'tenant';
  updated_at: string;
}

export type ChangeStatus = 'pending' | 'applied' | 'rejected' | 'withdrawn';

export interface Change {
  id: string;
  tenant_id: string;
  bo_key: string;
  staging_table: string;
  action: 'upsert' | 'delete';
  fields?: Record<string, string>;
  /** The binding's fields when the change was proposed (none: new binding). */
  before?: Record<string, string>;
  status: ChangeStatus;
  reason?: string;
  requested_by: string;
  requested_by_name?: string;
  requested_at: string;
  reviewed_by?: string;
  reviewed_by_name?: string;
  reviewed_at?: string;
  review_comment?: string;
}

export interface Proposal {
  bo_key: string;
  staging_table: string;
  action?: 'upsert' | 'delete';
  fields?: Record<string, string>;
  reason?: string;
}

const BASE = '/api/staging-bindings';
const json = (body: unknown) => ({ method: 'POST', body: JSON.stringify(body), headers: { 'Content-Type': 'application/json' } });

export const stagingBindingsApi = {
  list: () => apiClient<{ bindings: Binding[] }>(`${BASE}/`),
  changes: (status?: ChangeStatus) =>
    apiClient<{ changes: Change[] }>(`${BASE}/changes${status ? `?status=${status}` : ''}`),
  propose: (p: Proposal) => apiClient<{ change: Change }>(`${BASE}/changes`, json(p)),
  approve: (id: string, comment?: string) => apiClient<{ change: Change }>(`${BASE}/changes/${id}/approve`, json({ comment })),
  reject: (id: string, comment?: string) => apiClient<{ change: Change }>(`${BASE}/changes/${id}/reject`, json({ comment })),
  withdraw: (id: string) => apiClient<{ change: Change }>(`${BASE}/changes/${id}/withdraw`, { method: 'POST' }),
};

/** What changed between the binding a change was proposed on and the proposal, field by field. */
export type FieldDiff = { field: string; before?: string; after?: string; kind: 'added' | 'removed' | 'changed' | 'same' };

export function diffFields(before: Record<string, string> = {}, after: Record<string, string> = {}): FieldDiff[] {
  const names = Array.from(new Set([...Object.keys(before), ...Object.keys(after)])).sort();
  return names.map((field) => {
    const b = before[field];
    const a = after[field];
    const kind: FieldDiff['kind'] = b === undefined ? 'added' : a === undefined ? 'removed' : a === b ? 'same' : 'changed';
    return { field, before: b, after: a, kind };
  });
}
