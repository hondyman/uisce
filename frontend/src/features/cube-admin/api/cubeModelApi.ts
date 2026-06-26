/**
 * Cube Model API Client
 * 
 * TypeScript client for the Cube model management, catalog integration, 
 * and security policy APIs.
 */

// --- Types ---

export interface CoreCubeModel {
  id: string;
  tenant_id: string;
  tenant_instance_id: string;
  catalog_node_id: string;
  name: string;
  display_name: string;
  sql_table: string;
  data_source: string;
  description: string;
  generated_yaml: string;
  measures: MeasureDefinition[];
  dimensions: DimensionDefinition[];
  joins: JoinDefinition[];
  sync_status: 'synced' | 'pending' | 'error';
  last_synced_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface CustomCubeModel {
  id: string;
  tenant_id: string;
  tenant_instance_id: string;
  core_model_id: string | null;
  name: string;
  description: string;
  extension_type: 'extend' | 'override' | 'standalone';
  custom_config: CustomConfig;
  version: number;
  is_active: boolean;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface CustomConfig {
  measures?: CustomMeasure[];
  dimensions?: CustomDimension[];
  joins?: CustomJoin[];
  pre_aggregations?: CustomPreAgg[];
  segments?: CustomSegment[];
}

export interface MeasureDefinition {
  name: string;
  type: string;
  sql: string;
  title: string;
  description?: string;
  format?: string;
  drill_members?: string[];
  filters?: { sql: string }[];
  shown?: boolean;
}

export interface DimensionDefinition {
  name: string;
  type: string;
  sql: string;
  title: string;
  description?: string;
  primary_key?: boolean;
  shown?: boolean;
  sub_query?: boolean;
}

export interface JoinDefinition {
  name: string;
  target_cube: string;
  relationship: 'one_to_one' | 'one_to_many' | 'many_to_one';
  sql: string;
}

export interface CustomMeasure extends MeasureDefinition {
  is_override?: boolean;
}

export interface CustomDimension extends DimensionDefinition {
  is_override?: boolean;
}

export interface CustomJoin extends JoinDefinition {
  is_override?: boolean;
}

export interface CustomPreAgg {
  name: string;
  type: 'rollup' | 'rollupLambda' | 'rollupJoin' | 'originalSql';
  measures: string[];
  dimensions: string[];
  time_dimension?: string;
  granularity?: string;
  partition_granularity?: string;
  refresh_key?: string;
  build_range_start?: string;
  build_range_end?: string;
}

export interface CustomSegment {
  name: string;
  sql: string;
}

// Security Types
export interface SecurityPolicy {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  policy_type: 'row' | 'column' | 'access' | 'query';
  priority: number;
  enabled: boolean;
  target_cubes: string[];
  target_members: string[];
  conditions: PolicyConditions;
  effects: PolicyEffects;
  version: number;
  created_at: string;
  updated_at: string;
  created_by: string;
}

export interface PolicyConditions {
  roles?: string[];
  groups?: string[];
  attributes?: Record<string, any>;
  time_window?: TimeWindowCondition;
  ip_ranges?: string[];
  data_classification?: string[];
}

export interface TimeWindowCondition {
  allowed_days?: string[];
  allowed_hours_start?: number;
  allowed_hours_end?: number;
  timezone?: string;
  effective_from?: string;
  effective_until?: string;
}

export interface PolicyEffects {
  action: 'allow' | 'deny' | 'filter' | 'mask' | 'limit';
  row_filters?: RowFilter[];
  column_masks?: ColumnMask[];
  query_limits?: QueryLimits;
  audit_log?: boolean;
  alert_on_match?: boolean;
}

export interface RowFilter {
  cube?: string;
  dimension: string;
  operator: string;
  values: any[];
  dynamic?: boolean;
  expression?: string;
}

export interface ColumnMask {
  cube?: string;
  member: string;
  mask_type: 'redact' | 'hash' | 'truncate' | 'nullify' | 'partial' | 'custom';
  mask_pattern?: string;
  allowed_roles?: string[];
}

export interface QueryLimits {
  max_rows?: number;
  max_execution_time_ms?: number;
  max_concurrency?: number;
  allowed_cubes?: string[];
  denied_cubes?: string[];
  allowed_measures?: string[];
  denied_measures?: string[];
  allowed_dimensions?: string[];
  denied_dimensions?: string[];
  allow_pre_agg?: boolean;
  pre_agg_only?: boolean;
}

export interface SecurityContext {
  user_id: string;
  tenant_id: string;
  tenant_instance_id: string;
  roles: string[];
  groups: string[];
  attributes: Record<string, any>;
  session_id?: string;
  ip_address?: string;
}

export interface SecurityDecision {
  allowed: boolean;
  row_filters: RowFilter[];
  column_masks: ColumnMask[];
  query_limits: QueryLimits | null;
  applied_policies: string[];
  denial_reason?: string;
  audit_metadata?: Record<string, any>;
}

// Wizard Types
export interface WizardSession {
  id: string;
  tenant_id: string;
  tenant_instance_id: string;
  session_type: 'core' | 'custom' | 'extension';
  current_step: number;
  total_steps: number;
  session_data: Record<string, any>;
  status: 'in_progress' | 'completed' | 'cancelled';
  created_by: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
  result_model_id?: string;
  steps?: WizardStep[];
}

export interface WizardStep {
  step_number: number;
  step_type: string;
  step_data: Record<string, any>;
  completed: boolean;
}

// Catalog Types
export interface CatalogTable {
  id: string;
  name: string;
  display_name: string;
  description: string;
  schema: string;
}

export interface CatalogColumn {
  id: string;
  name: string;
  display_name: string;
  data_type: string;
  description: string;
  is_primary_key: boolean;
  is_foreign_key: boolean;
}

export interface CatalogRelationship {
  id: string;
  source_table: string;
  source_column: string;
  target_table: string;
  target_column: string;
  relation_type: string;
}

// Cache Stats
export interface CacheStats {
  l1_cache_size: number;
  l1_total_hits: number;
  l2_cache_size: number;
  l2_total_hits: number;
  l1_ttl_seconds: number;
}

import apiClient from '../../../utils/apiClient';

// --- API Client ---

async function fetchWithTenant<T>(
  url: string,
  tenantId: string,
  datasourceId: string,
  options: RequestInit = {}
): Promise<T> {
  const params = new URLSearchParams({ tenant_id: tenantId, tenant_instance_id: datasourceId });
  const separator = url.includes('?') ? '&' : '?';
  const path = `/cube${url}${separator}${params}`;

  const response = await apiClient(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || 'API request failed');
  }

  // Handle YAML responses
  const contentType = response.headers.get('content-type');
  if (contentType?.includes('application/x-yaml')) {
    return response.text() as Promise<T>;
  }

  return response.json();
}

// --- Core Models API ---

export const coreModelsApi = {
  list: (tenantId: string, datasourceId: string): Promise<CoreCubeModel[]> =>
    fetchWithTenant('/models/core', tenantId, datasourceId),

  get: (tenantId: string, datasourceId: string, modelId: string): Promise<CoreCubeModel> =>
    fetchWithTenant(`/models/core/${modelId}`, tenantId, datasourceId),

  getYaml: (tenantId: string, datasourceId: string, modelId: string): Promise<string> =>
    fetchWithTenant(`/models/core/${modelId}/yaml`, tenantId, datasourceId),

  syncFromCatalog: (
    tenantId: string,
    datasourceId: string,
    force = false
  ): Promise<{ synced_count: number; models: CoreCubeModel[] }> =>
    fetchWithTenant('/models/sync-catalog', tenantId, datasourceId, {
      method: 'POST',
      body: JSON.stringify({ force }),
    }),

  delete: (tenantId: string, datasourceId: string, modelId: string): Promise<{ status: string }> =>
    fetchWithTenant(`/models/core/${modelId}`, tenantId, datasourceId, {
      method: 'DELETE',
    }),
};

// --- Custom Models API ---

export const customModelsApi = {
  list: (tenantId: string, datasourceId: string): Promise<CustomCubeModel[]> =>
    fetchWithTenant('/models/custom', tenantId, datasourceId),

  get: (tenantId: string, datasourceId: string, modelId: string): Promise<CustomCubeModel> =>
    fetchWithTenant(`/models/custom/${modelId}`, tenantId, datasourceId),

  getYaml: (tenantId: string, datasourceId: string, modelId: string): Promise<string> =>
    fetchWithTenant(`/models/custom/${modelId}/yaml`, tenantId, datasourceId),

  getMergedYaml: (tenantId: string, datasourceId: string, modelId: string): Promise<string> =>
    fetchWithTenant(`/models/custom/${modelId}/merged-yaml`, tenantId, datasourceId),

  create: (
    tenantId: string,
    datasourceId: string,
    model: Partial<CustomCubeModel> & { created_by: string }
  ): Promise<CustomCubeModel> =>
    fetchWithTenant('/models/custom', tenantId, datasourceId, {
      method: 'POST',
      body: JSON.stringify({ ...model, tenant_id: tenantId, tenant_instance_id: datasourceId }),
    }),

  update: (
    tenantId: string,
    datasourceId: string,
    modelId: string,
    updates: Partial<CustomCubeModel>
  ): Promise<CustomCubeModel> =>
    fetchWithTenant(`/models/custom/${modelId}`, tenantId, datasourceId, {
      method: 'PUT',
      body: JSON.stringify(updates),
    }),

  delete: (tenantId: string, datasourceId: string, modelId: string): Promise<{ status: string }> =>
    fetchWithTenant(`/models/custom/${modelId}`, tenantId, datasourceId, {
      method: 'DELETE',
    }),
};

// --- YAML Generation API ---

export interface YamlSpec {
  cube_name: string;
  sql_table: string;
  data_source: string;
  description?: string;
  measures: MeasureDefinition[];
  dimensions: DimensionDefinition[];
  joins?: JoinDefinition[];
  pre_aggregations?: CustomPreAgg[];
}

export const yamlApi = {
  generateFromSpec: (tenantId: string, datasourceId: string, spec: YamlSpec): Promise<string> =>
    fetchWithTenant('/models/generate-yaml', tenantId, datasourceId, {
      method: 'POST',
      body: JSON.stringify(spec),
    }),

  preview: (
    tenantId: string,
    datasourceId: string,
    params: {
      core_model_id?: string;
      custom_config?: CustomConfig;
      extension_type?: string;
    }
  ): Promise<string> =>
    fetchWithTenant('/models/preview-yaml', tenantId, datasourceId, {
      method: 'POST',
      body: JSON.stringify(params),
    }),

  validate: (
    tenantId: string,
    datasourceId: string,
    yaml: string
  ): Promise<{ valid: boolean; errors: string[] }> =>
    fetchWithTenant('/models/validate-yaml', tenantId, datasourceId, {
      method: 'POST',
      body: JSON.stringify({ yaml }),
    }),
};

// --- Security Policies API ---

export const securityApi = {
  listPolicies: (tenantId: string, datasourceId: string): Promise<SecurityPolicy[]> =>
    fetchWithTenant('/security/policies', tenantId, datasourceId),

  getPolicy: (tenantId: string, datasourceId: string, policyId: string): Promise<SecurityPolicy> =>
    fetchWithTenant(`/security/policies/${policyId}`, tenantId, datasourceId),

  createPolicy: (
    tenantId: string,
    datasourceId: string,
    policy: Partial<SecurityPolicy>
  ): Promise<SecurityPolicy> =>
    fetchWithTenant('/security/policies', tenantId, datasourceId, {
      method: 'POST',
      body: JSON.stringify({ ...policy, tenant_id: tenantId }),
    }),

  updatePolicy: (
    tenantId: string,
    datasourceId: string,
    policyId: string,
    policy: Partial<SecurityPolicy>
  ): Promise<SecurityPolicy> =>
    fetchWithTenant(`/security/policies/${policyId}`, tenantId, datasourceId, {
      method: 'PUT',
      body: JSON.stringify(policy),
    }),

  deletePolicy: (tenantId: string, datasourceId: string, policyId: string): Promise<{ status: string }> =>
    fetchWithTenant(`/security/policies/${policyId}`, tenantId, datasourceId, {
      method: 'DELETE',
    }),

  evaluate: (
    tenantId: string,
    datasourceId: string,
    context: SecurityContext,
    cubes?: string[]
  ): Promise<SecurityDecision> =>
    fetchWithTenant('/security/evaluate', tenantId, datasourceId, {
      method: 'POST',
      body: JSON.stringify({ security_context: context, cubes }),
    }),

  getCacheStats: (tenantId: string, datasourceId: string): Promise<CacheStats> =>
    fetchWithTenant('/security/cache/stats', tenantId, datasourceId),

  invalidateCache: (tenantId: string, datasourceId: string): Promise<{ status: string }> =>
    fetchWithTenant('/security/cache/invalidate', tenantId, datasourceId, {
      method: 'POST',
    }),
};

// --- Wizard API ---

export const wizardApi = {
  createSession: (
    tenantId: string,
    datasourceId: string,
    sessionType: 'core' | 'custom' | 'extension',
    createdBy: string
  ): Promise<WizardSession> =>
    fetchWithTenant('/wizard/sessions', tenantId, datasourceId, {
      method: 'POST',
      body: JSON.stringify({
        tenant_id: tenantId,
        tenant_instance_id: datasourceId,
        session_type: sessionType,
        created_by: createdBy,
      }),
    }),

  getSession: (tenantId: string, datasourceId: string, sessionId: string): Promise<WizardSession> =>
    fetchWithTenant(`/wizard/sessions/${sessionId}`, tenantId, datasourceId),

  updateStep: (
    tenantId: string,
    datasourceId: string,
    sessionId: string,
    stepNumber: number,
    stepType: string,
    stepData: Record<string, any>,
    completed = true
  ): Promise<{ status: string }> =>
    fetchWithTenant(`/wizard/sessions/${sessionId}/steps/${stepNumber}`, tenantId, datasourceId, {
      method: 'PUT',
      body: JSON.stringify({ step_type: stepType, step_data: stepData, completed }),
    }),

  complete: (
    tenantId: string,
    datasourceId: string,
    sessionId: string
  ): Promise<{ status: string; result_model_id: string }> =>
    fetchWithTenant(`/wizard/sessions/${sessionId}/complete`, tenantId, datasourceId, {
      method: 'POST',
    }),

  deleteSession: (tenantId: string, datasourceId: string, sessionId: string): Promise<{ status: string }> =>
    fetchWithTenant(`/wizard/sessions/${sessionId}`, tenantId, datasourceId, {
      method: 'DELETE',
    }),
};

// --- Catalog Browsing API ---

export const catalogApi = {
  listTables: (tenantId: string, datasourceId: string): Promise<CatalogTable[]> =>
    fetchWithTenant('/catalog/tables', tenantId, datasourceId),

  listColumns: (tenantId: string, datasourceId: string, tableId: string): Promise<CatalogColumn[]> =>
    fetchWithTenant(`/catalog/tables/${tableId}/columns`, tenantId, datasourceId),

  listRelationships: (tenantId: string, datasourceId: string): Promise<CatalogRelationship[]> =>
    fetchWithTenant('/catalog/relationships', tenantId, datasourceId),
};

// --- React Hooks ---

import { useState, useEffect, useCallback } from 'react';

export function useCoreModels(tenantId: string, datasourceId: string) {
  const [models, setModels] = useState<CoreCubeModel[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!tenantId || !datasourceId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await coreModelsApi.list(tenantId, datasourceId);
      setModels(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load core models');
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId]);

  useEffect(() => {
    load();
  }, [load]);

  const syncFromCatalog = useCallback(async (force = false) => {
    setLoading(true);
    try {
      const result = await coreModelsApi.syncFromCatalog(tenantId, datasourceId, force);
      setModels(result.models);
      return result;
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId]);

  return { models, loading, error, reload: load, syncFromCatalog };
}

export function useCustomModels(tenantId: string, datasourceId: string) {
  const [models, setModels] = useState<CustomCubeModel[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!tenantId || !datasourceId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await customModelsApi.list(tenantId, datasourceId);
      setModels(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load custom models');
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId]);

  useEffect(() => {
    load();
  }, [load]);

  return { models, loading, error, reload: load };
}

export function useSecurityPolicies(tenantId: string, datasourceId: string) {
  const [policies, setPolicies] = useState<SecurityPolicy[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!tenantId || !datasourceId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await securityApi.listPolicies(tenantId, datasourceId);
      setPolicies(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load security policies');
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId]);

  useEffect(() => {
    load();
  }, [load]);

  return { policies, loading, error, reload: load };
}

export function useCatalogTables(tenantId: string, datasourceId: string) {
  const [tables, setTables] = useState<CatalogTable[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!tenantId || !datasourceId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await catalogApi.listTables(tenantId, datasourceId);
      setTables(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load catalog tables');
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId]);

  useEffect(() => {
    load();
  }, [load]);

  return { tables, loading, error, reload: load };
}

export function useCatalogColumns(tenantId: string, datasourceId: string, tableId: string) {
  const [columns, setColumns] = useState<CatalogColumn[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!tenantId || !datasourceId || !tableId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await catalogApi.listColumns(tenantId, datasourceId, tableId);
      setColumns(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load catalog columns');
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId, tableId]);

  useEffect(() => {
    load();
  }, [load]);

  return { columns, loading, error, reload: load };
}

// Default export for convenience
export default {
  coreModels: coreModelsApi,
  customModels: customModelsApi,
  yaml: yamlApi,
  security: securityApi,
  wizard: wizardApi,
  catalog: catalogApi,
};
