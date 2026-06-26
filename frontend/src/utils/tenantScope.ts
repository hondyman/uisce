import type { Tenant, DataSource } from '../types';
import { TENANT_STORAGE_KEYS } from '../contexts/TenantContext';
import { devLog, devWarn } from './devLogger';

export interface TenantScope {
  tenantId: string;
  tenantName?: string;
  datasourceId: string;
  datasourceName?: string;
}

interface CachedSelection {
  tenant: Tenant | null;
  datasource: DataSource | null;
}

function safeParse<T>(value: string | null): T | null {
  if (!value) {
    return null;
  }
  try {
    return JSON.parse(value) as T;
  } catch (error) {
    devWarn('Failed to parse cached tenant selection value', { value, error });
    return null;
  }
}

export function readCachedSelection(): CachedSelection {
  const tenant = safeParse<Tenant>(localStorage.getItem(TENANT_STORAGE_KEYS.TENANT));
  const datasource = safeParse<DataSource>(localStorage.getItem(TENANT_STORAGE_KEYS.DATASOURCE));
  return { tenant, datasource };
}

export function getRequiredTenantScope(): TenantScope {
  const { tenant, datasource } = readCachedSelection();

  const tenantId = tenant?.id?.trim() || '';
  const datasourceId = (datasource?.id || datasource?.alpha_tenant_instance_id || '').trim();

  if (!tenantId || !datasourceId) {
    throw new Error('Tenant selection is required. Please select a tenant and datasource to continue.');
  }

  return {
    tenantId,
    tenantName: tenant?.display_name || tenant?.name,
    datasourceId,
    datasourceName: datasource?.source_name || datasource?.alpha_datasource?.datasource_name,
  };
}

export function hasTenantScope(): boolean {
  try {
    const scope = getRequiredTenantScope();
    return Boolean(scope.tenantId && scope.datasourceId);
  } catch (error) {
    return false;
  }
}

export function logTenantScope(): void {
  if (!hasTenantScope()) {
    return;
  }
  const scope = getRequiredTenantScope();
  devLog('Tenant scope in use', scope);
}
