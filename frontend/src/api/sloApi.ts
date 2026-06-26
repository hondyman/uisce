import axios from 'axios';

export interface SLODefinition {
    id: string;
    env: string;
    tenant_id?: string;
    scope_type: 'bo' | 'preagg' | 'entitlement' | 'planner';
    scope_id: string; // e.g. BO name
    slo_type: 'latency' | 'freshness' | 'error_rate' | 'preagg_hit_rate';
    target: number;
    time_window: string;
    enabled: boolean;
    created_at: string;
    updated_at: string;
}

export interface SLOEvaluation {
    id: string;
    slo_id: string;
    env: string;
    tenant_id?: string;
    scope_type: string;
    scope_id: string;
    window_start: string;
    window_end: string;
    measured_value: number;
    target_value: number;
    status: 'met' | 'violated' | 'unknown';
    delta_percent?: number;
    created_at: string;
}

export interface SLOViolation {
    id: string;
    slo_id: string;
    evaluation_id?: string;
    env: string;
    tenant_id?: string;
    scope_type: string;
    scope_id: string;
    slo_type: string;
    target_value: number;
    actual_value: number;
    severity: 'critical' | 'warning' | 'info';
    created_at: string;
}

interface ListSLOParams {
    env?: string;
    scope_type?: string;
    scope_id?: string;
}

import { getRequiredTenantScope, hasTenantScope } from '../utils/tenantScope';
import { getSelectedRegion } from '../lib/region';

const getHeaders = () => {
    const headers: Record<string, string> = {
        'X-Tenant-Region': getSelectedRegion() || '',
    };
    if (hasTenantScope()) {
        const { tenantId, datasourceId } = getRequiredTenantScope();
        headers['X-Tenant-ID'] = tenantId;
        headers['X-Tenant-Datasource-ID'] = datasourceId;
    }
    return headers;
};

export const sloApi = {
    listSLOs: async (params: ListSLOParams = {}): Promise<SLODefinition[]> => {
        const resp = await axios.get('/api/slos', { params, headers: getHeaders() });
        return resp.data;
    },

    createSLO: async (slo: Partial<SLODefinition>): Promise<SLODefinition> => {
        const resp = await axios.post('/api/slos', slo, { headers: getHeaders() });
        return resp.data;
    },

    updateSLO: async (id: string, slo: Partial<SLODefinition>): Promise<SLODefinition> => {
        const resp = await axios.put(`/api/slos/${id}`, slo, { headers: getHeaders() });
        return resp.data;
    },

    deleteSLO: async (id: string): Promise<void> => {
        await axios.delete(`/api/slos/${id}`, { headers: getHeaders() });
    },
};
