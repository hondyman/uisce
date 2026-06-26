import { fetchAPI } from '../../../api';
import { ImpactGraphData, ImpactSummary, NodeType } from '../types';

export const impactApi = {
    getGraph: async (nodeType: NodeType, nodeId: string, depth: number = 3): Promise<ImpactGraphData> => {
        return fetchAPI(`/impact/graph/${nodeType}/${nodeId}?depth=${depth}`);
    },

    // New method to fetch lineage graph (uses catalog_node/edge tables via relational backend)
    getLineageGraph: async (nodeId: string, depth: number = 3): Promise<ImpactGraphData> => {
        return fetchAPI(`/lineage/node/${nodeId}/graph?depth=${depth}`);
    },

    getExplanation: async (nodeType: NodeType, nodeId: string, depth: number = 3): Promise<ImpactSummary> => {
        return fetchAPI(`/impact/explain/${nodeType}/${nodeId}?depth=${depth}`);
    },

    query: async (query: string, context?: { nodeType: NodeType; nodeId: string }): Promise<any> => {
        return fetchAPI('/impact/query', {
            method: 'POST',
            body: JSON.stringify({
                query,
                entityId: context?.nodeId,
                entityType: context?.nodeType
            }),
        });
    }
};
