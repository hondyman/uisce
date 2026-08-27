import { getRequiredTenantScope, hasTenantScope, readCachedSelection } from './tenantScope';
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

        try {
            const { tenant, datasource } = readCachedSelection();
            if (tenant?.id && !headers.has('X-Tenant-ID')) {
                headers.set('X-Tenant-ID', tenant.id);
            }
            const datasourceId = datasource?.id || datasource?.alpha_tenant_instance_id;
            if (datasourceId && !headers.has('X-Tenant-Datasource-ID')) {
                headers.set('X-Tenant-Datasource-ID', datasourceId);
            }
        } catch (_) {}

        // Inject chat metadata headers so the backend ChatEngine can persist
        // every message with the correct view_type / embedded / surface flags.
        if (path.includes('/chat') || path.includes('/genui') || path.includes('/ai/chat')) {
            try {
                if (typeof localStorage !== 'undefined') {
                    const viewType = localStorage.getItem('chat_view_type') || 'admin';
                    if (!headers.has('X-Chat-View-Type')) headers.set('X-Chat-View-Type', viewType);
                    const embedded = localStorage.getItem('chat_embedded') || 'false';
                    if (!headers.has('X-Chat-Embedded')) headers.set('X-Chat-Embedded', embedded);
                    const surface = localStorage.getItem('chat_embed_surface') || 'studio';
                    if (!headers.has('X-Chat-Embed-Surface')) headers.set('X-Chat-Embed-Surface', surface);
                }
            } catch (_) {}
        }
    }

    // Inject Authorization Token - check multiple storage locations for OIDC tokens
    let token: string | null = null;
    if (typeof localStorage !== 'undefined') {
        // 1. Try standard auth_token key
        token = localStorage.getItem('auth_token');

        // 2. Fallback: Parse oidc-client-ts namespaced storage if present
        if (!token) {
            const oidcKey = Object.keys(localStorage).find(k => k.startsWith('oidc.user:'));
            if (oidcKey) {
                try {
                    const userData = JSON.parse(localStorage.getItem(oidcKey) || '{}');
                    token = userData.access_token || userData.id_token || null;
                } catch (e) {
                    console.warn('[apiClient] Failed to parse OIDC storage user object', e);
                }
            }
        }
    }

    if (token && !headers.has('Authorization')) {
        // Basic JWT check to avoid sending placeholder tokens that break dev envs
        if (token.split('.').length === 3 && !token.includes('demo')) {
            headers.set('Authorization', `Bearer ${token}`);
        } else {
            console.warn(`[apiClient] Token found but doesn't appear to be a valid JWT: ${token.substring(0, 50)}...`);
        }
    } else if (!token) {
        console.warn(`[apiClient] No bearer token found in storage for request to ${path}`);
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
        let errDetails = '';
        try {
            errDetails = await response.text();
        } catch (_) {}
        console.error(`[apiClient] Request to ${url} failed with ${response.status} ${response.statusText}:`, errDetails);
        throw new Error(`API Error: ${response.status} ${response.statusText}${errDetails ? ` - ${errDetails}` : ''}`);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
        return response.json() as unknown as T;
    }

    return response as unknown as T;
}

export default apiClient;
