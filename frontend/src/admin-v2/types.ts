// TypeScript types for Admin UI
// Aligned with backend JSON contracts

export interface Tenant {
  id: string;
  name: string;
  code?: string;
  region?: string;
  plan: "free" | "pro" | "enterprise";
  max_requests?: number;
  window_seconds?: number;
  is_suspended: boolean;
  created_at: string;
  updated_at: string;
  user_id?: string;
  last_used_at?: string;
}

export interface CreateTenantRequest {
  name: string;
  code?: string;
  region?: string;
  plan: "free" | "pro" | "enterprise";
  max_requests?: number;
  window_seconds?: number;
}

export interface UpdateTenantRequest {
  name?: string;
  region?: string;
  plan?: "free" | "pro" | "enterprise";
  max_requests?: number;
  window_seconds?: number;
}

export interface APIKey {
  id: string;
  name: string;
  user_id: string;
  roles: string[];
  tenant_ids: string[];
  is_revoked: boolean;
  created_at: string;
  updated_at: string;
  last_used_at?: string;
}

export interface CreateAPIKeyRequest {
  user_id: string;
  name: string;
  description?: string;
  roles: string[];
  tenant_ids: string[];
}

export interface APIKeyUsage {
  id: string;
  api_key_id: string;
  user_id?: string;
  tenant_id?: string;
  path: string;
  method: string;
  region?: string;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

export interface DailyUsage {
  day: string;
  count: number;
}

export interface LatencyPoint {
  timestamp: string;
  p50: number;
  p95: number;
  p99?: number;
}

export interface ErrorPoint {
  timestamp: string;
  errors: number;
  error_rate?: number;
}

export interface UsagePoint {
  timestamp: string;
  count: number;
}

export interface EndpointUsage {
  path: string;
  requests: number;
  errors: number;
  avg_latency_ms: number;
}

export interface TopTenant {
  tenant_id: string;
  name: string;
  code?: string;
  region?: string;
  plan: string;
  requests: number;
  errors: number;
  avg_latency_ms: number;
}

export interface RecentError {
  timestamp: string;
  tenant_id?: string;
  tenant_name?: string;
  path: string;
  method?: string;
  error: string;
  status_code?: number;
}

// API Response envelopes
export interface ListResponse<T> {
  data?: T[];
  items?: T[];
  [key: string]: any;
}

export interface SingleResponse<T> {
  data?: T;
  [key: string]: any;
}

// Common error shape
export interface APIError {
  error: string;
  message?: string;
  details?: Record<string, any>;
}
// ========== Ops System Types ==========

// Alerts
export interface Alert {
  id: string;
  name: string;
  scope: "global" | "tenant" | "endpoint";
  metric: string;
  threshold: number;
  comparison: ">" | "<" | ">=" | "<=" | "==";
  window_secs: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface AlertEvent {
  id: string;
  alert_id: string;
  scope_id?: string;
  endpoint?: string;
  value: number;
  triggered_at: string;
}

// Health Scores
export interface TenantHealth {
  tenant_id: string;
  health_score: number; // 0-100
  components: {
    availability?: number;
    latency?: number;
    error_rate?: number;
    rate_limits?: number;
  };
  computed_at: string;
  updated_at: string;
}

export interface EndpointHealth {
  endpoint: string;
  health_score: number; // 0-100
  error_rate: number;
  p95_ms: number;
  requests_1h: number;
  components?: {
    error_rate?: number;
    latency?: number;
  };
  computed_at: string;
  updated_at: string;
}

// Latency Heatmaps
export interface HeatmapSeriesPoint {
  time: string;
  value: number;
  p95_ms?: number;
  p99_ms?: number;
}

export interface HeatmapSeries {
  key: string;
  values: HeatmapSeriesPoint[];
}

export interface Heatmap {
  buckets: string[];
  series: HeatmapSeries[];
}

// Error Fingerprints
export interface ErrorFingerprint {
  id: string;
  fingerprint: string;
  path: string;
  status_code: number;
  sample_message: string;
  first_seen: string;
  last_seen: string;
  count: number;
  created_at: string;
}

export interface ErrorEvent {
  id: string;
  fingerprint_id: string;
  tenant_id?: string;
  endpoint: string;
  status_code: number;
  message: string;
  request_id?: string;
  occurred_at: string;
}

// Health Status
export type HealthStatus = "healthy" | "degraded" | "critical";

export function getHealthStatus(score: number): HealthStatus {
  if (score >= 80) return "healthy";
  if (score >= 50) return "degraded";
  return "critical";
}