import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api';
import type { Alert, AlertEvent, TenantHealth, EndpointHealth, Heatmap, ErrorFingerprint, ErrorEvent } from '../types';

// ========== Alerts ==========

export interface ListAlertsResponse {
  data: Alert[];
}

export interface AlertEventsResponse {
  data: AlertEvent[];
}

export function useAlerts(enabled: boolean = true) {
  return useQuery({
    queryKey: ['alerts'],
    queryFn: () => api<ListAlertsResponse>('/admin/alerts'),
    staleTime: 5 * 60 * 1000,
    enabled,
  });
}

export function useAlert(alertId: string | null) {
  return useQuery({
    queryKey: ['alert', alertId],
    queryFn: () => api<{ data: Alert }>(`/admin/alerts/${alertId}`),
    staleTime: 5 * 60 * 1000,
    enabled: !!alertId,
  });
}

export function useAlertEvents(alertId: string | null, limit: number = 100) {
  return useQuery({
    queryKey: ['alertEvents', alertId, limit],
    queryFn: () => api<AlertEventsResponse>(`/admin/alerts/${alertId}/events?limit=${limit}`),
    staleTime: 1 * 60 * 1000,
    enabled: !!alertId,
  });
}

export function useCreateAlert() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<Alert, 'id' | 'created_at' | 'updated_at'>) =>
      api<{ data: Alert }>('/admin/alerts', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] });
    },
  });
}

export function useUpdateAlert(alertId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: Omit<Alert, 'id' | 'created_at' | 'updated_at'>) =>
      api<{ data: Alert }>(`/admin/alerts/${alertId}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] });
      queryClient.invalidateQueries({ queryKey: ['alert', alertId] });
    },
  });
}

export function useDeleteAlert() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (alertId: string) =>
      api(`/admin/alerts/${alertId}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] });
    },
  });
}

export function useEvaluateAlerts() {
  return useMutation({
    mutationFn: () =>
      api('/admin/alerts/evaluate', { method: 'POST' }),
  });
}

// ========== Tenant Health ==========

export interface TenantHealthResponse {
  data: TenantHealth;
}

export function useTenantHealth(tenantId: string | null, windowSecs: number = 3600) {
  return useQuery({
    queryKey: ['tenantHealth', tenantId, windowSecs],
    queryFn: () => {
      const window = `${Math.floor(windowSecs / 60)}m`;
      return api<TenantHealthResponse>(`/admin/tenants/${tenantId}/health?window=${window}`);
    },
    staleTime: 5 * 60 * 1000,
    refetchInterval: 1 * 60 * 1000, // Refresh every minute
    enabled: !!tenantId,
  });
}

// ========== Endpoint Health ==========

export interface EndpointHealthListResponse {
  data: EndpointHealth[];
}

export interface EndpointHealthResponse {
  data: EndpointHealth;
}

export function useEndpointHealthList(limit: number = 50) {
  return useQuery({
    queryKey: ['endpointHealthList', limit],
    queryFn: () => api<EndpointHealthListResponse>(`/admin/endpoints/health?limit=${limit}`),
    staleTime: 5 * 60 * 1000,
    refetchInterval: 2 * 60 * 1000, // Refresh every 2 minutes
  });
}

export function useEndpointHealth(endpoint: string | null, windowSecs: number = 1800) {
  return useQuery({
    queryKey: ['endpointHealth', endpoint, windowSecs],
    queryFn: () => {
      const window = `${Math.floor(windowSecs / 60)}m`;
      return api<EndpointHealthResponse>(`/admin/endpoints/${endpoint}/health?window=${window}`);
    },
    staleTime: 5 * 60 * 1000,
    refetchInterval: 1 * 60 * 1000,
    enabled: !!endpoint,
  });
}

// ========== Latency Heatmaps ==========

export interface HeatmapResponse {
  data: Heatmap;
}

export function useLatencyHeatmap(groupBy: 'region' | 'tenant' | 'endpoint' = 'region') {
  return useQuery({
    queryKey: ['latencyHeatmap', groupBy],
    queryFn: () => {
      const url = {
        region: '/admin/latency/heatmap/regions',
        tenant: '/admin/latency/heatmap/tenants',
        endpoint: '/admin/latency/heatmap/endpoints',
      }[groupBy];
      return api<HeatmapResponse>(url);
    },
    staleTime: 2 * 60 * 1000,
    refetchInterval: 5 * 60 * 1000, // Refresh every 5 minutes
  });
}

export function useRegionHeatmap() {
  return useLatencyHeatmap('region');
}

export function useTenantHeatmap() {
  return useLatencyHeatmap('tenant');
}

export function useEndpointHeatmap() {
  return useLatencyHeatmap('endpoint');
}

// ========== Error Fingerprints ==========

export interface FingerprintListResponse {
  data: ErrorFingerprint[];
}

export interface FingerprintHistoryResponse {
  data: ErrorEvent[];
}

export function useErrorFingerprints(limit: number = 50) {
  return useQuery({
    queryKey: ['errorFingerprints', limit],
    queryFn: () => api<FingerprintListResponse>(`/admin/errors/fingerprints?limit=${limit}`),
    staleTime: 2 * 60 * 1000,
    refetchInterval: 5 * 60 * 1000,
  });
}

export function useErrorFingerprintHistory(fingerprintId: string | null, limit: number = 100) {
  return useQuery({
    queryKey: ['fingerprintHistory', fingerprintId, limit],
    queryFn: () => api<FingerprintHistoryResponse>(`/admin/errors/fingerprints/${fingerprintId}?limit=${limit}`),
    staleTime: 1 * 60 * 1000,
    refetchInterval: 2 * 60 * 1000,
    enabled: !!fingerprintId,
  });
}
