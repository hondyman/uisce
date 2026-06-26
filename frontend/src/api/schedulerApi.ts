/**
 * Scheduler Intelligence API Client
 * Provides typed access to the scheduler backend API
 */

import axios from 'axios';
import { useState, useEffect, useCallback } from 'react';

// ============================================================================
// Types
// ============================================================================

export interface Compliance {
    pii: boolean;
    residency: string; // US, EU, GLOBAL
    sensitivity: string; // LOW, MEDIUM, HIGH
}

export interface Job {
    id: string;
    tenant_id: string;
    datasource_id?: string;
    name: string;
    description?: string;
    category: string;
    job_type: string;
    parameters?: Record<string, unknown>;
    semantic_bindings?: SemanticBinding;
    schedule_type: 'cron' | 'event' | 'predictive' | 'manual';
    cron_expression?: string;
    timezone: string;
    calendar_ids?: string[];
    timeout_seconds: number;
    priority: number;
    retry_policy?: RetryPolicy;
    is_active: boolean;
    slo_critical: boolean;
    compliance_tags?: string[];
    compliance?: Compliance;
    last_run_at?: string;
    next_run_at?: string;
    created_at: string;
    updated_at: string;
}

export interface SemanticBinding {
    bo_ids?: string[];
    api_ids?: string[];
    page_ids?: string[];
    workflow_ids?: string[];
    preagg_ids?: string[];
}

export interface RetryPolicy {
    max_attempts: number;
    initial_interval_seconds: number;
    backoff_coefficient: number;
    max_interval_seconds?: number;
}

export interface DAG {
    id: string;
    tenant_id: string;
    name: string;
    description?: string;
    category?: string;
    nodes: DAGNode[];
    edges: DAGEdge[];
    schedule_type?: string;
    cron_expression?: string;
    max_parallel_jobs: number;
    fail_fast: boolean;
    timeout_seconds: number;
    semantic_bindings?: SemanticBinding;
    is_active: boolean;
    slo_critical: boolean;
    last_run_at?: string;
    next_run_at?: string;
    created_at: string;
    updated_at: string;
}

export interface DAGNode {
    id: string;
    job_id: string;
    conditions?: Record<string, unknown>;
    position?: { x: number; y: number };
}

export interface DAGEdge {
    from_node_id: string;
    to_node_id: string;
    type?: 'success' | 'completion' | 'any';
    conditions?: Record<string, unknown>;
}

export interface JobRun {
    id: string;
    job_id: string;
    dag_run_id?: string;
    tenant_id: string;
    temporal_workflow_id?: string;
    temporal_run_id?: string;
    status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'paused';
    attempt_number: number;
    trigger_type: 'scheduled' | 'manual' | 'event' | 'api';
    triggered_by?: string;
    scheduled_at?: string;
    started_at?: string;
    completed_at?: string;
    duration_ms?: number;
    error_message?: string;
    slo_target_ms?: number;
    slo_breached: boolean;
    semantic_bindings?: SemanticBinding;
    created_at: string;
}

export interface DAGRun {
    id: string;
    dag_id: string;
    tenant_id: string;
    temporal_workflow_id?: string;
    temporal_run_id?: string;
    status: string;
    trigger_type: string;
    triggered_by?: string;
    scheduled_at?: string;
    started_at?: string;
    completed_at?: string;
    duration_ms?: number;
    completed_jobs: number;
    failed_jobs: number;
    skipped_jobs: number;
    error_message?: string;
    created_at: string;
}

export interface AISuggestion {
    id: string;
    tenant_id: string;
    suggestion_type: string;
    target_type?: string;
    target_id?: string;
    title: string;
    description?: string;
    impact_summary?: string;
    risk_level: 'low' | 'medium' | 'high' | 'critical';
    proposed_changes: Record<string, unknown>;
    status: 'pending' | 'accepted' | 'dismissed' | 'snoozed';
    created_at: string;
}

export interface JobStats {
    total_jobs: number;
    active_jobs: number;
    running_jobs: number;
    failed_last_24h: number;
    succeeded_last_24h: number;
    slo_critical_jobs: number;
    slo_breached_jobs: number;
    error_budget_consumed: number; // Percentage 0-100
    active_tenants?: number;
}

