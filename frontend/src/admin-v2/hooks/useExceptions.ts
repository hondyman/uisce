import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";

/**
 * Types mirroring backend/internal/platform_intelligence/exceptions/aggregator.go
 */
export type ExceptionType =
  | "slo_breach"
  | "semantic_drift"
  | "security_anomaly"
  | "data_quality"
  | "residency_violation"
  | "accessibility_violation"
  | "api_inconsistency"
  | "preagg_inconsistency"
  | "tenant_anomaly"
  | "pii_exposure";

export type ExceptionStatus =
  | "open"
  | "acknowledged"
  | "auto_fix_pending"
  | "auto_fixed"
  | "resolved"
  | "closed"
  | "ignored";

export interface AutofixAttempt {
  attempted_at: string;
  action: string;
  success: boolean;
  verified: boolean;
  detail?: string;
}

export interface PlatformException {
  id: string;
  tenant_id: string;
  type: ExceptionType;
  severity: "critical" | "high" | "medium" | "low";
  source: string;
  description: string;
  evidence: string[];
  fingerprint: string;
  occurrence_count: number;
  first_seen: string;
  last_seen: string;
  status: ExceptionStatus;
  resolved_at?: string;
  resolved_by?: string;
  closed_by_ai: boolean;
  autofix_attempts: AutofixAttempt[];
}

export interface ExceptionSummary {
  total_exceptions: number;
  critical_count: number;
  high_count: number;
  medium_count: number;
  low_count: number;
  by_type: Record<string, number>;
  recent_exceptions: PlatformException[];
  top_affected_tenants: string[];
}

export interface AutofixPolicy {
  id: string;
  tenant_id: string;
  user_id?: string;
  exception_type: ExceptionType;
  enabled: boolean;
  requires_approval: boolean;
  updated_by: string;
  updated_at: string;
}

const BASE = "/api/platform-intelligence/exceptions";

/** Fetch every open/closed exception for the current tenant. */
export function useExceptions() {
  return useQuery({
    queryKey: ["exceptions", "all"],
    queryFn: () => api<PlatformException[]>(BASE + "/all"),
  });
}

/** Fetch the exceptions summary (counts by severity/type) for dashboard tiles. */
export function useExceptionSummary() {
  return useQuery({
    queryKey: ["exceptions", "summary"],
    queryFn: () => api<ExceptionSummary>(BASE + "/summary"),
  });
}

/** Fetch the tenant's per-exception-type autofix toggles. */
export function useAutofixPolicies() {
  return useQuery({
    queryKey: ["exceptions", "autofix-policy"],
    queryFn: () => api<AutofixPolicy[]>(BASE + "/autofix-policy"),
  });
}

/**
 * Set a single (tenant[, user], exception_type) autofix toggle. Never a
 * global switch — always scoped to one exception type at a time.
 */
export function useSetAutofixPolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (policy: Partial<AutofixPolicy> & { exception_type: ExceptionType; enabled: boolean }) =>
      api<void>(BASE + "/autofix-policy", {
        method: "PUT",
        body: JSON.stringify(policy),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["exceptions", "autofix-policy"] });
    },
  });
}

/** Manually trigger a re-verify of one exception. */
export function useRerunException() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (exceptionId: string) =>
      api<void>(`${BASE}/${exceptionId}/rerun`, { method: "POST" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["exceptions"] });
    },
  });
}

/** Close an exception as a human operator. */
export function useCloseException() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (exceptionId: string) =>
      api<void>(`${BASE}/${exceptionId}/close`, { method: "POST" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["exceptions"] });
    },
  });
}

/** AI-generated root-cause explanation + fix suggestion for one exception. */
export function useExceptionAISuggestion(exceptionId: string | null) {
  return useQuery({
    queryKey: ["exceptions", "ai-suggestion", exceptionId],
    queryFn: () =>
      api<{ suggestion: string }>(`${BASE}/${exceptionId}/ai-suggestion`),
    enabled: !!exceptionId,
  });
}
