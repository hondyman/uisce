import * as React from 'react';
import { useMemo, useState } from 'react';
import { CircularProgress, IconButton, Tooltip } from '@mui/material';
import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTenant } from '../../../contexts/TenantContext';
import apiClient from '../../../utils/apiClient';
import type { CatalogNode } from '../../../api/glossary';
import { getPredicate } from '../constants/predicates';
import { useRejectionStore } from '../hooks/useRejectionStore';

/**
 * The 4 high-signal predicates surfaced to users as AI candidates.
 * Other predicates (HAS_BUSINESS_TERM, BO_RELATIONSHIP) are typically
 * authoring-side and don't need an AI confirmation step.
 */
const AI_CANDIDATE_PREDICATES = [
  'IS_SPECIALIZATION_OF',
  'IS_PEER_IDENTIFIER_OF',
  'DIFFERENTIATED_FROM',
  'RELATES_TO',
] as const;

export interface AISuggestion {
  term_id: string;
  term_name: string;
  qualified_path?: string;
  category?: string;
  data_type?: string;
  domain?: string;
  role?: string;
  relationship_type: string;
  differentiation_notes?: string;
  format_pattern?: string;
  standard?: string;
  confidence: number;
  is_gold_copy?: boolean;
}

export interface AISuggestionCardProps {
  entityType: 'business_term' | 'semantic_term' | 'business_object';
  entityId: string;
  focalNode?: CatalogNode;
  /**
   * Optional override for the suggestion fetch endpoint. Defaults to
   * `/api/semantic-terms/{id}/related` (works for both business_term and
   * semantic_term). For business_object callers should pass their own endpoint.
   */
  fetchEndpoint?: string;
  /** Optional suggestion override — if set, skip the fetch and use these directly. */
  suggestions?: AISuggestion[];
  /** Called after an Accept or Reject completes (e.g. to refetch the relationship list). */
  onSuggestionApplied?: () => void;
  /** Hide the card entirely. */
  disabled?: boolean;
}

interface TermDisambiguationResponse {
  primary_term?: AISuggestion;
  related_terms?: AISuggestion[];
  differentiator_summary?: string;
  domain_scope?: string;
}

/**
 * AISuggestionCard — reads candidate edges from the backend and lets the user
 * Accept (creates an edge via /api/semantic-terms/relationships) or Reject
 * (persists to catalog_edge_rejection_store via /api/semantic-mapper/rejections).
 * Already-rejected suggestions are filtered out client-side using
 * useRejectionStore.
 */
