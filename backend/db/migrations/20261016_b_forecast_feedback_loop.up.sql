-- Migration: 20261016_b_forecast_feedback_loop.up.sql
-- Forecast Outcome Feedback Loop & Calibration Engine
--
-- Depends on: 20261016_predictive_finops_and_smoothing.up.sql

-- ---------------------------------------------------------------------------
-- 1. Forecast Feedback — actual vs. predicted outcomes
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS finops.forecast_feedback (
    feedback_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    forecast_id     UUID NOT NULL REFERENCES finops.compute_demand_forecasts(forecast_id) ON DELETE CASCADE,

    -- Operator-supplied outcome classification
    outcome         VARCHAR(30) NOT NULL,
    -- ACCURATE       : spike happened and magnitude was within ±25% of projection
    -- FALSE_POSITIVE : high peak_probability but no spike materialised
    -- MISSED_SPIKE   : low peak_probability but severe spike occurred
    -- PARTIAL_SPIKE  : spike occurred but significantly smaller than projected

    -- Actual measured values (populated from audit telemetry post-hoc, or entered manually)
    actual_scanned_bytes   BIGINT         DEFAULT NULL,
    actual_cpu_duration_ms BIGINT         DEFAULT NULL,
    actual_cost_usd        NUMERIC(10, 4) DEFAULT NULL,

    -- Derived accuracy ratio = actual_cost / projected_cost (computed on insert via trigger)
    accuracy_ratio  NUMERIC(8, 4)  GENERATED ALWAYS AS (
        CASE
            WHEN actual_cost_usd IS NOT NULL AND actual_cost_usd > 0
            THEN ROUND(actual_cost_usd /
                    NULLIF((
                        SELECT projected_cost_usd
                        FROM   finops.compute_demand_forecasts cdf
                        WHERE  cdf.forecast_id = forecast_feedback.forecast_id
                        LIMIT  1
                    ), 0), 4)
            ELSE NULL
        END
    ) STORED,

    notes           TEXT,
    recorded_by     UUID,        -- user_id who submitted the feedback (nullable for system submissions)
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_forecast_feedback UNIQUE (forecast_id),  -- one feedback record per forecast
    CONSTRAINT chk_outcome CHECK (
        outcome IN ('ACCURATE', 'FALSE_POSITIVE', 'MISSED_SPIKE', 'PARTIAL_SPIKE')
    )
);

CREATE INDEX IF NOT EXISTS idx_forecast_feedback_tenant_time
    ON finops.forecast_feedback (tenant_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS idx_forecast_feedback_outcome
    ON finops.forecast_feedback (tenant_id, outcome, recorded_at DESC);

COMMENT ON TABLE finops.forecast_feedback IS
    'Operator or system-recorded outcomes for each compute demand forecast. '
    'The accuracy_ratio column (actual/projected cost) is the primary signal '
    'consumed by the calibration engine to correct future predictions.';

-- ---------------------------------------------------------------------------
-- 2. Tenant Calibration Summary (materialised per tenant for fast reads)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS finops.forecast_calibration_state (
    tenant_id           UUID PRIMARY KEY REFERENCES public.tenants(id) ON DELETE CASCADE,
    calibration_factor  NUMERIC(8, 4) NOT NULL DEFAULT 1.0000,
    -- Values > 1.0 → we have been under-predicting; boost projections
    -- Values < 1.0 → we have been over-predicting; reduce projections
    sample_count        INT           NOT NULL DEFAULT 0,
    last_computed_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_calibration_factor CHECK (calibration_factor > 0)
);

COMMENT ON TABLE finops.forecast_calibration_state IS
    'Per-tenant rolling calibration factor derived from the last 30 ACCURATE/PARTIAL_SPIKE '
    'feedback entries. Read by DemandForecaster at generation time to self-correct predictions.';
