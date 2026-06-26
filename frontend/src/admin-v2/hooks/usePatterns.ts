import { useQuery } from "@tanstack/react-query";
import { useAdminAPIClient } from "./useAdminAPI";

// Pattern types (matching backend)
export interface IncidentPattern {
  id: string;
  event_signature: string[];
  severity: "info" | "warning" | "error" | "critical";
  timeline_minutes: number;
  affected_services: string[];
  recurrence_count: number;
  first_seen: string;
  last_seen: string;
  successful_fixes: string[];
  average_duration: number;
  confidence: number;
}

export interface IncidentSimilarity {
  incident_1_id: string;
  incident_2_id: string;
  similarity_score: number;
  matched_events: number;
  pattern_id?: string;
}

/**
 * useIncidentPattern - Fetch pattern fingerprint for an incident
 * @param incidentId - UUID of the incident
 * @returns Pattern with event signature and metadata
 */
export const useIncidentPattern = (incidentId: string) => {
  const client = useAdminAPIClient();

  return useQuery({
    queryKey: ["pattern", incidentId],
    queryFn: async () => {
      const response = await client.get<IncidentPattern>(
        `/ops/incidents/${incidentId}/pattern`
      );
      return response;
    },
    enabled: !!incidentId,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
};

/**
 * useSimilarIncidents - Find historically similar incidents
 * @param incidentId - UUID of the incident
 * @returns List of similar incidents with similarity scores
 */
export const useSimilarIncidents = (incidentId: string) => {
  const client = useAdminAPIClient();

  return useQuery({
    queryKey: ["similar", incidentId],
    queryFn: async () => {
      const response = await client.get<{
        incident_id: string;
        similarities: IncidentSimilarity[];
      }>(`/ops/incidents/${incidentId}/similar`);
      return response;
    },
    enabled: !!incidentId,
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
};

/**
 * formatEventSignature - Convert event signature array to human-readable string
 */
export const formatEventSignature = (signature: string[]): string => {
  if (!signature || signature.length === 0) {
    return "No events";
  }

  return signature
    .map((event) => event.replace(/_/g, " ").toUpperCase())
    .join(" → ");
};

/**
 * formatPattern - Create human-readable pattern description
 */
export const formatPattern = (pattern: IncidentPattern): string => {
  const duration = pattern.timeline_minutes;
  const durationStr =
    duration < 60 ? `${duration} min` : `${Math.round(duration / 60)} hr`;

  return `${formatEventSignature(pattern.event_signature)} (${durationStr})`;
};

/**
 * getSimilarityColor - Get color for similarity confidence
 */
export const getSimilarityColor = (score: number): string => {
  if (score >= 0.9) return "#00a854"; // Green
  if (score >= 0.7) return "#108ee9"; // Blue
  if (score >= 0.5) return "#faad14"; // Orange
  return "#999999"; // Gray
};

/**
 * getSimilarityLabel - Get human-readable similarity label
 */
export const getSimilarityLabel = (score: number): string => {
  if (score >= 0.9) return "Very Similar";
  if (score >= 0.7) return "Similar";
  if (score >= 0.5) return "Moderately Similar";
  return "Possibly Related";
};

/**
 * getConfidencePercentage - Convert 0-1 confidence to percentage
 */
export const getConfidencePercentage = (confidence: number): number => {
  return Math.round(confidence * 100);
};
