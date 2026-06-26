import { useState, useCallback, useEffect } from 'react';

// ============================================================================
// Types
// ============================================================================

export interface ASOPolicy {
    id: string;
    env: string;
    tenant_id?: string;
    enabled: boolean;
    mode: 'advisory' | 'auto_tune' | 'auto_apply';
    max_new_preaggs_per_day: number;
    max_changes_per_day: number;
    min_score_for_new_preagg: number;
    min_usage_for_retirement: number;
    hot_path_threshold_ms: number;
    lookback_window_seconds: number;
    prewarm_enabled: boolean;
    prewarm_lead_time_minutes: number;
    created_at: string;
    updated_at: string;
    created_by: string;
    updated_by: string;
}

export interface ASOOptimization {
    id: string;
    env: string;
    tenant_id?: string;
    scope: 'core' | 'tenant';
    optimization_type: 'tune_refresh' | 'tune_definition' | 'create_preagg' | 'retire_asset' | 'prewarm';
    target_type: 'preagg' | 'bo' | 'calc' | 'term';
    target_id: string;
    target_name: string;
    status: 'proposed' | 'approved' | 'applied' | 'rejected' | 'failed' | 'superseded';
    mode: string;
    score: number;
    reason: string;
    details: Record<string, any>;
    workload_window_days: number;
    queries_per_day?: number;
    avg_latency_ms?: number;
    p95_latency_ms?: number;
    avg_rows_scanned?: number;
    policy_id?: string;
    created_at: string;
    created_by: string;
    approved_at?: string;
    approved_by?: string;
    applied_at?: string;
    applied_by?: string;
    rejected_at?: string;
    rejected_by?: string;
    rejection_reason?: string;
    before_config?: Record<string, any>;
    after_config?: Record<string, any>;

    // ML & Simulation
    ml_score?: number;
    predicted_speedup?: number;
    predicted_cost_savings?: number;
    risk_score?: number;
    confidence?: number;
    top_factors?: any[];
}

export interface ASOSummary {
    env: string;
    policy_enabled: boolean;
    policy_mode: string;
    optimizations_today: number;
    optimizations_pending: number;
    optimizations_applied_7d: number;
    hot_paths_detected: number;
    retirement_candidates: number;
    last_evaluated_at?: string;
}

// ============================================================================
// API Functions
// ============================================================================

const getHeaders = (tenantId?: string, datasourceId?: string) => {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json',
    };
    if (tenantId) headers['X-Tenant-ID'] = tenantId;
    if (datasourceId) headers['X-Tenant-Datasource-ID'] = datasourceId;
    return headers;
};

export async function fetchASOSummary(): Promise<Record<string, ASOSummary>> {
    const res = await fetch('/api/aso/summary', { headers: getHeaders() });
    if (!res.ok) throw new Error('Failed to fetch ASO summary');
    return res.json();
}

export async function fetchASOSummaryByEnv(env: string): Promise<ASOSummary> {
    const res = await fetch(`/api/aso/summary/${env}`, { headers: getHeaders() });
    if (!res.ok) throw new Error('Failed to fetch ASO summary');
    return res.json();
}

export async function fetchASOPolicies(): Promise<ASOPolicy[]> {
    const res = await fetch('/api/aso/policies', { headers: getHeaders() });
    if (!res.ok) throw new Error('Failed to fetch ASO policies');
    return res.json();
}

export async function fetchASOPoliciesByEnv(env: string): Promise<ASOPolicy[]> {
    const res = await fetch(`/api/aso/policies/${env}`, { headers: getHeaders() });
    if (!res.ok) throw new Error('Failed to fetch ASO policies');
    return res.json();
}

export async function upsertASOPolicy(policy: Partial<ASOPolicy>): Promise<ASOPolicy> {
    const res = await fetch('/api/aso/policies', {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify(policy),
    });
    if (!res.ok) throw new Error('Failed to upsert ASO policy');
    return res.json();
}

export async function deleteASOPolicy(id: string): Promise<void> {
    const res = await fetch(`/api/aso/policies/${id}`, {
        method: 'DELETE',
        headers: getHeaders(),
    });
    if (!res.ok) throw new Error('Failed to delete ASO policy');
}

