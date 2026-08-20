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
  let tenant = safeParse<Tenant>(localStorage.getItem(TENANT_STORAGE_KEYS.TENANT));
  let datasource = safeParse<DataSource>(localStorage.getItem(TENANT_STORAGE_KEYS.DATASOURCE));

  // Fallback 1: check operating_scope if legacy keys are missing
  if (!tenant?.id || !datasource?.id) {
    const scope = safeParse<any>(localStorage.getItem('operating_scope'));
    if (scope) {
      if (!tenant?.id && scope.tenantId) {
        tenant = {
          id: scope.tenantId,
          name: scope.tenantName || scope.tenantId,
          display_name: scope.tenantName || scope.tenantId,
        } as Tenant;
      }
      if (!datasource?.id && scope.datasourceId) {
        datasource = {
          id: scope.datasourceId,
          source_name: scope.datasourceName || scope.datasourceId,
        } as DataSource;
      }
    }
  }

  // Fallback 2: check auth_user / JWT token if tenant is still missing
  if (!tenant?.id) {
    try {
      const authUser = safeParse<any>(localStorage.getItem('auth_user'));
      const token = localStorage.getItem('auth_token');
      let jwtPayload: any = null;
      if (token && token.split('.').length === 3) {
        try {
          const base64Url = token.split('.')[1];
          const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
          const jsonPayload = decodeURIComponent(
            atob(base64)
              .split('')
              .map(c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
              .join('')
          );
          jwtPayload = JSON.parse(jsonPayload);
        } catch (_) {}
      }

      const candidateTenantId =
        jwtPayload?.scoped_tenant ||
        jwtPayload?.tenant_id ||
        authUser?.tenant_id ||
        authUser?.tenant_assignments?.[0]?.tenantId;

      const candidateTenantName =
        jwtPayload?.scoped_tenant_name ||
        authUser?.tenant_name ||
        authUser?.tenant_assignments?.[0]?.tenantName ||
        candidateTenantId;

      if (candidateTenantId) {
        tenant = {
          id: candidateTenantId,
          name: candidateTenantName,
          display_name: candidateTenantName,
        } as Tenant;
      }
    } catch (_) {}
  }

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

