import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';
import { useTenant } from '../contexts/TenantContext';
import { devDebug } from '../utils/devLogger';
import { getSelectedRegion } from '../lib/region';
import { apiFetch } from '../lib/apiClient';
import { useNodeTypes } from './nodeTypes';

// Types for Glossary
export interface CatalogNode {
  id: string;
  tenant_datasource_id?: string;
  catalog_type?: string; // From /api/catalog/nodes
  catalog_type_name?: string; // From /api/glossary/* endpoints
  description?: string;
  is_active?: boolean;
  parent_type_id?: string | null;
  parent_id?: string | null; // From /api/catalog/nodes
  config?: string;
  created_at: string;
  updated_at?: string;
  tenant_id?: string;
  core_id?: string | null;
  node_name?: string;
  qualified_path?: string;
  properties?: NodeProperty[] | Record<string, unknown>;
  node_type_id?: string;
  is_mapped?: boolean;
}

export interface NodeProperty {
  name: string;
  label: string;
  order: number;
  nullable: boolean;
  data_type: string;
  input_type: string;
}

export interface CatalogEdge {
  id: string;
  edge_type_name: string;
  description: string;
  object_node_type_id: string;
  properties: EdgeProperty[] | Record<string, any>;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  tenant_id: string;
  core_id: string | null;
}

export interface EdgeProperty {
  name: string;
  label: string;
  order: number;
  nullable: boolean;
  data_type: string;
  input_type: string;
}

export interface SemanticTerm extends CatalogNode {
  // Semantic terms are catalog nodes with catalog_type_name = 'semantic_term'
}

export interface BusinessTerm extends CatalogNode {
  // Business terms are catalog nodes with catalog_type_name = 'business_term'
}

// Query keys
export const glossaryKeys = {
  all: ['glossary'] as const,
  semanticTerms: () => [...glossaryKeys.all, 'semantic-terms'] as const,
  businessTerms: () => [...glossaryKeys.all, 'business-terms'] as const,
  edges: () => [...glossaryKeys.all, 'edges'] as const,
  term: (id: string) => [...glossaryKeys.all, 'term', id] as const,
  catalogNodes: () => [...glossaryKeys.all, 'catalog-nodes'] as const,
  semanticData: () => [...glossaryKeys.all, 'semantic-data'] as const,
};

// Fetch all semantic terms
// NOTE: Like useBusinessTerms(), this falls back to /api/catalog/nodes and filters
// client-side because /api/glossary/semantic-terms returns 404 on the platform backend.
// (Hasura has been removed permanently, so the GraphQL endpoint is unavailable.)
export function useSemanticTerms() {
  const { tenant, datasource } = useTenant();

  return useQuery({
    queryKey: glossaryKeys.semanticTerms(),
    queryFn: async () => {
      const params = new URLSearchParams();
      if (tenant?.id) {
        params.append('tenant_id', tenant.id);
      }
      if (datasource?.id) {
        params.append('tenant_instance_id', datasource.id);
      }

      const res = await apiFetch(`/api/catalog/nodes?${params.toString()}`);

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to fetch semantic terms');
      }

      const allNodes = ((await res.json()) as CatalogNode[]) || [];
      return allNodes.filter((node) => node.catalog_type === 'semantic_term');
    },
    enabled: !!(tenant?.id && datasource?.id),
  });
}

// Fetch all business terms
// NOTE: Uses /api/catalog/nodes as the canonical source. Backend does not implement
// /api/glossary/business-terms (returns 404), and Hasura/GraphQL has been removed
// permanently, so we filter client-side.
export function useBusinessTerms() {
  const { tenant, datasource } = useTenant();

  return useQuery({
    queryKey: glossaryKeys.businessTerms(),
    queryFn: async () => {
      const params = new URLSearchParams();
      if (tenant?.id) {
        params.append('tenant_id', tenant.id);
      }
      if (datasource?.id) {
        params.append('tenant_instance_id', datasource.id);
      }

      const res = await apiFetch(`/api/catalog/nodes?${params.toString()}`);

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to fetch business terms');
      }

      const allNodes = ((await res.json()) as CatalogNode[]) || [];
      return allNodes.filter((node) => node.catalog_type === 'business_term');
    },
    enabled: !!(tenant?.id && datasource?.id),
  });
}

