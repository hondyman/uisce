import { apiGet, apiPost } from '../utils/api';

export interface OffboardingRequest {
    user_id: string;
    reassign_to_user_id: string;
    reason: string;
}

export interface OffboardingRecord {
    id: string;
    tenant_id: string;
    user_id: string;
    reassigned_to: string;
    initiated_by: string;
    reason: string;
    status: string;
    created_at: string;
}

export const offboardingApi = {
    offboardUser: async (data: OffboardingRequest): Promise<{ offboarding_id: string }> => {
        return apiPost('admin/offboard', data);
    },

    listOffboardings: async (limit = 50, offset = 0): Promise<{ offboardings: OffboardingRecord[], total: number }> => {
        return apiGet(`admin/offboarding?limit=${limit}&offset=${offset}`);
    },

    reverseOffboarding: async (id: string): Promise<void> => {
        return apiPost(`admin/offboarding/${id}/reverse`);
    },
};
