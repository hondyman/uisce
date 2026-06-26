import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";

// ---- Types ----
export interface SourcePreference {
    id: string;
    tenant_id: string;
    business_object: string;
    semantic_term: string;
    region: string;
    priority: number;
    source_system: string;
    confidence: number;
    status: "draft" | "testing" | "staging" | "production";
    version: number;
    core_id?: string;
    override_reason?: string;
    valid_from: string;
    valid_to?: string;
    impact_analysis: ImpactAnalysis;
    created_at: string;
    updated_at: string;
}

export interface ImpactAnalysis {
    affected_dates: number;
    confidence_delta: number;
    business_impact: string;
    confidence_before: number;
    confidence_after: number;
    changed_dates: ChangedDate[];
}

export interface ChangedDate {
    date: string;
    old_source: string;
    new_source: string;
    old_confidence: number;
    new_confidence: number;
}

export interface SourceRanking {
    source_system: string;
    first_preference_count: number;
    second_preference_count: number;
    third_preference_count: number;
    other_preference_count: number;
    total_selections: number;
    first_preference_percent: number;
    avg_confidence: number;
}

export interface AnalyticsReport {
    rankings: SourceRanking[];
    business_object?: string;
    semantic_term?: string;
    region?: string;
    generated_at: string;
}

export interface SourceException {
    id: string;
    tenant_id: string;
    business_object: string;
    semantic_term?: string;
    region?: string;
    source_system: string;
    exception_type: string;
    description: string;
    impact_level: number;
    critical_path: boolean;
    status: "open" | "in_progress" | "resolved";
    metadata: Record<string, unknown>;
    created_at: string;
    resolved_at?: string;
}

// ---- Preferences ----
export function useSourcePreferences(bo?: string, term?: string, region?: string) {
    const params = new URLSearchParams();
    if (bo) params.set("business_object", bo);
    if (term) params.set("semantic_term", term);
    if (region) params.set("region", region);

    return useQuery<SourcePreference[]>({
        queryKey: ["source-preferences", bo, term, region],
        queryFn: () => api<SourcePreference[]>(`/sources/preferences?${params}`),
    });
}

export function useCreatePreference() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (pref: Partial<SourcePreference>) =>
            api<SourcePreference>("/sources/preferences", { method: "POST", body: JSON.stringify(pref) }),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["source-preferences"] }),
    });
}

export function useRequestOverride() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, reason, valid_to }: { id: string; reason: string; valid_to: string }) =>
            api<SourcePreference>(`/sources/preferences/${id}/override`, {
                method: "POST",
                body: JSON.stringify({ reason, valid_to }),
            }),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["source-preferences"] }),
    });
}

export function useApproveOverride() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, notes }: { id: string; notes?: string }) =>
            api<SourcePreference>(`/sources/preferences/${id}/approve`, {
                method: "POST",
                body: JSON.stringify({ notes }),
            }),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["source-preferences"] }),
    });
}

export function usePromoteStage() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) =>
            api<SourcePreference>(`/sources/preferences/${id}/promote`, { method: "POST" }),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["source-preferences"] }),
    });
}

// ---- Analytics ----
export function useSourceAnalytics(bo?: string, term?: string, region?: string) {
    const params = new URLSearchParams();
    if (bo) params.set("business_object", bo);
    if (term) params.set("semantic_term", term);
    if (region) params.set("region", region);

    return useQuery<AnalyticsReport>({
        queryKey: ["source-analytics", bo, term, region],
        queryFn: () => api<AnalyticsReport>(`/sources/analytics?${params}`),
    });
}

// ---- Exceptions ----
export function useSourceExceptions(status?: string) {
    const params = status ? `?status=${status}` : "";
    return useQuery<SourceException[]>({
        queryKey: ["source-exceptions", status],
        queryFn: () => api<SourceException[]>(`/sources/exceptions${params}`),
    });
}

export function useCreateException() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (exc: Partial<SourceException>) =>
            api<SourceException>("/sources/exceptions", { method: "POST", body: JSON.stringify(exc) }),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["source-exceptions"] }),
    });
}

export function useResolveException() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) =>
            api<void>(`/sources/exceptions/${id}/resolve`, { method: "POST" }),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["source-exceptions"] }),
    });
}

// ---- Impact Simulation ----
export function useImpactSimulation() {
    const [result, setResult] = useState<ImpactAnalysis | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const simulate = async (oldPref: SourcePreference, _newSystem: string, newConfidence: number) => {
        setLoading(true);
        setError(null);
        try {
            // Locally simulate; in production this would call /sources/preferences/{id}/override
            const delta = newConfidence - oldPref.confidence;
            const simulatedResult: ImpactAnalysis = {
                affected_dates: Math.round(365 * (Math.abs(delta) / 100)),
                confidence_delta: delta,
                business_impact: Math.abs(delta) <= 5 ? "low" : Math.abs(delta) <= 15 ? "moderate" : "high",
                confidence_before: oldPref.confidence,
                confidence_after: newConfidence,
                changed_dates: [],
            };
            setResult(simulatedResult);
        } catch (e: unknown) {
            setError(e instanceof Error ? e.message : "Simulation failed");
        } finally {
            setLoading(false);
        }
    };

    return { result, loading, error, simulate, reset: () => setResult(null) };
}
