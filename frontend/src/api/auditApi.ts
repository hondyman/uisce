import axios from 'axios';

const API_BASE = '/api/audit';

export interface EntitySnapshot {
    version_id: string;
    valid_from: string;
    valid_to?: string;
    system_from: string;
    system_to?: string;
    change_type: 'INSERT' | 'UPDATE' | 'DELETE' | 'RESTORE';
    changed_by: string;
    change_reason?: string;
    entity_data: Record<string, any>;
    is_current: boolean;
    is_deleted: boolean;
}

export interface HistoryResponse {
    entity_type: string;
    entity_id: string;
    history: EntitySnapshot[];
    count: number;
}

export interface HistoryFilters {
    from?: string;
    to?: string;
    validFrom?: string;
    validTo?: string;
    includeDeleted?: boolean;
    limit?: number;
    offset?: number;
}

export interface RestoreRequest {
    restoreToTime: string;
    reason: string;
}

export interface RestoreResponse {
    success: boolean;
    entity_type: string;
    entity_id: string;
    restore_to_time: string;
    reason: string;
}

export const auditApi = {
    /**
     * Get full history of an entity
     */
    getEntityHistory: async (
        entityType: string,
        entityId: string,
        filters?: HistoryFilters
    ): Promise<HistoryResponse> => {
        const params = new URLSearchParams();
        if (filters?.from) params.append('from', filters.from);
        if (filters?.to) params.append('to', filters.to);
        if (filters?.validFrom) params.append('validFrom', filters.validFrom);
        if (filters?.validTo) params.append('validTo', filters.validTo);
        if (filters?.includeDeleted) params.append('includeDeleted', 'true');
        if (filters?.limit) params.append('limit', filters.limit.toString());
        if (filters?.offset) params.append('offset', filters.offset.toString());

        const response = await axios.get(
            `${API_BASE}/history/${entityType}/${entityId}?${params.toString()}`
        );
        return response.data;
    },

    /**
     * Get entity state at a specific point in time
     */
    getEntityAtTime: async (
        entityType: string,
        entityId: string,
        timestamp: string
    ): Promise<EntitySnapshot> => {
        const response = await axios.get(
            `${API_BASE}/history/${entityType}/${entityId}/at/${timestamp}`
        );
        return response.data;
    },

    /**
     * Restore entity to a previous state
     */
    restoreEntity: async (
        entityType: string,
        entityId: string,
        request: RestoreRequest
    ): Promise<RestoreResponse> => {
        const response = await axios.post(
            `${API_BASE}/restore/${entityType}/${entityId}`,
            request
        );
        return response.data;
    },

    /**
     * Get all audit changes with filters
     */
    getAllChanges: async (filters?: HistoryFilters): Promise<any> => {
        const params = new URLSearchParams();
        if (filters?.from) params.append('from', filters.from);
        if (filters?.to) params.append('to', filters.to);
        if (filters?.limit) params.append('limit', filters.limit.toString());
        if (filters?.offset) params.append('offset', filters.offset.toString());

        const response = await axios.get(`${API_BASE}/changes?${params.toString()}`);
        return response.data;
    },
};
