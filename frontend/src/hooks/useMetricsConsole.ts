/**
 * Metrics Console - TanStack Query Hooks
 * Typed data access, caching, and mutation flows for metric CRUD and compute operations
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listMetrics,
  getMetric,
  createMetric,
  updateMetric,
  deleteMetric,
  getPop,
  getAnomalies,
  listRuns,
  triggerPop,
  triggerAnomaly,
  getGoldenPathReadiness,
  promoteToGoldenPath,
  getRun,
} from '../api/metrics-console';
import {
  UUID,
  CreateMetricRequest,
  UpdateMetricRequest,
  ComputePopRequest,
  DetectAnomaliesRequest,
} from '../types/metrics-console';

/**
 * List all metrics with optional filters
 */
export const useMetrics = (filters?: {
  q?: string;
  domain?: string;
  golden?: boolean;
  status?: string;
  limit?: number;
  offset?: number;
}) =>
  useQuery({
    queryKey: ['metrics', filters],
    queryFn: () => listMetrics(filters),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

/**
 * Get single metric definition
 */
export const useMetric = (metric_id?: UUID) =>
  useQuery({
    queryKey: ['metric', metric_id],
    queryFn: () => getMetric(metric_id!),
    enabled: !!metric_id,
    staleTime: 5 * 60 * 1000,
  });

/**
 * Get PoP (Period-over-Period) results
 */
export const usePop = (metric_id?: UUID, range?: { from?: string; to?: string }) =>
  useQuery({
    queryKey: ['pop', metric_id, range],
    queryFn: () => getPop(metric_id!, range),
    enabled: !!metric_id,
    staleTime: 2 * 60 * 1000, // 2 minutes
  });

/**
 * Get anomalies detected for a metric
 */
export const useAnomalies = (
  metric_id?: UUID,
  range?: { from?: string; to?: string; status?: string }
) =>
  useQuery({
    queryKey: ['anomalies', metric_id, range],
    queryFn: () => getAnomalies(metric_id!, range),
    enabled: !!metric_id,
    staleTime: 2 * 60 * 1000,
  });

/**
 * Get job runs (Temporal workflows)
 */
export const useRuns = (metric_id?: UUID, params?: { calc_type?: string; status?: string }) =>
  useQuery({
    queryKey: ['runs', metric_id, params],
    queryFn: () => listRuns(metric_id, params),
    staleTime: 1 * 60 * 1000, // 1 minute
  });

/**
 * Get single run details
 */
export const useRun = (run_id?: UUID) =>
  useQuery({
    queryKey: ['run', run_id],
    queryFn: () => getRun(run_id!),
    enabled: !!run_id,
    staleTime: 2 * 60 * 1000,
  });

/**
 * Get golden path readiness (SLA health)
 */
export const useGoldenPathReadiness = () =>
  useQuery({
    queryKey: ['golden-path-readiness'],
    queryFn: () => getGoldenPathReadiness(),
    staleTime: 10 * 60 * 1000, // 10 minutes
  });

// ============ Mutations ============

/**
 * Create new metric
 */
export const useCreateMetric = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: createMetric,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['metrics'] });
    },
  });
};

/**
 * Update metric definition
 */
export const useUpdateMetric = (metric_id: UUID) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateMetricRequest) => updateMetric(metric_id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['metric', metric_id] });
      qc.invalidateQueries({ queryKey: ['metrics'] });
    },
  });
};

/**
 * Delete metric
 */
export const useDeleteMetric = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: deleteMetric,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['metrics'] });
    },
  });
};

/**
 * Trigger PoP computation (real-time lane)
 */
export const useTriggerPop = (metric_id: UUID) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: ComputePopRequest) => triggerPop(metric_id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['pop', metric_id] });
      qc.invalidateQueries({ queryKey: ['runs', metric_id] });
    },
  });
};

/**
 * Trigger anomaly detection
 */
export const useTriggerAnomaly = (metric_id: UUID) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: DetectAnomaliesRequest) => triggerAnomaly(metric_id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['anomalies', metric_id] });
      qc.invalidateQueries({ queryKey: ['runs', metric_id] });
    },
  });
};

/**
 * Promote metric to golden path
 */
export const usePromoteToGoldenPath = (metric_id: UUID) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => promoteToGoldenPath(metric_id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['metric', metric_id] });
      qc.invalidateQueries({ queryKey: ['metrics'] });
      qc.invalidateQueries({ queryKey: ['golden-path-readiness'] });
    },
  });
};
