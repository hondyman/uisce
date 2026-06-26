import { fetchAPI } from '../api';

export interface ExecutionLog {
    id: string;
    event_type: string;
    status: string;
    engine: string;
    payload: any;
    result: any;
    started_at: string;
    completed_at?: string;
    duration_ms?: number;
    user_id?: string;
    tenant_id?: string;
    error_message?: string;
    calculation_id?: string;
    workflow_id?: string;
    run_id?: string;
    created_at: string;
}

export function listExecutionLogs(limit: number = 50, offset: number = 0): Promise<ExecutionLog[]> {
    return fetchAPI(`/execution-logs?limit=${limit}&offset=${offset}`);
}
