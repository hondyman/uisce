import apiClient from '../../utils/apiClient';

// Types mirror backend/internal/msgcat (handler.go, store.go).

export type Severity = 'Message' | 'Warning' | 'Error' | 'Fatal';
export const SEVERITIES: Severity[] = ['Message', 'Warning', 'Error', 'Fatal'];
export type Scope = 'core' | 'tenant';

export interface Language {
  code: string;
  name: string;
  rtl?: boolean;
}

export interface MessageSet {
  set_nbr: number;
  name: string;
  module: string;
  type: 'core' | 'custom' | 'client';
  description: string;
}

export interface Entry {
  set_nbr: number;
  message_nbr: number;
  language: string;
  severity: Severity;
  text: string;
  description?: string;
  user_action?: string;
  updated_by?: string;
  updated_at: string;
}

export interface MessageView {
  set_nbr: number;
  message_nbr: number;
  code: string;
  severity: Severity;
  core: Record<string, Entry>;
  tenant: Record<string, Entry>;
  pending: number;
}

export interface Change {
  id: string;
  scope: Scope;
  set_nbr: number;
  message_nbr: number;
  language: string;
  action: 'upsert' | 'delete';
  severity?: Severity;
  text?: string;
  description?: string;
  user_action?: string;
  before?: { severity: Severity; text: string; description?: string; user_action?: string } | null;
  status: 'pending' | 'applied' | 'rejected' | 'withdrawn';
  requested_by: string;
  requested_by_name?: string;
  requested_at: string;
  reason?: string;
  reviewed_by?: string;
  reviewed_by_name?: string;
  reviewed_at?: string;
  review_comment?: string;
  applied_at?: string;
}

export interface ChangeRequest {
  scope: Scope;
  set_nbr: number;
  message_nbr: number;
  language: string;
  action?: 'upsert' | 'delete';
  severity?: Severity;
  text?: string;
  description?: string;
  user_action?: string;
  reason?: string;
}

export interface Me {
  user_id: string;
  tenant_id: string;
  can_edit_core: boolean;
  can_edit_tenant: boolean;
  core_requires_approval: boolean;
  tenant_requires_approval?: boolean;
}

const BASE = '/api/message-catalog';

export const msgcatApi = {
  me: () => apiClient<Me>(`${BASE}/me`),
  languages: () => apiClient<{ languages: Language[]; base: string }>(`${BASE}/languages`),
  sets: () => apiClient<{ sets: MessageSet[] }>(`${BASE}/sets`),
  messages: (params: { set?: number; q?: string }) => {
    const qs = new URLSearchParams();
    if (params.set) qs.set('set', String(params.set));
    if (params.q) qs.set('q', params.q);
    return apiClient<{ messages: MessageView[]; total: number }>(`${BASE}/messages?${qs}`);
  },
  message: (set: number, nbr: number) =>
    apiClient<{ message: MessageView; history: Change[] | null }>(`${BASE}/messages/${set}/${nbr}`),
  changes: (status: 'pending' | 'all' = 'pending') =>
    apiClient<{ changes: Change[] }>(`${BASE}/changes?status=${status}`),
  propose: (changes: ChangeRequest[]) =>
    apiClient<{ changes: Change[]; applied: boolean }>(`${BASE}/changes`, {
      method: 'POST',
      body: JSON.stringify({ changes }),
    }),
  approve: (id: string, comment?: string) =>
    apiClient<Change>(`${BASE}/changes/${id}/approve`, { method: 'POST', body: JSON.stringify({ comment }) }),
  reject: (id: string, comment?: string) =>
    apiClient<Change>(`${BASE}/changes/${id}/reject`, { method: 'POST', body: JSON.stringify({ comment }) }),
  withdraw: (id: string) => apiClient<Change>(`${BASE}/changes/${id}/withdraw`, { method: 'POST', body: '{}' }),
};

// Client-side twins of backend msgcat/params.go, for live checks and preview.
const PLACEHOLDER = /%[1-9]/g;

export function placeholders(text: string): string[] {
  return Array.from(new Set(text.match(PLACEHOLDER) ?? [])).sort();
}

export function samePlaceholders(a: string, b: string): boolean {
  return placeholders(a).join(',') === placeholders(b).join(',');
}

export function formatMessage(text: string, params: string[]): string {
  return text.replace(PLACEHOLDER, (p) => params[Number(p[1]) - 1] ?? '');
}

/** The text a user sees for a message in a language, and where it came from. */
export function effective(
  m: MessageView,
  lang: string,
): { entry: Entry | undefined; source: 'tenant' | 'core' | 'fallback' | 'none' } {
  if (m.tenant[lang]) return { entry: m.tenant[lang], source: 'tenant' };
  if (m.core[lang]) return { entry: m.core[lang], source: 'core' };
  const en = m.tenant.en ?? m.core.en;
  return en ? { entry: en, source: 'fallback' } : { entry: undefined, source: 'none' };
}

/** A date-time in the UI language. */
export function formatWhen(iso: string, lang: string): string {
  try {
    return new Intl.DateTimeFormat(lang, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(iso));
  } catch {
    return iso;
  }
}
