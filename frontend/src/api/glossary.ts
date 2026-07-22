import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';
import { useTenant } from '../contexts/TenantContext';
import { devDebug } from '../utils/devLogger';
import { getSelectedRegion } from '../lib/region';
import { useNodeTypes } from './nodeTypes';

// ============================================================================
// Catalog node type UUIDs (sourced from the gold copy tenant's catalog_node_type).
// These are pinned in the frontend so we can filter by `node_type_id=<UUID>`
// directly on /api/catalog/nodes, instead of the (slower, string-based) name
// filter. They were sourced from the SQL dump on 2026-07-02 — the underlying
// values live in the central platform DB and are stable.
// ============================================================================
export const BUSINESS_TERM_TYPE_ID = '21645d21-de5f-4feb-af99-99273ea75626';
export const SEMANTIC_TERM_TYPE_ID  = '820b942a-9c9e-4abc-acdc-84616db33098';

// Effective auth token: impersonation scoped token wins if present, otherwise
// the primary OIDC token persisted by AuthContext.
function getEffectiveAuthToken(): string | null {
  try {
    return localStorage.getItem('uisce_impersonation_token') || localStorage.getItem('auth_token');
  } catch {
    return null;
  }
}

// Build headers for glossary catalog-node calls. Business and semantic terms are
// scoped by TENANT only; they never carry a datasource/instance header because
// those node types have no tenant_datasource_id / tenant_instance_id.
function getGlossaryHeaders(tenantId?: string | null): Record<string, string> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = getEffectiveAuthToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;
  if (tenantId) headers['X-Tenant-ID'] = tenantId;
  const region = getSelectedRegion();
  if (region) headers['X-Tenant-Region'] = region;
  return headers;
}

// Fetch wrapper for glossary endpoints that intentionally omits the auto-injected
// datasource header. Caller-supplied headers win, otherwise we inject auth,
// tenant and region only.
async function glossaryFetch(
  input: RequestInfo | URL,
  tenantId?: string | null,
  init: RequestInit = {},
): Promise<Response> {
  const headers = new Headers(init.headers || {});
  const glossaryHeaders = getGlossaryHeaders(tenantId);
  Object.entries(glossaryHeaders).forEach(([key, value]) => {
    if (!headers.has(key)) headers.set(key, value);
  });
  return fetch(input, { ...init, headers });
}

// Types for Glossary
export interface CatalogNode {
  id: string;
  tenant_datasource_id?: string | null;
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
  edge_type_id?: string;
  edge_type_name: string;
  description: string;
  subject_node_id: string;
  subject_node_type_id?: string;
  object_node_id: string;
  object_node_type_id: string;
  properties: EdgeProperty[] | Record<string, any>;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  tenant_id: string;
  core_id: string | null;
  // Backwards-compatible aliases used by legacy GraphQL consumers
  source_node_id?: string;
  target_node_id?: string;
  relationship_type?: string;
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
  // Semantic terms are catalog nodes with node_type_id = SEMANTIC_TERM_TYPE_ID
}

export interface BusinessTerm extends CatalogNode {
  // Business terms are catalog nodes with node_type_id = BUSINESS_TERM_TYPE_ID
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

// =============================================================================
// Glossary READ hooks
// =============================================================================
//
// IMPORTANT: semantic terms and business terms are tied to the TENANT (not the
// datasource). They are a merge of the gold copy tenant's core catalog and the
// caller's scoped tenant. The frontend therefore queries with:
//   ?tenant_id=<scope_or_current>  &  node_type_id=<UUID>  &  limit=...
// (NO `tenant_instance_id` / datasource param.)
// =============================================================================

// Fetch all semantic terms
export function useSemanticTerms(opts?: { tenantOverride?: string }) {
  const { tenant } = useTenant();
  const scopeTenant = opts?.tenantOverride || tenant?.id;

  return useQuery({
    queryKey: [...glossaryKeys.semanticTerms(), scopeTenant ?? null],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (scopeTenant) params.append('tenant_id', scopeTenant);
      params.append('node_type_id', SEMANTIC_TERM_TYPE_ID);
      params.append('limit', '100000');

      const url = `/api/catalog/nodes?${params.toString()}`;
      devDebug('[useSemanticTerms] GET', url);
      const res = await glossaryFetch(url, scopeTenant);

      if (!res.ok) {
        const error = await res.text();
        devDebug('[useSemanticTerms] ERROR', res.status, error.slice(0, 200));
        throw new Error(error || 'Failed to fetch semantic terms');
      }

      const allNodes = ((await res.json()) as CatalogNode[]) || [];
      devDebug('[useSemanticTerms] returned', allNodes.length, 'nodes; first sample:', allNodes[0] ? {
        id: allNodes[0].id,
        node_name: allNodes[0].node_name,
        node_type_id: allNodes[0].node_type_id,
        catalog_type: allNodes[0].catalog_type,
        catalog_type_name: allNodes[0].catalog_type_name,
      } : 'none');
      return allNodes.filter((node) => node.node_type_id === SEMANTIC_TERM_TYPE_ID);
    },
    enabled: !!scopeTenant,
  });
}

