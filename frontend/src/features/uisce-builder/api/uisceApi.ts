import axios from '@/utils/axiosClient';

// Types matching backend TraceResult
export interface DebugStep {
    filterName: string;
    stage?: string;
    inputSnapshot: Record<string, any>;
    status: 'PASS' | 'FAIL';
    errorDetails?: string;
    durationMs: number;
}

export interface TraceResult {
    tradeId: string;
    steps: DebugStep[];
    success: boolean;
}

export interface DebugRequest {
    tradeData: Record<string, any>;
}

const API_BASE = '/api';

export interface SimulationResult {
    success: boolean;
    events: string[];
    output?: Record<string, any>;
    errorMessage?: string;
}

// Deprecated: use runSimulation
export async function runDebugTrace(tradeData: Record<string, any>): Promise<TraceResult> {
    const response = await axios.post<TraceResult>(`${API_BASE}/uisce/debug`, {
        tradeData,
    });
    return response.data;
}

export async function runSimulation(pipelineId: string, formData: Record<string, any>): Promise<SimulationResult> {
    const response = await axios.post<SimulationResult>(`${API_BASE}/v1/pipelines/${pipelineId}/simulate`, {
        formData,
    });
    return response.data;
}
