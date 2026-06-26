import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type {
  EdgeType,
  CreateEdgeTypeRequest,
  UpdateEdgeTypeRequest,
  CreatePropertyRequest,
  UpdatePropertyRequest,
  DeletePropertyRequest,
} from '../types/edgeTypes';

// Re-export EdgeType for convenience
export type { EdgeType } from '../types/edgeTypes';

// Query keys
export const edgeTypesKeys = {
  all: ['edge-types'] as const,
  lists: () => [...edgeTypesKeys.all, 'list'] as const,
  list: (tenantId: string) => [...edgeTypesKeys.lists(), tenantId] as const,
  details: () => [...edgeTypesKeys.all, 'detail'] as const,
  detail: (id: string, tenantId: string) => [...edgeTypesKeys.details(), id, tenantId] as const,
  properties: (id: string, tenantId: string) => [...edgeTypesKeys.all, 'properties', id, tenantId] as const,
  search: (tenantId: string, q: string) => [...edgeTypesKeys.all, 'search', tenantId, q] as const,
};

// List all edge types for a tenant
export function useEdgeTypes(tenantId: string) {
  return useQuery({
    queryKey: edgeTypesKeys.list(tenantId),
    queryFn: async () => {
      const res = await fetch(`/api/edge-types?tenant_id=${tenantId}`, {
        credentials: 'include',
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to fetch edge types');
      }
      return res.json() as Promise<EdgeType[]>;
    },
    enabled: !!tenantId,
  });
}

// Search edge types server-side for a tenant with a query string
export function useSearchEdgeTypes(tenantId: string, q: string) {
  return useQuery({
    queryKey: edgeTypesKeys.search(tenantId, q),
    queryFn: async () => {
      const res = await fetch(`/api/edge-types?tenant_id=${tenantId}&q=${encodeURIComponent(q)}`, {
        credentials: 'include',
      });
      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || 'Failed to search edge types');
      }
      return res.json() as Promise<EdgeType[]>;
    },
    enabled: !!tenantId && !!q && q.trim() !== '',
  });
}

// Get a single edge type by ID
export function useEdgeType(id: string, tenantId: string) {
  return useQuery({
    queryKey: edgeTypesKeys.detail(id, tenantId),
    queryFn: async () => {
      const res = await fetch(`/api/edge-types/${id}?tenant_id=${tenantId}`, {
        credentials: 'include',
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to fetch edge type');
      }
      return res.json() as Promise<EdgeType>;
    },
    enabled: !!id && !!tenantId,
  });
}

// Create a new edge type
export function useCreateEdgeType() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (data: CreateEdgeTypeRequest) => {
      const res = await fetch('/api/edge-types', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        const errorText = await res.text();
        const err = new Error(errorText || 'Failed to create edge type');
        (err as any).status = res.status;
        throw err;
      }
      return res.json() as Promise<EdgeType>;
    },
    onSuccess: (data) => {
      // Invalidate the list query for this tenant
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list(data.tenant_id) });
    },
  });
}

// Update an existing edge type
export function useUpdateEdgeType() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ id, tenantId, data }: { id: string; tenantId: string; data: UpdateEdgeTypeRequest }) => {
      const res = await fetch(`/api/edge-types/${id}?tenant_id=${tenantId}`, {
        method: 'PATCH',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        const errorText = await res.text();
        const err = new Error(errorText || 'Failed to update edge type');
        (err as any).status = res.status;
        throw err;
      }
      return res.json() as Promise<EdgeType>;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list(variables.tenantId) });
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.detail(variables.id, variables.tenantId) });
    },
  });
}

// Delete an edge type
export function useDeleteEdgeType() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ id, tenantId }: { id: string; tenantId: string }) => {
      const res = await fetch(`/api/edge-types/${id}?tenant_id=${tenantId}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to delete edge type');
      }
      return;
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list(variables.tenantId) });
    },
  });
}

// Property management hooks
export function useCreateEdgeProperty() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ edge_type_id, tenant_id, property }: CreatePropertyRequest) => {
      const res = await fetch(`/api/edge-types/${edge_type_id}/properties`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tenant_id, property }),
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to create property');
      }
      return res.json() as Promise<EdgeType>;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.detail(variables.edge_type_id, variables.tenant_id) });
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list(variables.tenant_id) });
    },
  });
}

export function useUpdateEdgeProperty() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ edge_type_id, tenant_id, property_name, property }: UpdatePropertyRequest) => {
      const res = await fetch(`/api/edge-types/${edge_type_id}/properties/${property_name}`, {
        method: 'PATCH',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tenant_id, property }),
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to update property');
      }
      return res.json() as Promise<EdgeType>;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.detail(variables.edge_type_id, variables.tenant_id) });
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list(variables.tenant_id) });
    },
  });
}

export function useDeleteEdgeProperty() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ edge_type_id, tenant_id, property_name }: DeletePropertyRequest) => {
      const res = await fetch(`/api/edge-types/${edge_type_id}/properties/${property_name}?tenant_id=${tenant_id}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to delete property');
      }
      return res.json() as Promise<EdgeType>;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.detail(variables.edge_type_id, variables.tenant_id) });
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list(variables.tenant_id) });
    },
  });
}
