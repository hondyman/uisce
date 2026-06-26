import { useState, useCallback } from 'react';

export interface DirectRelationship {
  relatedEntityId: string;
  relatedEntityName: string;
  linkType: 'DIRECT_FK' | 'SEMANTIC' | 'MULTI_HOP';
  confidence: number;
  cardinality: '1:1' | '1:N' | 'N:1' | 'N:M';
  foreignKeyPath: Array<{
    sourceColumn: string;
    targetColumn: string;
    tableName: string;
  }>;
  columnMapping: Array<{
    sourceColumn: string;
    targetColumn: string;
  }>;
}

export interface RelationshipHop {
  entityId: string;
  entityName: string;
  linkType: string;
  cardinality: string;
  foreignKeyPath: any[];
}

export interface MultiHopPath {
  pathId: string;
  hops: RelationshipHop[];
  depth: number;
  confidence: number;
  cardinality: string;
  lastUpdated: string;
}

export interface DiscoverRelationshipsRequest {
  entityId: string;
  maxHopDepth?: number;
  includeSemanticLinks?: boolean;
}

export interface DiscoverRelationshipsResponse {
  directRelationships: DirectRelationship[];
  multiHopPaths: MultiHopPath[];
}

export interface ApplyRelationshipRequest {
  sourceEntityId: string;
  targetEntityId: string;
  linkType: 'DIRECT_FK' | 'SEMANTIC' | 'MULTI_HOP';
  confidence: number;
  cardinality: string;
  foreignKeyPath: any[];
  columnMapping: any[];
}

export const useRelationshipDiscovery = (
  tenantId: string,
  datasourceId: string
) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const discoverRelationships = useCallback(
    async (
      request: DiscoverRelationshipsRequest
    ): Promise<DiscoverRelationshipsResponse | null> => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch('/api/relationships/discover', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
          body: JSON.stringify(request),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.error || `Failed to discover relationships: ${response.statusText}`
          );
        }

        const data = await response.json();
        return data;
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to discover relationships';
        setError(errorMsg);
        return null;
      } finally {
        setLoading(false);
      }
    },
    [tenantId, datasourceId]
  );

  const applyRelationship = useCallback(
    async (request: ApplyRelationshipRequest): Promise<boolean> => {
      setLoading(true);
      setError(null);

      try {
        const response = await fetch('/api/relationships/apply', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
          body: JSON.stringify(request),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.error || `Failed to apply relationship: ${response.statusText}`
          );
        }

        return true;
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to apply relationship';
        setError(errorMsg);
        return false;
      } finally {
        setLoading(false);
      }
    },
    [tenantId, datasourceId]
  );

  return {
    discoverRelationships,
    applyRelationship,
    loading,
    error,
  };
};