export interface OptimizationFilter {
    env?: string;
    status?: string;
    type?: string;
    target_type?: string;
    limit?: number;
    offset?: number;
    tenantId?: string;
}

export async function fetchASOOptimizations(filter: OptimizationFilter = {}): Promise<ASOOptimization[]> {
    const params = new URLSearchParams();
    if (filter.env) params.append('env', filter.env);
    if (filter.status) params.append('status', filter.status);
    if (filter.type) params.append('type', filter.type);
    if (filter.target_type) params.append('target_type', filter.target_type);
    if (filter.limit) params.append('limit', filter.limit.toString());
    if (filter.offset) params.append('offset', filter.offset.toString());

    const res = await fetch(`/api/aso/optimizations?${params}`, { headers: getHeaders() });
    if (!res.ok) throw new Error('Failed to fetch ASO optimizations');
    return res.json();
}

export async function fetchASOOptimization(id: string): Promise<ASOOptimization> {
    const res = await fetch(`/api/aso/optimizations/${id}`, { headers: getHeaders() });
    if (!res.ok) throw new Error('Failed to fetch ASO optimization');
    return res.json();
}

export async function applyASOOptimization(id: string): Promise<ASOOptimization> {
    const res = await fetch(`/api/aso/optimizations/${id}/apply`, {
        method: 'POST',
        headers: getHeaders(),
    });
    if (!res.ok) throw new Error('Failed to apply ASO optimization');
    return res.json();
}

export async function approveASOOptimization(id: string): Promise<ASOOptimization> {
    const res = await fetch(`/api/aso/optimizations/${id}/approve`, {
        method: 'POST',
        headers: getHeaders(),
    });
    if (!res.ok) throw new Error('Failed to approve ASO optimization');
    return res.json();
}

export async function rejectASOOptimization(id: string, reason: string): Promise<ASOOptimization> {
    const res = await fetch(`/api/aso/optimizations/${id}/reject`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ reason }),
    });
    if (!res.ok) throw new Error('Failed to reject ASO optimization');
    return res.json();
}

export async function triggerASOEvaluation(env: string, tenantId?: string): Promise<{ optimizations_found: number }> {
    const url = tenantId ? `/api/aso/evaluate/${env}/${tenantId}` : `/api/aso/evaluate/${env}`;
    const res = await fetch(url, {
        method: 'POST',
        headers: getHeaders(),
    });
    if (!res.ok) throw new Error('Failed to trigger ASO evaluation');
    return res.json();
}

// ============================================================================
// Hooks
// ============================================================================

export function useASOSummary() {
    const [summaries, setSummaries] = useState<Record<string, ASOSummary>>({});
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const refresh = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const data = await fetchASOSummary();
            setSummaries(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        refresh();
    }, [refresh]);

    return { summaries, loading, error, refresh };
}

export function useASOPolicies(env?: string) {
    const [policies, setPolicies] = useState<ASOPolicy[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const refresh = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const data = env ? await fetchASOPoliciesByEnv(env) : await fetchASOPolicies();
            setPolicies(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load');
        } finally {
            setLoading(false);
        }
    }, [env]);

    useEffect(() => {
        refresh();
    }, [refresh]);

    return { policies, loading, error, refresh };
}

export function useASOOptimizations(filter: OptimizationFilter = {}) {
    const [optimizations, setOptimizations] = useState<ASOOptimization[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const refresh = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const data = await fetchASOOptimizations(filter);
            setOptimizations(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load');
        } finally {
            setLoading(false);
        }
    }, [JSON.stringify(filter)]);

    useEffect(() => {
        refresh();
    }, [refresh]);

    return { optimizations, loading, error, refresh };
}

export function useASOOptimization(id: string | undefined) {
    const [optimization, setOptimization] = useState<ASOOptimization | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const refresh = useCallback(async () => {
        if (!id) return;
        setLoading(true);
        setError(null);
        try {
            const data = await fetchASOOptimization(id);
            setOptimization(data);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load');
        } finally {
            setLoading(false);
        }
    }, [id]);

    useEffect(() => {
        refresh();
    }, [refresh]);

    return { optimization, loading, error, refresh };
}
