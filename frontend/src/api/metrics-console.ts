/**
 * Metrics Console - API Client
 * Axios instance with tenant-aware headers and all CRUD/compute endpoints
 */

import metricsApi from '../utils/axiosClient';
import {
  MetricRegistry,
  PopRow,
  AnomalyRow,
  JobRun,
  UUID,
  CreateMetricRequest,
  UpdateMetricRequest,
  ComputePopRequest,
  DetectAnomaliesRequest,
} from '../types/metrics-console';
export { metricsApi };

// ============ Registry CRUD ============

export async function listMetrics(params?: {
  q?: string;
  domain?: string;
  golden?: boolean;
  status?: string;
  limit?: number;
  offset?: number;
}) {
  const { data } = await metricsApi.get<MetricRegistry[]>('/api/metrics/definitions', { params });
  return data;
}

export async function getMetric(id: UUID) {
  const { data } = await metricsApi.get<MetricRegistry>(`/api/metrics/definitions/${id}`);
  return data;
}

export async function createMetric(input: CreateMetricRequest) {
  const { data } = await metricsApi.post<MetricRegistry>('/api/metrics/definitions', input);
  return data;
}

export async function updateMetric(id: UUID, input: UpdateMetricRequest) {
  const { data } = await metricsApi.put<MetricRegistry>(`/api/metrics/definitions/${id}`, input);
  return data;
}

export async function deleteMetric(id: UUID) {
  await metricsApi.delete(`/api/metrics/definitions/${id}`);
}

// ============ PoP and Anomalies ============

export async function getPop(metric_id: UUID, params?: { from?: string; to?: string }) {
  const { data } = await metricsApi.get<PopRow[]>(`/api/pop/metrics/${metric_id}`, { params });
  return data;
}

export async function getAnomalies(
  metric_id: UUID,
  params?: { from?: string; to?: string; status?: string }
) {
  const { data } = await metricsApi.get<AnomalyRow[]>(`/api/pop/anomalies/${metric_id}`, { params });
  return data;
}

// ============ Compute Triggers (Temporal-backed) ============

export async function triggerPop(metric_id: UUID, body: ComputePopRequest) {
  const { data } = await metricsApi.post(`/api/pop/metrics/${metric_id}/analyze-pop`, body);
  return data;
}

export async function triggerAnomaly(metric_id: UUID, body: DetectAnomaliesRequest) {
  const { data } = await metricsApi.post(`/api/pop/metrics/${metric_id}/analyze`, body);
  return data;
}

// ============ Job Runs & Monitoring ============

export async function listRuns(metric_id?: UUID, params?: { calc_type?: string; status?: string }) {
  const { data } = await metricsApi.get<JobRun[]>('/api/runs', {
    params: { metric_id, ...params },
  });
  return data;
}

export async function getRun(run_id: UUID) {
  const { data } = await metricsApi.get<JobRun>(`/api/runs/${run_id}`);
  return data;
}

// ============ SLA & Quality ============

export async function getGoldenPathReadiness() {
  const { data } = await metricsApi.get('/api/metrics/golden-path/readiness');
  return data;
}

export async function promoteToGoldenPath(metric_id: UUID) {
  const { data } = await metricsApi.post(`/api/metrics/${metric_id}/promote-golden`, {});
  return data;
}
