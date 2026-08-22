import { useCallback, useMemo } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTenant } from '../../../contexts/TenantContext';
import apiClient from '../../../utils/apiClient';

export interface RejectionEntry {
  rejection_id: string;
  tenant_id?: string;
  source_node_id: string;
  rejected_target_id: string;
  edge_type_id: string;
  rejected_by?: string;
  reason?: string;
  created_at?: string;
}

interface ListRejectionsResponse {
  data?: RejectionEntry[];
}

/**
 * Hook for the catalog_edge_rejection_store. Wraps the backend's
 * /api/semantic-mapper/rejections endpoints so callers can:
 *   1. Query existing rejections for a tenant (used to filter AI suggestions).
 *   2. Record a new rejection when the user dismisses a suggestion.
 *
 * Rejections are keyed by (source_node_id, rejected_target_id, edge_type_id).
 */
export function useRejectionStore() {
  const { tenant } = useTenant();
  const tenantId = tenant?.id;
  const queryClient = useQueryClient();
  const queryKey = ['rejections', tenantId] as const;

  const listQuery = useQuery({
    queryKey,
    queryFn: async (): Promise<RejectionEntry[]> => {
      if (!tenantId) return [];
      const res = await apiClient<ListRejectionsResponse | RejectionEntry[]>(
        '/api/semantic-mapper/rejections'
      );
      if (Array.isArray(res)) return res;
      return res.data ?? [];
    },
    enabled: !!tenantId,
    staleTime: 60_000,
  });

  const recordMutation = useMutation({
    mutationFn: async (payload: {
      sourceNodeId: string;
      rejectedTargetId: string;
      edgeTypeId: string;
      reason?: string;
    }) => {
      const body = {
        source_node_id: payload.sourceNodeId,
        rejected_target_id: payload.rejectedTargetId,
        edge_type_id: payload.edgeTypeId,
        reason: payload.reason ?? 'user_dismissed',
      };
      const res = await apiClient<{ status?: string }>(
        '/api/semantic-mapper/rejections',
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        }
      );
      return res;
    },
    onSuccess: () => {
      // Refresh the rejection list so subsequent queries see the new entry.
      void queryClient.invalidateQueries({ queryKey });
    },
  });

  const rejections = listQuery.data ?? [];

  /**
   * O(1) lookup by (source_node_id, rejected_target_id, edge_type_id).
   * Multiple rejections of the same triplet are deduped naturally.
   */
  const rejectionSet = useMemo(() => {
    const s = new Set<string>();
    for (const r of rejections) {
      s.add(makeRejectionKey(r.source_node_id, r.rejected_target_id, r.edge_type_id));
    }
    return s;
  }, [rejections]);

  const isRejected = useCallback(
    (sourceNodeId: string, rejectedTargetId: string, edgeTypeId?: string): boolean => {
      if (!edgeTypeId) {
        // Without an edge type, treat *any* rejection of the (source, target) pair as a hit.
        for (const r of rejections) {
          if (r.source_node_id === sourceNodeId && r.rejected_target_id === rejectedTargetId) {
            return true;
          }
        }
        return false;
      }
      return rejectionSet.has(makeRejectionKey(sourceNodeId, rejectedTargetId, edgeTypeId));
    },
    [rejections, rejectionSet]
  );

  const recordRejection = useCallback(
    async (payload: {
      sourceNodeId: string;
      targetNodeId: string;
      predicate: string;
      edgeTypeId?: string;
      reason?: string;
    }) => {
      await recordMutation.mutateAsync({
        sourceNodeId: payload.sourceNodeId,
        rejectedTargetId: payload.targetNodeId,
        edgeTypeId: payload.edgeTypeId ?? payload.predicate,
        reason: payload.reason,
      });
    },
    [recordMutation]
  );

  return {
    rejections,
    isLoading: listQuery.isLoading,
    error: listQuery.error as Error | null,
    isRejected,
    recordRejection,
    isRecording: recordMutation.isPending,
    recordError: recordMutation.error as Error | null,
    refetch: listQuery.refetch,
  };
}

/** Canonical key for a rejection record (used for O(1) Set lookup). */
export function makeRejectionKey(
  sourceNodeId: string,
  rejectedTargetId: string,
  edgeTypeId: string
): string {
  return `${sourceNodeId}::${rejectedTargetId}::${edgeTypeId}`;
}
