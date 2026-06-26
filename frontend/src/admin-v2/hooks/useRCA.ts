import { useQuery } from "@tanstack/react-query";
import { useAdminAPIClient } from "./useAdminAPI";

// RCA types (matching backend)
export interface ScoredEvent {
  event: any; // Full event object
  causality_score: number;
  impact_score: number;
  correlated_events?: CorrelationScore[];
}

export interface CorrelationScore {
  from_event_id: string;
  to_event_id: string;
  score: number;
  time_gap_ms: number;
  reason_scores: Record<string, number>;
  primary_reason: string;
}

export interface RemediationSuggestion {
  action_type: string;
  priority: "high" | "medium" | "low";
  confidence: number;
  reason: string;
  recurrence_count: number;
}

export interface RCAResult {
  suspected_root_cause: ScoredEvent | null;
  causality_chain: ScoredEvent[];
  affected_services: string[];
  suggested_remediations: RemediationSuggestion[];
  confidence_score: number;
}

/**
 * useRCA - Fetch intelligent RCA analysis for an incident
 * @param incidentId - UUID of the incident
 * @returns RCA result with root cause, causality chain, and suggestions
 */
export const useRCA = (incidentId: string) => {
  const client = useAdminAPIClient();

  return useQuery({
    queryKey: ["rca", incidentId],
    queryFn: async () => {
      const response = await client.get<RCAResult>(
        `/ops/incidents/${incidentId}/rca`
      );
      return response;
    },
    enabled: !!incidentId,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
};

/**
 * formatSeverityScore - Convert confidence score to human-readable label
 */
export const formatConfidenceScore = (score: number): string => {
  if (score >= 0.9) return "Very High Confidence";
  if (score >= 0.7) return "High Confidence";
  if (score >= 0.5) return "Moderate Confidence";
  if (score >= 0.3) return "Low Confidence";
  return "Uncertain";
};

/**
 * formatConfidenceColor - Get color class for confidence level
 */
export const formatConfidenceColor = (score: number): string => {
  if (score >= 0.9) return "#00a854"; // Green
  if (score >= 0.7) return "#108ee9"; // Blue
  if (score >= 0.5) return "#faad14"; // Orange
  if (score >= 0.3) return "#f5222d"; // Red
  return "#999999"; // Gray
};

/**
 * formatEventChain - Convert causality chain to readable description
 */
export const formatEventChain = (chain: ScoredEvent[]): string => {
  if (chain.length === 0) return "No clear causality chain";

  return chain
    .map((scored, idx) => {
      const event = scored.event;
      const typeLabel = event.event_type?.replace(/_/g, " ") || "Unknown";
      const confidence = Math.round(scored.causality_score * 100);
      return `${idx + 1}. ${typeLabel} (${confidence}% causality)`;
    })
    .join(" → ");
};

/**
 * getRemediationIcon - Get icon for remediation action type
 */
export const getRemediationIcon = (actionType: string): string => {
  switch (actionType) {
    case "restart_worker":
      return "🔄";
    case "throttle_tenant":
      return "🚦";
    case "circuit_breaker_toggle":
      return "⚡";
    case "failover_toggle":
      return "🔀";
    case "trigger_runbook":
      return "📋";
    default:
      return "⚙️";
  }
};

/**
 * getRemediationLabel - Get human-readable action label
 */
export const getRemediationLabel = (actionType: string): string => {
  switch (actionType) {
    case "restart_worker":
      return "Restart Worker";
    case "throttle_tenant":
      return "Throttle Tenant";
    case "circuit_breaker_toggle":
      return "Toggle Circuit Breaker";
    case "failover_toggle":
      return "Failover to Replica";
    case "trigger_runbook":
      return "Trigger Runbook";
    default:
      return actionType;
  }
};

/**
 * getPriorityColor - Get color for remediation priority
 */
export const getPriorityColor = (priority: string): string => {
  switch (priority) {
    case "high":
      return "#f5222d";
    case "medium":
      return "#faad14";
    case "low":
      return "#13c2c2";
    default:
      return "#999999";
  }
};
