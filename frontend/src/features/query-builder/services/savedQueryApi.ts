/**
 * Saved Query API service - persists a reusable, parameterized QueryDef
 * (see backend/internal/querybuilder/saved_query_handler.go) and exposes it
 * as its own REST endpoint (GET .../preview, with `?<paramName>=value`
 * overrides) that both this query builder's own "run" and any external
 * caller (or a Page Studio widget's "use a saved query" binding) can hit.
 */

import { apiFetch } from '../../../lib/apiClient';
import type { SavedQuery, SavedQueryChartType, SavedQueryState } from '../types/queryDef';

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await apiFetch(path, {
    headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
    ...init,
  });
  if (!response.ok) {
    let detail = '';
    try {
      const body = await response.json();
      detail = body.error || body.details || JSON.stringify(body);
    } catch {
      detail = await response.text().catch(() => '');
    }
    throw new Error(`${response.status} ${response.statusText}${detail ? `: ${detail}` : ''}`);
  }
  if (response.status === 204) return undefined as unknown as T;
  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) return response.json() as Promise<T>;
  return (await response.text()) as unknown as T;
}

export interface SavedQueryInput {
  name: string;
  description?: string;
  boId: string;
  bindingId?: string;
  chartType: SavedQueryChartType;
  state: SavedQueryState;
  tags?: string[];
}

export async function listSavedQueries(boId?: string): Promise<SavedQuery[]> {
  const qs = boId ? `?boId=${encodeURIComponent(boId)}` : '';
  const data = await fetchJSON<{ savedQueries: SavedQuery[] }>(`/api/explorer/saved-queries${qs}`);
  return data.savedQueries || [];
}

export async function getSavedQuery(id: string): Promise<SavedQuery> {
  return fetchJSON<SavedQuery>(`/api/explorer/saved-queries/${encodeURIComponent(id)}`);
}

export async function createSavedQuery(input: SavedQueryInput): Promise<SavedQuery> {
  return fetchJSON<SavedQuery>('/api/explorer/saved-queries', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export async function updateSavedQuery(id: string, input: SavedQueryInput): Promise<SavedQuery> {
  return fetchJSON<SavedQuery>(`/api/explorer/saved-queries/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  });
}

export async function deleteSavedQuery(id: string): Promise<void> {
  await fetchJSON<void>(`/api/explorer/saved-queries/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export interface SavedQueryRunResult {
  columns: { name: string; type: string }[];
  rows: Record<string, unknown>[];
  rowCount: number;
  chartType: SavedQueryChartType;
  name: string;
}

/**
 * Runs a saved query - this is exactly the same request an external REST
 * consumer or a Page Studio widget makes (GET .../preview?<param>=value),
 * so "preview inside the builder" and "actually use this query" always
 * behave identically.
 */
export async function runSavedQuery(id: string, params?: Record<string, string | number | boolean>): Promise<SavedQueryRunResult> {
  const qs = params
    ? '?' + new URLSearchParams(Object.entries(params).map(([k, v]) => [k, String(v)])).toString()
    : '';
  return fetchJSON<SavedQueryRunResult>(`/api/explorer/saved-queries/${encodeURIComponent(id)}/preview${qs}`);
}

/** The stable REST URL for a saved query, for display/copy in the builder UI. */
export function savedQueryRestUrl(id: string): string {
  return `${window.location.origin}/api/explorer/saved-queries/${id}/preview`;
}