// Fetch edges between business terms and semantic terms
// NOTE: Returns empty array because /api/glossary/edges and /api/catalog/edges are
// not yet implemented on the platform backend, and GraphQL (which previously served
// edges) has been removed permanently. TODO: wire a REST endpoint when available.
export function useGlossaryEdges() {
  const { tenant, datasource } = useTenant();

  return useQuery({
    queryKey: glossaryKeys.edges(),
    queryFn: async () => {
      return [] as CatalogEdge[];
    },
    enabled: !!(tenant?.id && datasource?.id),
  });
}

// Fetch all catalog nodes for the active tenant/datasource (REST-only).
// Returns the full list — callers can filter by `catalog_type` or `node_type_id`.
function useAllCatalogNodes() {
  const { tenant, datasource } = useTenant();

  return useQuery({
    queryKey: [...glossaryKeys.catalogNodes(), tenant?.id, datasource?.id],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (tenant?.id) params.append('tenant_id', tenant.id);
      if (datasource?.id) params.append('tenant_instance_id', datasource.id);

      const res = await apiFetch(`/api/catalog/nodes?${params.toString()}`);
      if (!res.ok) {
        throw new Error(await res.text() || 'Failed to fetch catalog nodes');
      }
      return ((await res.json()) as CatalogNode[]) || [];
    },
    enabled: !!(tenant?.id && datasource?.id),
  });
}

// Attach a denormalized `node_type` object to each catalog node based on the
// shared nodeTypes list. Matches the shape the GraphQL version produced.
function attachNodeType(nodes: CatalogNode[], nodeTypesList?: { id: string; catalog_type_name?: string }[] | null) {
  return nodes.map((node) => {
    const typeDef = nodeTypesList?.find((t) => t.id === node.node_type_id);
    return {
      ...node,
      node_type: {
        catalog_type_name: typeDef?.catalog_type_name || 'unknown',
      },
    };
  });
}

// Fetch all semantic data (business terms, semantic terms, semantic columns,
// calculation terms, and edges) using REST only.
//
// Previously this used Apollo/GraphQL to query catalog_node / catalog_edge in a
// single round-trip. With Hasura removed permanently, this now:
//   1. Fetches the full list of node types via /api/node-types (REST).
//   2. Fetches the full list of catalog nodes via /api/catalog/nodes (REST).
//   3. Filters client-side by `node_type_id` for each type bucket.
//   4. Returns an empty array for `semantic_edges` (no REST edge endpoint yet).
export function useAllSemanticData() {
  const { tenant, datasource } = useTenant();
  const NIL_UUID = '00000000-0000-0000-0000-000000000000';

  // 1. Node types (REST)
  const { data: nodeTypesList, isLoading: isNodeTypesLoading } = useNodeTypes();

  // 2. Resolve type IDs by name → id
  const typeMap = useMemo(() => {
    const empty = {
      business_term: NIL_UUID,
      semantic_term: NIL_UUID,
      semantic_column: NIL_UUID,
      calculation: NIL_UUID,
      calculation_term: NIL_UUID,
      metric: NIL_UUID,
    };
    if (!nodeTypesList) return empty;
    return {
      business_term: nodeTypesList.find((t) => t.catalog_type_name === 'business_term')?.id || NIL_UUID,
      semantic_term: nodeTypesList.find((t) => t.catalog_type_name === 'semantic_term')?.id || NIL_UUID,
      semantic_column: nodeTypesList.find((t) => t.catalog_type_name === 'semantic_column')?.id || NIL_UUID,
      calculation: nodeTypesList.find((t) => t.catalog_type_name === 'calculation')?.id || NIL_UUID,
      calculation_term: nodeTypesList.find((t) => t.catalog_type_name === 'calculation_term')?.id || NIL_UUID,
      metric: nodeTypesList.find((t) => t.catalog_type_name === 'metric')?.id || NIL_UUID,
    };
  }, [nodeTypesList]);

  // 3. All catalog nodes (REST)
  const nodesQuery = useAllCatalogNodes();

  // 4. Filter and shape data the way the GraphQL version did
  const transformedData = useMemo(() => {
    const allNodes = nodesQuery.data || [];
    const allWithType = attachNodeType(allNodes, nodeTypesList);

    const business_terms = allWithType.filter((n) => n.node_type_id === typeMap.business_term);
    const semantic_terms = allWithType.filter((n) => n.node_type_id === typeMap.semantic_term);
    const semantic_columns = allWithType.filter((n) => n.node_type_id === typeMap.semantic_column);
    const calculation_terms = allWithType.filter((n) =>
      n.node_type_id === typeMap.calculation ||
      n.node_type_id === typeMap.calculation_term ||
      n.node_type_id === typeMap.metric
    );

    return {
      business_terms,
      semantic_terms,
      semantic_edges: [] as CatalogEdge[], // REST edge endpoint not yet implemented
      all_nodes: allWithType,
      node_types: nodeTypesList || [],
      calculation_terms,
      semantic_columns,
    };
  }, [nodesQuery.data, nodeTypesList, typeMap]);

  return {
    data: transformedData,
    isLoading: isNodeTypesLoading || nodesQuery.isLoading,
    error: (nodesQuery.error as Error | null)?.message || null,
    enabled: !!(tenant?.id && datasource?.id),
    refetch: () => {
      void nodesQuery.refetch();
    },
  };
}

