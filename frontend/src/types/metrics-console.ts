/**
 * Metrics Console - Core TypeScript Types
 * Mirrors backend schema for metric registry, PoP results, anomalies, and job runs
 */

export type UUID = string;

// ============ Metric Registry ============

export interface MetricRegistry {
  tenant_id: UUID;
  id: UUID;
  name: string;
  display_name?: string;
  description?: string;
  domain: string;
  category?: string;
  granularity: 'day' | 'month' | 'quarter' | 'year';
  aggregation_function: 'SUM' | 'AVG' | 'COUNT' | 'RATIO' | string;
  metric_type?: string;
  base_query?: string;
  source_system?: string;
  source_formula?: string;
  comparison_periods?: string[];           // e.g., ['previous_period', 'year_over_year']
  sla_freshness_hours?: number;
  sla_completeness_threshold?: number;
  golden_path: boolean;
  status?: 'active' | 'deprecated' | 'pending' | string;
  owner_user_id?: string;
  steward_group?: string;
  created_at?: string;
  updated_at?: string;
  created_by?: string;
  updated_by?: string;
}

// ============ PoP (Period-over-Period) Results ============

export interface PopRow {
  tenant_id: UUID;
  metric_id: UUID;
  period_start: string;     // date YYYY-MM-DD
  period_end: string;       // date YYYY-MM-DD
  period_label: string;     // 'YYYY-MM' or custom format
  record_count: number;
  current_value: string;    // decimal as string for precision
  previous_value: string | null;
  delta: string | null;
  percent_change: number | null;
  computation_status: 'success' | 'failed' | 'running' | string;
  last_updated: string;     // ISO timestamp
  granularity?: string;
}

// ============ Anomaly Detection Results ============

export interface AnomalyRow {
  tenant_id: UUID;
  metric_id: UUID;
  computation_id?: UUID | null;
  anomaly_type: 'z_score' | 'threshold' | 'trend' | string;
  detected_at: string;      // ISO timestamp
  severity: 'low' | 'medium' | 'high' | 'critical';
  confidence?: number;      // 0-1 probability score
  actual_value?: string;
  expected_value?: string | null;
  expected_range_min?: string | null;
  expected_range_max?: string | null;
  z_score?: number;
  detection_params?: Record<string, unknown>;
  status: 'open' | 'resolved' | 'acknowledged' | string;
  created_at?: string;
  resolved_at?: string | null;
}

// ============ Temporal Job Runs ============

export interface JobRun {
  run_id: UUID;
  tenant_id: UUID;
  metric_id: UUID;
  calc_type: 'pop' | 'anomaly' | 'comparison' | string;
  period_label?: string;
  period_start?: string;
  period_end?: string;
  status: 'pending' | 'running' | 'success' | 'failed';
  stats?: Record<string, unknown>;
  error_message?: string;
  started_at: string;       // ISO timestamp
  ended_at?: string | null; // ISO timestamp
}

// ============ SLA & Quality ============

export interface SLAViolation {
  tenant_id: UUID;
  metric_id: UUID;
  violation_type: 'freshness' | 'completeness' | 'quality';
  expected_threshold: number;
  actual_value: number;
  breached_at: string;
  status: 'open' | 'acknowledged' | 'resolved';
}

export interface GoldenPathReadiness {
  metric_id: UUID;
  name: string;
  ready: boolean;
  freshness_ok: boolean;
  completeness_ok: boolean;
  last_check: string;
  breaches_count: number;
}

// ============ API Request/Response Models ============

export interface ListMetricsResponse {
  data: MetricRegistry[];
  total: number;
  limit?: number;
  offset?: number;
}

export interface CreateMetricRequest
  extends Partial<Omit<MetricRegistry, 'metric_id' | 'tenant_id' | 'created_at' | 'updated_at'>> { }

export interface UpdateMetricRequest extends Partial<MetricRegistry> { }

export interface ComputePopRequest {
  period_label?: string;
  period_start?: string;
  period_end?: string;
}

export interface DetectAnomaliesRequest {
  window_days?: number;
  threshold?: number;
  min_data_points?: number;
}

// ============ UI State Models ============

export interface PaginationParams {
  limit: number;
  offset: number;
}

export interface FilterParams {
  q?: string;
  domain?: string;
  golden?: boolean;
  status?: string;
}

export interface DateRangeFilter {
  from?: string;
  to?: string;
}
