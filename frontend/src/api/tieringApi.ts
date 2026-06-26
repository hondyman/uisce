import axios from 'axios';

export type StorageTier = 'hot' | 'warm' | 'cold' | 'archive';
export type PlanStatus = 'pending' | 'migrating' | 'completed' | 'dismissed';

export interface TieringRule {
    tableName: string;
    condition: string;
    targetTier: StorageTier;
    rationale: string;
    dataVolume: string;
    costSavings: string;
}

export interface TieringPlan {
    id: string;
    tenant_id: string;
    rules: TieringRule[];
    summary: string;
    status: PlanStatus;
    created_at: string;
    updated_at: string;
}

const API_BASE = '/api/intelligence/storage';

export const tieringApi = {
    listPlans: async (tenantId: string): Promise<TieringPlan[]> => {
        const response = await axios.get(`${API_BASE}/plans`, {
            params: { tenant_id: tenantId }
        });
        return response.data;
    },

    generatePlan: async (tenantId: string): Promise<TieringPlan> => {
        const response = await axios.post(`${API_BASE}/plans/generate`, {
            tenant_id: tenantId
        });
        return response.data;
    },

    getPlan: async (id: string): Promise<TieringPlan> => {
        const response = await axios.get(`${API_BASE}/plans/${id}`);
        return response.data;
    },

    executePlan: async (id: string): Promise<{ status: string }> => {
        const response = await axios.post(`${API_BASE}/plans/${id}/execute`);
        return response.data;
    },

    updateStatus: async (id: string, status: PlanStatus): Promise<void> => {
        await axios.post(`${API_BASE}/plans/${id}/status`, { status });
    }
};
