import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';
import { useTenant } from '../contexts/TenantContext';
import { useApolloClient, useQuery as useApolloQuery, gql } from '@apollo/client';
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
// NOTE: Edges are now provided by GraphQL query (catalog_edge), so this REST call is disabled
export function useGlossaryEdges() {
  const { tenant, datasource } = useTenant();

  return useQuery({
    queryKey: glossaryKeys.edges(),
    queryFn: async () => {
      // Return empty array - edges are fetched via GraphQL instead
      return [] as CatalogEdge[];
    },
    enabled: !!(tenant?.id && datasource?.id),
  });
}

// GraphQL query for all semantic data (includes qualified_path for relationships display)
const GET_ALL_SEMANTIC_DATA = gql`
  query GetAllSemanticData(
    $datasourceId: uuid!, 
    $businessTermTypeId: uuid!, 
    $semanticTermTypeId: uuid!, 
    $semanticColumnTypeId: uuid!,
    $calculationTypeId: uuid!,
    $calculationTermTypeId: uuid!,
    $metricTypeId: uuid!
  ) {
    # Business Terms
    business_terms: catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId },
        node_type_id: { _eq: $businessTermTypeId }
      }
      order_by: { node_name: asc }
    ) {
      id
      node_type_id
      node_name
      description
      qualified_path
      parent_id
      properties
      created_at
      updated_at
    }
    
    # Semantic Terms
    semantic_terms: catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId },
        node_type_id: { _eq: $semanticTermTypeId }
      }
      order_by: { node_name: asc }
    ) {
      id
      node_type_id
      node_name
      description
      qualified_path
      parent_id
      properties
      created_at
      updated_at
    }
    
    # Semantic Columns
    semantic_columns: catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId },
        node_type_id: { _eq: $semanticColumnTypeId }
      }
      order_by: { node_name: asc }
    ) {
      id
      node_type_id
      node_name
      description
      qualified_path
      parent_id
      properties
      created_at
      updated_at
    }

    # Calculation Terms - using _in for multiple IDs if they are provided, essentially OR logic
    calculation_terms: catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId },
        node_type_id: { _in: [$calculationTypeId, $calculationTermTypeId, $metricTypeId] }
      }
      order_by: { node_name: asc }
    ) {
      id
      node_type_id
      node_name
      description
      qualified_path
      parent_id
      properties
      created_at
      updated_at
    }
    
    # All catalog nodes for qualified path lookup
    all_nodes: catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId }
      }
    ) {
      id
      node_name
      qualified_path
      node_type_id
      # Removed node_type relationship selection as it was causing schema errors
    }
    
    # Semantic Edges/Relationships
    # Semantic Edges/Relationships
    semantic_edges: catalog_edge(
      where: {
        tenant_datasource_id: { _eq: $datasourceId }
      }
      order_by: { created_at: desc }
    ) {
      id
      source_node_id
      target_node_id
      edge_type_id
      relationship_type
      properties
      created_at
      updated_at
      # Relationship 'edge_type' not exposed in Hasura yet, fetching separately
    }
    
    # Edge Types for lookups - REMOVED: catalog_edge_type not available in Hasura schema
    # Will fetch edge type names from edge objects directly if needed
    
    # Node Types for type name lookups
    node_types: catalog_node_type {
      id
      catalog_type_name
    }
  }
`;

import { useNodeTypes } from './nodeTypes';

