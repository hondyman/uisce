import { apiFetch } from '../../lib/apiClient';

/**
 * Data domains: one global taxonomy shared by every tenant. Any tenant reads it;
 * only core administrators change it (the API enforces that).
 */

/** A domain as the page uses it. */
export interface DataDomainRecord {
  id: string;
  name: string;
  slug: string;
  parent_id: string | null;
  level: number;
  description: string;
}

interface RawDomain {
  id?: unknown;
  name?: unknown;
  slug?: unknown;
  parent_id?: unknown;
  level?: unknown;
  description?: unknown;
}

/** description arrives as a plain string or as a sql.NullString shape. */
function text(v: unknown): string {
  if (typeof v === 'string') return v;
  if (v && typeof v === 'object' && 'String' in v) return String((v as { String: unknown }).String ?? '');
  return '';
}

export function normalizeDomain(raw: RawDomain): DataDomainRecord {
  return {
    id: String(raw.id ?? ''),
    name: String(raw.name ?? ''),
    slug: String(raw.slug ?? ''),
    parent_id: raw.parent_id ? String(raw.parent_id) : null,
    level: typeof raw.level === 'number' ? raw.level : 0,
    description: text(raw.description),
  };
}

async function failOn(res: Response, what: string): Promise<void> {
  if (res.ok) return;
  const body = (await res.text()).trim();
  throw new Error(body || `${what} failed (${res.status})`);
}

export async function listDomains(): Promise<DataDomainRecord[]> {
  const res = await apiFetch('/api/data-domains', { credentials: 'include' });
  await failOn(res, 'Loading domains');
  const json: unknown = await res.json();
  return Array.isArray(json) ? json.map((r) => normalizeDomain(r as RawDomain)) : [];
}

export interface DomainSaveInput {
  /** Set to update; absent to create. */
  id?: string;
  name: string;
  slug: string;
  parent_id: string | null;
  /** Computed by the caller from the parent: a top-level domain is level 1. */
  level: number;
  description: string;
}

export async function saveDomain(input: DomainSaveInput): Promise<void> {
  // The API takes description as a sql.NullString shape.
  const payload = {
    name: input.name,
    slug: input.slug,
    parent_id: input.parent_id,
    level: input.level,
    description: { String: input.description, Valid: input.description.length > 0 },
  };
  const res = await apiFetch(input.id ? `/api/data-domains/${input.id}` : '/api/data-domains', {
    method: input.id ? 'PUT' : 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  await failOn(res, 'Saving the domain');
}

export async function deleteDomain(id: string): Promise<void> {
  const res = await apiFetch(`/api/data-domains/${id}`, { method: 'DELETE', credentials: 'include' });
  await failOn(res, 'Deleting the domain');
}
