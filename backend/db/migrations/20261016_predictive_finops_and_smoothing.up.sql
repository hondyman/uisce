-- Migration: 20261016_predictive_finops_and_smoothing.up.sql
-- Predictive Compute Demand Forecasting, Workload Smoothing Policies &
-- Off-Peak Pre-Warm Execution Ledger
--
-- Depends on: 20260910_finops_and_governance_extensions.sql (finops schema)
--             public.tenants(id) primary key

-- ---------------------------------------------------------------------------
-- 0. Audit Telemetry Source Extensions
-- ---------------------------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS audit;

-- audit.analytical_query_execution_logs was created in 20260824_002 with timestamp 'created_at'
-- Ensure optional columns for FinOps telemetry exist if not already present
ALTER TABLE audit.analytical_query_execution_logs
    ADD COLUMN IF NOT EXISTS cpu_duration_ms     BIGINT       NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS attributed_cost_usd NUMERIC(10, 6) NOT NULL DEFAULT 0.000000,
    ADD COLUMN IF NOT EXISTS user_id             UUID,
    ADD COLUMN IF NOT EXISTS bo_id               UUID,
    ADD COLUMN IF NOT EXISTS session_id          UUID;

CREATE INDEX IF NOT EXISTS idx_aqel_tenant_created
    ON audit.analytical_query_execution_logs (tenant_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- 1. Projected Hourly Compute Demand per Tenant
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS finops.compute_demand_forecasts (
    forecast_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    forecast_window_start    TIMESTAMPTZ NOT NULL,
    forecast_window_end      TIMESTAMPTZ NOT NULL,
    projected_scanned_bytes  BIGINT      NOT NULL DEFAULT 0,
    projected_cpu_duration_ms BIGINT     NOT NULL DEFAULT 0,
    projected_cost_usd       NUMERIC(10, 4) NOT NULL DEFAULT 0.0000,
    confidence_score         NUMERIC(4, 3)  NOT NULL DEFAULT 0.500,  -- 0.000–1.000
    peak_probability         NUMERIC(4, 3)  NOT NULL DEFAULT 0.000,  -- Likelihood of autoscale trigger
    contributing_factors     JSONB          NOT NULL DEFAULT '[]'::jsonb,
    -- e.g. ["CALENDAR_MONTH_END", "BATCH_REPORT_BURST(4)", "CALENDAR_QUARTER_END"]
    generated_at             TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tenant_forecast_window
        UNIQUE (tenant_id, forecast_window_start)
);

CREATE INDEX IF NOT EXISTS idx_demand_forecast_lookup
    ON finops.compute_demand_forecasts (tenant_id, forecast_window_start, forecast_window_end);

COMMENT ON TABLE finops.compute_demand_forecasts IS
    'Persisted per-tenant hourly compute demand projections produced by the DemandForecaster. '
    'Upserted on each forecast run; retained for trend analysis and cost attribution.';

-- ---------------------------------------------------------------------------
-- 2. Workload Smoothing Policies (Rule 1: Config-Before-Code)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS finops.workload_smoothing_policies (
    policy_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    policy_name VARCHAR(100) NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,

    -- Cron expression defining the off-peak pre-warming window (tenant timezone implied)
    off_peak_cron VARCHAR(50) NOT NULL DEFAULT '0 2 * * *',  -- 02:00 UTC

    -- Multiplier above baseline that classifies an impending window as a spike requiring pre-warming
    prewarm_threshold_multiplier NUMERIC(4, 2) NOT NULL DEFAULT 2.50,

    -- Whether elastic/background workloads may be deferred into cost troughs
    enable_burst_deferral BOOLEAN NOT NULL DEFAULT TRUE,
    max_deferral_minutes  INT     NOT NULL DEFAULT 180,

    -- Minimum peak probability (0–1) that must be forecast before pre-warming fires
    min_peak_probability_to_prewarm NUMERIC(4, 3) NOT NULL DEFAULT 0.700,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tenant_smoothing_policy
        UNIQUE (tenant_id, policy_name)
);

COMMENT ON TABLE finops.workload_smoothing_policies IS
    'Tenant-level configuration governing pre-warming schedules, spike thresholds, and '
    'burst deferral limits. Consumed by the PrewarmCoordinator at runtime.';

-- (updated_at is maintained directly by UpsertPolicy in application code)

-- ---------------------------------------------------------------------------
-- 3. Pre-Warm Execution Ledger (Rule 8: Observability & Billing)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS finops.prewarm_execution_ledger (
    execution_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    bo_id                     UUID NOT NULL,
    target_metric             VARCHAR(50)    NOT NULL,  -- 'XIRR' | 'NAV_ROLLUP' | 'MODIFIED_DURATION'
    entities_prewarmed_count  INT            NOT NULL DEFAULT 0,
    compute_cost_incurred_usd NUMERIC(8, 4)  NOT NULL DEFAULT 0.0000,
    estimated_peak_savings_usd NUMERIC(8, 4) NOT NULL DEFAULT 0.0000,
    peak_probability_at_trigger NUMERIC(4, 3),
    policy_id                 UUID REFERENCES finops.workload_smoothing_policies(policy_id),
    status                    VARCHAR(30)    NOT NULL DEFAULT 'PENDING',
    -- PENDING | RUNNING | COMPLETED | PARTIAL | FAILED
    error_detail              TEXT,
    executed_at               TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    completed_at              TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_prewarm_ledger_tenant_time
    ON finops.prewarm_execution_ledger (tenant_id, executed_at DESC);

COMMENT ON TABLE finops.prewarm_execution_ledger IS
    'Immutable audit log of every off-peak pre-warming run: which BOs were seeded, '
    'compute cost incurred, and the estimated avoided on-demand scaling charges.';
