import { devLog, devError } from '../utils/devLogger';
import { getSelectedRegion } from '../lib/region';

function getAuthToken(): string {
  try { return localStorage.getItem('auth_token') || ''; } catch { return ''; }
}

function buildHeaders(tenantId: string, datasourceId: string): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId,
    'X-Tenant-Datasource-ID': datasourceId,
    'X-Tenant-Region': getSelectedRegion(),
  };
  const token = getAuthToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;
  return headers;
}

/**
 * Represents a related entity that can be linked to a source entity
 */
export interface RelatedEntity {
  id: string;
  sourceEntity: string;
  sourceName?: string;
  targetEntity: string;
  targetName?: string;
  cardinality: 'One-to-One' | 'One-to-Many' | 'Many-to-One' | 'Many-to-Many';
  keyFields: {
    source: string;
    target: string;
  };
  description?: string;
  edgeType?: string;
  tableName?: string;
  semanticName?: string;
  sourceFields?: any;
  targetFields?: any;
  isApplied?: boolean;
}

/**
 * Response from the relationships/objects endpoint
 */
export interface RelationshipsObjectsResponse {
  sourceEntity: string;
  relationships: RelatedEntity[];
  count: number;
}

/**
 * Fetches related/linkable entities for a given entity
 * @param tenantId - The tenant ID
 * @param datasourceId - The datasource ID
 * @param entityIdOrName - The entity ID (preferred) or entity name (fallback)
 * @returns Promise resolving to array of RelatedEntity objects
 */
export async function fetchRelatedObjects(
  tenantId: string,
  datasourceId: string,
  entityIdOrName: string
): Promise<RelatedEntity[]> {
  if (!tenantId || !datasourceId || !entityIdOrName) {
    devError('fetchRelatedObjects: Missing required parameters', { tenantId, datasourceId, entityIdOrName });
    throw new Error('Missing required parameters: tenantId, datasourceId, entityIdOrName');
  }

  devLog('📡 Fetching related objects for entity:', { entityIdOrName, tenantId, datasourceId });

  // NOTE: The /api/relationships/{entityId}/objects endpoint is not yet implemented on the backend.
  // Returning empty array to allow the UI to function normally.
  // TODO: Implement this endpoint when relationship data retrieval is needed.
  devLog('Related objects endpoint not yet implemented, returning empty array');
  return [];
}

export async function fetchRelationshipSuggestions(
  tenantId: string,
  datasourceId: string,
  entityIdOrName: string,
  limit: number = 5
): Promise<RelatedEntity[]> {
  if (!tenantId || !datasourceId || !entityIdOrName) {
    throw new Error('Missing required parameters');
  }

  devLog('📡 Fetching relationship suggestions for entity:', { entityIdOrName, limit });

  const params = new URLSearchParams({
    limit: String(limit),
  });

  try {
    const response = await fetch(
      `/api/relationships/${entityIdOrName}/suggestions?${params.toString()}`,
      {
        method: 'GET',
        headers: buildHeaders(tenantId, datasourceId),
      }
    );

    if (!response.ok) {
      devError('Failed to fetch relationship suggestions:', {
        status: response.status,
        statusText: response.statusText,
      });
      return [];
    }

    const data = await response.json();
    devLog('✅ Relationship suggestions fetched:', data);

    return Array.isArray(data) ? data : data.suggestions || [];
  } catch (err) {
    devError('Error fetching relationship suggestions:', err);
    return [];
  }
}

export async function applyRelationship(
  tenantId: string,
  datasourceId: string,
  sourceEntity: string,
  targetEntity: string,
  relationshipType: string = 'entity_relationship',
  cardinality: string = 'One-to-Many'
): Promise<{ success: boolean; edgeId?: string; error?: string }> {
  if (!tenantId || !datasourceId || !sourceEntity || !targetEntity) {
    throw new Error('Missing required parameters');
  }

  devLog('🔗 Applying relationship:', { sourceEntity, targetEntity, relationshipType, cardinality });

  try {
    const response = await fetch('/api/relationships/apply', {
      method: 'POST',
      headers: buildHeaders(tenantId, datasourceId),
      body: JSON.stringify({
        sourceEntity: sourceEntity,
        targetEntity: targetEntity,
        edgeType: relationshipType,
        cardinality: cardinality,
      }),
    });

    if (!response.ok) {
      const errorText = await response.text();
      devError('Failed to apply relationship:', {
        status: response.status,
        statusText: response.statusText,
        body: errorText,
      });

      return {
        success: false,
        error: `Failed to apply relationship: ${response.statusText}`,
      };
    }

    const data = await response.json();
    devLog('✅ Relationship applied:', data);

    return {
      success: true,
      edgeId: data.edge_id || data.id || 'applied',
    };
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    devError('Error applying relationship:', message);
    return {
      success: false,
      error: message,
    };
  }
}

export async function unlinkRelationship(
  tenantId: string,
  datasourceId: string,
  sourceEntity: string,
  targetEntity: string
): Promise<{ success: boolean; error?: string }> {
  if (!tenantId || !datasourceId || !sourceEntity || !targetEntity) {
    throw new Error('Missing required parameters');
  }

  devLog('🔗 Unlinking relationship:', { sourceEntity, targetEntity });

  try {
    const response = await fetch('/api/relationships/remove', {
      method: 'POST',
      headers: buildHeaders(tenantId, datasourceId),
      body: JSON.stringify({
        sourceEntity,
        targetEntity,
      }),
    });

    if (!response.ok) {
      const errorText = await response.text().catch(() => '');
      devError('Failed to unlink relationship:', {
        status: response.status,
        statusText: response.statusText,
        body: errorText,
      });

      return { success: false, error: `Failed to unlink relationship: ${response.statusText}` };
    }

    devLog('✅ Relationship unlinked');
    return { success: true };
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    devError('Error unlinking relationship:', message);
    return { success: false, error: message };
  }
}

export async function dismissRelationshipSuggestion(
  tenantId: string,
  datasourceId: string,
  suggestionId: string
): Promise<boolean> {
  if (!tenantId || !datasourceId || !suggestionId) {
    throw new Error('Missing required parameters');
  }

  devLog('🗑️ Dismissing suggestion:', { suggestionId });

  try {
    const response = await fetch('/api/relationships/suggestions/dismiss', {
      method: 'POST',
      headers: buildHeaders(tenantId, datasourceId),
      body: JSON.stringify({
        suggestion_id: suggestionId,
      }),
    });

    if (!response.ok) {
      devError('Failed to dismiss suggestion:', {
        status: response.status,
        statusText: response.statusText,
      });
      return false;
    }

    devLog('✅ Suggestion dismissed');
    return true;
  } catch (err) {
    devError('Error dismissing suggestion:', err);
    return false;
  }
}
