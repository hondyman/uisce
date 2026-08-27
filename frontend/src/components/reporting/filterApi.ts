import { FilterModel, TenantDefaults, TenantCalendar, ReportParameter } from './filterTypes';

function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const tenantId = (window as any).__TENANT_ID__;
  if (tenantId) headers['X-Tenant-ID'] = tenantId;
  return headers;
}

function getTenantId(): string {
  return (window as any).__TENANT_ID__ || '';
}

export async function loadFilterModel(reportId: string): Promise<FilterModel> {
  const res = await fetch(`/api/reports/${reportId}/filters`, {
    headers: getAuthHeaders(),
  });
  if (!res.ok) {
    throw new Error(`Failed to load filters: ${res.statusText}`);
  }
  const json = await res.json();
  return json.model;
}

export async function saveFilterModel(
  reportId: string,
  model: FilterModel,
  parameters: ReportParameter[] = [],
  defaults: TenantDefaults
): Promise<string> {
  const res = await fetch(`/api/reports/${reportId}/filters`, {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify({ model, parameters, defaults }),
  });
  if (!res.ok) {
    throw new Error(`Failed to save filters: ${res.statusText}`);
  }
  const json = await res.json();
  return json.compiledWhere;
}

export async function loadTenantDefaults(): Promise<TenantDefaults> {
  const tenantId = getTenantId();
  if (!tenantId) {
    return { defaultCalendarCode: 'US', defaultFiscalYear: new Date().getFullYear(), defaultRegion: 'us-east-1' };
  }
  const res = await fetch(`/api/tenants/${tenantId}/defaults`, {
    headers: getAuthHeaders(),
  });
  if (!res.ok) {
    return { defaultCalendarCode: 'US', defaultFiscalYear: new Date().getFullYear(), defaultRegion: 'us-east-1' };
  }
  return res.json();
}

export async function listTenantCalendars(): Promise<TenantCalendar[]> {
  const tenantId = getTenantId();
  if (!tenantId) return [];
  const res = await fetch(`/api/tenants/${tenantId}/calendars`, {
    headers: getAuthHeaders(),
  });
  if (!res.ok) return [];
  const json = await res.json();
  return json.calendars || [];
}

export function emptyFilterModel(): FilterModel {
  return { groups: [], groupCombinator: 'AND' };
}

export function createFilter(partial: Partial<import('./filterTypes').Filter> = {}): import('./filterTypes').Filter {
  return {
    id: `f_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
    field: '',
    operator: 'equals',
    valueSource: { kind: 'constant', value: '' },
    values: [],
    enabled: true,
    ...partial,
  };
}

export function createGroup(partial: Partial<import('./filterTypes').FilterGroup> = {}): import('./filterTypes').FilterGroup {
  return {
    id: `g_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
    combinator: 'AND',
    filters: [],
    ...partial,
  };
}
