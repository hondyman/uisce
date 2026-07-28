-- Migration: What-If Scenario Simulation & Projections Engine
-- Date: 2026-07-31
-- Purpose: Schema for user-defined macro-scenarios, interest rate shocks, and AST projection multipliers.

CREATE TABLE IF NOT EXISTS public.simulation_scenarios (
    scenario_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    scenario_name VARCHAR(150) NOT NULL,
    description TEXT,
    target_bo_id VARCHAR(128) NOT NULL DEFAULT 'customers',
    -- JSON structure defining shocks: [{"field": "market_value", "operator": "MULTIPLY", "value": 0.85}, ...]
    shock_rules JSONB NOT NULL,
    is_global BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    created_by VARCHAR(255) NOT NULL DEFAULT 'system',
    CONSTRAINT uk_tenant_scenario UNIQUE (tenant_id, scenario_name)
);

CREATE INDEX IF NOT EXISTS idx_sim_scenarios_tenant ON public.simulation_scenarios(tenant_id, target_bo_id);
