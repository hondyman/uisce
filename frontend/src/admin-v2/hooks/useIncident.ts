import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { OpsEvent, OpsIncident, IncidentResponse } from "./useOpsTimeline";

/**
 * Hook for fetching a single incident with all its events
 */
export function useIncident(id: string | null) {
  return useQuery({
    queryKey: ["incident", id],
    queryFn: async () => {
      const { incident, events } = await api<IncidentResponse>(
        `/admin/ops/incidents/${id}`
      );
      return { incident, events };
    },
    enabled: !!id,
  });
}

/**
 * Hook for closing an incident with summary and root cause
 */
export function useCloseIncidentWithAnalysis(incidentId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: { summary?: string; rootCause?: string }) => {
      const response = await api<{ closed: boolean }>(
        `/admin/ops/incidents/${incidentId}/close`,
        {
          method: "POST",
          body: JSON.stringify({
            summary: data.summary,
            root_cause: data.rootCause,
          }),
        }
      );
      return response;
    },
    onSuccess: () => {
      // Invalidate incident query to refetch updated data
      queryClient.invalidateQueries({ queryKey: ["incident", incidentId] });
      // Invalidate timeline so incident shows as closed
      queryClient.invalidateQueries({ queryKey: ["opsTimeline"] });
    },
  });
}

/**
 * Types for incident view
 */
export interface IncidentDetailState {
  incidentId: string;
  incident?: OpsIncident;
  events: OpsEvent[];
  isLoading: boolean;
  isError: boolean;
}

export interface IncidentRCAResult {
  suspectedRootCause?: OpsEvent;
  affectedTenants: Set<string>;
  affectedEndpoints: Set<string>;
  eventTimeline: OpsEvent[];
}

/**
 * Helper function to compute suspected root cause from incident events
 * RCA Strategy: earliest critical event is likely root cause
 */
export function computeIncidentRCA(
  events: OpsEvent[]
): IncidentRCAResult {
  // Find earliest critical event
  const criticalEvents = events.filter((e) => e.severity === "critical");
  const suspectedRootCause =
    criticalEvents.length > 0
      ? criticalEvents.reduce((earliest, current) =>
          new Date(current.occurred_at) < new Date(earliest.occurred_at)
            ? current
            : earliest
        )
      : undefined;

  // Affected blast radius
  const affectedTenants = new Set<string>();
  const affectedEndpoints = new Set<string>();

  for (const e of events) {
    if (e.tenant_id) affectedTenants.add(e.tenant_id);
    if (e.endpoint_path) affectedEndpoints.add(e.endpoint_path);
  }

  return {
    suspectedRootCause,
    affectedTenants,
    affectedEndpoints,
    eventTimeline: events,
  };
}
