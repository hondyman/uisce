import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../lib/apiClient';

export interface Tenant {
  id: string;
  display_name: string;
  description?: string;
  is_active: boolean;
  gold_copy: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateTenantRequest {
  display_name: string;
  description?: string;
  is_active: boolean;
}

export interface UpdateTenantRequest {
  display_name?: string;
  description?: string;
  is_active?: boolean;
}

export const tenantKeys = {
  all: ['tenants'] as const,
  lists: () => [...tenantKeys.all, 'list'] as const,
  list: () => [...tenantKeys.lists()] as const,
  details: () => [...tenantKeys.all, 'detail'] as const,
  detail: (id: string) => [...tenantKeys.details(), id] as const,
};

export function useTenants() {
  return useQuery({
    queryKey: tenantKeys.list(),
    queryFn: async () => {
      const res = await apiFetch('/api/rest/tenants');
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<Tenant[]>;
    },
  });
}

export function useTenant(id: string) {
  return useQuery({
    queryKey: tenantKeys.detail(id),
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/tenants/${id}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      const data = await res.json();
      return Array.isArray(data) ? data[0] : data;
    },
    enabled: !!id,
  });
}

export function useCreateTenant() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateTenantRequest) => {
      const res = await apiFetch('/api/rest/tenants', {
        method: 'POST',
        body: JSON.stringify(input),
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tenantKeys.lists() });
    },
  });
}

export function useUpdateTenant() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: UpdateTenantRequest }) => {
      const res = await apiFetch(`/api/rest/tenants/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: tenantKeys.detail(vars.id) });
      queryClient.invalidateQueries({ queryKey: tenantKeys.lists() });
    },
  });
}

export function useDeleteTenant() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await apiFetch(`/api/rest/tenants/${id}`, {
        method: 'DELETE',
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: tenantKeys.lists() });
    },
  });
}
