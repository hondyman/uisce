-- ASO (Autonomous Semantic Optimization) Schema
-- This migration creates tables for ASO policies and optimization records

-- Ensure semantic schema exists
CREATE SCHEMA IF NOT EXISTS semantic;

-- ============================================================================
-- ASO Policy Table
-- Stores optimization policies per environment with core→tenant inheritance
-- ============================================================================

CREATE TABLE IF NOT EXISTS semantic.aso_policy (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    env text NOT NULL CHECK (env IN ('dev', 'staging', 'prod')),
    tenant_id uuid NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Policy state
    enabled boolean NOT NULL DEFAULT false,
    mode text NOT NULL DEFAULT 'advisory' CHECK (mode IN ('advisory', 'auto_tune', 'auto_apply')),
    
    -- Thresholds and limits
    max_new_preaggs_per_day integer NOT NULL DEFAULT 3,
    max_changes_per_day integer NOT NULL DEFAULT 10,
    min_score_for_new_preagg double precision NOT NULL DEFAULT 1.0,
    min_usage_for_retirement integer NOT NULL DEFAULT 0,
    hot_path_threshold_ms integer NOT NULL DEFAULT 1000,
    lookback_window_seconds integer NOT NULL DEFAULT 604800, -- 7 days
    
    -- Pre-warm settings
    prewarm_enabled boolean NOT NULL DEFAULT false,
    prewarm_lead_time_minutes integer NOT NULL DEFAULT 15,
    
    -- Audit
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'system',
    updated_by text NOT NULL DEFAULT 'system',
    
    -- Ensure unique per env + tenant (NULL tenant = core policy)
    CONSTRAINT uq_aso_policy_env_tenant UNIQUE (env, tenant_id)
);

-- Index for policy lookup
CREATE INDEX IF NOT EXISTS idx_aso_policy_env ON semantic.aso_policy(env);
CREATE INDEX IF NOT EXISTS idx_aso_policy_tenant ON semantic.aso_policy(tenant_id);

-- ============================================================================
-- ASO Optimization Table
-- Stores optimization records (proposed, approved, applied, rejected)
-- ============================================================================

CREATE TABLE IF NOT EXISTS semantic.aso_optimization (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    env text NOT NULL CHECK (env IN ('dev', 'staging', 'prod')),
    tenant_id uuid NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    scope text NOT NULL CHECK (scope IN ('core', 'tenant')),
    
    -- Optimization type and target
    optimization_type text NOT NULL CHECK (optimization_type IN (
        'tune_refresh',
        'tune_definition', 
        'create_preagg',
        'retire_asset',
        'prewarm'
    )),
    target_type text NOT NULL CHECK (target_type IN ('preagg', 'bo', 'calc', 'term')),
    target_id uuid NOT NULL,
    target_name text NOT NULL,
    
    -- Status and mode
    status text NOT NULL DEFAULT 'proposed' CHECK (status IN (
        'proposed',
        'approved', 
        'applied',
        'rejected',
        'failed',
        'superseded'
    )),
    mode text NOT NULL CHECK (mode IN ('advisory', 'auto')),
    
    -- Scoring and reasoning
    score double precision NOT NULL DEFAULT 0.0,
    reason text NOT NULL,
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    
    -- Workload evidence
    workload_window_days integer NOT NULL DEFAULT 7,
    queries_per_day double precision,
    avg_latency_ms double precision,
    p95_latency_ms double precision,
    avg_rows_scanned bigint,
    
    -- Policy reference
    policy_id uuid REFERENCES semantic.aso_policy(id) ON DELETE SET NULL,
    
    -- Lifecycle
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by text NOT NULL DEFAULT 'aso_engine',
    approved_at timestamptz,
    approved_by text,
    applied_at timestamptz,
    applied_by text,
    rejected_at timestamptz,
    rejected_by text,
    rejection_reason text,
    
    -- For rollback/undo
    before_config jsonb,
    after_config jsonb
);

