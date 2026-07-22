import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../lib/apiClient';

export interface FabricDefn {
  id: string;
  tenant_id: string;
  tenant_datasource_id: string;
  model_key: string;
  version: number;
  title?: string;
  description?: string;
  source_config?: any;
  resolved_config?: any;
  status: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  checksum_sha256?: string;
}

export interface CreateFabricDefnRequest {
  model_key: string;
  version: number;
  title?: string;
  description?: string;
  source_config?: any;
  resolved_config?: any;
}

export const fabricKeys = {
  all: ['fabric-defns'] as const,
  lists: () => [...fabricKeys.all, 'list'] as const,
  list: (tenantId: string, datasourceId?: string) => [...fabricKeys.lists(), tenantId, datasourceId ?? 'all'] as const,
  details: () => [...fabricKeys.all, 'detail'] as const,
  detail: (id: string) => [...fabricKeys.details(), id] as const,
};

export function useFabricDefns(tenantId: string, datasourceId?: string) {
  return useQuery({
    queryKey: fabricKeys.list(tenantId, datasourceId),
    queryFn: async () => {
      let url = `/api/rest/fabric-defns?tenant_id=${encodeURIComponent(tenantId)}`;
      if (datasourceId) {
        url += `&datasource_id=${encodeURIComponent(datasourceId)}`;
      }
      const res = await apiFetch(url);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<FabricDefn[]>;
    },
    enabled: !!tenantId,
  });
}

export function useFabricDefn(id: string) {
  return useQuery({
    queryKey: fabricKeys.detail(id),
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/fabric-defns/${id}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      const data = await res.json();
      return Array.isArray(data) ? data[0] : data;
    },
    enabled: !!id,
  });
}

export function useCreateFabricDefn() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateFabricDefnRequest) => {
      const res = await apiFetch('/api/rest/fabric-defn', {
        method: 'POST',
        body: JSON.stringify({ input }),
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: fabricKeys.lists() });
    },
  });
}
