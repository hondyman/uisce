-- Enhanced SLO schema for unified observability
-- Migration: 20260152_enhanced_slo_schema.sql

-- SLO definitions with scope types
CREATE TABLE IF NOT EXISTS semantic_slos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env TEXT NOT NULL DEFAULT 'prod',
    tenant_id UUID,
    scope_type TEXT NOT NULL, -- 'bo' | 'preagg' | 'entitlement' | 'planner'
    scope_id TEXT NOT NULL,   -- e.g. 'Positions' or 'preagg:positions_daily'
    slo_type TEXT NOT NULL,   -- 'latency' | 'freshness' | 'error_rate' | 'entitlement_latency' | 'preagg_hit_rate'
    target NUMERIC NOT NULL,  -- e.g. 500 (ms), 60 (sec), 0.01 (1%), 0.8 (80%)
    time_window TEXT NOT NULL DEFAULT '7d', -- '7d', '30d'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT,
    enabled BOOLEAN DEFAULT TRUE,
    UNIQUE(env, tenant_id, scope_type, scope_id, slo_type)
);

CREATE INDEX IF NOT EXISTS idx_semantic_slos_env_tenant ON semantic_slos(env, tenant_id);
CREATE INDEX IF NOT EXISTS idx_semantic_slos_scope ON semantic_slos(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_semantic_slos_enabled ON semantic_slos(enabled) WHERE enabled = true;

-- SLO evaluation results
CREATE TABLE IF NOT EXISTS semantic_slo_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slo_id UUID NOT NULL REFERENCES semantic_slos(id) ON DELETE CASCADE,
    env TEXT NOT NULL,
    tenant_id UUID,
    scope_type TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    measured_value NUMERIC NOT NULL,
    target_value NUMERIC NOT NULL,
    status TEXT NOT NULL, -- 'met' | 'violated' | 'unknown'
    delta_percent NUMERIC, -- How far off from target (negative = under, positive = over)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_slo_evaluations_slo_id ON semantic_slo_evaluations(slo_id);
CREATE INDEX IF NOT EXISTS idx_slo_evaluations_created ON semantic_slo_evaluations(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_slo_evaluations_status ON semantic_slo_evaluations(status);

-- SLO violations (for alerting and policy changes)
CREATE TABLE IF NOT EXISTS semantic_slo_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slo_id UUID NOT NULL REFERENCES semantic_slos(id) ON DELETE CASCADE,
    evaluation_id UUID REFERENCES semantic_slo_evaluations(id) ON DELETE SET NULL,
    env TEXT NOT NULL,
    tenant_id UUID,
    scope_type TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    slo_type TEXT NOT NULL,
    target_value NUMERIC NOT NULL,
    actual_value NUMERIC NOT NULL,
    severity TEXT NOT NULL DEFAULT 'warning', -- 'info' | 'warning' | 'critical'
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_by TEXT,
    acknowledged_at TIMESTAMPTZ,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_slo_violations_slo_id ON semantic_slo_violations(slo_id);
CREATE INDEX IF NOT EXISTS idx_slo_violations_unresolved ON semantic_slo_violations(resolved) WHERE resolved = false;
CREATE INDEX IF NOT EXISTS idx_slo_violations_created ON semantic_slo_violations(created_at DESC);

-- ASO tuning hints (feedback from SLO violations)
CREATE TABLE IF NOT EXISTS aso_tuning_hints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env TEXT NOT NULL DEFAULT 'prod',
    tenant_id UUID,
    bo_name TEXT NOT NULL,
    priority_boost NUMERIC DEFAULT 1.0, -- >1.0 for hot BOs
    max_aggressiveness NUMERIC DEFAULT 1.0, -- 0-1 scale
    auto_apply_enabled BOOLEAN DEFAULT TRUE,
    reason TEXT, -- Why this hint was set
    source TEXT, -- 'slo_violation' | 'manual' | 'aso_recommendation'
    expires_at TIMESTAMPTZ, -- Optional expiration
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(env, tenant_id, bo_name)
);

CREATE INDEX IF NOT EXISTS idx_aso_tuning_hints_bo ON aso_tuning_hints(env, tenant_id, bo_name);
CREATE INDEX IF NOT EXISTS idx_aso_tuning_hints_active ON aso_tuning_hints(expires_at);

-- Query telemetry for planner feedback (extends existing query_events if needed)
CREATE TABLE IF NOT EXISTS planner_telemetry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env TEXT NOT NULL DEFAULT 'prod',
    tenant_id UUID,
    bo_name TEXT NOT NULL,
    plan_type TEXT NOT NULL, -- 'base' | 'preagg' | 'hybrid' | 'cached'
    preagg_name TEXT,
    entitlement_strategy TEXT,
    estimated_latency_ms NUMERIC,
    actual_latency_ms NUMERIC,
    estimated_scan_bytes NUMERIC,
    actual_scan_bytes NUMERIC,
    slo_satisfied BOOLEAN,
    candidates_evaluated INTEGER,
    planning_time_ms NUMERIC,
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    user_id TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_planner_telemetry_bo ON planner_telemetry(env, tenant_id, bo_name);
CREATE INDEX IF NOT EXISTS idx_planner_telemetry_plan_type ON planner_telemetry(plan_type);
CREATE INDEX IF NOT EXISTS idx_planner_telemetry_created ON planner_telemetry(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_planner_telemetry_preagg ON planner_telemetry(preagg_name) WHERE preagg_name IS NOT NULL;

-- BO features (aggregated telemetry for cost estimation)
CREATE TABLE IF NOT EXISTS bo_features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env TEXT NOT NULL DEFAULT 'prod',
    tenant_id UUID,
    bo_name TEXT NOT NULL,
    time_window TEXT NOT NULL DEFAULT '7d', -- '1d', '7d', '30d'
    p50_latency_ms NUMERIC,
    p95_latency_ms NUMERIC,
    p99_latency_ms NUMERIC,
    avg_scan_bytes NUMERIC,
    query_count BIGINT,
    error_rate NUMERIC,
    cache_hit_rate NUMERIC,
    preagg_hit_rate NUMERIC,
    last_query_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(env, tenant_id, bo_name, time_window)
);

CREATE INDEX IF NOT EXISTS idx_bo_features_lookup ON bo_features(env, tenant_id, bo_name, time_window);

-- Pre-agg features (aggregated telemetry for cost estimation)
CREATE TABLE IF NOT EXISTS preagg_features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env TEXT NOT NULL DEFAULT 'prod',
    tenant_id UUID,
    preagg_name TEXT NOT NULL,
    time_window TEXT NOT NULL DEFAULT '7d',
    avg_speedup NUMERIC,
    hit_count BIGINT,
    miss_count BIGINT,
    hit_rate NUMERIC,
    storage_bytes BIGINT,
    refresh_frequency_sec INTEGER,
    last_refresh_at TIMESTAMPTZ,
    avg_freshness_lag_sec NUMERIC,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(env, tenant_id, preagg_name, time_window)
);

CREATE INDEX IF NOT EXISTS idx_preagg_features_lookup ON preagg_features(env, tenant_id, preagg_name, time_window);