export const AISuggestionCard: React.FC<AISuggestionCardProps> = ({
  entityType,
  entityId,
  focalNode,
  fetchEndpoint,
  suggestions: providedSuggestions,
  onSuggestionApplied,
  disabled,
}) => {
  const { tenant } = useTenant();
  const tenantId = tenant?.id;
  const queryClient = useQueryClient();

  const { isRejected, recordRejection, isRecording } = useRejectionStore();

  const endpoint =
    fetchEndpoint ??
    (entityType === 'business_term' || entityType === 'semantic_term'
      ? `/api/semantic-terms/${encodeURIComponent(entityId)}/related`
      : null);

  // For business_object the backend doesn't expose a /related endpoint yet,
  // so callers must supply suggestions explicitly.
  const enabled =
    !disabled &&
    !!tenantId &&
    !!endpoint &&
    providedSuggestions === undefined;

  const disambigQuery = useQuery({
    queryKey: ['ai-suggestions', entityType, entityId, tenantId],
    queryFn: async (): Promise<TermDisambiguationResponse> => {
      if (!endpoint) return {};
      const res = await apiClient<TermDisambiguationResponse>(endpoint);
      return res;
    },
    enabled,
    staleTime: 60_000,
  });

  const allSuggestions: AISuggestion[] = useMemo(() => {
    if (providedSuggestions) return providedSuggestions;
    return disambigQuery.data?.related_terms ?? [];
  }, [providedSuggestions, disambigQuery.data]);

  // Filter out rejected suggestions (client-side; the backend will also
  // exclude them next fetch thanks to useRejectionStore.refetch on success).
  const visibleSuggestions = useMemo(
    () =>
      allSuggestions.filter(
        (s) => !isRejected(entityId, s.term_id, s.relationship_type)
      ),
    [allSuggestions, isRejected, entityId]
  );

  // Local optimistic removal — the suggestion is hidden immediately when
  // rejected, before the server confirms.
  const [locallyDismissed, setLocallyDismissed] = useState<Set<string>>(() => new Set());

  const fullyVisible = useMemo(
    () => visibleSuggestions.filter((s) => !locallyDismissed.has(`${s.term_id}::${s.relationship_type}`)),
    [visibleSuggestions, locallyDismissed]
  );

  // Edge creation — POST /api/semantic-terms/relationships
  const acceptMutation = useMutation({
    mutationFn: async (s: AISuggestion) => {
      const body = {
        source_node_id: entityId,
        target_node_id: s.term_id,
        edge_type_name: s.relationship_type,
        properties: {
          confidence: s.confidence,
          reason: s.differentiation_notes ?? '',
          source: 'ai_suggestion',
        },
      };
      return apiClient<{ edge_id?: string; status?: string }>(
        '/api/semantic-terms/relationships',
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        }
      );
    },
    onSuccess: () => {
      // Refresh both the suggestions and the relationships list so the new
      // edge appears immediately.
      void queryClient.invalidateQueries({ queryKey: ['entity-relationships'] });
      void queryClient.invalidateQueries({ queryKey: ['ai-suggestions'] });
      onSuggestionApplied?.();
    },
  });

  const handleAccept = (s: AISuggestion) => {
    acceptMutation.mutate(s);
  };

  const handleReject = async (s: AISuggestion) => {
    setLocallyDismissed((prev) => new Set(prev).add(`${s.term_id}::${s.relationship_type}`));
    try {
      await recordRejection({
        sourceNodeId: entityId,
        targetNodeId: s.term_id,
        predicate: s.relationship_type,
        edgeTypeId: s.relationship_type,
        reason: s.differentiation_notes ?? 'user_dismissed',
      });
    } catch (e) {
      // The local dismiss is still in effect; the rejection will retry on
      // next mount. We don't block the user on transient failures.
    }
  };

  if (disabled) return null;

  const loading = providedSuggestions === undefined && disambigQuery.isLoading;
  const error = providedSuggestions === undefined ? (disambigQuery.error as Error | null) : null;

  if (loading) {
    return (
      <div data-testid="ai-suggestion-card-loading" style={{ display: 'flex', alignItems: 'center', gap: 10, color: '#A5B4FC', fontSize: 13, padding: '8px 4px' }}>
        <CircularProgress size={14} sx={{ color: '#818CF8' }} />
        <span>Loading AI suggestions…</span>
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid="ai-suggestion-card-error" style={{ color: '#EF4444', fontSize: 12, padding: '8px 4px' }}>
        AI suggestions unavailable: {error.message}
      </div>
    );
  }

  if (fullyVisible.length === 0) {
    return (
      <div data-testid="ai-suggestion-card-empty" style={{ color: '#8892A4', fontSize: 12, fontStyle: 'italic', padding: '8px 4px' }}>
        ✨ No active AI suggestions for this term.
      </div>
    );
  }

  return (
    <div
      data-testid="ai-suggestion-card"
      data-entity-type={entityType}
      data-entity-id={entityId}
      style={{ display: 'flex', flexDirection: 'column', gap: 10 }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#A5B4FC', fontSize: 13, fontWeight: 700 }}>
        <AutoAwesomeIcon sx={{ fontSize: 16 }} />
        <span>{fullyVisible.length} AI suggestion{fullyVisible.length !== 1 ? 's' : ''}</span>
      </div>
      {fullyVisible.map((s) => {
        const predicateMeta = getPredicate(s.relationship_type);
        const confidencePct = Math.round((s.confidence ?? 0) * 100);
        const isAccepting = acceptMutation.isPending && acceptMutation.variables?.term_id === s.term_id;
        return (
          <div
            key={`${s.term_id}-${s.relationship_type}`}
            data-testid="ai-suggestion-row"
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: 6,
              padding: '10px 12px',
              background: 'rgba(99,102,241,0.06)',
              border: '1px solid rgba(99,102,241,0.25)',
              borderRadius: 8,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, minWidth: 0, flex: 1 }}>
                <span style={{ fontFamily: 'monospace', fontWeight: 700, color: '#E2E8F0', fontSize: 13 }}>
                  {focalNode?.node_name ?? entityId}
                </span>
                <span style={{ color: predicateMeta.color, fontWeight: 700, fontSize: 11 }}>
                  {predicateMeta.icon} {predicateMeta.label}
                </span>
                <span style={{ color: '#8892A4', fontSize: 12 }}>→</span>
                <span style={{ color: '#E2E8F0', fontWeight: 600, fontSize: 13, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {s.term_name}
                </span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <Tooltip title="Accept (creates this edge)">
                  <span>
                    <IconButton
                      size="small"
                      onClick={() => handleAccept(s)}
                      disabled={isAccepting}
                      sx={{ color: '#10B981' }}
                      data-testid="ai-suggestion-accept"
                    >
                      {isAccepting ? <CircularProgress size={14} /> : <CheckIcon fontSize="small" />}
                    </IconButton>
                  </span>
                </Tooltip>
                <Tooltip title="Reject (saves to rejection store)">
                  <span>
                    <IconButton
                      size="small"
                      onClick={() => handleReject(s)}
                      disabled={isRecording}
                      sx={{ color: '#EF4444' }}
                      data-testid="ai-suggestion-reject"
                    >
                      <CloseIcon fontSize="small" />
                    </IconButton>
                  </span>
                </Tooltip>
              </div>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 11, color: '#8892A4' }}>
              <div style={{ flex: 1, height: 6, background: 'rgba(255,255,255,0.05)', borderRadius: 3, overflow: 'hidden' }}>
                <div style={{ width: `${confidencePct}%`, height: '100%', background: predicateMeta.color }} />
              </div>
              <span style={{ minWidth: 36, textAlign: 'right', fontWeight: 700, color: predicateMeta.color }}>
                {confidencePct}%
              </span>
              {s.domain && (
                <span style={{ padding: '2px 6px', borderRadius: 4, background: 'rgba(255,255,255,0.04)' }}>
                  {s.domain}
                </span>
              )}
            </div>
            {s.differentiation_notes && (
              <div style={{ fontSize: 11, color: '#A5B4FC', lineHeight: 1.4 }}>
                {s.differentiation_notes}
              </div>
            )}
          </div>
        );
      })}
      {/* Soft indication that the candidate predicate set is bounded to
          the high-signal ones; non-listed predicates surface elsewhere. */}
      <div style={{ fontSize: 10, color: '#64748B', marginTop: 4 }}>
        Showing candidates for: {AI_CANDIDATE_PREDICATES.map((p) => getPredicate(p).label).join(' · ')}
      </div>
    </div>
  );
};

export default AISuggestionCard;
