import type JSONValue from '../types/json';
import { useAuthFetch } from '../utils/authFetch';
import resolveApiUrl from '../utils/resolveApiUrl';

export interface ExtensionSavePayload {
  base_model_key: string;
  model_key?: string;
  title?: string;
  description?: string;
  status?: 'draft' | 'published';
  core_version?: number;
  extension_cube: Record<string, JSONValue>;
  actor_id?: string;
}

export interface ValidationIssue {
  level: 'error' | 'warning';
  code: string;
  message: string;
  details?: Record<string, JSONValue> | null;
}

export interface ExtensionCompatibilityRow {
  extension_model_key: string;
  base_model_key: string;
  base_cube_name: string;
  base_version: number;
  extension_core_version_target?: number | null;
  version_mismatch: boolean;
  status: string;
  issues: ValidationIssue[];
  extension_changes?: Record<string, JSONValue> | null;
}

function tryExtractIssues(obj: unknown): ValidationIssue[] {
  if (!obj || typeof obj !== 'object') return [];
  const record = obj as Record<string, unknown>;
  const candidate = (record.issues ?? record.errors) as unknown;
  if (!candidate || !Array.isArray(candidate)) return [];
  // Narrow each item into ValidationIssue as best-effort
  return candidate
    .filter((it) => typeof it === 'object' && it !== null)
    .map((it) => {
      const r = it as Record<string, unknown>;
      return {
        level: r.level === 'warning' ? 'warning' : 'error',
        code: String(r.code ?? r.code ?? ''),
        message: String(r.message ?? ''),
        details: (r.details as Record<string, unknown>) ?? null,
      } as ValidationIssue;
    });
}

// Hooked variants using centralized authFetch
export function useExtensionsService() {
  const { authFetch } = useAuthFetch();

  const listExtensions = async (datasourceId: string) => {
  const resp = await authFetch<unknown>(resolveApiUrl(`/api/fabric/extensions?datasource_id=${encodeURIComponent(datasourceId)}`));
    if (!resp.ok) throw new Error(resp.error || `Failed to list extensions: ${resp.status}`);
    return resp.data;
  };

  const saveExtension = async (datasourceId: string, payload: ExtensionSavePayload) => {
    const resp = await authFetch<{ model: Record<string, JSONValue>; issues: ValidationIssue[] }>(
      resolveApiUrl(`/api/fabric/extensions?datasource_id=${encodeURIComponent(datasourceId)}`),
      { method: 'POST', json: payload }
    );
    if (!resp.ok) {
      // Try to surface issues from backend
      const issues = tryExtractIssues(resp.data) || [];
      if (issues.length) {
        try { window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues } })); } catch {}
      }
      throw new Error(resp.error || `Failed to save extension: ${resp.status}`);
    }
    return resp.data as { model: Record<string, JSONValue>; issues: ValidationIssue[] };
  };

  const getCompatibilityReport = async (datasourceId: string) => {
    const resp = await authFetch<{ report: ExtensionCompatibilityRow[]; issues: ValidationIssue[] }>(
      resolveApiUrl(`/api/fabric/extensions/compatibility-report?datasource_id=${encodeURIComponent(datasourceId)}`)
    );
    if (!resp.ok) throw new Error(resp.error || `Failed to fetch compatibility report: ${resp.status}`);
    return resp.data as { report: ExtensionCompatibilityRow[]; issues: ValidationIssue[] };
  };

  const validateExtension = async (datasourceId: string, payload: Record<string, any>) => {
    const resp = await authFetch<{ issues: ValidationIssue[] }>(
      resolveApiUrl(`/api/fabric/models/validate?datasource_id=${encodeURIComponent(datasourceId)}`),
      { method: 'POST', json: payload }
    );
    if (!resp.ok) {
      // Try to extract issues from error payload and broadcast so tiles turn red
      let issues = tryExtractIssues(resp.data) || [];
      if (!issues || !Array.isArray(issues) || issues.length === 0) {
        issues = [{ level: 'error', code: 'validation_error', message: resp.error || `HTTP ${resp.status}` }];
      }
      try { window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues } })); } catch {}
      throw new Error(resp.error || `Failed to validate extension: ${resp.status}`);
    }
    return resp.data as { issues: ValidationIssue[] };
  };

  return { listExtensions, saveExtension, getCompatibilityReport, validateExtension };
}

// Backwards-compatible non-hook functions (for legacy callers). These will use window.fetch
// but keep credentials and error parsing consistent.
export async function listExtensions(datasourceId: string) {
  const res = await fetch(resolveApiUrl(`/api/fabric/extensions?datasource_id=${encodeURIComponent(datasourceId)}`), { credentials: 'include' });
  if (!res.ok) throw new Error(`Failed to list extensions: ${res.status} ${res.statusText}`);
  return res.json();
}

export async function saveExtension(datasourceId: string, payload: ExtensionSavePayload) {
  const res = await fetch(resolveApiUrl(`/api/fabric/extensions?datasource_id=${encodeURIComponent(datasourceId)}`), {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const data = await res.json().catch(async () => ({ message: await res.text().catch(() => '') }));
    let issues = tryExtractIssues(data);
    const fallbackMessage = data && typeof data === 'object' ? String((data as Record<string, unknown>).message ?? '') : '';
    if (!issues || !Array.isArray(issues) || issues.length === 0) {
      issues = [{ level: 'error', code: 'save_error', message: fallbackMessage || `${res.status} ${res.statusText}` }];
    }
    try { window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues } })); } catch {}
    throw new Error(`Failed to save extension: ${res.status} ${res.statusText}${data?.message ? ' - ' + data.message : ''}`);
  }
  return res.json() as Promise<{ model: Record<string, JSONValue>; issues: ValidationIssue[] }>;
}

export async function getCompatibilityReport(datasourceId: string) {
  const res = await fetch(resolveApiUrl(`/api/fabric/extensions/compatibility-report?datasource_id=${encodeURIComponent(datasourceId)}`), { credentials: 'include' });
  if (!res.ok) throw new Error(`Failed to fetch compatibility report: ${res.status} ${res.statusText}`);
  return res.json() as Promise<{ report: ExtensionCompatibilityRow[]; issues: ValidationIssue[] }>;
}

export async function validateExtension(datasourceId: string, payload: Record<string, any>) {
  const res = await fetch(resolveApiUrl(`/api/fabric/models/validate?datasource_id=${encodeURIComponent(datasourceId)}`), {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const data = await res.json().catch(async () => ({ message: await res.text().catch(() => '') }));
    let issues = tryExtractIssues(data);
    const fallbackMessage2 = data && typeof data === 'object' ? String((data as Record<string, unknown>).message ?? '') : '';
    if (!issues || !Array.isArray(issues) || issues.length === 0) {
      issues = [{ level: 'error', code: 'validation_error', message: fallbackMessage2 || `${res.status} ${res.statusText}` }];
    }
    try { window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues } })); } catch {}
    throw new Error(`Failed to validate extension: ${res.status} ${res.statusText}${data?.message ? ' - ' + data.message : ''}`);
  }
  return res.json() as Promise<{ issues: ValidationIssue[] }>;
}
