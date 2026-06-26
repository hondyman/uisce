import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTenant } from '../contexts/TenantContext';

export interface NodeType {
  id: string;
  tenant_id: string;
  catalog_type_name: string;
  description: string;
  is_active: boolean;
  parent_type_id?: string | null;
  config: Record<string, any>;
  properties: NodeProperty[];
  created_at: string;
  updated_at: string;
}

export interface NodeProperty {
  name: string;
  label: string;
  data_type: string;
  nullable: boolean;
  default_value?: any;
  input_type: string;
  format?: string;
  validation?: Record<string, any>;
  options?: string[];
  order: number;
  lookup_id?: string;
  cascade_from?: string;
}

export interface CatalogNode {
  id: string;
  tenant_id: string;
  tenant_datasource_id?: string;
  node_type_id: string;
  node_name: string;
  qualified_path?: string;
  description?: string;
  parent_id?: string | null;
  properties?: Record<string, any>;
  config?: Record<string, any>;
  created_at: string;
  updated_at: string;
  is_active?: boolean;
  score?: number; // Search score
}

// Query keys for React Query
export const nodeTypesKeys = {
  all: ['nodeTypes'] as const,
  lists: () => [...nodeTypesKeys.all, 'list'] as const,
  list: (filters: Record<string, any>) => [...nodeTypesKeys.lists(), filters] as const,
  details: () => [...nodeTypesKeys.all, 'detail'] as const,
  detail: (id: string) => [...nodeTypesKeys.details(), id] as const,
  nodes: (typeId: string) => [...nodeTypesKeys.detail(typeId), 'nodes'] as const,
};

// Fetch all node types
export function useNodeTypes(search?: string) {
  const { tenant } = useTenant();

  return useQuery({
    queryKey: nodeTypesKeys.list({ search }),
    queryFn: async (): Promise<NodeType[]> => {
      if (!tenant?.id) {
        return [];
      }

      const params = new URLSearchParams();
      params.append('tenant_id', tenant.id);
      // Fix: Some legacy callers pass tenant.id as the first argument, which is treated as 'search'. 
      // We ignore it if it matches the tenant ID.
      if (search && search !== tenant.id) {
        params.append('q', search);
      }

      const res = await fetch(`/api/node-types?${params.toString()}`, {
        headers: {
          'X-Tenant-ID': tenant.id,
        },
      });

      if (!res.ok) {
        throw new Error('Failed to fetch node types');
      }

      const data = await res.json();
      return data.data || data; // Handle both wrapped and unwrapped responses
    },
    enabled: !!tenant?.id,
  });
}

// Fetch a single node type
export function useNodeType(id: string) {
  const { tenant } = useTenant();

  return useQuery({
    queryKey: nodeTypesKeys.detail(id),
    queryFn: async (): Promise<NodeType> => {
      if (!tenant?.id || !id) {
        throw new Error('Missing tenant or node type ID');
      }

      const params = new URLSearchParams();
      params.append('tenant_id', tenant.id);

      const res = await fetch(`/api/node-types/${id}?${params.toString()}`, {
        headers: {
          'X-Tenant-ID': tenant.id,
        },
      });

      if (!res.ok) {
        throw new Error('Failed to fetch node type');
      }

      const data = await res.json();
      return data.data || data;
    },
    enabled: !!tenant?.id && !!id,
  });
}

// Fetch all nodes of a specific type
export function useNodesByType(typeId: string, search?: string) {
  const { tenant, datasource } = useTenant();

  return useQuery({
    queryKey: nodeTypesKeys.nodes(typeId),
    queryFn: async (): Promise<CatalogNode[]> => {
      if (!tenant?.id || !typeId) {
        return [];
      }

      const params = new URLSearchParams();
      params.append('tenant_id', tenant.id);
      if (datasource?.id) {
        params.append('tenant_instance_id', datasource.id);
      }
      if (search) {
        params.append('q', search);
      }

      // We'll use the generic search endpoint or a specific nodes endpoint if available
      // For now, assuming we filter client-side or use a specific endpoint
      // Using /api/catalog/nodes which was implied in previous context or we can use list
      const res = await fetch(`/api/node-types/${typeId}/nodes?${params.toString()}`, {
        headers: {
          'X-Tenant-ID': tenant.id,
        },
      });

      if (res.status === 404) {
        // If the specific endpoint doesn't exist, try fetching all nodes and filtering (fallback)
        const fallbackRes = await fetch(`/api/glossary/semantic-terms?tenant_id=${tenant.id}`, {
          headers: { 'X-Tenant-ID': tenant.id }
        });
        if (!fallbackRes.ok) return [];
        const allNodes: CatalogNode[] = await fallbackRes.json();
        return allNodes.filter(n => n.node_type_id === typeId);
      }

      if (!res.ok) {
        throw new Error('Failed to fetch nodes');
      }

      const data = await res.json();
      return data.data || data;
    },
    enabled: !!tenant?.id && !!typeId,
  });
}
// Create a new node type
export function useCreateNodeType() {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (newNodeType: Partial<NodeType>) => {
      if (!tenant?.id) throw new Error('Tenant ID is required');

      const res = await fetch('/api/node-types', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenant.id,
        },
        body: JSON.stringify(newNodeType),
      });

      if (!res.ok) {
        throw new Error('Failed to create node type');
      }

      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: nodeTypesKeys.lists() });
    },
  });
}

// Update an existing node type
export function useUpdateNodeType() {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, ...updates }: Partial<NodeType> & { id: string }) => {
      if (!tenant?.id) throw new Error('Tenant ID is required');

      const res = await fetch(`/api/node-types/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenant.id,
        },
        body: JSON.stringify(updates),
      });

      if (!res.ok) {
        throw new Error('Failed to update node type');
      }

      return res.json();
    },
    onSuccess: (data: NodeType, variables: Partial<NodeType> & { id: string }) => {
      queryClient.invalidateQueries({ queryKey: nodeTypesKeys.lists() });
      queryClient.invalidateQueries({ queryKey: nodeTypesKeys.detail(variables.id) });
    },
  });
}

// Delete a node type
export function useDeleteNodeType() {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      if (!tenant?.id) throw new Error('Tenant ID is required');

      const res = await fetch(`/api/node-types/${id}`, {
        method: 'DELETE',
        headers: {
          'X-Tenant-ID': tenant.id,
        },
      });

      if (!res.ok) {
        throw new Error('Failed to delete node type');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: nodeTypesKeys.lists() });
    },
  });
}

// Alias for search specific use case
export const useSearchNodeTypes = useNodeTypes;
