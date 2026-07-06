/**
 * Alpha Query Builder API service.
 *
 * This service consumes the QueryDef contract. It never constructs SQL.
 * All semantic resolution, binding traversal, join generation, dialect
 * translation, and tenant safety happen on the backend.
 */

import { apiFetch } from '../../../lib/apiClient';
import type {
  QueryDef,
  SemanticTermView,
  BindingView,
  PreviewResult,
  QueryExecuteResult,
} from '../types/queryDef';
import { installQueryBuilderMock } from './queryBuilderMock';

// In development, install a lightweight mock so the feature is demoable
// without the backend endpoints being live.
if (import.meta.env.DEV) {
  installQueryBuilderMock();
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await apiFetch(path, {
    headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
    ...init,
  });

  if (!response.ok) {
    let detail = '';
    try {
      const body = await response.json();
      detail = body.error || body.message || JSON.stringify(body);
    } catch {
      try {
        detail = await response.text();
      } catch {
        detail = '';
      }
    }
    throw new Error(`${response.status} ${response.statusText}${detail ? `: ${detail}` : ''}`);
  }

  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    return response.json() as Promise<T>;
  }
  return (await response.text()) as unknown as T;
}

/**
 * Fetch the bindings available for a Business Object.
 */
export async function fetchBusinessObjectBindings(boId: string): Promise<BindingView[]> {
  const data = await fetchJSON<unknown>(`/api/business-objects/${encodeURIComponent(boId)}/bindings`);
  const raw = Array.isArray(data) ? data : (data as any)?.bindings || [];

  return raw.map((b: any): BindingView => ({
    bindingId: b.bindingId || b.bo_binding_id || b.id || '',
    bindingName: b.bindingName || b.binding_name || b.name || 'Binding',
    backendId: b.backendId || b.backend_id || '',
    backendName: b.backendName || b.backend_name || 'Backend',
    drivingTableId: b.drivingNodeId || b.driving_node_id || b.driving_table_id,
    drivingTableName: b.drivingTableName || b.driving_table_name,
    isDefault: b.isDefault ?? b.is_default ?? false,
  }));
}

/**
 * Fetch semantic terms that are valid for the active binding.
 *
 * The backend should only return terms whose field_binding is RESOLVED for
 * the given bindingId.
 */
export async function fetchBOTerms(
  boId: string,
  bindingId: string
): Promise<SemanticTermView[]> {
  const params = new URLSearchParams({ bindingId });
  const data = await fetchJSON<unknown>(
    `/api/business-objects/${encodeURIComponent(boId)}/terms?${params.toString()}`
  );
  const raw = Array.isArray(data) ? data : (data as any)?.terms || (data as any)?.items || [];

  return raw.map((t: any): SemanticTermView => ({
    termNodeId: t.termNodeId || t.id || t.node_id || '',
    termKey: t.termKey || t.term_key || t.key || '',
    termName: t.termName || t.term_name || t.name || t.node_name || '',
    displayName: t.displayName || t.display_name || t.termName || t.name || t.node_name || '',
    description: t.description,
    dataType: t.dataType || t.data_type || t.type || 'text',
    role: normalizeRole(t.role),
    bindingStatus: t.bindingStatus || t.binding_status || 'UNRESOLVED',
    defaultAggregation: t.defaultAggregation || t.default_aggregation,
  }));
}

function normalizeRole(role: unknown): SemanticTermView['role'] {
  const r = String(role || 'DIMENSION').toUpperCase();
  if (r === 'MEASURE') return 'MEASURE';
  if (r === 'CALCULATED') return 'CALCULATED';
  return 'DIMENSION';
}

/**
 * Ask the backend to translate the QueryDef into SQL without executing it.
 */
export async function previewQuery(queryDef: QueryDef): Promise<PreviewResult> {
  return fetchJSON<PreviewResult>('/api/query/preview', {
    method: 'POST',
    body: JSON.stringify(queryDef),
  });
}

/**
 * Execute the QueryDef and return results plus the SQL that was run.
 */
export async function executeQuery(queryDef: QueryDef): Promise<QueryExecuteResult> {
  return fetchJSON<QueryExecuteResult>('/api/query/execute', {
    method: 'POST',
    body: JSON.stringify(queryDef),
  });
}
