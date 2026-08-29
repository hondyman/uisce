import { apiClient } from '../../utils/apiClient';

export interface FilterModel {
  id?: string;
  name: string;
  conditions: unknown[];
  logic: 'and' | 'or';
}

export interface FilterCategory {
  id: string;
  name: string;
  filters: FilterModel[];
}

export async function loadFilterModel(id: string): Promise<FilterModel> {
  return apiClient<FilterModel>(`/api/filters/${id}`);
}

export async function saveFilterModel(filter: Partial<FilterModel>): Promise<FilterModel> {
  return apiClient<FilterModel>('/api/filters', {
    method: 'POST',
    body: JSON.stringify(filter),
    headers: { 'Content-Type': 'application/json' }
  });
}

export async function loadTenantDefaults(tenantId: string): Promise<FilterCategory[]> {
  return apiClient<FilterCategory[]>(`/api/filters/tenant/${tenantId}/defaults`);
}