// Fetch all business terms
export function useBusinessTerms(opts?: { tenantOverride?: string }) {
  const { tenant } = useTenant();
  const scopeTenant = opts?.tenantOverride || tenant?.id;

  return useQuery({
    queryKey: [...glossaryKeys.businessTerms(), scopeTenant ?? null],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (scopeTenant) params.append('tenant_id', scopeTenant);
      params.append('node_type_id', BUSINESS_TERM_TYPE_ID);
      params.append('limit', '100000');

      const url = `/api/catalog/nodes?${params.toString()}`;
      devDebug('[useBusinessTerms] GET', url);
      const res = await glossaryFetch(url, scopeTenant);

      if (!res.ok) {
        const error = await res.text();
        devDebug('[useBusinessTerms] ERROR', res.status, error.slice(0, 200));
        throw new Error(error || 'Failed to fetch business terms');
      }

      const allNodes = ((await res.json()) as CatalogNode[]) || [];
      devDebug('[useBusinessTerms] returned', allNodes.length, 'nodes; first sample:', allNodes[0] ? {
        id: allNodes[0].id,
        node_name: allNodes[0].node_name,
        node_type_id: allNodes[0].node_type_id,
        catalog_type: allNodes[0].catalog_type,
        catalog_type_name: allNodes[0].catalog_type_name,
      } : 'none');
      return allNodes.filter((node) => node.node_type_id === BUSINESS_TERM_TYPE_ID);
    },
    enabled: !!scopeTenant,
  });
}

// Fetch edges between business terms and semantic terms from the REST endpoint.
// Falls back to an empty array if the backend returns no data.
export function useGlossaryEdges(opts?: { tenantOverride?: string }) {
  const { tenant } = useTenant();
  const scopeTenant = opts?.tenantOverride || tenant?.id;

  return useQuery({
    queryKey: [...glossaryKeys.edges(), scopeTenant ?? null],
    queryFn: async () => {
      const params = new URLSearchParams();
      if (scopeTenant) params.append('tenant_id', scopeTenant);

      const url = `/api/glossary/edges?${params.toString()}`;
      devDebug('[useGlossaryEdges] GET', url);
      const res = await glossaryFetch(url, scopeTenant);

      if (!res.ok) {
        const error = await res.text();
        devDebug('[useGlossaryEdges] ERROR', res.status, error.slice(0, 200));
        // Gracefully degrade if the GET endpoint is not deployed yet; this keeps
        // the glossary UI usable while the backend endpoint is rolled out.
        if (res.status === 404 || res.status === 501) {
          return [] as CatalogEdge[];
        }
        throw new Error(error || 'Failed to fetch glossary edges');
      }

      const payload = (await res.json()) as CatalogEdge[] | { edges?: CatalogEdge[] };
      const edges: CatalogEdge[] = Array.isArray(payload)
        ? payload
        : Array.isArray((payload as any).edges)
          ? (payload as any).edges
          : [];

      // Normalize both naming conventions (subject/object from the REST spec and
      // source/target returned by the current backend) so every downstream consumer
      // sees the fields it expects.
      return edges.map((edge: CatalogEdge) => {
        const subjectId = edge.subject_node_id || edge.source_node_id || '';
        const objectId = edge.object_node_id || edge.target_node_id || '';
        return {
          ...edge,
          subject_node_id: subjectId,
          object_node_id: objectId,
          source_node_id: edge.source_node_id || subjectId,
          target_node_id: edge.target_node_id || objectId,
          relationship_type: edge.edge_type_name || edge.relationship_type || '',
        } as CatalogEdge;
      });
    },
    enabled: !!scopeTenant,
  });
}

