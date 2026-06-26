import { getRequiredTenantScope, hasTenantScope } from './tenantScope';
import resolveApiUrl from './resolveApiUrl';
import { getSelectedRegion } from '../lib/region';

/**
 * Standard API client for semlayer.
 * Replaces the need for the global window.fetch patch.
 */
export async function apiClient<T = Response>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
    let urlString = typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url;

    // Resolve URL (handles /api prefixing and base URL rebasing)
    let path = urlString;
    // Fix: Ensure we only skip if it starts with /api/ or is exactly /api
    if (!urlString.match(/^\/api(\/|$)/)) {
        path = urlString.startsWith('/') ? `/api${urlString}` : `/api/${urlString}`;
    }
    const url = resolveApiUrl(path);

    const headers = new Headers(init?.headers ?? (input instanceof Request ? input.headers : undefined));

    // Inject Tenant Scope (but not for auth endpoints - they don't require tenant context)
    const skipTenantHeaderPaths = ['/api/auth/login', '/api/auth/register', '/api/auth/refresh', '/api/auth/logout'];
    const shouldSkipTenantHeaders = skipTenantHeaderPaths.some(p => path.includes(p));

    if (!shouldSkipTenantHeaders) {
        // Always inject region
        if (!headers.has('X-Tenant-Region')) {
            const region = getSelectedRegion();
            if (region) headers.set('X-Tenant-Region', region);
        }

        if (hasTenantScope()) {
            const { tenantId, datasourceId } = getRequiredTenantScope();
            if (!headers.has('X-Tenant-ID')) {
                headers.set('X-Tenant-ID', tenantId);
            }
            if (!headers.has('X-Tenant-Datasource-ID')) {
                headers.set('X-Tenant-Datasource-ID', datasourceId);
            }
        }
    }

    // Inject Authorization Token
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
    if (token && !headers.has('Authorization')) {
        // Basic JWT check to avoid sending placeholder tokens that break dev envs
        if (token.split('.').length === 3 && !token.includes('demo')) {
            headers.set('Authorization', `Bearer ${token}`);
        }
    }

    // Ensure Content-Type is set for JSON requests
    if (init?.body && typeof init.body === 'string' && !headers.has('Content-Type')) {
        try {
            JSON.parse(init.body);
            headers.set('Content-Type', 'application/json');
        } catch (e) {
            // Not JSON, ignore
        }
    }

    const response = await fetch(url, {
        ...init,
        headers,
        credentials: init?.credentials ?? (input instanceof Request ? input.credentials : 'include')
    });

    if (!response.ok) {
        throw new Error(`API Error: ${response.status} ${response.statusText}`);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
        return response.json() as unknown as T;
    }

    return response as unknown as T;
}

export default apiClient;
