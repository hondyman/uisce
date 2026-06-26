// Analytics and operational types matching Go backend

export interface Chain {
  id: string;
  tenant_id: string;
  name: string;
  region: string;
  status: 'active' | 'inactive' | 'degraded';
  created_at: string;
}

export interface SLAComplianceTrend {
  id: string;
  chain_id: string;
  tenant_id: string;
  compliance_score: number; // 0-100
  success_rate_trend: number; // percentage change
  latency_trend: number; // percentage change
  percentile_99: number;
  status: 'improving' | 'stable' | 'degrading';
  reported_at: string;
  created_at: string;
}

export interface ConflictResolutionTrend {
  id: string;
  tenant_id: string;
  total_conflicts: number;
  resolved_count: number;
  failed_count: number;
  resolution_rate: number; // percentage
  avg_resolution_ms: number;
  most_common_rule: 'priority' | 'first_win' | 'serial_execute';
  period_start: string;
  period_end: string;
  created_at: string;
}

export interface ChainExecutionStats {
  id: string;
  chain_id: string;
  tenant_id: string;
  total_executions: number;
  successful_executions: number;
  failed_executions: number;
  success_rate_pct: number; // 0-100
  avg_execution_ms: number;
  max_execution_ms: number;
  min_execution_ms: number;
  last_success_at: string | null;
  last_failure_at: string | null;
  period_start: string;
  period_end: string;
  created_at: string;
}

export interface ChainHealthReport {
  id: string;
  chain_id: string;
  tenant_id: string;
  region: string;
  overall_health: number; // 0-100 score
  last_execution_status: 'success' | 'failure' | 'running';
  consecutive_failures: number;
  is_healthy: boolean;
  recommended_action: 'investigate' | 'retry' | 'disable' | 'none';
  action_executed: boolean;
  reported_at: string;
  created_at: string;
}

export interface ChainPrediction {
  id: string;
  chain_id: string;
  tenant_id: string;
  region: string;
  prediction_ts: string;
  failure_prob: number; // 0.0-1.0
  recommended_action: string;
  model_version: string;
  top_features: Array<{
    name: string;
    importance: number;
  }>;
  created_at: string;
}

export interface RealTimeEvent {
  id: string;
  type: string;
  tenant_id: string;
  region: string;
  timestamp: string;
  data: Record<string, unknown>;
}

export interface MetricCard {
  label: string;
  value: string | number;
  change?: number;
  trend?: 'up' | 'down' | 'stable';
  color?: 'success' | 'warning' | 'danger' | 'info';
}

export interface DashboardFilters {
  tenant_id?: string;
  region?: string;
  time_range?: 'hour' | 'day' | 'week' | 'month';
  status?: string;
}

export interface ApiError {
  message: string;
  code?: string;
  details?: unknown;
}

// ML Predictions & Explainability Types (Phase 3.17)
export interface Prediction {
  id: string;
  chain_id: string;
  region: string;
  tenant_id: string;
  failure_probability: number;
  confidence: number;
  risk_level: 'low' | 'medium' | 'high' | 'critical';
  predicted_at: string;
  horizon_hours: number;
  top_risk_factors: RiskFactor[];
  model_version: string;
  explainability?: Explainability;
}

export interface RiskFactor {
  name: string;
  contribution: number;
  current_value: number | string;
  threshold: number;
  direction: 'increasing' | 'decreasing' | 'stable';
}

export interface Explainability {
  shap_values: Record<string, number>;
  base_value: number;
  feature_importance: Record<string, number>;
  feature_values: Record<string, any>;
  local_contributions: LocalContribution[];
  interaction_pairs: InteractionPair[];
  explanation_type: string;
  computation_time_ms: number;
}

export interface LocalContribution {
  feature: string;
  shap_value: number;
  abs_shap_value: number;
  actual_value: any;
  range?: FeatureRange;
  impact: 'positive' | 'negative' | 'neutral';
  percentile: number;
}

export interface InteractionPair {
  feature_1: string;
  feature_2: string;
  interaction: number;
}

export interface FeatureRange {
  min: number;
  max: number;
  mean: number;
  std_dev: number;
  q1: number;
  median: number;
  q3: number;
}
