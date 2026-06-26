import { fetchAPI } from '../api';

export interface BusinessTermCompliance {
    id: string;
    name: string;
    description: string;
    piiFlag: boolean;
    residency: string;
    sensitivity: string;
    semanticTerms: SemanticTermSummary[];
}

export interface SemanticTermSummary {
    id: string;
    name: string;
}

export interface AddMappingsRequest {
    semanticTermIds: string[];
}

export interface UpdateComplianceRequest {
    piiFlag?: boolean;
    residency?: string;
    sensitivity?: string;
}

export const catalogApi = {
    getBusinessTerm: async (id: string): Promise<BusinessTermCompliance> => {
        return fetchAPI(`/catalog/business-terms/${id}`);
    },

    addMappings: async (id: string, req: AddMappingsRequest): Promise<void> => {
        return fetchAPI(`/catalog/business-terms/${id}/mappings`, {
            method: 'POST',
            body: JSON.stringify(req),
        });
    },

    removeMapping: async (id: string, semId: string): Promise<void> => {
        return fetchAPI(`/catalog/business-terms/${id}/mappings/${semId}`, {
            method: 'DELETE',
        });
    },

    updateCompliance: async (id: string, req: UpdateComplianceRequest): Promise<void> => {
        return fetchAPI(`/catalog/business-terms/${id}/compliance`, {
            method: 'PUT',
            body: JSON.stringify(req),
        });
    },

    // AI Suggestions
    listSuggestions: async (): Promise<AIBusinessTermDraft[]> => {
        return fetchAPI('/catalog/ai/suggestions');
    },

    generateSuggestion: async (input: { tableNames: string[]; context?: string }): Promise<AIBusinessTermDraft> => {
        return fetchAPI('/catalog/ai/generate', {
            method: 'POST',
            body: JSON.stringify(input),
        });
    },

    approveSuggestion: async (id: string): Promise<void> => {
        return fetchAPI(`/catalog/ai/suggestions/${id}/approve`, {
            method: 'POST',
        });
    },

    rejectSuggestion: async (id: string, reason: string): Promise<void> => {
        return fetchAPI(`/catalog/ai/suggestions/${id}/reject`, {
            method: 'POST',
            body: JSON.stringify({ reason }),
        });
    },

    getSemanticTermsByTable: async (tableId: string, datasourceId: string): Promise<any[]> => {
        return fetchAPI(`/catalog/semantic-terms-by-table/${tableId}`, {
            headers: {
                'X-Tenant-Datasource-ID': datasourceId,
            },
        }).then((response: any) => response.semanticTerms || []);
    },
};

export interface AIBusinessTermDraft {
    id: string; // matches businessTermId from AI
    name: string;
    definition: string;
    piiFlag: boolean;
    sensitivity: 'LOW' | 'MEDIUM' | 'HIGH';
    residency: string;
    hierarchy: {
        level1: string;
        level2: string;
        level3: string;
    };
    sourceSemanticTerms: string[];
    sourceColumns: string[];
    tags: string[];
    status: 'DRAFT_AI' | 'APPROVED' | 'REJECTED';
    createdAt: string;
}
