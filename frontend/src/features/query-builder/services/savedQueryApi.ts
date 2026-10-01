/**
 * Saved Query API service - persists a reusable, parameterized QueryDef
 * (see backend/internal/querybuilder/saved_query_handler.go) and exposes it
 * as its own REST endpoint (GET .../preview, with `?<paramName>=value`
 * overrides) that both this query builder's own "run" and any external
 * caller (or a Page Studio widget's "use a saved query" binding) can hit.
 */

import { apiFetch } from '../../../lib/apiClient';
import type { SavedQuery, SavedQueryChartType, SavedQueryState, SavedQueryFolder } from '../types/queryDef';

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
  /** Editable after creation. boId/bindingId are locked once the query
   * exists and are ignored by the backend on update. */
  relatedBoIds?: string[];
  chartType: SavedQueryChartType;
  state: SavedQueryState;
  tags?: string[];
  folderId?: string;
}

export async function listSavedQueries(opts?: { boId?: string; folderId?: string }): Promise<SavedQuery[]> {
  const params = new URLSearchParams();
  if (opts?.boId) params.set('boId', opts.boId);
  if (opts?.folderId) params.set('folderId', opts.folderId);
  const qs = params.toString() ? `?${params.toString()}` : '';
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

export interface SavedQueryReference {
  type: 'page_draft' | 'page_published' | 'drill_through_target' | 'core_adoption';
  id?: string;
  name?: string;
  location?: string;
  version?: number;
  sourceQueryId?: string;
  target?: string;
  tenantId?: string;
  count?: number;
}

export interface SavedQueryUsageReport {
  inUse: boolean;
  references: SavedQueryReference[];
}

export async function getSavedQueryUsage(id: string): Promise<SavedQueryUsageReport> {
  return fetchJSON<SavedQueryUsageReport>(`/api/explorer/saved-queries/${encodeURIComponent(id)}/usage`);
}

export async function patchSavedQueryStatus(
  id: string,
  status: 'active' | 'deprecated' | 'archived'
): Promise<SavedQuery> {
  return fetchJSON<SavedQuery>(`/api/explorer/saved-queries/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify({ status }),
  });
}

export async function cloneSavedQuery(id: string): Promise<SavedQuery> {
  return fetchJSON<SavedQuery>(`/api/explorer/saved-queries/${encodeURIComponent(id)}/clone`, { method: 'POST' });
}

export async function setSavedQueryFavorite(id: string, isFavorite: boolean): Promise<SavedQuery> {
  return fetchJSON<SavedQuery>(`/api/explorer/saved-queries/${encodeURIComponent(id)}/favorite`, {
    method: 'PUT',
    body: JSON.stringify({ isFavorite }),
  });
}

export async function setSavedQueryVisibility(id: string, visibility: 'private' | 'shared'): Promise<SavedQuery> {
  return fetchJSON<SavedQuery>(`/api/explorer/saved-queries/${encodeURIComponent(id)}/share`, {
    method: 'POST',
    body: JSON.stringify({ visibility }),
  });
}

// --- Core Lifecycle ---

export interface CoreGroupChange {
  path: Array<{ kind: string; value?: string }>;
  op: 'add' | 'remove' | 'change' | 'reorder';
  old?: unknown;
  new?: unknown;
}

export interface CoreGroup {
  id: string;
  kind: string;
  label: string;
  changes: CoreGroupChange[];
}

export interface QueryComparisonResult {
  baseVersion: number;
  coreVersion: number;
  upgradeAvailable: boolean;
  customizations: CoreGroup[];
  coreUpdates: CoreGroup[];
  conflicts?: string[];
}

export async function extendSavedQuery(id: string, input: SavedQueryInput): Promise<SavedQuery> {
  return fetchJSON<SavedQuery>(`/api/explorer/saved-queries/${encodeURIComponent(id)}/extend`, {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export async function compareSavedQuery(id: string): Promise<QueryComparisonResult> {
  return fetchJSON<QueryComparisonResult>(`/api/explorer/saved-queries/${encodeURIComponent(id)}/compare`);
}

export async function upgradeSavedQuery(id: string, opts?: { remove?: string[]; dryRun?: boolean }): Promise<{ dryRun?: boolean; preview?: SavedQuery } | SavedQuery> {
  const qs = opts?.dryRun ? '?dryRun=true' : '';
  return fetchJSON(`/api/explorer/saved-queries/${encodeURIComponent(id)}/upgrade${qs}`, {
    method: 'POST',
    body: JSON.stringify({ remove: opts?.remove || [] }),
  });
}

export async function revertSavedQuery(id: string): Promise<SavedQuery> {
  return fetchJSON<SavedQuery>(`/api/explorer/saved-queries/${encodeURIComponent(id)}/revert`, {
    method: 'POST',
  });
}

// --- Folders ---

export async function listSavedQueryFolders(): Promise<SavedQueryFolder[]> {
  const data = await fetchJSON<{ folders: SavedQueryFolder[] }>('/api/explorer/saved-query-folders');
  return data.folders || [];
}

export async function createSavedQueryFolder(name: string, parentId?: string): Promise<SavedQueryFolder> {
  return fetchJSON<SavedQueryFolder>('/api/explorer/saved-query-folders', {
    method: 'POST',
    body: JSON.stringify({ name, parentId }),
  });
}

export async function renameSavedQueryFolder(id: string, name: string, parentId?: string): Promise<SavedQueryFolder> {
  return fetchJSON<SavedQueryFolder>(`/api/explorer/saved-query-folders/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify({ name, parentId }),
  });
}

