import { useQuery, useMutation } from "@tanstack/react-query";
import { api } from "../api";

// Types
export interface OpsEvent {
  id: string;
  incident_id?: string;
  event_type:
    | "alert"
    | "fingerprint"
    | "tenant_health"
    | "endpoint_health"
    | "latency_anomaly"
    | "incident_opened"
    | "incident_closed";
  scope: "global" | "tenant" | "endpoint" | "region";
  tenant_id?: string;
  endpoint_path?: string;
  region?: string;
  fingerprint_id?: string;
  alert_id?: string;
  severity: "info" | "warning" | "error" | "critical";
  title: string;
  details: Record<string, any>;
  occurred_at: string;
  created_at: string;
}

export interface OpsIncident {
  id: string;
  status: "open" | "closed";
  severity: "info" | "warning" | "error" | "critical";
  title: string;
  summary?: string;
  root_cause?: string;
  started_at: string;
  ended_at?: string;
  created_at: string;
  updated_at: string;
  events?: OpsEvent[];
}

export interface TimelineResponse {
  events: OpsEvent[];
  total: number;
}

export interface IncidentResponse {
  incident: OpsIncident;
  events: OpsEvent[];
}

// Hooks

/**
 * Fetch recent events from ops timeline
 * @param since Duration string like "1h", "24h", "7d" (default: "1h")
 * @param limit Maximum events to fetch (default: 200)
 */
export function useOpsTimeline(since: string = "1h", limit: number = 200) {
  return useQuery({
    queryKey: ["opsTimeline", since, limit],
    queryFn: () =>
      api<TimelineResponse>(
        `/api/admin/ops/timeline?since=${since}&limit=${limit}`
      ),
    refetchInterval: 30_000, // Refetch every 30 seconds
  });
}

/**
 * Fetch a specific incident with all its events
 * @param incidentId UUID of incident to fetch
 */
export function useOpsIncident(incidentId: string | null) {
  return useQuery({
    queryKey: ["opsIncident", incidentId],
    queryFn: () => api<IncidentResponse>(`/api/admin/ops/incidents/${incidentId}`),
    enabled: !!incidentId, // Only fetch if ID is provided
  });
}

/**
 * Close an incident with optional summary and root cause
 */
export function useCloseIncident() {
  return useMutation({
    mutationFn: (params: {
      incidentId: string;
      summary?: string;
      rootCause?: string;
    }) =>
      api(`/api/admin/ops/incidents/${params.incidentId}/close`, {
        method: "POST",
        body: JSON.stringify({
          summary: params.summary,
          root_cause: params.rootCause,
        }),
      }),
  });
}
