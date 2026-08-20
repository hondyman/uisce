import axios from 'axios';
import { getRequiredTenantScope, hasTenantScope, readCachedSelection } from './tenantScope';
import resolveApiUrl from './resolveApiUrl';

const axiosClient = axios.create();

axiosClient.interceptors.request.use((config) => {
    // Resolve base URL
    if (config.url && (config.url.startsWith('/api') || config.url.startsWith('/'))) {
        config.url = resolveApiUrl(config.url.startsWith('/api') ? config.url : `/api${config.url}`);
    }

    // Inject Tenant Scope
    try {
        const { tenant, datasource } = readCachedSelection();
        if (tenant?.id) {
            config.headers['X-Tenant-ID'] = tenant.id;
        }
        const datasourceId = datasource?.id || datasource?.alpha_tenant_instance_id;
        if (datasourceId) {
            config.headers['X-Tenant-Datasource-ID'] = datasourceId;
        }
    } catch (_) {}

    // Inject Authorization Token
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
    if (token && token.split('.').length === 3 && !token.includes('demo')) {
        config.headers['Authorization'] = `Bearer ${token}`;
    }

    config.withCredentials = true;
    return config;
});

export default axiosClient;
