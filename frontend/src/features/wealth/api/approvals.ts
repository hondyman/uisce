import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiGet, apiPost } from '../../../utils/api';

export interface ApprovalRequest {
    id: string;
    tenant_id: string;
    client_id: string;
    action_type: string;
    action_details: Record<string, any>;
    requester_id: string;
    status: 'pending' | 'approved' | 'rejected';
    approver_id?: string;
    approval_comment?: string;
    created_at: string;
    updated_at: string;
    workflow_id: string;
    run_id: string;
}

export interface ApproveRequest {
    approver_id: string;
    comment: string;
}

export const fetchPendingApprovals = async (): Promise<ApprovalRequest[]> => {
    return await apiGet('wealth/approvals/pending');
};

export const approveRequest = async (id: string, data: ApproveRequest): Promise<ApprovalRequest> => {
    return await apiPost(`wealth/approvals/${id}/approve`, data);
};

export const rejectRequest = async (id: string, data: ApproveRequest): Promise<ApprovalRequest> => {
    return await apiPost(`wealth/approvals/${id}/reject`, data);
};

export const executeFeedAction = async (cardId: string, clientId: string, actionDetails: Record<string, any> = {}) => {
    return await apiPost('wealth/actions/execute', {
        card_id: cardId,
        client_id: clientId,
        action_details: actionDetails,
    });
};

export const usePendingApprovals = () => {
    return useQuery({
        queryKey: ['wealth-approvals'],
        queryFn: () => fetchPendingApprovals(),
        refetchInterval: 10000,
    });
};

export const useApproveRequest = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: ApproveRequest }) => approveRequest(id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['wealth-approvals'] });
        },
    });
};

export const useRejectRequest = () => {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: ApproveRequest }) => rejectRequest(id, data),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['wealth-approvals'] });
        },
    });
};

export const useExecuteFeedAction = () => {
    return useMutation({
        mutationFn: ({ cardId, clientId, actionDetails }: {
            cardId: string;
            clientId: string;
            actionDetails?: Record<string, any>;
        }) => executeFeedAction(cardId, clientId, actionDetails),
    });
};