-- Indexes for optimization queries
CREATE INDEX IF NOT EXISTS idx_aso_opt_env_status ON semantic.aso_optimization(env, status);
CREATE INDEX IF NOT EXISTS idx_aso_opt_tenant ON semantic.aso_optimization(tenant_id);
CREATE INDEX IF NOT EXISTS idx_aso_opt_target ON semantic.aso_optimization(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_aso_opt_created ON semantic.aso_optimization(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_aso_opt_type ON semantic.aso_optimization(optimization_type);

-- ============================================================================
-- ASO Optimization Audit Table
-- Detailed audit trail for applied and rejected optimizations
-- ============================================================================

CREATE TABLE IF NOT EXISTS semantic.aso_optimization_audit (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    optimization_id uuid NOT NULL REFERENCES semantic.aso_optimization(id) ON DELETE CASCADE,
    action text NOT NULL CHECK (action IN ('proposed', 'approved', 'applied', 'rejected', 'failed', 'rolled_back')),
    actor text NOT NULL,
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_aso_audit_opt ON semantic.aso_optimization_audit(optimization_id);
CREATE INDEX IF NOT EXISTS idx_aso_audit_created ON semantic.aso_optimization_audit(created_at DESC);

-- ============================================================================
-- ASO Daily Stats Table
-- Tracks daily optimization activity for rate limiting
-- ============================================================================

CREATE TABLE IF NOT EXISTS semantic.aso_daily_stats (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    env text NOT NULL,
    tenant_id uuid NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    stat_date date NOT NULL DEFAULT CURRENT_DATE,
    
    -- Counts for rate limiting
    preaggs_created integer NOT NULL DEFAULT 0,
    changes_applied integer NOT NULL DEFAULT 0,
    optimizations_proposed integer NOT NULL DEFAULT 0,
    optimizations_rejected integer NOT NULL DEFAULT 0,
    
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    
    CONSTRAINT uq_aso_stats_env_tenant_date UNIQUE (env, tenant_id, stat_date)
);

CREATE INDEX IF NOT EXISTS idx_aso_stats_date ON semantic.aso_daily_stats(stat_date);

-- ============================================================================
-- Pre-agg Usage Stats View
-- Aggregates telemetry for pre-agg optimization decisions
-- ============================================================================

-- This view assumes query_telemetry table exists with pre-agg routing info
-- Adjust based on actual telemetry schema

-- CREATE OR REPLACE VIEW semantic.preagg_usage_stats AS
-- SELECT 
--     preagg_id,
--     tenant_id,
--     DATE_TRUNC('day', executed_at) AS stat_date,
--     COUNT(*) AS query_count,
--     AVG(duration_ms) AS avg_duration_ms,
--     PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) AS p95_duration_ms,
--     AVG(rows_scanned) AS avg_rows_scanned
-- FROM query_telemetry
-- WHERE preagg_id IS NOT NULL
-- GROUP BY preagg_id, tenant_id, DATE_TRUNC('day', executed_at);

-- ============================================================================
-- Insert default core policies (one per environment)
-- ============================================================================

INSERT INTO semantic.aso_policy (env, tenant_id, enabled, mode, created_by, updated_by)
VALUES 
    ('dev', NULL, true, 'auto_apply', 'system', 'system'),
    ('staging', NULL, true, 'auto_tune', 'system', 'system'),
    ('prod', NULL, false, 'advisory', 'system', 'system')
ON CONFLICT (env, tenant_id) DO NOTHING;

-- ============================================================================
-- Update trigger for updated_at
-- ============================================================================

CREATE OR REPLACE FUNCTION semantic.update_aso_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_aso_policy_updated
    BEFORE UPDATE ON semantic.aso_policy
    FOR EACH ROW
    EXECUTE FUNCTION semantic.update_aso_updated_at();

CREATE TRIGGER trg_aso_optimization_updated
    BEFORE UPDATE ON semantic.aso_optimization
    FOR EACH ROW
    EXECUTE FUNCTION semantic.update_aso_updated_at();

CREATE TRIGGER trg_aso_stats_updated
    BEFORE UPDATE ON semantic.aso_daily_stats
    FOR EACH ROW
    EXECUTE FUNCTION semantic.update_aso_updated_at();
