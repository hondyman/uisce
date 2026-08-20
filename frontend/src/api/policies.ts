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
  list: () => [...policyKeys.lists()] as const,
  details: () => [...policyKeys.all, 'detail'] as const,
  detail: (id: string) => [...policyKeys.details(), id] as const,
  simulations: () => [...policyKeys.all, 'simulations'] as const,
  driftRuns: () => [...policyKeys.all, 'drift-runs'] as const,
};

export function usePolicies() {
  return useQuery({
    queryKey: policyKeys.list(),
    queryFn: async () => {
      const res = await apiFetch('/api/rest/policies');
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<Policy[]>;
    },
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

export function usePolicySimulations() {
  return useQuery({
    queryKey: policyKeys.simulations(),
    queryFn: async () => {
      const res = await apiFetch('/api/rest/policy-simulations');
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
  });
}

export function useDriftRuns() {
  return useQuery({
    queryKey: policyKeys.driftRuns(),
    queryFn: async () => {
      const res = await apiFetch('/api/rest/drift-runs');
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json();
    },
  });
}