// Fetch all semantic data (business terms, semantic terms, and edges)
export function useAllSemanticData() {
  const { datasource } = useTenant();
  const apolloClient = useApolloClient();

  // 1. Fetch node types first to get IDs
  const { data: nodeTypesList, isLoading: isNodeTypesLoading } = useNodeTypes();

  // 2. Resolve IDs
  // Use Nil UUID for missing types to prevent "unexpected null value" errors in GraphQL
  // and ensuring we just match nothing instead of crashing.
  const NIL_UUID = '00000000-0000-0000-0000-000000000000';

  // Check for invalid IDs as requested
  if (nodeTypesList) {
    const invalidTypes = nodeTypesList.filter(t => !t.id || t.id.trim() === '');
    if (invalidTypes.length > 0) {
      console.error('[SemanticTerms] Found invalid node types with null/empty IDs (Action required: Delete these):', invalidTypes);
    }
  }

  const typeMap = useMemo(() => {
    if (!nodeTypesList) return {
      business_term: NIL_UUID,
      semantic_term: NIL_UUID,
      semantic_column: NIL_UUID,
      calculation: NIL_UUID,
      calculation_term: NIL_UUID,
      metric: NIL_UUID,
    };
    return {
      business_term: nodeTypesList.find(t => t.catalog_type_name === 'business_term')?.id || NIL_UUID,
      semantic_term: nodeTypesList.find(t => t.catalog_type_name === 'semantic_term')?.id || NIL_UUID,
      semantic_column: nodeTypesList.find(t => t.catalog_type_name === 'semantic_column')?.id || NIL_UUID,
      calculation: nodeTypesList.find(t => t.catalog_type_name === 'calculation')?.id || NIL_UUID,
      calculation_term: nodeTypesList.find(t => t.catalog_type_name === 'calculation_term')?.id || NIL_UUID,
      metric: nodeTypesList.find(t => t.catalog_type_name === 'metric')?.id || NIL_UUID,
    };
  }, [nodeTypesList]);

  // 3. Query with resolved IDs
  const { data, loading: isGraphLoading, error, refetch } = useApolloQuery(GET_ALL_SEMANTIC_DATA, {
    variables: {
      datasourceId: datasource?.id || '',
      businessTermTypeId: typeMap.business_term,
      semanticTermTypeId: typeMap.semantic_term,
      semanticColumnTypeId: typeMap.semantic_column,
      calculationTypeId: typeMap.calculation,
      calculationTermTypeId: typeMap.calculation_term,
      metricTypeId: typeMap.metric,
    },
    skip: !datasource?.id || isNodeTypesLoading, // Wait for types
    client: apolloClient,
  });

  const transformedData = useMemo(() => {
    if (!data) return {
      business_terms: [],
      semantic_terms: [],
      semantic_edges: [],
      all_nodes: [],
      node_types: [],
      calculation_terms: [],
    };

    // Helper to attach node_type object manually
    const attachNodeType = (nodes: any[]) => {
      return nodes.map(node => {
        // Find the type name using the ID from our loaded types list
        const typeDef = nodeTypesList?.find(t => t.id === node.node_type_id);
        return {
          ...node,
          node_type: {
            catalog_type_name: typeDef?.catalog_type_name || 'unknown'
          }
        };
      });
    };

    // Helper to attach edge_type info manually
    const attachEdgeType = (edges: any[]) => {
      return edges.map(edge => {
        const typeDef = data.edge_types?.find((t: any) => t.id === edge.edge_type_id);
        return {
          ...edge,
          edge_type_name: typeDef?.edge_type_name, // Flatten for easy access
          edge_type: typeDef // Keep structured object for compatibility
        };
      });
    };

    return {
      business_terms: attachNodeType(data.business_terms || []),
      semantic_terms: attachNodeType(data.semantic_terms || []),
      semantic_edges: attachEdgeType(data.semantic_edges || []),
      // Also attach to all_nodes so lookups work
      all_nodes: attachNodeType(data.all_nodes || []),
      node_types: data.node_types || [],
      calculation_terms: attachNodeType(data.calculation_terms || []),
    };
  }, [data, nodeTypesList]);

  return {
    data: transformedData,
    isLoading: isNodeTypesLoading || isGraphLoading,
    error: error?.message || null,
    enabled: !!datasource?.id,
    refetch
  };
}

export const useAllSemanticDataQuery = useAllSemanticData;

// Update a semantic term or business term
export function useUpdateTerm() {
  const queryClient = useQueryClient();
  const apolloClient = useApolloClient();
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

      devDebug('[useUpdateTerm.onSuccess] All React Query caches invalidated');

      // Invalidate Apollo GraphQL cache
      void apolloClient.cache.evict({ fieldName: 'catalog_node' });
      void apolloClient.cache.gc();
      // Also refetch active GraphQL queries to ensure UI updates
      try {
        void apolloClient.refetchQueries({ include: 'active' });
      } catch (e) {
        // Best-effort; ignore errors here
      }

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
  const apolloClient = useApolloClient();
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

      // Invalidate Apollo GraphQL cache
      void apolloClient.cache.evict({ fieldName: 'catalog_node' });
      void apolloClient.cache.gc();
      // Also refetch active GraphQL queries to ensure UI updates
      try {
        void apolloClient.refetchQueries({ include: 'active' });
      } catch (e) {
        // Best-effort; ignore errors here
      }
    },
  });
}

// Delete a semantic term or business term
export function useDeleteTerm() {
  const queryClient = useQueryClient();
  const apolloClient = useApolloClient();
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

      // Invalidate Apollo GraphQL cache
      void apolloClient.cache.evict({ fieldName: 'catalog_node' });
      void apolloClient.cache.gc();
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
