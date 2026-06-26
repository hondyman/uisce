import apiClient from '@/utils/apiClient';

const API_BASE = '/cube';

// Types
export interface WorkerPool {
  id: string;
  name: string;
  display_name: string;
  description: string;
  tier: 'standard' | 'professional' | 'enterprise' | 'priority';
  min_workers: number;
  max_workers: number;
  current_workers: number;
  target_workers: number;
  memory_limit_mb: number;
  cpu_limit_cores: number;
  concurrent_jobs: number;
  queue_size: number;
  auto_scale_enabled: boolean;
  scale_up_threshold: number;
  scale_down_threshold: number;
  scale_cooldown_seconds: number;
  status: 'active' | 'scaling' | 'degraded' | 'inactive';
  last_scale_at: string | null;
  health_check_at: string | null;
  metadata?: Record<string, unknown>;
}

export interface WorkerInstance {
  id: string;
  pool_id: string;
  instance_id: string;
  hostname: string;
  ip_address: string;
  status: 'starting' | 'idle' | 'busy' | 'unhealthy' | 'draining' | 'terminated';
  current_job_id: string | null;
  jobs_completed: number;
  jobs_failed: number;
  memory_used_mb: number;
  cpu_used_percent: number;
  started_at: string;
  last_heartbeat_at: string;
  last_job_at: string | null;
  metadata?: Record<string, unknown>;
}

export interface PreAggDefinition {
  id: string;
  tenant_id: string;
  tenant_instance_id: string;
  cube_name: string;
  preagg_name: string;
  measures: string[];
  dimensions: string[];
  time_dimension: string;
  granularity: string;
  partition_granularity: string;
  refresh_key: Record<string, unknown>;
  scheduled_refresh: boolean;
  refresh_cron: string;
  refresh_interval_minutes: number;
  refresh_timezone: string;
  external_storage: boolean;
  storage_engine: string;
  table_name: string;
  indexes: Record<string, unknown>;
  build_range_start: string | null;
  build_range_end: string | null;
  priority: number;
  worker_pool_id: string | null;
  status: string;
  last_build_at: string | null;
  last_build_duration_ms: number | null;
  last_build_rows: number | null;
  last_error: string;
  yaml_definition: string;
}

export interface PreAggJob {
  id: string;
  preagg_id: string;
  tenant_id: string;
  tenant_instance_id: string;
  job_type: 'full_build' | 'incremental' | 'partition_build' | 'rebuild';
  partition_key: string;
  priority: number;
  worker_pool_id: string | null;
  assigned_worker_id: string | null;
  status: 'pending' | 'queued' | 'running' | 'completed' | 'failed' | 'cancelled';
  progress_percent: number;
  current_step: string;
  scheduled_at: string;
  queued_at: string | null;
  started_at: string | null;
  completed_at: string | null;
  timeout_at: string | null;
  rows_processed: number;
  bytes_written: number;
  duration_ms: number | null;
  retry_count: number;
  max_retries: number;
  error_message: string;
  error_stack: string;
  build_options: Record<string, unknown>;
  result_metadata: Record<string, unknown>;
}

export interface PreAggPartition {
  id: string;
  preagg_id: string;
  partition_key: string;
  status: string;
  table_name: string;
  row_count: number;
  size_bytes: number;
  data_from: string | null;
  data_to: string | null;
  built_at: string | null;
  expires_at: string | null;
  refresh_key_value: string;
  build_duration_ms: number | null;
  last_error: string;
}

export interface QueueStats {
  pending: number;
  queued: number;
  running: number;
  completed_1h: number;
  failed_1h: number;
  avg_duration_ms: number;
}

export interface PoolHealthSummary {
  id: string;
  name: string;
  display_name: string;
  tier: string;
  status: string;
  current_workers: number;
  target_workers: number;
  idle_workers: number;
  busy_workers: number;
  unhealthy_workers: number;
  avg_cpu_percent: number;
  avg_memory_mb: number;
  pending_jobs: number;
  running_jobs: number;
  total_jobs_completed: number;
  total_jobs_failed: number;
}

// API Functions

