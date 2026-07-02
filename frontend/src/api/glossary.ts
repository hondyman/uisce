import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';
import { useTenant } from '../contexts/TenantContext';
import { devDebug } from '../utils/devLogger';
import { getSelectedRegion } from '../lib/region';

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
};

// Fetch all semantic terms
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

      const res = await fetch(`/api/glossary/semantic-terms?${params.toString()}`, {
        credentials: 'include',
        headers: {
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
          ...(datasource?.id && { 'X-Tenant-Datasource-ID': datasource.id }),
          'X-Tenant-Region': getSelectedRegion(),
        },
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to fetch semantic terms');
      }

      return res.json() as Promise<SemanticTerm[]>;
    },
    enabled: !!(tenant?.id && datasource?.id),
  });
}

// Fetch all business terms
// NOTE: Uses /api/catalog/nodes as fallback since /api/glossary/business-terms returns 404
// TODO: Remove this fallback once backend glossary routes are fixed
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

      // Fallback to /api/catalog/nodes which works; filter client-side for business_term type
      const res = await fetch(`/api/catalog/nodes?${params.toString()}`, {
        credentials: 'include',
        headers: {
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
          ...(datasource?.id && { 'X-Tenant-Datasource-ID': datasource.id }),
          'X-Tenant-Region': getSelectedRegion(),
        },
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to fetch business terms');
      }

      const allNodes = (await res.json() as CatalogNode[]) || [];
      // Filter for business_term nodes
      const businessTerms = allNodes.filter(
        (node) => node.catalog_type === 'business_term'
      );
      return businessTerms;
    },
    enabled: !!(tenant?.id && datasource?.id),
  });
}

// Fetch edges between business terms and semantic terms
// Backed by the REST /api/glossary/edges endpoint (the previous GraphQL
// implementation was disabled along with Hasura).
export function useGlossaryEdges() {
  const { tenant, datasource } = useTenant();

  return useQuery({
    queryKey: glossaryKeys.edges(),
    queryFn: async () => {
      if (!tenant?.id) return [] as CatalogEdge[];
      const params = new URLSearchParams();
      params.append('tenant_id', tenant.id);
      if (datasource?.id) params.append('tenant_instance_id', datasource.id);

      const res = await fetch(`/api/glossary/edges?${params.toString()}`, {
        credentials: 'include',
        headers: {
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
          ...(datasource?.id && { 'X-Tenant-Datasource-ID': datasource.id }),
          'X-Tenant-Region': getSelectedRegion(),
        },
      });
      if (!res.ok) {
        devDebug(`[useGlossaryEdges] /api/glossary/edges returned ${res.status}`);
        return [] as CatalogEdge[];
      }
      return (await res.json()) as CatalogEdge[];
    },
    enabled: !!(tenant?.id && datasource?.id),
  });
}

// Fetch all semantic data (business terms, semantic terms, semantic columns,
// calculation terms, edges) using the existing REST endpoints.
//
// The previous implementation was built on Hasura GraphQL which has been
// removed. We fan out to the same REST routes the rest of the app already
// uses, then re-assemble the shape the page expects:
//
//   {
//     business_terms:     CatalogNode[]  (filtered to type=business_term)
//     semantic_terms:     CatalogNode[]  (filtered to type=semantic_term)
//     semantic_columns:   CatalogNode[]  (filtered to type=semantic_column)
//     calculation_terms:  CatalogNode[]  (union of calculation, calculation_term, metric)
//     semantic_edges:     CatalogEdge[]
//     all_nodes:          CatalogNode[]  (every node in the datasource, for qualified_path lookup)
//     node_types:         NodeType[]     (passthrough from useNodeTypes)
//   }
//
// Every node is decorated with a synthetic `node_type: { catalog_type_name }`
// object so existing consumers (BusinessTermsTab, SemanticTermsTab, etc.) that
// read `node.node_type.catalog_type_name` keep working.
import { useNodeTypes } from './nodeTypes';

