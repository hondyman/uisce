/**
 * Global API client with automatic tenant + region + auth header injection
 * 
 * This module ensures all tenant-scoped API calls automatically include:
 * - Authorization (Bearer JWT)
 * - X-Tenant-ID
 * - X-Tenant-Datasource-ID
 * - X-Tenant-Region
 * - Content-Type
 * 
 * This prevents accidental region violations and keeps multi-region logic centralized.
 */

import { getSelectedRegion } from './region';
import { readCachedSelection } from '../utils/tenantScope';

/**
 * Helper to get all required headers from localStorage / AccessContext
 * This is used by apiFetch to automatically inject headers
 */
function getTenantHeadersInternal(): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  // Inject Authorization token - check multiple storage locations for OIDC tokens
  try {
    let token = localStorage.getItem('auth_token');

    // Fallback: Parse oidc-client-ts namespaced storage if present
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

    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
  } catch (_) {
    // Silently fail
  }

  // Resolve tenant and datasource from unified scope helper
  try {
    const { tenant, datasource } = readCachedSelection();
    if (tenant?.id) {
      headers['X-Tenant-ID'] = tenant.id;
    }
    const datasourceId = datasource?.id || datasource?.alpha_tenant_instance_id;
    if (datasourceId) {
      headers['X-Tenant-Datasource-ID'] = datasourceId;
    }
  } catch (_) {
    // Silently fail
  }

  const region = getSelectedRegion();
  if (region) {
    headers['X-Tenant-Region'] = region;
  }

  return headers;
}

/**
 * Wrapper around native fetch that automatically injects tenant + region headers
 * 
 * Usage:
 *   const response = await apiFetch('/api/validation-rules?...');
 *   const data = await apiFetch('/api/semantic-terms', { method: 'POST', body: JSON.stringify(...) });
 */
export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly statusText: string,
    public readonly response: Response
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export async function apiFetch(
  input: RequestInfo | URL,
  init: RequestInit = {}
): Promise<Response> {
  const headers = new Headers(init.headers || {});

  // Inject tenant headers automatically — but only when the caller has NOT
  // already supplied a value for that header. Caller-supplied values win,
  // so a component that has the React tenant/datasource in scope can pass
  // them explicitly and avoid races with stale localStorage.
  const tenantHeaders = getTenantHeadersInternal();

  Object.entries(tenantHeaders).forEach(([key, value]) => {
    if (!headers.has(key)) {
      headers.set(key, value);
    }
  });

  const response = await fetch(input, { ...init, headers });

  if (!response.ok) {
    const body = await response.text().catch(() => '');
    throw new ApiError(
      `API request failed: ${response.status} ${response.statusText}${body ? ': ' + body.slice(0, 200) : ''}`,
      response.status,
      response.statusText,
      response
    );
  }

  return response;
}

/**
 * Axios-compatible API client (if using axios instead of fetch)
 */
import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';

let axiosInstance: AxiosInstance | null = null;

export function getApiClient(): AxiosInstance {
  if (!axiosInstance) {
    axiosInstance = axios.create();

    // Request interceptor: inject tenant + region headers
    axiosInstance.interceptors.request.use((config: any) => {
      if (!config.headers) {
        config.headers = {};
      }

      const tenantHeaders = getTenantHeadersInternal();
      Object.entries(tenantHeaders).forEach(([key, value]) => {
        config.headers![key] = value;
      });

      return config;
    });

    // Response interceptor: optional error handling
    axiosInstance.interceptors.response.use(
      (response) => response,
      (error) => {
        // Log tenant-related errors for debugging
        if (error.response?.status === 403 || error.response?.status === 400) {
          const url = error.config?.url;
          if (url && url.includes('/api/')) {
            const tenantHeaders = getTenantHeadersInternal();
            console.warn('[API Client] Tenant/region error:', {
              url,
              status: error.response?.status,
              headers: tenantHeaders
            });
          }
        }
        return Promise.reject(error);
      }
    );
  }

  return axiosInstance;
}

/**
 * Helper to manually build headers for specific cases
 * (e.g., when you need headers but aren't making a fetch/axios call)
 */
export function getTenantHeaders(): Record<string, string> {
  return getTenantHeadersInternal();
}
