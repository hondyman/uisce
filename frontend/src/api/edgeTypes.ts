import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type {
  EdgeType,
  CreateEdgeTypeRequest,
  UpdateEdgeTypeRequest,
  CreatePropertyRequest,
  UpdatePropertyRequest,
  DeletePropertyRequest,
} from '../types/edgeTypes';
import { apiFetch } from '../lib/apiClient';

// Re-export EdgeType for convenience
export type { EdgeType } from '../types/edgeTypes';

// Query keys
export const edgeTypesKeys = {
  all: ['edge-types'] as const,
  lists: () => [...edgeTypesKeys.all, 'list'] as const,
  list: () => [...edgeTypesKeys.lists()] as const,
  details: () => [...edgeTypesKeys.all, 'detail'] as const,
  detail: (id: string) => [...edgeTypesKeys.details(), id] as const,
  properties: (id: string) => [...edgeTypesKeys.all, 'properties', id] as const,
  search: (q: string) => [...edgeTypesKeys.all, 'search', q] as const,
};

// List all edge types for a tenant
export function useEdgeTypes() {
  return useQuery({
    queryKey: edgeTypesKeys.list(),
    queryFn: async () => {
      const res = await apiFetch('/api/edge-types');
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to fetch edge types');
      }
      return res.json() as Promise<EdgeType[]>;
    },
  });
}

// Search edge types server-side for a tenant with a query string
export function useSearchEdgeTypes(q: string) {
  return useQuery({
    queryKey: edgeTypesKeys.search(q),
    queryFn: async () => {
      const res = await apiFetch(`/api/edge-types?q=${encodeURIComponent(q)}`);
      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || 'Failed to search edge types');
      }
      return res.json() as Promise<EdgeType[]>;
    },
    enabled: !!q && q.trim() !== '',
  });
}

// Get a single edge type by ID
export function useEdgeType(id: string) {
  return useQuery({
    queryKey: edgeTypesKeys.detail(id),
    queryFn: async () => {
      const res = await apiFetch(`/api/edge-types/${id}`);
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to fetch edge type');
      }
      return res.json() as Promise<EdgeType>;
    },
    enabled: !!id,
  });
}

// Create a new edge type
export function useCreateEdgeType() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: CreateEdgeTypeRequest) => {
      const res = await apiFetch('/api/edge-types', {
        method: 'POST',
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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list() });
    },
  });
}

// Update an existing edge type
export function useUpdateEdgeType() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: UpdateEdgeTypeRequest }) => {
      const res = await apiFetch(`/api/edge-types/${id}`, {
        method: 'PATCH',
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
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list() });
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.detail(variables.id) });
    },
  });
}

// Delete an edge type
export function useDeleteEdgeType() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id }: { id: string }) => {
      const res = await apiFetch(`/api/edge-types/${id}`, {
        method: 'DELETE',
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to delete edge type');
      }
      return;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list() });
    },
  });
}

// Property management hooks
export function useCreateEdgeProperty() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ edge_type_id, property }: CreatePropertyRequest) => {
      const res = await apiFetch(`/api/edge-types/${edge_type_id}/properties`, {
        method: 'POST',
        body: JSON.stringify({ property }),
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to create property');
      }
      return res.json() as Promise<EdgeType>;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.detail(variables.edge_type_id) });
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list() });
    },
  });
}

export function useUpdateEdgeProperty() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ edge_type_id, property_name, property }: UpdatePropertyRequest) => {
      const res = await apiFetch(`/api/edge-types/${edge_type_id}/properties/${property_name}`, {
        method: 'PATCH',
        body: JSON.stringify({ property }),
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to update property');
      }
      return res.json() as Promise<EdgeType>;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.detail(variables.edge_type_id) });
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list() });
    },
  });
}

export function useDeleteEdgeProperty() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ edge_type_id, property_name }: DeletePropertyRequest) => {
      const res = await apiFetch(`/api/edge-types/${edge_type_id}/properties/${property_name}`, {
        method: 'DELETE',
      });
      if (!res.ok) {
        const error = await res.text();
        throw new Error(error || 'Failed to delete property');
      }
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.detail(variables.edge_type_id) });
      queryClient.invalidateQueries({ queryKey: edgeTypesKeys.list() });
    },
  });
}