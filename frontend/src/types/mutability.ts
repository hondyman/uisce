/**
 * Phase 5 — Three-tier storage mesh: hydration-aware UI types.
 *
 * These mirror the backend's `layout.PageResolution` payload
 * (GET /api/v1/layout/resolve). The AdaptiveSemanticForm / AdaptiveSemanticCanvas
 * components consume them and adjust their controls based on the resolved
 * capabilities + per-field hydration state.
 */

export type MutabilityMode = 'DIRECT_OLTP_SQL' | 'ASYNCHRONOUS_CQRS_QUEUE';

export type TemporalStrategy =
  | 'NONE'
  | 'VALID_TIME'
  | 'BITEMPORAL'
  | 'SYSTEM_TIME_SNAPSHOT';

export type HydrationState = 'RESOLVED' | 'UNBOUND_FALLBACK_NULL';

export interface ResolvedCapabilities {
  mutabilityMode: MutabilityMode;
  temporalStrategy: TemporalStrategy;
  allowDirectCrudFormButtons: boolean;
  activeRouteBackendId: string;
  commandTopicTemplate?: string;
}

export interface ResolvedField {
  semanticTermKey: string;
  displayLabel: string;
  hydrationState: HydrationState;
  isEditable: boolean;
  bindingBackendId?: string;
  fieldRole?: string;
  dataType?: string;
}

export interface PageResolution {
  pageKey: string;
  primaryBusinessObject: string;
  bindingId?: string;
  capabilities: ResolvedCapabilities;
  schemaViewport: ResolvedField[];
  resolvedAt: string;
  precedenceRank: number;
}

/**
 * Mutation payload accepted by POST /api/v1/mutations/dispatch. Mirrors
 * the backend's services.MutationRequest. The pageKey + bindingId pair
 * comes from the resolved page; the StatePayload is what the form collects.
 */
export interface MutationRequest {
  businessObjectKey: string;
  businessObjectId?: string;
  bindingId?: string;
  mutationType: 'CREATE' | 'UPDATE' | 'DELETE' | 'CLONE';
  tenantId: string;
  userId?: string;
  statePayload: Record<string, unknown>;
  syncRequest?: boolean;
}

export interface MutationResponse {
  commandId: string;
  correlationId: string;
  route: 'DIRECT_OLTP_SQL' | 'ASYNCHRONOUS_CQRS_QUEUE' | 'REJECTED';
  topic?: string;
  status: 'pending' | 'success' | 'failed';
  timestamp: string;
  error?: string;
}

/**
 * Helper for the AdaptiveSemanticCanvas badge: which tier is the
 * field sourced from? Derived from bindingBackendId (with fallback to
 * capabilities.activeRouteBackendId).
 *
 * Cardinal Rule 1: this is a pure function over resolved data — no
 * engine_type string branches in component code, just a tier tag.
 */
export function getFieldTierTag(
  field: ResolvedField,
  capabilities: ResolvedCapabilities,
): 'hot' | 'cold' | 'oltp' | null {
  const id = (field.bindingBackendId || capabilities.activeRouteBackendId || '').toLowerCase();
  if (!id) return null;
  if (id.includes('starrocks') || id.endsWith('_hot')) return 'hot';
  if (id.includes('iceberg') || id.endsWith('_coldstore') || id.endsWith('_cold')) return 'cold';
  return 'oltp';
}