export async function listWorkerPools(): Promise<WorkerPool[]> {
  const response = await apiClient(`${API_BASE}/worker-pools`);
  if (!response.ok) throw new Error('Failed to list worker pools');
  return response.json();
}

export async function getWorkerPool(poolId: string): Promise<WorkerPool> {
  const response = await apiClient(`${API_BASE}/worker-pools/${poolId}`);
  if (!response.ok) throw new Error('Failed to get worker pool');
  return response.json();
}

export async function scaleWorkerPool(poolId: string, targetWorkers: number): Promise<void> {
  const response = await apiClient(`${API_BASE}/worker-pools/${poolId}/scale`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ target_workers: targetWorkers }),
  });
  if (!response.ok) throw new Error('Failed to scale worker pool');
}

export async function listWorkerInstances(poolId: string): Promise<WorkerInstance[]> {
  const response = await apiClient(`${API_BASE}/worker-pools/${poolId}/workers`);
  if (!response.ok) throw new Error('Failed to list workers');
  return response.json();
}

export async function listPreAggDefinitions(
  tenantId: string,
  datasourceId: string
): Promise<PreAggDefinition[]> {
  const params = new URLSearchParams({ tenant_id: tenantId, tenant_instance_id: datasourceId });
  const response = await apiClient(`${API_BASE}/preagg-definitions?${params}`);
  if (!response.ok) throw new Error('Failed to list pre-aggregations');
  return response.json();
}

export async function createPreAggDefinition(
  definition: Partial<PreAggDefinition>
): Promise<PreAggDefinition> {
  const response = await apiClient(`${API_BASE}/preagg-definitions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(definition),
  });
  if (!response.ok) throw new Error('Failed to create pre-aggregation');
  return response.json();
}

export async function getPreAggDefinition(defId: string): Promise<PreAggDefinition> {
  const response = await apiClient(`${API_BASE}/preagg-definitions/${defId}`);
  if (!response.ok) throw new Error('Failed to get pre-aggregation');
  return response.json();
}

export async function updatePreAggDefinition(
  defId: string,
  updates: Partial<PreAggDefinition>
): Promise<PreAggDefinition> {
  const response = await apiClient(`${API_BASE}/preagg-definitions/${defId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  });
  if (!response.ok) throw new Error('Failed to update pre-aggregation');
  return response.json();
}

export async function deletePreAggDefinition(defId: string): Promise<void> {
  const response = await apiClient(`${API_BASE}/preagg-definitions/${defId}`, {
    method: 'DELETE',
  });
  if (!response.ok) throw new Error('Failed to delete pre-aggregation');
}

export async function listPartitions(defId: string): Promise<PreAggPartition[]> {
  const response = await apiClient(`${API_BASE}/preagg-definitions/${defId}/partitions`);
  if (!response.ok) throw new Error('Failed to list partitions');
  return response.json();
}

export async function triggerBuild(
  defId: string,
  options: {
    job_type?: string;
    partition_key?: string;
    priority?: number;
    options?: Record<string, unknown>;
  },
  tenantId: string,
  datasourceId: string
): Promise<PreAggJob> {
  const params = new URLSearchParams({ tenant_id: tenantId, tenant_instance_id: datasourceId });
  const response = await apiClient(`${API_BASE}/preagg-definitions/${defId}/build?${params}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(options),
  });
  if (!response.ok) throw new Error('Failed to trigger build');
  return response.json();
}

export async function listJobs(
  tenantId?: string,
  status?: string,
  limit?: number
): Promise<PreAggJob[]> {
  const params = new URLSearchParams();
  if (tenantId) params.append('tenant_id', tenantId);
  if (status) params.append('status', status);
  if (limit) params.append('limit', limit.toString());

  const response = await apiClient(`${API_BASE}/jobs?${params}`);
  if (!response.ok) throw new Error('Failed to list jobs');
  return response.json();
}

export async function getJobQueueStats(): Promise<QueueStats> {
  const response = await apiClient(`${API_BASE}/jobs/stats`);
  if (!response.ok) throw new Error('Failed to get queue stats');
  return response.json();
}

export async function getJob(jobId: string): Promise<PreAggJob> {
  const response = await apiClient(`${API_BASE}/jobs/${jobId}`);
  if (!response.ok) throw new Error('Failed to get job');
  return response.json();
}

export async function cancelJob(jobId: string): Promise<void> {
  const response = await apiClient(`${API_BASE}/jobs/${jobId}/cancel`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error('Failed to cancel job');
}

export async function retryJob(jobId: string): Promise<void> {
  const response = await apiClient(`${API_BASE}/jobs/${jobId}/retry`, {
    method: 'POST',
  });
  if (!response.ok) throw new Error('Failed to retry job');
}

// Worker Registration APIs (for worker instances)

export async function registerWorker(
  poolId: string,
  instanceId: string,
  hostname: string,
  ipAddress: string
): Promise<WorkerInstance> {
  const response = await apiClient(`${API_BASE}/workers/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      pool_id: poolId,
      instance_id: instanceId,
      hostname,
      ip_address: ipAddress,
    }),
  });
  if (!response.ok) throw new Error('Failed to register worker');
  return response.json();
}

