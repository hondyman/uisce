/**
 * Hook to resolve entity keys to their UUIDs from fabric_defn
 * This allows validation rules to be linked by UUID instead of name
 */

import { useCallback, useEffect, useState } from 'react';
import { devLog, devError } from '../utils/devLogger';

export interface EntityResolution {
  [entityKey: string]: {
    id: string;     // UUID from fabric_defn
    key: string;    // Entity key (model_key)
    name: string;   // Display name (title)
  };
}

/**
 * Hook to fetch and cache entity ID resolutions
 * @param tenantId - Tenant ID
 * @param datasourceId - Datasource ID
 * @returns Object mapping entity keys to their IDs and names
 */
export function useEntityResolution(tenantId?: string, datasourceId?: string) {
  const [entityMap, setEntityMap] = useState<EntityResolution>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchEntityResolution = useCallback(async () => {
    if (!tenantId || !datasourceId) {
      devLog('[useEntityResolution] Skipping fetch - missing tenant/datasource IDs');
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const response = await fetch('/api/entities/resolve', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
      });

      if (!response.ok) {
        // If endpoint doesn't exist (404), return empty map to allow component to function
        if (response.status === 404) {
          devLog('[useEntityResolution] Entity resolution endpoint not found, using empty map');
          setEntityMap({});
          return;
        }
        throw new Error(`Failed to resolve entities: ${response.statusText}`);
      }

      const data = await response.json();
      devLog('[useEntityResolution] Resolved entities:', data);
      setEntityMap(data);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Unknown error';
      devError('[useEntityResolution] Failed to resolve entities:', errorMsg);
      setError(errorMsg);
    } finally {
      setLoading(false);
    }
  }, [tenantId, datasourceId]);

  useEffect(() => {
    fetchEntityResolution();
  }, [fetchEntityResolution]);

  /**
   * Get the UUID for a given entity key
   * @param entityKey - The entity key (model_key)
   * @returns The UUID, or undefined if not found
   */
  const getEntityId = useCallback((entityKey: string): string | undefined => {
    return entityMap[entityKey]?.id;
  }, [entityMap]);

  /**
   * Get the display name for a given entity key
   * @param entityKey - The entity key (model_key)
   * @returns The display name, or undefined if not found
   */
  const getEntityName = useCallback((entityKey: string): string | undefined => {
    return entityMap[entityKey]?.name;
  }, [entityMap]);

  return {
    entityMap,
    loading,
    error,
    getEntityId,
    getEntityName,
    refetch: fetchEntityResolution,
  };
}

export default useEntityResolution;
