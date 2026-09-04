/**
 * Alpha Query Builder API service.
 *
 * This service consumes the QueryDef contract. It never constructs SQL.
 * All semantic resolution, binding traversal, join generation, dialect
 * translation, and tenant safety happen on the backend.
 */

import { apiFetch } from '../../../lib/apiClient';
import { devError } from '../../../utils/devLogger';
import type {
  QueryDef,
  SemanticTermView,
  BindingView,
  PreviewResult,
  QueryExecuteResult,
  BOSchema,
} from '../types/queryDef';
import { installQueryBuilderMock } from './queryBuilderMock';

// In development, install a lightweight mock so the feature is demoable
// without the backend endpoints being live.
if (false && import.meta.env.DEV) {
  installQueryBuilderMock();
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await apiFetch(path, {
    headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
    ...init,
  });

  if (!response.ok) {
    let detail = '';
    let errorBody: unknown = null;
    try {
      errorBody = await response.json();
      // .details carries the actual backend error (e.g. why SQL generation
      // failed); .error is just the generic HTTP status text ("Bad Request")
      // and was masking the real reason for every failure.
      detail = (errorBody as any)?.details || (errorBody as any)?.error || (errorBody as any)?.message || JSON.stringify(errorBody);
    } catch {
      try {
        detail = await response.text();
      } catch {
        detail = '';
      }
    }
    devError('[QueryBuilder]', response.url || response.status, response.status, response.statusText, errorBody || detail);
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
  const data = await fetchJSON<unknown>(`/api/business-objects/bindings?bo_id=${encodeURIComponent(boId)}`);
  const raw = Array.isArray(data) ? data : (data as any)?.bindings || [];

  return raw.map((b: any): BindingView => ({
    bindingId: b.id || b.bindingId || b.binding_id || '',
    bindingName: b.displayName || b.display_name || b.backendType || b.backend_type || 'Binding',
    backendId: b.alphaDatasourceId || b.alpha_datasource_id || '',
    backendName: b.displayName || b.display_name || b.backendType || b.backend_type || 'Datasource',
    drivingTableId: b.drivingNodeId || b.driving_node_id,
    isDefault: b.isDefault ?? b.is_default ?? false,
  }));
}

/**
 * Resolve a binding's declared datasource slot to this tenant's own
 * connection (tenant_product_datasource.id) — the value to send as
 * X-Datasource-Id on Preview/Execute, and what the query-builder picker
 * compares against the tenant-scoped datasource to pick a default. Returns
 * undefined if this tenant has no connection wired for that binding's
 * datasource type.
 */
export async function resolveBindingDatasource(
  boId: string,
  bindingId?: string
): Promise<string | undefined> {
  const params = bindingId ? `?binding_id=${encodeURIComponent(bindingId)}` : '';
  try {
    const data = await fetchJSON<{ datasource_id?: string }>(
      `/api/business-objects/${encodeURIComponent(boId)}/resolve-datasource${params}`
    );
    return data.datasource_id || undefined;
  } catch (err) {
    devError('[QueryBuilder] Failed to resolve binding datasource', boId, bindingId, err);
    return undefined;
  }
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
 * Fetch the self-describing BO schema from the Meta-API.
 */
export async function fetchBOSchema(boId: string): Promise<BOSchema> {
  const data = await fetchJSON<unknown>(`/api/metadata/bo/${encodeURIComponent(boId)}`);
  return data as BOSchema;
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