// Fetch all catalog nodes for specific node_type_id UUIDs (REST-only).
// Drops `tenant_instance_id`; semantic/business terms are scoped by tenant only.
function useCatalogNodesByTypeIds(
  typeIds: string[],
  opts?: { tenantOverride?: string }
) {
  const { tenant } = useTenant();
  const scopeTenant = opts?.tenantOverride || tenant?.id;

  return useQuery({
    queryKey: [...glossaryKeys.catalogNodes(), scopeTenant ?? null, typeIds.join(',')],
    queryFn: async () => {
      const results = await Promise.all(
        typeIds.map(async (tid) => {
          const params = new URLSearchParams();
          if (scopeTenant) params.append('tenant_id', scopeTenant);
          params.append('node_type_id', tid);
          params.append('limit', '100000');

          const res = await glossaryFetch(`/api/catalog/nodes?${params.toString()}`, scopeTenant);
          if (!res.ok) {
            devDebug(`[useCatalogNodesByTypeIds] failed to fetch type_id ${tid}`);
            return [];
          }
          return ((await res.json()) as CatalogNode[]) || [];
        })
      );
      return results.flat();
    },
    enabled: !!scopeTenant && typeIds.length > 0,
  });
}

// Attach a denormalized `node_type` object to each catalog node based on the
// shared nodeTypes list. Matches the shape the GraphQL version produced.
function attachNodeType(
  nodes: CatalogNode[],
  nodeTypesList?: { id: string; catalog_type_name?: string }[] | null
) {
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
// calculation terms) using REST only. Backed by node_type_id UUIDs only — no
// string name filters.
export function useAllSemanticData(opts?: { tenantOverride?: string }) {
  const { tenant } = useTenant();
  const scopeTenant = opts?.tenantOverride || tenant?.id;

  // 1. Node types (REST) — only used to attach denormalized `node_type` info
  const { data: nodeTypesList, isLoading: isNodeTypesLoading } = useNodeTypes({ tenantId: scopeTenant });

  // 2. Catalog nodes fetched by node_type_id UUID
  const requiredTypeIds = useMemo(
    () => [
      BUSINESS_TERM_TYPE_ID,
      SEMANTIC_TERM_TYPE_ID,
      // Other technical types — kept in case downstream code reads them,
      // but the user only asked for business + semantic here.
      // '1439f761-606a-44cb-b4f8-7aa6b27a9bf5', // semantic_column
      // '2e6aa1bb-2582-4a0b-8a19-e4753f1ff5a8', // calculation_term
      // 'a3b8d1b6-0b3b-4b1a-9c1a-1a2b3c4d5e6f', // delta_kpi
    ],
    []
  );
  const nodesQuery = useCatalogNodesByTypeIds(requiredTypeIds, { tenantOverride: scopeTenant });

  // 3. Edges between business and semantic terms (used for mapped/unmapped filters)
  const { data: edgesData, isLoading: edgesLoading, refetch: refetchEdges } = useGlossaryEdges({ tenantOverride: scopeTenant });

  // 4. Filter and shape data the way the GraphQL version did
  const transformedData = useMemo(() => {
    const allNodes = nodesQuery.data || [];
    const allWithType = attachNodeType(allNodes, nodeTypesList);

    const business_terms = allWithType.filter(
      (n) => n.node_type_id === BUSINESS_TERM_TYPE_ID
    );
    const semantic_terms = allWithType.filter(
      (n) => n.node_type_id === SEMANTIC_TERM_TYPE_ID
    );

    // Compute is_mapped for semantic terms based on edges
    const edges = edgesData || [];
    const connectedSemanticIds = new Set<string>();
    edges.forEach((edge: CatalogEdge) => {
      const subjectId = edge.subject_node_id || edge.source_node_id;
      const objectId = edge.object_node_id || edge.target_node_id;
      if (subjectId) connectedSemanticIds.add(subjectId);
      if (objectId) connectedSemanticIds.add(objectId);
    });
    const semantic_terms_with_mapping = semantic_terms.map((term) => ({
      ...term,
      is_mapped: connectedSemanticIds.has(term.id),
    }));

    return {
      business_terms,
      semantic_terms: semantic_terms_with_mapping,
      semantic_edges: edges,
      all_nodes: allWithType,
      node_types: nodeTypesList || [],
      // Keep these keys in the response shape so downstream callers don't break:
      calculation_terms: [] as CatalogNode[],
      semantic_columns: [] as CatalogNode[],
    };
  }, [nodesQuery.data, nodeTypesList, edgesData]);

  return {
    data: transformedData,
    isLoading: isNodeTypesLoading || nodesQuery.isLoading || edgesLoading,
    error: (nodesQuery.error as Error | null)?.message || null,
    enabled: !!scopeTenant,
    refetch: () => {
      void nodesQuery.refetch();
      void refetchEdges();
    },
  };
}

export const useAllSemanticDataQuery = useAllSemanticData;

// =============================================================================
// Glossary CRUD hooks
// =============================================================================
//
// For semantic/business terms: tenant_datasource_id is NEVER set (these types
// have no datasource). The CRUD payloads below explicitly omit it.
// =============================================================================

interface GlossaryMutationOpts { tenantOverride?: string; }

// Update a semantic term or business term
export function useUpdateTerm(opts?: GlossaryMutationOpts) {
  const queryClient = useQueryClient();
  const { tenant } = useTenant();
  const scopeTenant = opts?.tenantOverride || tenant?.id;

  return useMutation({
    mutationFn: async (data: { id: string; updates: Partial<CatalogNode> }) => {
      const params = new URLSearchParams();
      if (scopeTenant) {
        params.append('tenant_id', scopeTenant);
      }

      devDebug('[useUpdateTerm] Starting update for term:', data.id);
      devDebug('[useUpdateTerm] Updates to send:', JSON.stringify(data.updates, null, 2));
      devDebug('[useUpdateTerm] parent_id value:', data.updates.parent_id);
      devDebug('[useUpdateTerm] catalog_type:', data.updates.catalog_type);

      // For semantic/business terms, NEVER set tenant_datasource_id
      const updatesWithoutDatasource: any = { ...data.updates };
      if (updatesWithoutDatasource.tenant_datasource_id !== undefined) {
        delete updatesWithoutDatasource.tenant_datasource_id;
      }

      // Ensure parent_id is explicitly included for semantic terms
      // ALSO ensure properties is always an object, never an array
      let updatePayload: any = {
        ...updatesWithoutDatasource,
        ...(data.updates.catalog_type === 'semantic_term' && {
          parent_id: data.updates.parent_id ?? null,
        }),
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

      const res = await glossaryFetch(url, scopeTenant, {
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
export function useCreateTerm(opts?: GlossaryMutationOpts) {
  const queryClient = useQueryClient();
  const { tenant } = useTenant();
  const scopeTenant = opts?.tenantOverride || tenant?.id;

  return useMutation({
    mutationFn: async (data: Omit<CatalogNode, 'id' | 'created_at' | 'updated_at'>) => {
      const params = new URLSearchParams();
      if (scopeTenant) {
        params.append('tenant_id', scopeTenant);
      }

      // For semantic/business terms, NEVER set tenant_datasource_id
      const createPayload: any = { ...data };
      if (createPayload.tenant_datasource_id !== undefined) {
        delete createPayload.tenant_datasource_id;
      }

      // Normalize properties to always be an object (not array)
      if (createPayload.properties && Array.isArray(createPayload.properties)) {
        devDebug('[useCreateTerm] Properties came as array, converting to empty object for proper storage');
        createPayload.properties = {};
      }

      const res = await glossaryFetch(`/api/glossary/terms?${params.toString()}`, scopeTenant, {
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
export function useDeleteTerm(opts?: GlossaryMutationOpts) {
  const queryClient = useQueryClient();
  const { tenant } = useTenant();
  const scopeTenant = opts?.tenantOverride || tenant?.id;

  return useMutation({
    mutationFn: async (id: string) => {
      const params = new URLSearchParams();
      if (scopeTenant) {
        params.append('tenant_id', scopeTenant);
      }

      // Step 1: Check BO dependencies before attempting delete
      const checkRes = await glossaryFetch(
        `/api/glossary/nodes/${id}/dependencies?${params.toString()}`,
        scopeTenant
      );

      if (!checkRes.ok) {
        const errorText = await checkRes.text();
        // Try to parse as JSON in case it contains dependency info even on error
        try {
          const errJson = JSON.parse(errorText);
          if (errJson.dependencies && errJson.dependencies.length > 0) {
            const deps = errJson.dependencies;
            const boNames = [...new Set(deps.map((d: any) => d.bo_name))].join(', ');
            const details = deps.map((d: any) => `- ${d.bo_name}: ${d.ref_detail || d.bo_key}`).join('\n');
            const depError = new Error(
              `Cannot delete: This term is linked to ${deps.length} BO field(s) in: ${boNames || 'unknown business objects'}. Unlink the fields first.\n\nDetails:\n${details}`
            );
            (depError as any).code = 'BO_DEPENDENCIES_BLOCK_DELETION';
            (depError as any).dependencies = deps;
            (depError as any).dependencyReport = errJson;
            throw depError;
          }
        } catch (parseErr) {
          if (parseErr instanceof Error && (parseErr as any).code === 'BO_DEPENDENCIES_BLOCK_DELETION') {
            throw parseErr;
          }
        }
        throw new Error(`Dependency check failed: ${errorText}`);
      }

      const depCheck = await checkRes.json() as {
        can_delete: boolean;
        dependencies?: Array<{
          ref_table: string;
          ref_id: string;
          bo_id: string;
          bo_key: string;
          bo_name: string;
          ref_detail: string;
        }>;
        message?: string;
        edge_count?: number;
        validation_count?: number;
        suggestion_count?: number;
      };

      if (!depCheck.can_delete) {
        const deps = depCheck.dependencies ?? [];
        const boNames = [...new Set(deps.map((d: any) => d.bo_name))].join(', ');
        const details = deps.map((d: any) => `- ${d.bo_name}: ${d.ref_detail || d.bo_key}`).join('\n');
        const depError = new Error(
          `Cannot delete: This term is linked to ${deps.length} BO field(s) in: ${boNames || 'unknown business objects'}. Unlink the fields first.\n\nDetails:\n${details}`
        );
        (depError as any).code = 'BO_DEPENDENCIES_BLOCK_DELETION';
        (depError as any).dependencies = deps;
        (depError as any).dependencyReport = depCheck;
        throw depError;
      }

      // Step 2: Proceed with deletion
      const res = await glossaryFetch(`/api/glossary/terms/${id}?${params.toString()}`, scopeTenant, {
        method: 'DELETE',
      });

      if (!res.ok) {
        const errorText = await res.text();
        let errorCode = 'DELETE_FAILED';
        let errorMessage = 'Failed to delete term';
        let dependencies: any[] = [];
        let dependencyReport: any = null;
        try {
          const errJson = JSON.parse(errorText);
          if (errJson.code === 'BO_DEPENDENCIES_BLOCK_DELETION' || errJson.error === 'BO_DEPENDENCIES_BLOCK_DELETION') {
            errorCode = 'BO_DEPENDENCIES_BLOCK_DELETION';
          }
          // Preserve dependency details from the response
          if (errJson.dependencies && Array.isArray(errJson.dependencies)) {
            dependencies = errJson.dependencies;
          }
          if (errJson.dependency_report) {
            dependencyReport = errJson.dependency_report;
          } else if (errJson.dependencyReport) {
            dependencyReport = errJson.dependencyReport;
          }
          errorMessage = errJson.message || errJson.error || errorText;
        } catch {
          errorMessage = errorText || 'Failed to delete term';
        }

        // Clean up double "Failed to delete term" prefixes (with colon)
        if (errorMessage.startsWith('Failed to delete term: ')) {
          errorMessage = errorMessage.substring('Failed to delete term: '.length);
        }
        // Also handle bare "Failed to delete term" (no colon) by stripping if it appears at start
        if (errorMessage === 'Failed to delete term') {
          errorMessage = 'Delete operation failed';
        }

        const err = new Error(errorMessage);
        (err as any).code = errorCode;
        if (dependencies.length > 0) {
          (err as any).dependencies = dependencies;
        }
        if (dependencyReport) {
          (err as any).dependencyReport = dependencyReport;
        }
        throw err;
      }
      return res.json();
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticTerms() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.businessTerms() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.catalogNodes() });
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.semanticData() });
      // Invalidate BO-related caches since deletion may affect BO bindings
      void queryClient.invalidateQueries({ queryKey: ['business-objects'] });
      void queryClient.invalidateQueries({ queryKey: ['business-object'] });
      void queryClient.invalidateQueries({ queryKey: ['bo-bindings'] });
    },
  });
}

// Create a new edge between terms
export function useCreateTermEdge() {
  const queryClient = useQueryClient();
  const { tenant } = useTenant();

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

      const res = await glossaryFetch(`/api/glossary/edges?${params.toString()}`, tenant?.id, {
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
  const { tenant } = useTenant();

  return useMutation({
    mutationFn: async (data: { id: string; updates: Partial<CatalogEdge> }) => {
      const params = new URLSearchParams();
      if (tenant?.id) {
        params.append('tenant_id', tenant.id);
      }

      const res = await glossaryFetch(`/api/glossary/edges/${data.id}?${params.toString()}`, tenant?.id, {
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
  const { tenant } = useTenant();

  return useMutation({
    mutationFn: async (id: string) => {
      const params = new URLSearchParams();
      if (tenant?.id) {
        params.append('tenant_id', tenant.id);
      }

      const res = await glossaryFetch(`/api/glossary/edges/${id}?${params.toString()}`, tenant?.id, {
        method: 'DELETE',
      });

      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to delete edge');
      }
      // The backend returns 204 No Content on success; avoid parsing an empty body.
      if (res.status === 204) {
        return null;
      }
      return res.json();
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: glossaryKeys.edges() });
    },
  });
}

export const useDeleteSemanticTerm = useDeleteTerm;
