import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTenant } from '../contexts/TenantContext';
import { apiFetch } from '../lib/apiClient';

export interface ViewTypePolicy {
  defaultInclude?: boolean;
  nodeTypes?: Record<string, 'include' | 'exclude'>;
  edgeTypes?: Record<string, 'include' | 'exclude'>;
}

export interface ViewGroupingRule {
  childNodeType: string;
  parentRelation?: string;
  clusterLabel?: string;
  defaultCollapsed?: boolean;
  collapseThreshold?: number;
}

export interface ViewLayoutConfig {
  algorithm?: 'dagre' | 'bfs-level';
  direction?: 'LR' | 'TB' | 'up' | 'down' | 'both';
}

export interface ViewAssignedAssetTypes {
  boSubtypes?: string[];
  catalogNodeTypes?: string[];
}

export interface ViewDefinitionConfig {
  typePolicy?: ViewTypePolicy;
  grouping?: ViewGroupingRule[];
  layout?: ViewLayoutConfig;
  assignedAssetTypes?: ViewAssignedAssetTypes;
}

export interface ViewDefinition {
  id: string;
  tenant_id: string;
  view_key: string;
  display_name: string;
  description?: string;
  is_core: boolean;
  is_active: boolean;
  config: ViewDefinitionConfig;
  created_at: string;
  updated_at: string;
}

export interface CatalogResponseNode {
  id: string;
  type: string;
  label: string;
  parentId?: string;
  isCluster?: boolean;
  memberIds?: string[];
  properties: Record<string, any>;
}

export interface CatalogResponseEdge {
  id?: string;
  source: string;
  target: string;
  type: string;
  properties?: Record<string, any>;
}

export interface CatalogResponseGraph {
  nodes: CatalogResponseNode[];
  edges: CatalogResponseEdge[];
}

export const viewDefinitionsKeys = {
  all: ['viewDefinitions'] as const,
  lists: () => [...viewDefinitionsKeys.all, 'list'] as const,
  detail: (id: string) => [...viewDefinitionsKeys.all, 'detail', id] as const,
  forAsset: (assetType: string, assetSubtype: string) =>
    [...viewDefinitionsKeys.all, 'for-asset', assetType, assetSubtype] as const,
  graph: (viewId: string, rootNodeId: string, depth?: number, expandedClusters?: string[]) =>
    [...viewDefinitionsKeys.all, 'graph', viewId, rootNodeId, depth, expandedClusters?.slice().sort()] as const,
};

export function useViewDefinitions() {
  const { tenant } = useTenant();

  return useQuery({
    queryKey: viewDefinitionsKeys.lists(),
    queryFn: async (): Promise<ViewDefinition[]> => {
      const res = await apiFetch(`/api/view-definitions`);
      if (!res.ok) {
        throw new Error('Failed to fetch view definitions');
      }
      return res.json();
    },
    enabled: !!tenant?.id,
  });
}

export function useViewDefinition(id: string | undefined) {
  const { tenant } = useTenant();

  return useQuery({
    queryKey: viewDefinitionsKeys.detail(id || ''),
    queryFn: async (): Promise<ViewDefinition> => {
      const res = await apiFetch(`/api/view-definitions/${id}`);
      if (!res.ok) {
        throw new Error('Failed to fetch view definition');
      }
      return res.json();
    },
    enabled: !!tenant?.id && !!id,
  });
}

export function useViewDefinitionsForAsset(assetType: string, assetSubtype: string) {
  const { tenant } = useTenant();

  return useQuery({
    queryKey: viewDefinitionsKeys.forAsset(assetType, assetSubtype),
    queryFn: async (): Promise<ViewDefinition[]> => {
      const res = await apiFetch(
        `/api/view-definitions/for-asset/${encodeURIComponent(assetType)}/${encodeURIComponent(assetSubtype)}`
      );
      if (!res.ok) {
        throw new Error('Failed to fetch view definitions for asset');
      }
      return res.json();
    },
    enabled: !!tenant?.id && !!assetType,
  });
}

export function useCatalogGraph(
  viewId: string | undefined,
  rootNodeId: string | undefined,
  depth?: number,
  expandedClusters?: string[]
) {
  const { tenant } = useTenant();

  return useQuery({
    queryKey: viewDefinitionsKeys.graph(viewId || '', rootNodeId || '', depth, expandedClusters),
    queryFn: async (): Promise<CatalogResponseGraph> => {
      const params = new URLSearchParams();
      if (depth) params.append('depth', String(depth));
      if (expandedClusters && expandedClusters.length > 0) {
        params.append('expandClusters', expandedClusters.join(','));
      }
      const res = await apiFetch(
        `/api/view-definitions/${viewId}/graph/${encodeURIComponent(rootNodeId!)}?${params.toString()}`
      );
      if (!res.ok) {
        throw new Error('Failed to fetch catalog graph');
      }
      return res.json();
    },
    enabled: !!tenant?.id && !!viewId && !!rootNodeId,
  });
}

export function useCreateViewDefinition() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: Partial<ViewDefinition>): Promise<ViewDefinition> => {
      const res = await apiFetch(`/api/view-definitions`, {
        method: 'POST',
        body: JSON.stringify(input),
      });
      if (!res.ok) {
        throw new Error('Failed to create view definition');
      }
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: viewDefinitionsKeys.lists() });
    },
  });
}

export function useUpdateViewDefinition() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, ...input }: Partial<ViewDefinition> & { id: string }): Promise<ViewDefinition> => {
      const res = await apiFetch(`/api/view-definitions/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      });
      if (!res.ok) {
        throw new Error('Failed to update view definition');
      }
      return res.json();
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: viewDefinitionsKeys.lists() });
      queryClient.invalidateQueries({ queryKey: viewDefinitionsKeys.detail(variables.id) });
    },
  });
}

export function useDeleteViewDefinition() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string): Promise<void> => {
      const res = await apiFetch(`/api/view-definitions/${id}`, { method: 'DELETE' });
      if (!res.ok && res.status !== 204) {
        throw new Error('Failed to delete view definition');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: viewDefinitionsKeys.lists() });
    },
  });
}
