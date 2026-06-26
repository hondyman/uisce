// React Query hooks for tenant management
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { Tenant, CreateTenantRequest, UpdateTenantRequest } from "../types";

// ============================================================================
// QUERIES
// ============================================================================

export function useTenants() {
  return useQuery({
    queryKey: ["tenants"],
    queryFn: () => 
      api<{ tenants: Tenant[] }>("/admin/tenants").then(res => res.tenants ? { tenants: res.tenants } : res)
  });
}

export function useTenant(id: string | undefined) {
  return useQuery({
    queryKey: ["tenant", id],
    queryFn: () => 
      api<{ tenant: Tenant }>(`/admin/tenants/${id}`).then(res => res.tenant ? { tenant: res.tenant } : res),
    enabled: !!id
  });
}

// ============================================================================
// MUTATIONS
// ============================================================================

export function useCreateTenant() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (body: CreateTenantRequest) =>
      api<{ tenant: Tenant }>("/admin/tenants", {
        method: "POST",
        body: JSON.stringify(body)
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["tenants"] });
    }
  });
}

export function useUpdateTenant(id: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (body: UpdateTenantRequest) =>
      api<{ tenant: Tenant }>(`/admin/tenants/${id}`, {
        method: "PATCH",
        body: JSON.stringify(body)
      }),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ["tenants"] });
      qc.invalidateQueries({ queryKey: ["tenant", id] });
      if (data.tenant) {
        qc.setQueryData(["tenant", id], { tenant: data.tenant });
      }
    }
  });
}

export function useSuspendTenant(id: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: () =>
      api(`/admin/tenants/${id}/suspend`, {
        method: "POST"
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["tenants"] });
      qc.invalidateQueries({ queryKey: ["tenant", id] });
    }
  });
}

export function useUnsuspendTenant(id: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: () =>
      api(`/admin/tenants/${id}/unsuspend`, {
        method: "POST"
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["tenants"] });
      qc.invalidateQueries({ queryKey: ["tenant", id] });
    }
  });
}

export function useDeleteTenant(id: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: () =>
      api(`/admin/tenants/${id}`, {
        method: "DELETE"
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["tenants"] });
    }
  });
}