export const useAllSemanticDataQuery = useAllSemanticData;

// Update a semantic term or business term
export function useUpdateTerm() {
  const queryClient = useQueryClient();
  const { tenant, datasource } = useTenant();

  return useMutation({
    mutationFn: async (data: { id: string; updates: Partial<CatalogNode> }) => {
      const params = new URLSearchParams();
      if (tenant?.id) {
        params.append('tenant_id', tenant.id);
      }
      if (datasource?.id) {
        params.append('tenant_instance_id', datasource.id);
      }

      devDebug('[useUpdateTerm] Starting update for term:', data.id);
      devDebug('[useUpdateTerm] Updates to send:', JSON.stringify(data.updates, null, 2));
      devDebug('[useUpdateTerm] parent_id value:', data.updates.parent_id);
      devDebug('[useUpdateTerm] catalog_type:', data.updates.catalog_type);

      // Ensure parent_id is explicitly included for semantic terms
      // ALSO ensure properties is always an object, never an array
      let updatePayload: any = {
        ...data.updates,
        // Preserve parent_id for semantic terms - use the value from updates (could be null or a valid ID)
        ...(data.updates.catalog_type === 'semantic_term' && { parent_id: data.updates.parent_id ?? null }),
      };

      // Normalize properties to always be an object (not array)
      if (updatePayload.properties) {
        if (Array.isArray(updatePayload.properties)) {
          devDebug('[useUpdateTerm] Properties came as array, converting to empty object for proper storage');
          devDebug('[useUpdateTerm] Properties was array:', updatePayload.properties);
          updatePayload.properties = {};
        }
      }

      const url = `/api/glossary/terms/${data.id}?${params.toString()}`;
      const requestBody = JSON.stringify(updatePayload);

      devDebug('[useUpdateTerm] Request URL:', url);
      devDebug('[useUpdateTerm] Request body:', requestBody);
      devDebug('[useUpdateTerm] parent_id in payload:', updatePayload.parent_id);

      const res = await apiFetch(url, {
        method: 'PUT',
        body: requestBody,
      });

      const responseText = await res.text();
      devDebug('[useUpdateTerm] Response status:', res.status);
      devDebug('[useUpdateTerm] Response body:', responseText);

      if (!res.ok) {
        console.error('[useUpdateTerm] Update failed with status', res.status);
        throw new Error(responseText || 'Failed to update term');
      }

      const responseData = JSON.parse(responseText) as CatalogNode;
      devDebug('[useUpdateTerm] Update successful!');
      devDebug('[useUpdateTerm] Returned parent_id:', responseData.parent_id);
      devDebug('[useUpdateTerm] Full response:', JSON.stringify(responseData, null, 2));

      return responseData;
    },
    onSuccess: (responseData, variables) => {
      devDebug('[useUpdateTerm.onSuccess] Starting cache invalidation...');
      devDebug('[useUpdateTerm.onSuccess] Response had parent_id:', responseData.parent_id);

      // Optimistically update the term in the cache for immediate UI feedback
      const queryKey =
        responseData.catalog_type === 'semantic_term'
          ? glossaryKeys.semanticTerms()
          : glossaryKeys.businessTerms();

      queryClient.setQueryData<CatalogNode[]>(queryKey, (oldData) => {
        if (!oldData) return [];
        return oldData.map((term) => (term.id === variables.id ? { ...term, ...responseData } : term));
      });
      devDebug(`[useUpdateTerm.onSuccess] Optimistically updated cache for ${responseData.catalog_type}`);

      // Invalidate queries to ensure data consistency with the backend
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticTerms() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.businessTerms() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.term(variables.id) });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.catalogNodes() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticData() });

      devDebug('[useUpdateTerm.onSuccess] All React Query caches invalidated');
    },
    onError: (error) => {
      console.error('[useUpdateTerm] Mutation failed with error:', error);
      console.error('[useUpdateTerm] Error message:', error.message);
    },
  });
}

