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

/**
 * Ids that were once seeded into localStorage as dev placeholders (index.html
 * wrote them on localhost whenever no selection was cached). They name no
 * real tenant or datasource, so sending them only earns a rejected request.
 */
const PLACEHOLDER_SCOPE_IDS = new Set([
  '00000000-0000-0000-0000-000000000000',
  '11111111-1111-1111-1111-111111111111',
]);

export function isPlaceholderScopeId(id: string | null | undefined): boolean {
  return !!id && PLACEHOLDER_SCOPE_IDS.has(id.trim());
}

/** A cached selection that is a placeholder is dropped (and forgotten). */
function readRealSelection<T extends { id?: string }>(key: string): T | null {
  const value = safeParse<T>(localStorage.getItem(key));
  if (value && isPlaceholderScopeId(value.id)) {
    try {
      localStorage.removeItem(key);
    } catch (_) {}
    return null;
  }
  return value;
}

export function readCachedSelection(): CachedSelection {
  let tenant = readRealSelection<Tenant>(TENANT_STORAGE_KEYS.TENANT);
  let datasource = readRealSelection<DataSource>(TENANT_STORAGE_KEYS.DATASOURCE);

  // Fallback 1: check operating_scope if legacy keys are missing
  if (!tenant?.id || !datasource?.id) {
    const scope = safeParse<any>(localStorage.getItem('operating_scope'));
    if (scope) {
      if (!tenant?.id && scope.tenantId && !isPlaceholderScopeId(scope.tenantId)) {
        tenant = {
          id: scope.tenantId,
          name: scope.tenantName || scope.tenantId,
          display_name: scope.tenantName || scope.tenantId,
        } as Tenant;
      }
      if (!datasource?.id && scope.datasourceId && !isPlaceholderScopeId(scope.datasourceId)) {
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


// --- Scope restore gate ------------------------------------------------------
//
// After a reload the Operating Scope is restored asynchronously (AccessContext
// fetches the caller's tenants, then re-selects the persisted datasource).
// Requests that fire before then and have no cached datasource would go out
// unscoped, so the API clients hold them until the scope is restored. The gate
// is only armed while an AccessProvider is restoring; it is bounded by a
// timeout so a failed restore can never hang a request.

type ScopeRestoreState = 'idle' | 'restoring' | 'restored';

let scopeRestoreState: ScopeRestoreState = 'idle';
let scopeWaiters: Array<() => void> = [];

export const SCOPE_RESTORE_TIMEOUT_MS = 5000;

/** Requests that establish the scope itself, so must never wait for it. */
const SCOPE_FREE_PATHS = [/^\/api\/auth(\/|$)/, /^\/api\/tenants\/(all|accessible)(\/|\?|$)/];

function pathOf(url: string): string {
  try {
    return new URL(url, 'http://scope.invalid').pathname;
  } catch (_) {
    return url;
  }
}

/** Called by AccessProvider when it starts (re)restoring the scope. */
export function beginScopeRestore(): void {
  scopeRestoreState = 'restoring';
}

/** Called once a scope is selected, or when there is nothing to restore. */
export function markScopeRestored(): void {
  scopeRestoreState = 'restored';
  const waiters = scopeWaiters;
  scopeWaiters = [];
  waiters.forEach(release => release());
}

/** Test hook: back to the no-provider state. */
export function resetScopeRestoreGate(): void {
  scopeRestoreState = 'idle';
  scopeWaiters = [];
}

function hasCachedDatasource(): boolean {
  const { datasource } = readCachedSelection();
  return Boolean(datasource?.id || datasource?.alpha_tenant_instance_id);
}

/**
 * Resolves when a request to url may read the tenant scope: at once when no
 * restore is under way, the path does not need a scope, or a datasource is
 * already cached; otherwise when the restore finishes (or times out).
 */
export function whenTenantScopeReady(url: string, timeoutMs = SCOPE_RESTORE_TIMEOUT_MS): Promise<void> {
  if (scopeRestoreState !== 'restoring') return Promise.resolve();
  const path = pathOf(url);
  if (SCOPE_FREE_PATHS.some(re => re.test(path))) return Promise.resolve();
  if (hasCachedDatasource()) return Promise.resolve();
  return new Promise(resolve => {
    const release = () => {
      clearTimeout(timer);
      scopeWaiters = scopeWaiters.filter(w => w !== release);
      resolve();
    };
    const timer = setTimeout(() => {
      devWarn('Operating Scope was not restored in time; sending request without a datasource', { path });
      release();
    }, timeoutMs);
    scopeWaiters.push(release);
  });
}
