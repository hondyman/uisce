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

/**
 * Helper to get all required headers from localStorage / AccessContext
 * This is used by apiFetch to automatically inject headers
 */
function getTenantHeadersInternal(): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  // Inject Authorization token
  try {
    const token = localStorage.getItem('auth_token');
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
  } catch (_) {
    // Silently fail
  }

  // Try to get tenant/datasource from legacy storage keys
  try {
    const tenantData = localStorage.getItem('selected_tenant');
    const datasourceData = localStorage.getItem('selected_datasource');

    if (tenantData) {
      const tenant = JSON.parse(tenantData);
      if (tenant?.id) headers['X-Tenant-ID'] = tenant.id;
    }

    if (datasourceData) {
      const datasource = JSON.parse(datasourceData);
      if (datasource?.id) headers['X-Tenant-Datasource-ID'] = datasource.id;
    }
  } catch (_) {
    // Silently fail if localStorage parsing fails
  }

  // Fallback: read from AccessContext's operating_scope storage
  if (!headers['X-Tenant-ID'] || !headers['X-Tenant-Datasource-ID']) {
    try {
      const scopeData = localStorage.getItem('operating_scope');
      if (scopeData) {
        const scope = JSON.parse(scopeData);
        if (!headers['X-Tenant-ID'] && scope?.tenantId) {
          headers['X-Tenant-ID'] = scope.tenantId;
        }
        if (!headers['X-Tenant-Datasource-ID'] && scope?.datasourceId) {
          headers['X-Tenant-Datasource-ID'] = scope.datasourceId;
        }
      }
    } catch (_) {
      // Silently fail
    }
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
export async function apiFetch(
  input: RequestInfo | URL,
  init: RequestInit = {}
): Promise<Response> {
  const headers = new Headers(init.headers || {});

  // Inject tenant headers automatically
  const tenantHeaders = getTenantHeadersInternal();

  Object.entries(tenantHeaders).forEach(([key, value]) => {
    headers.set(key, value);
  });

  return fetch(input, { ...init, headers });
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
    axiosInstance.interceptors.request.use((config: AxiosRequestConfig) => {
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