export async function deleteSavedQueryFolder(id: string): Promise<void> {
  await fetchJSON<void>(`/api/explorer/saved-query-folders/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export interface SavedQueryRunResult {
  columns: { name: string; type: string }[];
  rows: Record<string, unknown>[];
  rowCount: number;
  chartType: SavedQueryChartType;
  name: string;
}

export interface RuntimeFilter {
  termNodeId: string;
  operator: string;
  value: unknown;
}

export interface ExecuteSavedQueryOptions {
  params?: Record<string, string | number | boolean>;
  runtimeFilters?: RuntimeFilter[];
  format?: 'json' | 'csv';
}

/**
 * Executes a saved query via POST /api/explorer/saved-queries/{id}/execute
 * allowing runtime dynamic filters (cross-filtering, drill-down) and format options.
 */
export async function executeSavedQuery(
  id: string,
  opts?: ExecuteSavedQueryOptions
): Promise<SavedQueryRunResult> {
  return fetchJSON<SavedQueryRunResult>(`/api/explorer/saved-queries/${encodeURIComponent(id)}/execute`, {
    method: 'POST',
    body: JSON.stringify({
      params: opts?.params,
      runtimeFilters: opts?.runtimeFilters,
      format: opts?.format || 'json',
    }),
  });
}

/**
 * Runs a saved query - this is exactly the same request an external REST
 * consumer or a Page Studio widget makes (GET .../preview?<param>=value),
 * so "preview inside the builder" and "actually use this query" always
 * behave identically.
 */
export async function runSavedQuery(
  id: string,
  params?: Record<string, string | number | boolean>,
  runtimeFilters?: RuntimeFilter[]
): Promise<SavedQueryRunResult> {
  if (runtimeFilters && runtimeFilters.length > 0) {
    return executeSavedQuery(id, { params, runtimeFilters });
  }
  const qs = params
    ? '?' + new URLSearchParams(Object.entries(params).map(([k, v]) => [k, String(v)])).toString()
    : '';
  return fetchJSON<SavedQueryRunResult>(`/api/explorer/saved-queries/${encodeURIComponent(id)}/preview${qs}`);
}

/** The stable REST URL for a saved query, for display/copy in the builder UI. */
export function savedQueryRestUrl(id: string): string {
  return `${window.location.origin}/api/explorer/saved-queries/${id}/preview`;
}

export interface BatchExecuteQueryItem {
  id: string;
  savedQueryId: string;
  params?: Record<string, unknown>;
  runtimeFilters?: RuntimeFilter[];
  limit?: number;
  routeTier?: 'hot' | 'warm' | 'cold';
}

export interface BatchExecuteItemResult {
  id: string;
  savedQueryId: string;
  columns?: { name: string; type: string }[];
  rows?: Record<string, unknown>[];
  rowCount?: number;
  resolvedTier?: string;
  mvHit?: boolean;
  cacheHit?: boolean;
  durationMs?: number;
  error?: string;
}

export interface BatchExecuteSavedQueryResponse {
  results: BatchExecuteItemResult[];
  totalDurationMs: number;
}

/**
 * Executes a batch of saved queries in parallel via POST /api/query/batch-execute
 * with server-side singleflight coalescing and watermark result caching.
 */
export async function batchExecuteSavedQueries(
  queries: BatchExecuteQueryItem[]
): Promise<BatchExecuteSavedQueryResponse> {
  return fetchJSON<BatchExecuteSavedQueryResponse>('/api/query/batch-execute', {
    method: 'POST',
    body: JSON.stringify({ queries }),
  });
}

