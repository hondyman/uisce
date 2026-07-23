-- Phase 7: ASO Enhancements - Drift Signals & Healing Actions

-- Drift Signal Table
CREATE TABLE IF NOT EXISTS semantic.drift_signal (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL CHECK (target_type IN ('preagg', 'bo', 'calc', 'term')),
    target_id uuid NOT NULL,
    target_name text NOT NULL DEFAULT '',
    tenant_id uuid NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    env text NOT NULL CHECK (env IN ('dev', 'staging', 'prod')),
    
    -- Signal classification
    signal_type text NOT NULL CHECK (signal_type IN (
        'pattern_change',
        'miss_rate_spike',
        'latency_regression',
        'refresh_failure',
        'schema_drift',
        'usage_decline',
        'stale_data'
    )),
    severity text NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')) DEFAULT 'medium',
    status text NOT NULL CHECK (status IN ('open', 'acknowledged', 'resolving', 'resolved', 'ignored')) DEFAULT 'open',
    
    -- Evidence and recommendation
    evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
    recommendation text NOT NULL DEFAULT '',
    
    -- Timestamps
    detected_at timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz,
    resolved_by text,
    auto_resolved boolean NOT NULL DEFAULT false
);

-- Indexes for drift_signal
CREATE INDEX IF NOT EXISTS idx_drift_signal_target ON semantic.drift_signal(target_id);
CREATE INDEX IF NOT EXISTS idx_drift_signal_env_status ON semantic.drift_signal(env, status);
CREATE INDEX IF NOT EXISTS idx_drift_signal_severity ON semantic.drift_signal(severity) WHERE status = 'open';
CREATE INDEX IF NOT EXISTS idx_drift_signal_tenant ON semantic.drift_signal(tenant_id);

-- Healing Action Table
CREATE TABLE IF NOT EXISTS semantic.healing_action (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL CHECK (target_type IN ('preagg', 'bo', 'calc', 'term')),
    target_id uuid NOT NULL,
    target_name text NOT NULL DEFAULT '',
    tenant_id uuid NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    env text NOT NULL DEFAULT 'prod',
    
    -- Action details
    action_type text NOT NULL CHECK (action_type IN (
        'retry_refresh',
        'rebuild_preagg',
        'update_definition',
        'adjust_interval',
        'expand_coverage',
        'deprecate_unused'
    )),
    trigger text NOT NULL,
    status text NOT NULL CHECK (status IN ('pending', 'in_progress', 'success', 'failed', 'skipped')) DEFAULT 'pending',
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    
    -- Timing and retry
    started_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    error text,
    retry_count int NOT NULL DEFAULT 0,
    
    -- Link to signal if applicable
    drift_signal_id uuid REFERENCES semantic.drift_signal(id)
);

-- Indexes for healing_action
CREATE INDEX IF NOT EXISTS idx_healing_action_target ON semantic.healing_action(target_id);
CREATE INDEX IF NOT EXISTS idx_healing_action_status ON semantic.healing_action(status) WHERE status IN ('pending', 'in_progress');
CREATE INDEX IF NOT EXISTS idx_healing_action_started ON semantic.healing_action(started_at DESC);

-- Cost Tracking View (aggregates cost metrics from optimizations)
CREATE OR REPLACE VIEW semantic.v_aso_cost_summary AS
SELECT 
    env,
    tenant_id,
    COUNT(*) FILTER (WHERE status = 'applied') as applied_count,
    SUM((details->'cost_metrics'->>'net_savings_per_day')::numeric) as daily_savings,
    SUM((details->'cost_metrics'->>'total_savings_to_date')::numeric) as total_savings,
    SUM((details->'cost_metrics'->>'total_costs_to_date')::numeric) as total_costs
FROM semantic.aso_optimization
WHERE details->'cost_metrics' IS NOT NULL
GROUP BY env, tenant_id;

-- Daily summary materialization for dashboards
CREATE TABLE IF NOT EXISTS semantic.aso_cost_daily (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    date date NOT NULL,
    env text NOT NULL,
    tenant_id uuid NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Metrics
    optimizations_applied int NOT NULL DEFAULT 0,
    compute_savings numeric(12,4) NOT NULL DEFAULT 0,
    storage_costs numeric(12,4) NOT NULL DEFAULT 0,
    net_savings numeric(12,4) NOT NULL DEFAULT 0,
    queries_accelerated bigint NOT NULL DEFAULT 0,
    
    created_at timestamptz NOT NULL DEFAULT now(),
    
    UNIQUE(date, env, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_aso_cost_daily_date ON semantic.aso_cost_daily(date DESC);
CREATE INDEX IF NOT EXISTS idx_aso_cost_daily_tenant ON semantic.aso_cost_daily(tenant_id, date DESC);
