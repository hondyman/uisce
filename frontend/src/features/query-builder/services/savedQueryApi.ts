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

export interface SavedQueryRunResultColumn {
  name: string;
  type: string;
  /**
   * "one" | "many" | "unresolved" | "" - fan-out relative to the primary
   * BO, folded over EVERY hop in this column's join path (see the
   * backend's JoinPath.Analyze/TraversalCardinality doc comments). Empty
   * for a related-BO query's own metadata gap (see hasRelatedBOs on
   * SavedQueryRunResult for the single-BO case, which this field alone
   * cannot distinguish from "unknown").
   */
  cardinality?: string;
  /**
   * "unique" | "shared" | "unresolved" | "" - whether THIS column's value
   * is reachable from more than one row of the primary BO. See the
   * backend's JoinPath.RootOwnership doc comment.
   *
   * This is a PER-COLUMN fact and answers a narrower question than "is it
   * safe to sum this column across the query's returned rows" - see
   * isRowGrainIntact below for why a clean column's own ownership is
   * necessary, not sufficient, for that.
   */
  rootOwnership?: string;
}

export interface SavedQueryRunResult {
  columns: SavedQueryRunResultColumn[];
  rows: Record<string, unknown>[];
  rowCount: number;
  chartType: SavedQueryChartType;
  name: string;
  /**
   * True iff this saved query traverses at least one related BO
   * (RelatedBOIDs non-empty). False means zero joins were involved at
   * all, so no row this query returns can be a duplicate relative to the
   * primary BO - every column is safe to roll up across rows BY
   * CONSTRUCTION, regardless of the (always-empty, for this case)
   * cardinality/rootOwnership fields above, which Preview's single-BO
   * branch never populates.
   *
   * This is what makes isRowGrainIntact answerable for single-BO queries
   * without needing per-column metadata that doesn't exist for them: the
   * fact isn't "what is this column's grain," it's "did this query join
   * to anything at all," which the backend already knows at the handler
   * seam and this field states directly instead of asking the frontend
   * to infer it from an absence.
   */
  hasRelatedBOs?: boolean;
}

/**
 * True iff NO join in this query's result fans out - i.e. every selected
 * column's Cardinality is "one" (or the query has zero related-BO joins
 * at all, per hasRelatedBOs). This is necessarily a QUERY-level fact, not
 * a per-column one: a column that is itself a clean root-owned 1:1
 * lookup can still ride alongside a DIFFERENT column whose join fans out,
 * and every row this query returns is still duplicated relative to root
 * because of THAT join - the grain belongs to the result, not to any
 * single column being summed. See the backend's RootOwnership doc
 * comment for the worked example (a `sku -1:1-> profile` measure next to
 * a `sku -M:1-> category -1:M-> subcategory` label column).
 *
 * Absence of a column's cardinality (empty/undefined) is treated as
 * UNKNOWN, hence unsafe - unless hasRelatedBOs is explicitly false, which
 * proves safety by construction (zero joins, nothing to fan out) without
 * needing per-column metadata Preview's single-BO branch doesn't produce.
 */
export function isRowGrainIntact(result: Pick<SavedQueryRunResult, 'columns' | 'hasRelatedBOs'>): boolean {
  if (result.hasRelatedBOs === false) return true;
  return result.columns.every((c) => c.cardinality === 'one');
}

/**
 * True iff a specific column is known-safe to SUM or AVERAGE across a
 * saved query's returned rows. This is the conjunction of TWO
 * independent conditions, both required:
 *
 *   1. isRowGrainIntact(result) - the QUERY's row grain is intact (no
 *      join anywhere in the result fans out). Necessary because a clean
 *      column can still sit next to a fanning-out one in the same
 *      result - see isRowGrainIntact's doc comment.
 *   2. This column's OWN ownership is explicitly "unique" (or the query
 *      has zero related-BO joins at all) - necessary because even with
 *      intact row grain, a shared-ownership column (e.g. a region name
 *      reachable from many orders) still double-counts if summed across
 *      root rows - a completely different hazard from fan-out.
 *
 * NOT covered here, deliberately - a third, independent condition: is
 * `+` even a meaningful way to combine this column's values across rows
 * at all (aggregation LINEARITY). A column holding MIN(cost) or
 * COUNT(DISTINCT sku) per group has correct per-row values regardless of
 * grain/ownership, but summing THOSE values with `+` across rows is
 * still wrong (a sum of minimums, or a double-counted distinct count).
 * This function has no way to check that today - SavedQueryRunResult
 * doesn't carry the underlying aggregation - so a caller must not read
 * "true" from here as "any `+`-based rollup of this column is fine."
 * See the backend's OwnershipFinding.AtRisk doc comment for the same
 * three-way split on the scanner side.
 */
export function isSafeToRollUpAcrossRows(
  result: Pick<SavedQueryRunResult, 'columns' | 'hasRelatedBOs'>,
  column: { cardinality?: string; rootOwnership?: string },
): boolean {
  if (!isRowGrainIntact(result)) return false;
  if (result.hasRelatedBOs === false) return true;
  return column.rootOwnership === 'unique';
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
