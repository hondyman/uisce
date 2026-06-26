import { useQuery } from '@tanstack/react-query';
import type { SearchResult } from '../types/search';
import type { NodeType } from '../types/nodeTypes';
import type { EdgeType } from '../types/edgeTypes';

// Combined search across node-types and edge-types. Returns SearchResult payloads

export function useCatalogSearch(tenantId: string, q: string) {
  return useQuery({
    queryKey: ['catalog-search', tenantId, q],
    queryFn: async () => {
      if (!tenantId) return [] as SearchResult[];
      const params = new URLSearchParams();
      params.set('tenant_id', tenantId);
      if (q && q.trim() !== '') params.set('q', q);

      const [nodesRes, edgesRes] = await Promise.all([
        fetch(`/api/node-types?${params.toString()}`, { credentials: 'include' }),
        fetch(`/api/edge-types?${params.toString()}`, { credentials: 'include' }),
      ]);

  const results: CatalogSearchResult[] = [];

      if (nodesRes.ok) {
        const nodes = (await nodesRes.json()) as NodeType[];
        for (const n of nodes) {
          // NodeType has catalog_type_name in its shape
          results.push({ id: n.id, text: (n as any).catalog_type_name || n.id, subtext: n.description || '', payload: n, kind: 'node' } as CatalogSearchResult);
        }
      }

      if (edgesRes.ok) {
        const edges = (await edgesRes.json()) as EdgeType[];
        for (const e of edges) {
          // EdgeType uses 'predicate' for the relationship name
          results.push({ id: e.id, text: e.edge_type_name || e.id, subtext: e.description || '', payload: e, kind: 'edge' } as CatalogSearchResult);
        }
      }

      return results;
    },
    enabled: !!tenantId && !!q && q.trim() !== '',
  });
}

export type CatalogSearchResult = SearchResult<NodeType | EdgeType> & { kind?: 'node' | 'edge' };
