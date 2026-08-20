import { apiClient } from '../utils/apiClient';
import { APIEndpoint, APICatalog, APITest, APITestRun } from '../types/apiStudio';

const API_BASE = '/api-studio';

export const ApiStudioApi = {
    // Endpoints
    listEndpoints: async (env: string): Promise<APIEndpoint[]> => {
        return apiClient<APIEndpoint[]>(`${API_BASE}/endpoints?env=${env}`);
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
    generateEndpointWithAI: async (prompt: string): Promise<APIEndpoint> => {
        return apiClient<APIEndpoint>(`${API_BASE}/endpoints/ai`, {
            method: 'POST',
            body: JSON.stringify({ prompt }),
            headers: { 'Content-Type': 'application/json' }
        });
    },

    // OpenAPI
    getOpenApiSpec: async (env: string): Promise<any> => {
        return apiClient<any>(`${API_BASE}/openapi?env=${env}`);
    },

    // Runtime Preview (calling the actual runtime)
    previewEndpoint: async (path: string, method: string, env: string, params: any): Promise<any> => {
        const url = `/runtime${path}`;
        // ApiClient handles API prefixing

        // Pass validation headers for the runtime
        const headers = {
            'Content-Type': 'application/json',
            'X-Env': env,
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
    getSdkURL: (lang: string, env: string) => {
        return `/api/api-studio/sdk/${lang}?env=${env}`;
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
