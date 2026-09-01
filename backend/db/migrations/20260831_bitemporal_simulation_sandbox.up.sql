-- 20260831_bitemporal_simulation_sandbox.up.sql
-- Bitemporal Simulation Sandbox & Copy-on-Write Delta Ledger

CREATE SCHEMA IF NOT EXISTS catalog_sandbox;

-- 1. Simulation Scenarios (Zero-Copy Branches)
CREATE TABLE IF NOT EXISTS catalog_sandbox.simulation_scenarios (
    scenario_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    scenario_key VARCHAR(100) NOT NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    base_portfolio_node_id UUID NOT NULL,
    effective_date_target DATE NOT NULL,
    knowledge_time_cutoff TIMESTAMPTZ NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT', -- DRAFT, RUNNING, EVALUATED, PROMOTED, ARCHIVED
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_scenario UNIQUE(tenant_id, scenario_key)
);

-- 2. Copy-on-Write Scenario Mutation Deltas
CREATE TABLE IF NOT EXISTS catalog_sandbox.scenario_mutation_delta (
    delta_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID NOT NULL REFERENCES catalog_sandbox.simulation_scenarios(scenario_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    target_entity_type VARCHAR(50) NOT NULL, -- POSITION, TAX_LOT, FEE_SCHEDULE, ALLOCATION_PCT
    entity_key VARCHAR(100) NOT NULL,
    operation VARCHAR(20) NOT NULL, -- INSERT, UPDATE, DELETE, OVERRIDE
    original_state JSONB,
    mutated_state JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sandbox_delta_lookup 
    ON catalog_sandbox.scenario_mutation_delta(scenario_id, target_entity_type, entity_key);

-- 3. Simulation Run Results & Merkle Replay Ledger
CREATE TABLE IF NOT EXISTS catalog_sandbox.scenario_execution_runs (
    run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID NOT NULL REFERENCES catalog_sandbox.simulation_scenarios(scenario_id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    total_positions_evaluated INT NOT NULL,
    baseline_nav NUMERIC(18, 4) NOT NULL,
    simulated_nav NUMERIC(18, 4) NOT NULL,
    simulated_xirr NUMERIC(8, 4),
    simulated_tracking_error NUMERIC(8, 4),
    merkle_execution_passport VARCHAR(64) NOT NULL,
    executed_at TIMESTAMPTZ DEFAULT NOW()
);