// Create a new semantic term or business term
export function useCreateTerm() {
  const queryClient = useQueryClient();
  const { tenant, datasource } = useTenant();

  return useMutation({
    mutationFn: async (data: Omit<CatalogNode, 'id' | 'created_at' | 'updated_at'>) => {
      const params = new URLSearchParams();
      if (tenant?.id) {
        params.append('tenant_id', tenant.id);
      }
      if (datasource?.id) {
        params.append('tenant_instance_id', datasource.id);
      }

      // Normalize properties to always be an object (not array)
      let createPayload = { ...data };
      if (createPayload.properties && Array.isArray(createPayload.properties)) {
        devDebug('[useCreateTerm] Properties came as array, converting to empty object for proper storage');
        createPayload.properties = {};
      }

      const res = await apiFetch(`/api/glossary/terms?${params.toString()}`, {
        method: 'POST',
        body: JSON.stringify(createPayload),
      });

      if (!res.ok) {
        // Try to parse structured validation errors (returned as JSON) and rethrow
        const text = await res.text();
        try {
          const parsed = JSON.parse(text);
          if (parsed && parsed.validation_errors) {
            const err: any = new Error('Validation failed');
            err.validation_errors = parsed.validation_errors;
            throw err;
          }
        } catch (e) {
          // not JSON
        }
        const error = text;
        throw new Error(error || 'Failed to create term');
      }
      return res.json() as Promise<CatalogNode>;
    },
    onSuccess: () => {
      // Invalidate queries to refetch the lists after creation
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticTerms() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.businessTerms() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.catalogNodes() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticData() });
    },
  });
}

// Delete a semantic term or business term
export function useDeleteTerm() {
  const queryClient = useQueryClient();
  const { tenant, datasource } = useTenant();

  return useMutation({
    mutationFn: async (id: string) => {
      const params = new URLSearchParams();
      if (tenant?.id) {
        params.append('tenant_id', tenant.id);
      }
      if (datasource?.id) {
        params.append('tenant_instance_id', datasource.id);
      }

      const res = await apiFetch(`/api/glossary/terms/${id}?${params.toString()}`, {
        method: 'DELETE',
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to delete term');
      }
      return res.json();
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticTerms() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.businessTerms() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.catalogNodes() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticData() });
    },
  });
}

// Create a new edge between terms
export function useCreateTermEdge() {
  const queryClient = useQueryClient();
  const { tenant, datasource } = useTenant();

  return useMutation({
    mutationFn: async (data: {
      subject_node_id: string;
      object_node_id: string;
      edge_type_id: string;
      properties?: Record<string, any>; // Custom edge properties
    }) => {
      const params = new URLSearchParams();
      if (tenant?.id) {
        params.append('tenant_id', tenant.id);
      }
      if (datasource?.id) {
        params.append('tenant_instance_id', datasource.id);
      }

      const res = await apiFetch(`/api/glossary/edges?${params.toString()}`, {
        method: 'POST',
        body: JSON.stringify(data),
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to create edge');
      }
      return res.json() as Promise<CatalogEdge>;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
    },
  });
}

// Update an existing edge
export function useUpdateTermEdge() {
  const queryClient = useQueryClient();
  const { tenant, datasource } = useTenant();

  return useMutation({
    mutationFn: async (data: { id: string; updates: Partial<CatalogEdge> }) => {
      const params = new URLSearchParams();
      if (tenant?.id) {
        params.append('tenant_id', tenant.id);
      }
      if (datasource?.id) {
        params.append('tenant_instance_id', datasource.id);
      }

      const res = await apiFetch(`/api/glossary/edges/${data.id}?${params.toString()}`, {
        method: 'PUT',
        body: JSON.stringify(data.updates),
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to update edge');
      }
      return res.json() as Promise<CatalogEdge>;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
    },
  });
}

// Delete an edge
export function useDeleteTermEdge() {
  const queryClient = useQueryClient();
  const { tenant, datasource } = useTenant();

  return useMutation({
    mutationFn: async (id: string) => {
      const params = new URLSearchParams();
      if (tenant?.id) {
        params.append('tenant_id', tenant.id);
      }
      if (datasource?.id) {
        params.append('tenant_instance_id', datasource.id);
      }

      const res = await apiFetch(`/api/glossary/edges/${id}?${params.toString()}`, {
        method: 'DELETE',
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to delete edge');
      }
      return res.json();
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
    },
  });
}

export const useDeleteSemanticTerm = useDeleteTerm;
