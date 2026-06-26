import axios from 'axios';
import { NLRequest, NLResponse, QueryPlan, IncidentExplanation, ChangeSetProposal, ForecastResult } from '../types/nl-intelligence';

const API_BASE = '/api/nl';

export const NLIntelligenceApi = {
    interpret: async (req: NLRequest): Promise<NLResponse> => {
        const resp = await axios.post(`${API_BASE}/interpret`, req);
        return resp.data;
    },
    execute: async (plan: QueryPlan): Promise<any> => {
        const resp = await axios.post(`${API_BASE}/execute`, plan);
        return resp.data;
    },
    summarize: async (question: string, result: any): Promise<string> => {
        const resp = await axios.post(`${API_BASE}/summarize`, { question, result });
        return resp.data.explanation;
    },
    explainIncident: async (incidentId: string, graphContext: any): Promise<IncidentExplanation> => {
        const resp = await axios.post(`${API_BASE}/explain-incident`, { incident_id: incidentId, graph_context: graphContext });
        return resp.data;
    },
    proposeChangeSet: async (problemContext: any): Promise<ChangeSetProposal> => {
        const resp = await axios.post(`${API_BASE}/propose-changeset`, { problem_context: problemContext });
        return resp.data;
    },
    predictFailures: async (history: any, graph: any): Promise<ForecastResult> => {
        const resp = await axios.post(`${API_BASE}/predict-failures`, { history, graph });
        return resp.data;
    }
};
