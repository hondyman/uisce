import apiClient from '../../../utils/apiClient';

interface RequestOptions {
  tenantId: string;
  datasourceId: string;
}

interface Organization {
  id: string;
  name: string;
  display_name: string;
  description: string;
  contact_email: string;
  tenant_count?: number;
  created_at: string;
  updated_at: string;
}

interface OrganizationTenant {
  tenant_id: string;
  tenant_name: string;
  role: string;
  added_at: string;
}

interface SemanticModel {
  id: string;
  name: string;
  description: string;
  cube_type: 'cube' | 'view';
  measures_count: number;
  dimensions_count: number;
  joins_count: number;
  version: string;
  status: 'active' | 'draft' | 'deprecated';
  created_at: string;
  updated_at: string;
}

interface QueryAnalytic {
  id: string;
  tenant_id: string;
  query_hash: string;
  cube_name: string;
  measures: string[];
  dimensions: string[];
  duration_ms: number;
  cache_hit: boolean;
  pre_agg_used: boolean;
  rows_returned: number;
  executed_at: string;
}

interface UsageMetric {
  id: string;
  tenant_id: string;
  metric_date: string;
  queries_count: number;
  cache_hit_count: number;
  pre_agg_hit_count: number;
  total_duration_ms: number;
  rows_processed: number;
  unique_users: number;
}

// Helper to build path with extra params
function buildPath(path: string, params?: Record<string, string>): string {
  const url = new URL(path, 'http://localhost'); // dummy base
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value) url.searchParams.set(key, value);
    });
  }
  return url.pathname + url.search;
}

// Generic fetch wrapper - now uses apiClient
async function fetchApi<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await apiClient(`/cube-admin${path}`, options);

  if (!res.ok) {
    const error = await res.text();
    throw new Error(error || `Request failed: ${res.status}`);
  }

  return res.json();
}

// ==================== Organizations ====================

export async function listOrganizations(_opts: RequestOptions): Promise<Organization[]> {
  const url = buildPath('/organizations');
  return fetchApi<Organization[]>(url);
}

export async function getOrganization(orgId: string, _opts: RequestOptions): Promise<Organization> {
  const url = buildPath(`/organizations/${orgId}`);
  return fetchApi<Organization>(url);
}

export async function createOrganization(
  data: Pick<Organization, 'name' | 'display_name' | 'description' | 'contact_email'>,
  _opts: RequestOptions
): Promise<Organization> {
  const url = buildPath('/organizations');
  return fetchApi<Organization>(url, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateOrganization(
  orgId: string,
  data: Partial<Pick<Organization, 'display_name' | 'description' | 'contact_email'>>,
  _opts: RequestOptions
): Promise<Organization> {
  const url = buildPath(`/organizations/${orgId}`);
  return fetchApi<Organization>(url, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function deleteOrganization(orgId: string, _opts: RequestOptions): Promise<void> {
  const url = buildPath(`/organizations/${orgId}`);
  await apiClient(`/cube-admin${url}`, { method: 'DELETE' });
}

// ==================== Organization Tenants ====================

export async function listOrganizationTenants(
  orgId: string,
  _opts: RequestOptions
): Promise<OrganizationTenant[]> {
  const url = buildPath(`/organizations/${orgId}/tenants`);
  return fetchApi<OrganizationTenant[]>(url);
}

export async function addTenantToOrganization(
  orgId: string,
  tenantId: string,
  role: string,
  _opts: RequestOptions
): Promise<void> {
  const url = buildPath(`/organizations/${orgId}/tenants`);
  await fetchApi(url, {
    method: 'POST',
    body: JSON.stringify({ tenant_id: tenantId, role }),
  });
}

export async function removeTenantFromOrganization(
  orgId: string,
  tenantId: string,
  _opts: RequestOptions
): Promise<void> {
  const url = buildPath(`/organizations/${orgId}/tenants/${tenantId}`);
  await apiClient(`/cube-admin${url}`, { method: 'DELETE' });
}

// ==================== Semantic Models ====================

export async function listSemanticModels(_opts: RequestOptions): Promise<SemanticModel[]> {
  const url = buildPath('/semantic-models');
  return fetchApi<SemanticModel[]>(url);
}

export async function getSemanticModel(modelId: string, _opts: RequestOptions): Promise<SemanticModel> {
  const url = buildPath(`/semantic-models/${modelId}`);
  return fetchApi<SemanticModel>(url);
}

// ==================== Query Analytics ====================

export interface QueryAnalyticsParams {
  startDate?: string;
  endDate?: string;
  cubeName?: string;
  limit?: number;
  offset?: number;
}

export async function getQueryAnalytics(
  params: QueryAnalyticsParams,
  _opts: RequestOptions
): Promise<QueryAnalytic[]> {
  const url = buildPath('/analytics/queries', {
    start_date: params.startDate || '',
    end_date: params.endDate || '',
    cube_name: params.cubeName || '',
    limit: params.limit?.toString() || '',
    offset: params.offset?.toString() || '',
  });
  return fetchApi<QueryAnalytic[]>(url);
}

// ==================== Usage Metrics ====================

export interface UsageMetricsParams {
  startDate?: string;
  endDate?: string;
}

export async function getUsageMetrics(
  params: UsageMetricsParams,
  _opts: RequestOptions
): Promise<UsageMetric[]> {
  const url = buildPath('/analytics/usage', {
    start_date: params.startDate || '',
    end_date: params.endDate || '',
  });
  return fetchApi<UsageMetric[]>(url);
}

// ==================== React Query Hooks (optional) ====================
// These can be used with React Query for better caching and state management

/*
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

export function useOrganizations(opts: RequestOptions) {
  return useQuery({
    queryKey: ['cube-admin', 'organizations', opts.tenantId],
    queryFn: () => listOrganizations(opts),
  });
}

export function useOrganization(orgId: string, opts: RequestOptions) {
  return useQuery({
    queryKey: ['cube-admin', 'organizations', orgId],
    queryFn: () => getOrganization(orgId, opts),
  });
}

export function useCreateOrganization(opts: RequestOptions) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof createOrganization>[0]) => 
      createOrganization(data, opts),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cube-admin', 'organizations'] });
    },
  });
}

export function useSemanticModels(opts: RequestOptions) {
  return useQuery({
    queryKey: ['cube-admin', 'semantic-models', opts.tenantId],
    queryFn: () => listSemanticModels(opts),
  });
}

export function useQueryAnalytics(params: QueryAnalyticsParams, opts: RequestOptions) {
  return useQuery({
    queryKey: ['cube-admin', 'analytics', 'queries', params],
    queryFn: () => getQueryAnalytics(params, opts),
  });
}

export function useUsageMetrics(params: UsageMetricsParams, opts: RequestOptions) {
  return useQuery({
    queryKey: ['cube-admin', 'analytics', 'usage', params],
    queryFn: () => getUsageMetrics(params, opts),
  });
}
*/

export type {
  Organization,
  OrganizationTenant,
  SemanticModel,
  QueryAnalytic,
  UsageMetric,
  RequestOptions,
};
