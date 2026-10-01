-- Migration: 20261121_001_shadow_validation.up.sql
-- Description: Creates shadow run log table for tracking live golden record vs candidate displacement simulations.

CREATE TABLE IF NOT EXISTS mdm_eval.shadow_run_log (
    run_id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    candidate_vendor_ids TEXT[] NOT NULL,
    dropped_vendor_ids TEXT[] DEFAULT '{}',
    replacement_vendor_ids TEXT[] DEFAULT '{}',
    universe_size INT NOT NULL,
    t1_concordance_delta NUMERIC(6,4),
    t2_concordance_delta NUMERIC(6,4),
    t3_concordance_delta NUMERIC(6,4),
    gross_annual_savings NUMERIC(14,2) DEFAULT 0.00,
    net_tco_benefit NUMERIC(14,2) DEFAULT 0.00,
    payback_months NUMERIC(6,1),
    solver_latency_ms NUMERIC(8,2),
    solver_strategy TEXT NOT NULL DEFAULT 'BITMASK_BRANCH_AND_BOUND',
    solver_partial BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'COMPLETED',
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_shadow_run_tenant_date
    ON mdm_eval.shadow_run_log (tenant_id, executed_at DESC);

COMMENT ON TABLE mdm_eval.shadow_run_log IS 'Audit trail for multi-vendor displacement scenarios executed in shadow validation mode.';