export interface CreateJobRequest {
    name: string;
    description?: string;
    category: string;
    job_type: string;
    input_parameters?: Record<string, unknown>;
    output_properties?: Record<string, unknown>;
    semantic_bindings?: SemanticBinding;
    created_at: string;
    cron_expression?: string;
    timezone?: string;
    calendar_ids?: string[];
    timeout_seconds?: number;
    priority?: number;
    retry_policy?: RetryPolicy;
    slo_critical?: boolean;
    compliance_tags?: string[];
}

export interface UpdateJobRequest {
    name?: string;
    description?: string;
    category?: string;
    parameters?: Record<string, unknown>;
    schedule_type?: string;
    cron_expression?: string;
    timezone?: string;
    timeout_seconds?: number;
    priority?: number;
    is_active?: boolean;
}

export interface JobListFilters {
    category?: string;
    is_active?: boolean;
    slo_critical?: boolean;
    limit?: number;
    offset?: number;
}

// ============================================================================
// API Functions
// ============================================================================

const schedulerAxios = axios.create({
    baseURL: '/api/scheduler'
});

// Interceptor to add actor and tenant headers
schedulerAxios.interceptors.request.use((config) => {
    const role = localStorage.getItem('scheduler_actor_role');
    const tenantId = localStorage.getItem('scheduler_tenant_id');

    if (role) {
        config.headers['X-Actor-Type'] = role;
    }
    if (tenantId) {
        config.headers['X-Tenant-Id'] = tenantId;
    }

    return config;
});

// Jobs
export async function listJobs(tenantId: string, filters?: JobListFilters): Promise<{ jobs: Job[]; total: number }> {
    const params = new URLSearchParams();
    if (tenantId) params.set('tenant_id', tenantId);
    if (filters?.category) params.set('category', filters.category);
    if (filters?.is_active !== undefined) params.set('is_active', String(filters.is_active));
    if (filters?.slo_critical !== undefined) params.set('slo_critical', String(filters.slo_critical));
    if (filters?.limit) params.set('limit', String(filters.limit));
    if (filters?.offset) params.set('offset', String(filters.offset));

    const response = await schedulerAxios.get(`/jobs?${params}`);
    return response.data;
}

export async function getJob(jobId: string): Promise<Job> {
    const response = await schedulerAxios.get(`/jobs/${jobId}`);
    return response.data;
}

export async function createJob(tenantId: string, request: CreateJobRequest): Promise<Job> {
    const response = await schedulerAxios.post(`/jobs?tenant_id=${tenantId}`, request);
    return response.data;
}

export async function updateJob(jobId: string, request: UpdateJobRequest): Promise<Job> {
    const response = await schedulerAxios.patch(`/jobs/${jobId}`, request);
    return response.data;
}

export async function deleteJob(jobId: string): Promise<void> {
    await schedulerAxios.delete(`/jobs/${jobId}`);
}

export async function triggerJob(jobId: string, parameters?: Record<string, unknown>, executionMode?: string): Promise<JobRun> {
    const response = await schedulerAxios.post(`/jobs/${jobId}/run`, { parameters, execution_mode: executionMode });
    return response.data;
}

export async function getJobRuns(jobId: string, limit?: number): Promise<JobRun[]> {
    const params = new URLSearchParams();
    if (limit) params.set('limit', String(limit));
    const response = await schedulerAxios.get(`/jobs/${jobId}/runs?${params}`);
    return response.data;
}

// DAGs
export async function listDAGs(tenantId: string, activeOnly?: boolean): Promise<DAG[]> {
    const params = new URLSearchParams();
    if (tenantId) params.set('tenant_id', tenantId);
    if (activeOnly) params.set('active_only', 'true');
    const response = await schedulerAxios.get(`/dags?${params}`);
    return response.data;
}

export async function getDAG(dagId: string): Promise<DAG> {
    const response = await schedulerAxios.get(`/dags/${dagId}`);
    return response.data;
}

export async function createDAG(tenantId: string, dag: Partial<DAG>): Promise<DAG> {
    const response = await schedulerAxios.post(`/dags?tenant_id=${tenantId}`, dag);
    return response.data;
}

export async function updateDAG(dagId: string, updates: Partial<DAG>): Promise<DAG> {
    const response = await schedulerAxios.patch(`/dags/${dagId}`, updates);
    return response.data;
}

