import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

export interface ExecuteActionRequest {
  action_type: string;
  parameters: Record<string, any>;
}

export interface ExecuteActionResponse {
  action_history_id: string;
  status: "success" | "failed";
  action_type: string;
  result?: Record<string, any>;
  error_msg?: string;
  timeline_event_id?: string;
}

export interface ActionHistory {
  id: string;
  incident_id: string;
  action_type: string;
  status: "pending" | "success" | "failed";
  parameters: Record<string, any>;
  result?: Record<string, any>;
  error_msg?: string;
  executed_at: string;
  created_at: string;
  updated_at: string;
}

/**
 * useExecuteOpsAction - Execute an ops action on an incident
 */
export function useExecuteOpsAction(incidentId: string | null) {
  return useMutation<ExecuteActionResponse, Error, ExecuteActionRequest>({
    mutationFn: async (req: ExecuteActionRequest) => {
      if (!incidentId) throw new Error("Incident ID required");

      const response = await fetch(
        `/api/admin/ops/incidents/${incidentId}/execute-action`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(req),
        }
      );

      if (!response.ok) {
        const error = await response.text();
        throw new Error(error || "Failed to execute action");
      }

      return response.json();
    },
    onSuccess: () => {
      // Invalidate both incident and timeline queries to refresh
      const queryClient = useQueryClient();
      queryClient.invalidateQueries({ queryKey: ["incident", incidentId] });
      queryClient.invalidateQueries({ queryKey: ["opsTimeline"] });
    },
  });
}

/**
 * useIncidentActions - Get action history for an incident
 */
export function useIncidentActions(incidentId: string | null, limit: number = 50) {
  return useQuery<ActionHistory[]>({
    queryKey: ["incidentActions", incidentId],
    queryFn: async () => {
      if (!incidentId) return [];

      const response = await fetch(
        `/api/admin/ops/incidents/${incidentId}/actions?limit=${limit}`
      );

      if (!response.ok) throw new Error("Failed to fetch actions");
      const data = await response.json();
      return data.data || [];
    },
    enabled: !!incidentId,
  });
}

/**
 * Available ops actions
 */
export const OPS_ACTIONS = [
  {
    id: "restart_worker",
    name: "Restart Worker",
    description: "Gracefully restart unhealthy worker service",
    icon: "🔄",
    severity: "medium" as const,
    requiresConfirm: true,
    parameters: [
      {
        name: "worker_id",
        label: "Worker ID",
        type: "text" as const,
        placeholder: "e.g., worker-1",
        required: true,
      },
      {
        name: "force",
        label: "Force Kill (skip graceful shutdown)",
        type: "checkbox" as const,
        required: false,
      },
    ],
  },
  {
    id: "throttle_tenant",
    name: "Throttle Tenant",
    description: "Rate-limit tenant to contain blast radius",
    icon: "🛑",
    severity: "high" as const,
    requiresConfirm: true,
    parameters: [
      {
        name: "tenant_id",
        label: "Tenant ID",
        type: "text" as const,
        placeholder: "e.g., tenant-abc123",
        required: true,
      },
      {
        name: "rate_limit_per_sec",
        label: "Rate Limit (requests/sec)",
        type: "number" as const,
        placeholder: "100",
        required: true,
        defaultValue: 100,
      },
      {
        name: "duration",
        label: "Duration",
        type: "select" as const,
        required: true,
        options: [
          { label: "5 minutes", value: "5m" },
          { label: "10 minutes", value: "10m" },
          { label: "15 minutes", value: "15m" },
          { label: "30 minutes", value: "30m" },
          { label: "1 hour", value: "1h" },
        ],
        defaultValue: "10m",
      },
    ],
  },
  {
    id: "trigger_runbook",
    name: "Trigger Runbook",
    description: "Execute workflow runbook to resolve incident",
    icon: "📋",
    severity: "high" as const,
    requiresConfirm: true,
    parameters: [
      {
        name: "runbook_id",
        label: "Runbook ID",
        type: "text" as const,
        placeholder: "e.g., incident-response-1",
        required: true,
      },
      {
        name: "variables",
        label: "Variables (JSON)",
        type: "text" as const,
        placeholder: '{"timeout": 300, "retries": 3}',
        required: false,
      },
    ],
  },
  {
    id: "circuit_breaker_toggle",
    name: "Toggle Circuit Breaker",
    description: "Open/close circuit breaker to prevent cascading failures",
    icon: "⚡",
    severity: "high" as const,
    requiresConfirm: true,
    parameters: [
      {
        name: "circuit_id",
        label: "Circuit ID",
        type: "text" as const,
        placeholder: "e.g., api-gateway-db",
        required: true,
      },
      {
        name: "target_state",
        label: "Target State",
        type: "select" as const,
        required: true,
        options: [
          { label: "Open (block traffic)", value: "open" },
          { label: "Closed (allow traffic)", value: "closed" },
          { label: "Half-open (test)", value: "half-open" },
        ],
        defaultValue: "open",
      },
      {
        name: "duration_secs",
        label: "Duration (seconds)",
        type: "number" as const,
        placeholder: "300",
        required: false,
        defaultValue: 300,
      },
    ],
  },
  {
    id: "failover_toggle",
    name: "Failover to Replica",
    description: "Failover traffic to standby region or replica",
    icon: "🔀",
    severity: "critical" as const,
    requiresConfirm: true,
    parameters: [
      {
        name: "source_region",
        label: "Source Region",
        type: "select" as const,
        required: true,
        options: [
          { label: "US-East", value: "us-east-1" },
          { label: "US-West", value: "us-west-2" },
          { label: "EU-West", value: "eu-west-1" },
          { label: "Asia-Pacific", value: "ap-southeast-1" },
        ],
      },
      {
        name: "target_region",
        label: "Target Region",
        type: "select" as const,
        required: true,
        options: [
          { label: "US-East", value: "us-east-1" },
          { label: "US-West", value: "us-west-2" },
          { label: "EU-West", value: "eu-west-1" },
          { label: "Asia-Pacific", value: "ap-southeast-1" },
        ],
      },
      {
        name: "immediate",
        label: "Immediate Failover (skip graceful migration)",
        type: "checkbox" as const,
        required: false,
      },
    ],
  },
];

export function getActionById(id: string) {
  return OPS_ACTIONS.find((a) => a.id === id);
}

export function formatActionResult(action: ActionHistory): string {
  if (action.status === "failed") {
    return action.error_msg || "Action failed";
  }

  if (!action.result) return "Action completed";

  const result = action.result as Record<string, any>;
  switch (action.action_type) {
    case "restart_worker":
      return `Restarted ${result.worker_id} (${result.jobs_requeued} jobs requeued)`;
    case "throttle_tenant":
      return `Throttled ${result.tenant_id} to ${result.rate_limit_per_sec} req/sec`;
    case "trigger_runbook":
      return `Executed runbook ${result.runbook_id} (${result.status})`;
    case "circuit_breaker_toggle":
      return `Circuit ${result.circuit_id} changed to ${result.new_state}`;
    case "failover_toggle":
      return `Failover from ${result.source_region} to ${result.target_region}`;
    default:
      return "Action completed";
  }
}
