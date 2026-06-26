import { apiClient } from '../utils/apiClient';
import { APIEndpoint, APICatalog, APITest, APITestRun } from '../types/apiStudio';

const API_BASE = '/api-studio';

export const ApiStudioApi = {
    // Endpoints
    listEndpoints: async (env: string, tenantId: string): Promise<APIEndpoint[]> => {
        return apiClient<APIEndpoint[]>(`${API_BASE}/endpoints?env=${env}&tenant_id=${tenantId}`);
    },
    getEndpoint: async (id: string): Promise<APIEndpoint> => {
        return apiClient<APIEndpoint>(`${API_BASE}/endpoints/${id}`);
    },
    saveEndpoint: async (endpoint: Partial<APIEndpoint>): Promise<APIEndpoint> => {
        return apiClient<APIEndpoint>(`${API_BASE}/endpoints`, {
            method: 'POST',
            body: JSON.stringify(endpoint),
            headers: { 'Content-Type': 'application/json' }
        });
    },
    deprecateEndpoint: async (id: string): Promise<APIEndpoint> => {
        return apiClient<APIEndpoint>(`${API_BASE}/endpoints/${id}/deprecate`, { method: 'POST' });
    },
    retireEndpoint: async (id: string): Promise<APIEndpoint> => {
        return apiClient<APIEndpoint>(`${API_BASE}/endpoints/${id}/retire`, { method: 'POST' });
    },
    generateEndpointWithAI: async (prompt: string, tenantId: string): Promise<APIEndpoint> => {
        return apiClient<APIEndpoint>(`${API_BASE}/endpoints/ai`, {
            method: 'POST',
            body: JSON.stringify({ prompt, tenant_id: tenantId }),
            headers: { 'Content-Type': 'application/json' }
        });
    },

    // OpenAPI
    getOpenApiSpec: async (env: string, tenantId: string): Promise<any> => {
        return apiClient<any>(`${API_BASE}/openapi?env=${env}&tenant_id=${tenantId}`);
    },

    // Runtime Preview (calling the actual runtime)
    previewEndpoint: async (path: string, method: string, env: string, tenantId: string, params: any): Promise<any> => {
        const url = `/runtime${path}`;
        // ApiClient handles API prefixing

        // Pass validation headers for the runtime
        const headers = {
            'Content-Type': 'application/json',
            'X-Env': env,
            'X-Tenant-ID': tenantId
        };

        const config: RequestInit = {
            method,
            headers
        };

        if (method === 'GET') {
            const qs = new URLSearchParams(params).toString();
            return apiClient(`${url}?${qs}`, config);
        } else {
            config.body = JSON.stringify(params);
            return apiClient(url, config);
        }
    },

    // Performance & DX
    getSdkURL: (lang: string, env: string, tenantId: string) => {
        // This is a direct URL for valid hrefs, so we keep using the API base directly regarding path
        // but removing the /api prefix from the constant if apiClient adds it? 
        // Actually apiClient adds /api if missing. Here we return a string URL.
        // Let's assume the component consuming this knows to prepend host or use relative path.
        // Since API_BASE is now just /api-studio (apiClient handles /api prefix for requests),
        // but for a link href we need the full path.
        return `/api/api-studio/sdk/${lang}?env=${env}&tenant_id=${tenantId}`;
    },
    getEndpointMetrics: async (id: string): Promise<any> => {
        // In a real system, this would call obs_metrics
        // For now, we return mock performance data
        return Promise.resolve({
            p50: 120,
            p95: 180,
            p99: 250,
            qps: 45.5,
            cacheHitRate: 0.82,
            preaggHitRate: 0.95
        });
    }
};