export async function sendHeartbeat(
  workerId: string,
  status: string,
  memoryMb: number,
  cpuPercent: number
): Promise<void> {
  const response = await apiClient(`${API_BASE}/workers/${workerId}/heartbeat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      status,
      memory_mb: memoryMb,
      cpu_percent: cpuPercent,
    }),
  });
  if (!response.ok) throw new Error('Failed to send heartbeat');
}

export async function deregisterWorker(workerId: string): Promise<void> {
  const response = await apiClient(`${API_BASE}/workers/${workerId}`, {
    method: 'DELETE',
  });
  if (!response.ok) throw new Error('Failed to deregister worker');
}

export async function claimJob(
  workerId: string,
  poolId?: string
): Promise<PreAggJob | null> {
  const response = await apiClient(`${API_BASE}/workers/${workerId}/claim-job`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ pool_id: poolId }),
  });
  if (response.status === 204) return null;
  if (!response.ok) throw new Error('Failed to claim job');
  return response.json();
}

export async function completeJob(
  workerId: string,
  jobId: string,
  rowsProcessed: number,
  bytesWritten: number,
  metadata?: Record<string, unknown>
): Promise<void> {
  const response = await apiClient(`${API_BASE}/workers/${workerId}/complete-job/${jobId}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      rows_processed: rowsProcessed,
      bytes_written: bytesWritten,
      metadata,
    }),
  });
  if (!response.ok) throw new Error('Failed to complete job');
}

export async function failJob(
  workerId: string,
  jobId: string,
  errorMessage: string,
  errorStack?: string
): Promise<void> {
  const response = await apiClient(`${API_BASE}/workers/${workerId}/fail-job/${jobId}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      error_message: errorMessage,
      error_stack: errorStack,
    }),
  });
  if (!response.ok) throw new Error('Failed to fail job');
}

// Utility functions

export function formatDuration(ms: number | null): string {
  if (ms === null || ms === undefined) return '-';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${(ms / 60000).toFixed(1)}m`;
  return `${(ms / 3600000).toFixed(1)}h`;
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function getStatusColor(
  status: string
): 'success' | 'error' | 'warning' | 'info' | 'default' {
  switch (status) {
    case 'active':
    case 'completed':
    case 'healthy':
    case 'idle':
      return 'success';
    case 'running':
    case 'processing':
    case 'busy':
      return 'info';
    case 'pending':
    case 'queued':
    case 'starting':
    case 'scaling':
    case 'draining':
      return 'warning';
    case 'failed':
    case 'error':
    case 'unhealthy':
    case 'degraded':
    case 'terminated':
      return 'error';
    default:
      return 'default';
  }
}

export function getTierColor(tier: string): string {
  switch (tier) {
    case 'enterprise':
      return '#9c27b0'; // Purple
    case 'professional':
      return '#2196f3'; // Blue
    case 'priority':
      return '#ff9800'; // Orange
    case 'standard':
    default:
      return '#4caf50'; // Green
  }
}
