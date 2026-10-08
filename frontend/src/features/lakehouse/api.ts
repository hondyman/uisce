import apiClient from '../../utils/apiClient';

// Mirrors backend/internal/lakehouse/status.go and
// backend/internal/handlers/lakehouse_status_handler.go. Two endpoints, two authz scopes,
// one payload shape — the tenant handler strips the cluster and resource-group sections
// before serializing, so the frontend reads what is in front of it without re-shaping.

// ---- Top-level payload shapes ----

export interface LakehouseStatus {
  generated_at: string;
  cluster: ClusterStatus;
  resource_groups: ResourceGroup[];
  tenants: TenantBlock[];
  notes?: string[];
}

export interface LakehouseTenantStatus {
  generated_at: string;
  tenant: TenantBlock;
  notes?: string[];
}

// ---- Cluster / resource groups / tenants ----

export interface ClusterStatus {
  frontends: FrontendStatus[];
  backends: BackendStatus[];
}

export interface FrontendStatus {
  name: string;
  host?: string;
  alive: boolean;
  version?: string;
}

export interface BackendStatus {
  host?: string;
  alive: boolean;
  total_capacity_bytes?: number;
  /** 0..1, fraction of capacity used. */
  used_pct?: number;
}

export interface ResourceGroup {
  name: string;
  cpu_weight?: number;
  mem_limit?: string;
  concurrency_limit?: number;
  big_query_mem_limit?: number;
  classifiers: ResourceClassifier[];
}

export interface ResourceClassifier {
  user?: string;
  role?: string;
  weight?: number;
}

// TenantBlock is the panel's row shape. warnings is the cross-check column — empty is a
// green row, any entry is a real configuration or wiring problem.
export interface TenantBlock {
  name: string;
  /** TenantEnvKey(name), uppercased with non-alphanumeric collapsed to underscores. */
  slug: string;
  tenant_id?: string;
  env: TenantEnv;
  database: TenantDB;
  audit_copy: TenantAuditCopy;
  warnings?: string[];
}

export interface TenantEnv {
  dsn_configured: boolean;
  init_resource_group: string;
}

export interface TenantDB {
  name?: string;
  exists: boolean;
  total_bytes?: number;
  table_count?: number;
  /** Human-readable size, e.g. "16 KB". Empty when total_bytes is 0. */
  size_text?: string;
}

export interface TenantAuditCopy {
  last_run_status?: string;
  last_run_time?: string;
  last_run_duration_ms?: number;
}

// ---- API client ----

async function call<T>(path: string, init?: RequestInit): Promise<T> {
  try {
    return await apiClient<T>(path, init);
  } catch (err) {
    throw new Error(err instanceof Error ? err.message : String(err));
  }
}

export const lakehouseStatusApi = {
  /** Platform admin only. Full payload: cluster, resource groups, every tenant. */
  platform: () => call<LakehouseStatus>('/admin/lakehouse/status'),
  /** Tenant-scoped: just the session tenant's row, no cluster/resource-group sections. */
  tenant: () => call<LakehouseTenantStatus>('/tenant/lakehouse/status'),
};