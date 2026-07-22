import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from '../lib/apiClient';

export interface Policy {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  policy_type: string;
  config: any;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreatePolicyRequest {
  name: string;
  description?: string;
  policy_type: string;
  config?: any;
  is_active?: boolean;
}

export interface UpdatePolicyRequest {
  name?: string;
  description?: string;
  config?: any;
  is_active?: boolean;
}

export const policyKeys = {
  all: ['policies'] as const,
  lists: () => [...policyKeys.all, 'list'] as const,
  list: (tenantId: string) => [...policyKeys.lists(), tenantId] as const,
  details: () => [...policyKeys.all, 'detail'] as const,
  detail: (id: string) => [...policyKeys.details(), id] as const,
  simulations: (tenantId: string) => [...policyKeys.all, 'simulations', tenantId] as const,
  driftRuns: (tenantId: string) => [...policyKeys.all, 'drift-runs', tenantId] as const,
};

export function usePolicies(tenantId: string) {
  return useQuery({
    queryKey: policyKeys.list(tenantId),
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/policies?tenant_id=${encodeURIComponent(tenantId)}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<Policy[]>;
    },
    enabled: !!tenantId,
  });
}

export function usePolicy(id: string) {
  return useQuery({
    queryKey: policyKeys.detail(id),
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/policies/${id}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      const data = await res.json();
      return Array.isArray(data) ? data[0] : data;
    },
    enabled: !!id,
  });
}

export function useCreatePolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreatePolicyRequest) => {
      const res = await apiFetch('/api/rest/policies', {
        method: 'POST',
        body: JSON.stringify(input),
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: policyKeys.lists() });
    },
  });
}

export function useUpdatePolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: UpdatePolicyRequest }) => {
      const res = await apiFetch(`/api/rest/policies/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: policyKeys.detail(vars.id) });
      queryClient.invalidateQueries({ queryKey: policyKeys.lists() });
    },
  });
}

export function useDeletePolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const res = await apiFetch(`/api/rest/policies/${id}`, {
        method: 'DELETE',
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: policyKeys.lists() });
    },
  });
}

export function usePolicySimulations(tenantId: string) {
  return useQuery({
    queryKey: policyKeys.simulations(tenantId),
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/policy-simulations?tenant_id=${encodeURIComponent(tenantId)}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
    enabled: !!tenantId,
  });
}

export function useDriftRuns(tenantId: string) {
  return useQuery({
    queryKey: policyKeys.driftRuns(tenantId),
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/drift-runs?tenant_id=${encodeURIComponent(tenantId)}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
    enabled: !!tenantId,
  });
}
