// Centralized helper for admin seeding endpoints
import { getSelectedRegion } from '../lib/region';

export type SeedResult = {
  validationRules?: number;
  approvalRules?: number;
  approverAssignments?: number;
} | null;

async function request(path: string, method: 'POST' | 'DELETE' = 'POST', body?: any, tenantId?: string, datasourceId?: string) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (tenantId) headers['X-Tenant-ID'] = tenantId;
  if (datasourceId) headers['X-Tenant-Datasource-ID'] = datasourceId;
  headers['X-Tenant-Region'] = getSelectedRegion();
  try {
    const token = localStorage.getItem('auth_token');
    if (token) headers['Authorization'] = `Bearer ${token}`;
  } catch { /* ignore */ }

  const res = await fetch(path, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
    credentials: 'include',
  });

  let data: any = null;
  try {
    const ct = res.headers.get('content-type') || '';
    if (ct.includes('application/json')) data = await res.json();
  } catch (e) {
    // ignore parse errors
  }

  return { ok: res.ok, status: res.status, data };
}

export async function seedAll(tenantId: string, datasourceId?: string) {
  return request('/api/admin/seed', 'POST', { tenantId, datasourceId }, tenantId, datasourceId);
}

export async function seedValidationRules(tenantId: string, datasourceId?: string) {
  return request('/api/admin/seed/validation-rules', 'POST', { tenantId, datasourceId }, tenantId, datasourceId);
}

export async function seedApprovalRules(tenantId: string) {
  return request('/api/admin/seed/approval-rules', 'POST', { tenantId }, tenantId, undefined);
}

export async function clearSeed(tenantId: string, datasourceId?: string) {
  return request('/api/admin/seed', 'DELETE', { tenantId, datasourceId }, tenantId, datasourceId);
}

export default {
  seedAll,
  seedValidationRules,
  seedApprovalRules,
  clearSeed,
};
