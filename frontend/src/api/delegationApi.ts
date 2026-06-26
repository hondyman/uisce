import { apiGet, apiPost } from '../utils/api';

export interface DelegationRequest {
    to_user_id: string;
    from_date: string;
    to_date: string;
    reason: string;
    roles: string[];
    workflows: string[];
}

export interface Delegation {
    id: string;
    from_user_id: string;
    to_user_id: string;
    start_date: string;
    end_date: string;
    status: string;
    reason: string;
    roles: string[];
    workflows: string[];
    created_at: string;
}

export interface DelegationResponse {
    delegations: Delegation[];
}

export const delegationApi = {
    createDelegation: async (data: DelegationRequest): Promise<{ delegation_id: string; status: string }> => {
        return apiPost('delegations', data);
    },

    getIncomingDelegations: async (): Promise<Delegation[]> => {
        const data = await apiGet('delegations/incoming');
        return data.delegations || [];
    },

    getOutgoingDelegations: async (): Promise<Delegation[]> => {
        const data = await apiGet('delegations/outgoing');
        return data.delegations || [];
    },

    revokeDelegation: async (delegationId: string): Promise<void> => {
        return apiPost(`delegations/${delegationId}/revoke`);
    },
};
