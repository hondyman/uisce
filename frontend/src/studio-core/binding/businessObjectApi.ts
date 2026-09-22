/**
 * Shared Business Object discovery API - the one place any studio (Query
 * Builder, Page Studio, Report Studio, the BO Binding wizard) fetches the
 * BO list and a BO's relationships from, mirroring boRelationships.ts's
 * "do not add a second API" rule for GET /business-objects/{id}/relationships.
 *
 * Binding and field-term fetches already had one shared home
 * (features/query-builder/services/queryBuilderApi.ts's
 * fetchBusinessObjectBindings/fetchBOTerms) - re-exported here so a
 * consumer of useBusinessObjectSelector only needs one import.
 */
import apiClient from '../../utils/apiClient';
import type { BORelationship } from './boRelationships';

export type { BORelationship } from './boRelationships';
export { fetchBusinessObjectBindings, fetchBOTerms } from '../../features/query-builder/services/queryBuilderApi';
export type { BindingView, SemanticTermView } from '../../features/query-builder/types/queryDef';

export interface BusinessObjectOption {
  id: string;
  /** bo_key - the technical name every bo-scoped endpoint actually expects. */
  key: string;
  name: string;
  displayName: string;
}

function normalizeBO(item: Record<string, unknown>): BusinessObjectOption {
  return {
    id: String(item.id ?? item.name ?? ''),
    key: String(item.key ?? item.technicalName ?? item.technical_name ?? item.name ?? ''),
    name: String(item.name ?? ''),
    displayName: String(item.displayName ?? item.display_name ?? item.name ?? ''),
  };
}

/** Every Business Object visible to the caller's tenant. */
export async function listBusinessObjects(): Promise<BusinessObjectOption[]> {
  const data = await apiClient<unknown>('/business-objects');
  const raw = Array.isArray(data)
    ? data
    : data && typeof data === 'object'
      ? (data as any).businessObjects || (data as any).items || Object.values(data as Record<string, unknown>)
      : [];
  return raw
    .filter((item: unknown): item is Record<string, unknown> => !!item && typeof item === 'object')
    .map(normalizeBO);
}

/**
 * Related Business Objects (and extra related tables) for `boId`, resolved
 * server-side via the relationship graph - the same source Page Studio's
 * ObjectPalette drag-and-drop and DataBindingsPanel already read. `kind`
 * distinguishes an actual related BO ('bo', the default) from an extra
 * table of the same BO ('relatedTable') - callers that only want objects a
 * user can add to a query/page should filter out 'relatedTable'.
 */
export async function fetchBORelationships(boId: string): Promise<BORelationship[]> {
  const data = await apiClient<{ relatedObjects?: BORelationship[] }>(`/business-objects/${encodeURIComponent(boId)}/relationships`);
  return data?.relatedObjects || [];
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** True when `value` is a raw UUID rather than a resolved display name -
 * every "show a Business Object's name" spot should check this so a
 * still-resolving or unresolvable name shows something readable instead
 * of a 36-character id. */
export function looksLikeUuid(value: string): boolean {
  return UUID_RE.test(value);
}