export async function deleteDAG(dagId: string): Promise<void> {
    await schedulerAxios.delete(`/dags/${dagId}`);
}

export async function triggerDAG(dagId: string, executionMode?: string): Promise<DAGRun> {
    const response = await schedulerAxios.post(`/dags/${dagId}/run`, { execution_mode: executionMode });
    return response.data;
}

export async function getDAGRuns(dagId: string, limit?: number): Promise<DAGRun[]> {
    const params = new URLSearchParams();
    if (limit) params.set('limit', String(limit));
    const response = await schedulerAxios.get(`/dags/${dagId}/runs?${params}`);
    return response.data;
}

// Runs
export async function getJobRun(runId: string): Promise<JobRun> {
    const response = await schedulerAxios.get(`/runs/jobs/${runId}`);
    return response.data;
}

export async function getDAGRun(runId: string): Promise<DAGRun> {
    const response = await schedulerAxios.get(`/runs/dags/${runId}`);
    return response.data;
}

// AI Suggestions
export async function getAISuggestions(tenantId: string): Promise<AISuggestion[]> {
    const response = await schedulerAxios.get(`/ai/suggestions?tenant_id=${tenantId}`);
    return response.data;
}

export async function acceptAISuggestion(suggestionId: string): Promise<void> {
    await schedulerAxios.post(`/ai/suggestions/${suggestionId}/accept`, {});
}

export async function dismissAISuggestion(suggestionId: string, reason?: string): Promise<void> {
    await schedulerAxios.post(`/ai/suggestions/${suggestionId}/dismiss`, { reason });
}

// Stats
export async function getSchedulerStats(tenantId: string): Promise<JobStats> {
    const response = await schedulerAxios.get(`/stats?tenant_id=${tenantId}`);
    return response.data;
}

// ============================================================================
// Hooks
// ============================================================================

export function useJobs(tenantId: string, filters?: JobListFilters) {
    const [jobs, setJobs] = useState<Job[]>([]);
    const [total, setTotal] = useState(0);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    const refetch = useCallback(async () => {
        if (!tenantId) return;
        setLoading(true);
        try {
            const result = await listJobs(tenantId, filters);
            setJobs(result.jobs);
            setTotal(result.total);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err : new Error('Failed to fetch jobs'));
        } finally {
            setLoading(false);
        }
    }, [tenantId, filters?.category, filters?.is_active, filters?.limit, filters?.offset]);

    useEffect(() => {
        refetch();
    }, [refetch]);

    return { jobs, total, loading, error, refetch };
}

export function useDAGs(tenantId: string, activeOnly?: boolean) {
    const [dags, setDAGs] = useState<DAG[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    const refetch = useCallback(async () => {
        if (!tenantId) return;
        setLoading(true);
        try {
            const result = await listDAGs(tenantId, activeOnly);
            setDAGs(result);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err : new Error('Failed to fetch DAGs'));
        } finally {
            setLoading(false);
        }
    }, [tenantId, activeOnly]);

    useEffect(() => {
        refetch();
    }, [refetch]);

    return { dags, loading, error, refetch };
}

export function useSchedulerStats(tenantId: string) {
    const [stats, setStats] = useState<JobStats | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    const refetch = useCallback(async () => {
        if (!tenantId) return;
        setLoading(true);
        try {
            const result = await getSchedulerStats(tenantId);
            setStats(result);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err : new Error('Failed to fetch stats'));
        } finally {
            setLoading(false);
        }
    }, [tenantId]);

    useEffect(() => {
        refetch();
    }, [refetch]);

    return { stats, loading, error, refetch };
}

export function useAISuggestions(tenantId: string) {
    const [suggestions, setSuggestions] = useState<AISuggestion[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    const refetch = useCallback(async () => {
        if (!tenantId) return;
        setLoading(true);
        try {
            const result = await getAISuggestions(tenantId);
            setSuggestions(result);
            setError(null);
        } catch (err) {
            setError(err instanceof Error ? err : new Error('Failed to fetch suggestions'));
        } finally {
            setLoading(false);
        }
    }, [tenantId]);

    useEffect(() => {
        refetch();
    }, [refetch]);

    return { suggestions, loading, error, refetch };
}