export function useAllSemanticData() {
  const { tenant, datasource } = useTenant();

  // 1. Pull node types so we can resolve node_type_id -> catalog_type_name
  const { data: nodeTypesList, isLoading: isNodeTypesLoading } = useNodeTypes();

  const buildHeaders = useCallback(() => ({
    ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
    ...(datasource?.id && { 'X-Tenant-Datasource-ID': datasource.id }),
    'X-Tenant-Region': getSelectedRegion(),
  }), [tenant?.id, datasource?.id]);

  // Fetch catalog nodes for a given catalog_type_name using /api/catalog/nodes?type=
  const fetchByType = useCallback(
    async (typeName: string): Promise<any[]> => {
      if (!tenant?.id) return [];
      const params = new URLSearchParams();
      params.append('tenant_id', tenant.id);
      if (datasource?.id) params.append('tenant_instance_id', datasource.id);
      params.append('type', typeName);
      const res = await fetch(`/api/catalog/nodes?${params.toString()}`, {
        credentials: 'include',
        headers: buildHeaders(),
      });
      if (!res.ok) {
        devDebug(`[useAllSemanticData] fetchByType(${typeName}) returned ${res.status}`);
        return [];
      }
      return (await res.json()) || [];
    },
    [tenant?.id, datasource?.id, buildHeaders],
  );

  // Fetch all glossary edges
  const fetchEdges = useCallback(async (): Promise<any[]> => {
    if (!tenant?.id) return [];
    const params = new URLSearchParams();
    params.append('tenant_id', tenant.id);
    if (datasource?.id) params.append('tenant_instance_id', datasource.id);
    const res = await fetch(`/api/glossary/edges?${params.toString()}`, {
      credentials: 'include',
      headers: buildHeaders(),
    });
    if (!res.ok) {
      devDebug(`[useAllSemanticData] fetchEdges returned ${res.status}`);
      return [];
    }
    return (await res.json()) || [];
  }, [tenant?.id, datasource?.id, buildHeaders]);

  const enabled = !!(tenant?.id && datasource?.id);

  const query = useQuery({
    queryKey: [
      ...glossaryKeys.all,
      'all-semantic-data',
      tenant?.id || null,
      datasource?.id || null,
    ],
    enabled,
    queryFn: async () => {
      // Fan out: 6 type-filtered node lookups + 1 edges lookup + 1 unfiltered node lookup.
      const [
        business,
        semantic,
        columns,
        calc,
        calcTerm,
        metric,
        allNodes,
        edges,
      ] = await Promise.all([
        fetchByType('business_term'),
        fetchByType('semantic_term'),
        fetchByType('semantic_column'),
        fetchByType('calculation'),
        fetchByType('calculation_term'),
        fetchByType('metric'),
        // all nodes - same /api/catalog/nodes but with no type filter
        (async () => {
          if (!tenant?.id) return [];
          const params = new URLSearchParams();
          params.append('tenant_id', tenant.id);
          if (datasource?.id) params.append('tenant_instance_id', datasource.id);
          const res = await fetch(`/api/catalog/nodes?${params.toString()}`, {
            credentials: 'include',
            headers: buildHeaders(),
          });
          if (!res.ok) return [];
          return (await res.json()) || [];
        })(),
        fetchEdges(),
      ]);

      // Decorate every node with a synthetic node_type so the
      // BusinessTermsTab/SemanticTermsTab/CalculationTermsTab can read
      // node.node_type.catalog_type_name just like before.
      const attachNodeType = (nodes: any[]) =>
        nodes.map((node) => {
          const typeDef = nodeTypesList?.find(
            (t: any) => t.id === node.node_type_id,
          );
          return {
            ...node,
            node_type: {
              catalog_type_name: typeDef?.catalog_type_name || node.catalog_type || 'unknown',
            },
          };
        });

      // Decorate edges with a friendly edge_type_name fallback.
      const attachEdgeType = (es: any[]) =>
        es.map((e) => ({
          ...e,
          edge_type_name: e.edge_type || e.relationship_type,
        }));

      return {
        business_terms: attachNodeType(business),
        semantic_terms: attachNodeType(semantic),
        semantic_columns: attachNodeType(columns),
        // calculation_terms unions the three calculation-style types
        // the GraphQL query used to group together.
        calculation_terms: attachNodeType([...calc, ...calcTerm, ...metric]),
        semantic_edges: attachEdgeType(edges),
        all_nodes: attachNodeType(allNodes),
        node_types: nodeTypesList || [],
      };
    },
    staleTime: 30_000,
  });

  const fallback = useMemo(
    () => ({
      business_terms: [],
      semantic_terms: [],
      semantic_columns: [],
      calculation_terms: [],
      semantic_edges: [],
      all_nodes: [],
      node_types: [],
    }),
    [],
  );

  return {
    data: query.data ?? fallback,
    isLoading: isNodeTypesLoading || query.isLoading,
    error: (query.error as Error | null)?.message ?? null,
    enabled,
    refetch: query.refetch,
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

      const res = await fetch(url, {
        method: 'PUT',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
          ...(datasource?.id && { 'X-Tenant-Datasource-ID': datasource.id }),
          'X-Tenant-Region': getSelectedRegion(),
        },
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
      const queryKey = responseData.catalog_type === 'semantic_term'
        ? glossaryKeys.semanticTerms()
        : glossaryKeys.businessTerms();

      queryClient.setQueryData<CatalogNode[]>(queryKey, (oldData) => {
        if (!oldData) return [];
        return oldData.map((term) =>
          term.id === variables.id ? { ...term, ...responseData } : term
        );
      });
      devDebug(`[useUpdateTerm.onSuccess] Optimistically updated cache for ${responseData.catalog_type}`);

      // Invalidate queries to ensure data consistency with the backend
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticTerms() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.businessTerms() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.term(variables.id) });

      // Refresh the aggregated glossary data so the page re-fetches terms/edges
      // with the new node attached.
      void queryClient.invalidateQueries({
        queryKey: [
          ...glossaryKeys.all,
          'all-semantic-data',
        ],
      });

      devDebug('[useUpdateTerm.onSuccess] Cache invalidation complete');
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

      const res = await fetch(`/api/glossary/terms?${params.toString()}`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
          ...(datasource?.id && { 'X-Tenant-Datasource-ID': datasource.id }),
          'X-Tenant-Region': getSelectedRegion(),
        },
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

      // Refresh the aggregated glossary data so the page re-fetches terms/edges.
      void queryClient.invalidateQueries({
        queryKey: [
          ...glossaryKeys.all,
          'all-semantic-data',
        ],
      });
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

      const res = await fetch(`/api/glossary/terms/${id}?${params.toString()}`, {
        method: 'DELETE',
        credentials: 'include',
        headers: {
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
          ...(datasource?.id && { 'X-Tenant-Datasource-ID': datasource.id }),
          'X-Tenant-Region': getSelectedRegion(),
        },
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

      // Refresh the aggregated glossary data so the page re-fetches terms/edges.
      void queryClient.invalidateQueries({
        queryKey: [
          ...glossaryKeys.all,
          'all-semantic-data',
        ],
      });
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

      const res = await fetch(`/api/glossary/edges?${params.toString()}`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
          ...(datasource?.id && { 'X-Tenant-Datasource-ID': datasource.id }),
        },
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

      const res = await fetch(`/api/glossary/edges/${data.id}?${params.toString()}`, {
        method: 'PUT',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
          ...(datasource?.id && { 'X-Tenant-Datasource-ID': datasource.id }),
          'X-Tenant-Region': getSelectedRegion(),
        },
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

      const res = await fetch(`/api/glossary/edges/${id}?${params.toString()}`, {
        method: 'DELETE',
        credentials: 'include',
        headers: {
          ...(tenant?.id && { 'X-Tenant-ID': tenant.id }),
          ...(datasource?.id && { 'X-Tenant-Datasource-ID': datasource.id }),
          'X-Tenant-Region': getSelectedRegion(),
        },
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
