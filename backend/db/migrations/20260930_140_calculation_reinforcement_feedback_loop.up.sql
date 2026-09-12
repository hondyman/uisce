-- 20260930_140_calculation_reinforcement_feedback_loop.up.sql
-- Dynamic Confidence Adjustment & Reinforcement Learning Feedback Loop for AI Calculation Suggestions

ALTER TABLE catalog_calc.ai_calculation_suggestions 
ADD COLUMN IF NOT EXISTS acceptance_count INTEGER NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS rejection_count INTEGER NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS dynamic_weight NUMERIC(5, 4) NOT NULL DEFAULT 1.0000;

-- Telemetry log of every feedback decision for Bayesian update calibration
CREATE TABLE IF NOT EXISTS catalog_calc.calculation_feedback_telemetry (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    user_id            VARCHAR(100) NOT NULL,
    suggested_calc_key VARCHAR(100) NOT NULL,
    applicable_bo_key  VARCHAR(100) NOT NULL,
    action             VARCHAR(20) NOT NULL CHECK (action IN ('ACCEPTED', 'REJECTED')),
    applied_to_bo      BOOLEAN NOT NULL DEFAULT FALSE,
    previous_weight    NUMERIC(5, 4) NOT NULL,
    new_weight         NUMERIC(5, 4) NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_calc_feedback_telemetry 
ON catalog_calc.calculation_feedback_telemetry (tenant_id, applicable_bo_key, created_at DESC);
