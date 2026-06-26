import axios from 'axios';
import { getRequiredTenantScope, hasTenantScope } from './tenantScope';
import resolveApiUrl from './resolveApiUrl';

const axiosClient = axios.create();

axiosClient.interceptors.request.use((config) => {
    // Resolve base URL
    if (config.url && (config.url.startsWith('/api') || config.url.startsWith('/'))) {
        config.url = resolveApiUrl(config.url.startsWith('/api') ? config.url : `/api${config.url}`);
    }

    // Inject Tenant Scope
    if (hasTenantScope()) {
        const { tenantId, datasourceId } = getRequiredTenantScope();
        config.headers['X-Tenant-ID'] = tenantId;
        config.headers['X-Tenant-Datasource-ID'] = datasourceId;
    }

    // Inject Authorization Token
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
    if (token && token.split('.').length === 3 && !token.includes('demo')) {
        config.headers['Authorization'] = `Bearer ${token}`;
    }

    config.withCredentials = true;
    return config;
});

export default axiosClient;
