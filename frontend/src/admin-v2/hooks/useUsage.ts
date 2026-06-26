// React Query hooks for usage analytics
import { useQuery } from "@tanstack/react-query";
import { api } from "../api";
import {
  DailyUsage,
  APIKeyUsage,
  LatencyPoint,
  ErrorPoint,
  UsagePoint,
  EndpointUsage,
  TopTenant,
  RecentError
} from "../types";

// ============================================================================
// TENANT USAGE
// ============================================================================

export function useTenantDailyUsage(id: string | undefined, days = 30) {
  return useQuery({
    queryKey: ["tenantDailyUsage", id, days],
    queryFn: () =>
      api<{ data: DailyUsage[] }>(
        `/admin/tenants/${id}/usage/daily?days=${days}`
      ),
    enabled: !!id
  });
}

export function useTenantEndpointUsage(id: string | undefined, limit = 20) {
  return useQuery({
    queryKey: ["tenantEndpointUsage", id, limit],
    queryFn: () =>
      api<{ top_endpoints: EndpointUsage[] }>(
        `/admin/tenants/${id}/usage/endpoints?limit=${limit}`
      ),
    enabled: !!id
  });
}

export function useTenantRecentRequests(id: string | undefined, limit = 100) {
  return useQuery({
    queryKey: ["tenantRecentRequests", id, limit],
    queryFn: () =>
      api<{ requests: APIKeyUsage[] }>(
        `/admin/tenants/${id}/usage/recent?limit=${limit}`
      ),
    enabled: !!id
  });
}

// ============================================================================
// GLOBAL OPS
// ============================================================================

export function useGlobalUsage() {
  return useQuery({
    queryKey: ["globalUsage"],
    queryFn: () => api<{ data: UsagePoint[] }>("/admin/usage/global"),
    refetchInterval: 1000 * 60 // Refetch every minute
  });
}

export function useGlobalErrors() {
  return useQuery({
    queryKey: ["globalErrors"],
    queryFn: () => api<{ data: ErrorPoint[] }>("/admin/errors/global"),
    refetchInterval: 1000 * 60
  });
}

export function useGlobalLatency() {
  return useQuery({
    queryKey: ["globalLatency"],
    queryFn: () => api<{ data: LatencyPoint[] }>("/admin/latency/global"),
    refetchInterval: 1000 * 60
  });
}

export function useTopTenants(limit = 10) {
  return useQuery({
    queryKey: ["topTenants", limit],
    queryFn: () =>
      api<{ tenants: TopTenant[] }>(
        `/admin/tenants/top?limit=${limit}`
      ),
    refetchInterval: 1000 * 60
  });
}

export function useTopEndpoints(limit = 10) {
  return useQuery({
    queryKey: ["topEndpoints", limit],
    queryFn: () =>
      api<{ endpoints: EndpointUsage[] }>(
        `/admin/endpoints/top?limit=${limit}`
      ),
    refetchInterval: 1000 * 60
  });
}

export function useRecentErrors(limit = 50) {
  return useQuery({
    queryKey: ["recentErrors", limit],
    queryFn: () =>
      api<{ errors: RecentError[] }>(
        `/admin/errors/recent?limit=${limit}`
      ),
    refetchInterval: 1000 * 60
  });
}
