import axios from '@/utils/axiosClient';

// --- Types ---

export interface ModelChangeProposalPayload {
    clientId: string;
    fromModelId: string;
    toModelId: string;
    accountIds: string[];
    rationale: string;
    initiatorUserId: string;
}

export interface BPInstanceStartResponse {
    instanceId: string;
    status: string;
}

export interface ModelChangeInstanceDetails {
    id: string;
    status: string;
    currentStepId: string;
    proposal: {
        fromModelId: string;
        toModelId: string;
        accountIds: string[];
        rationale: string;
    };
    clientExplanation?: string;
    suitabilityResult?: {
        status: 'OK' | 'WARNING' | 'HARD_FAIL';
        warnings: string[];
    };
    clientApproval?: {
        outcome: 'approved' | 'rejected';
        timestamp: string;
    };
    events: any[]; // Matches TaskTimeline Event type
    externalTasks: any[]; // Matches ExternalTask type
    report?: any;
}

// --- API Calls ---

export const startModelChangeBP = async (payload: ModelChangeProposalPayload): Promise<BPInstanceStartResponse> => {
    // In a real app this would POST to /api/workflows/start
    // for now we mock it or assume the route exists
    const response = await axios.post('/api/workflows/start', {
        bpDefinitionId: 'bp.model_change_with_client_approval.v1',
        businessObjectId: payload.clientId,
        initiatorUserId: payload.initiatorUserId,
        payload: {
            proposal_free_text: payload.rationale,
            from_model: payload.fromModelId,
            to_model: payload.toModelId,
            selected_accounts: payload.accountIds,
        },
    });
    return response.data;
};

export const getModelChangeInstance = async (instanceId: string): Promise<ModelChangeInstanceDetails> => {
    const response = await axios.get(`/api/workflows/instances/${instanceId}`);
    return response.data;
};

export const approveModelChange = async (instanceId: string, outcome: 'approved' | 'rejected'): Promise<void> => {
    await axios.post(`/api/workflows/instances/${instanceId}/signal`, {
        signalName: 'ClientApprovalSignal',
        payload: { outcome }
    });
};

export const updateProposal = async (instanceId: string, updates: Partial<ModelChangeProposalPayload>): Promise<void> => {
    // Advisor updating proposal at S3
    await axios.post(`/api/workflows/instances/${instanceId}/Signal`, {
        signalName: 'AdvisorUpdateSignal',
        payload: { updates }
    });
};

export const sendToClient = async (instanceId: string, explanation: string): Promise<void> => {
    // Advisor completing S4
    await axios.post(`/api/workflows/instances/${instanceId}/step/S4/complete`, {
        output: { client_explanation: explanation }
    });
};
