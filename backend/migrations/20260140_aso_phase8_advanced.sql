-- Phase 8: ASO Advanced Features - A/B Testing, Simulation, ML Scoring

-- A/B Experiment Table
CREATE TABLE IF NOT EXISTS semantic.aso_experiment (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    optimization_id uuid NOT NULL REFERENCES semantic.aso_optimization(id) ON DELETE CASCADE,
    tenant_id uuid NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    env text NOT NULL CHECK (env IN ('dev', 'staging', 'prod')),
    name text NOT NULL,
    
    -- Status
    status text NOT NULL CHECK (status IN ('draft', 'running', 'completed', 'aborted')) DEFAULT 'draft',
    
    -- Configuration
    traffic_percent numeric(5,2) NOT NULL DEFAULT 10.0,
    config_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    
    -- Timing
    started_at timestamptz,
    scheduled_end_at timestamptz,
    ended_at timestamptz,
    
    -- Results
    metrics_json jsonb,
    outcome text NOT NULL CHECK (outcome IN ('pending', 'promoted', 'abandoned', 'inconclusive')) DEFAULT 'pending',
    
    -- Audit
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Experiment Metrics (per-query recording)
CREATE TABLE IF NOT EXISTS semantic.aso_experiment_metric (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id uuid NOT NULL REFERENCES semantic.aso_experiment(id) ON DELETE CASCADE,
    group_name text NOT NULL CHECK (group_name IN ('control', 'test')),
    latency_ms numeric(10,2) NOT NULL,
    hit boolean NOT NULL DEFAULT false,
    error boolean NOT NULL DEFAULT false,
    recorded_at timestamptz NOT NULL DEFAULT now()
);

-- Indexes for experiment metrics
CREATE INDEX IF NOT EXISTS idx_exp_metric_experiment ON semantic.aso_experiment_metric(experiment_id);
CREATE INDEX IF NOT EXISTS idx_exp_metric_group ON semantic.aso_experiment_metric(experiment_id, group_name);

-- Simulation Results
CREATE TABLE IF NOT EXISTS semantic.aso_simulation (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id uuid NOT NULL,
    result_json jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_simulation_target ON semantic.aso_simulation(target_id);
CREATE INDEX IF NOT EXISTS idx_simulation_created ON semantic.aso_simulation(created_at DESC);

-- ML Training Data
CREATE TABLE IF NOT EXISTS semantic.ml_training_data (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    optimization_id uuid NOT NULL REFERENCES semantic.aso_optimization(id) ON DELETE CASCADE,
    input_json jsonb NOT NULL,
    actual_speedup numeric(6,2) NOT NULL,
    actual_roi numeric(8,2) NOT NULL,
    was_successful boolean NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ml_training_opt ON semantic.ml_training_data(optimization_id);
CREATE INDEX IF NOT EXISTS idx_ml_training_success ON semantic.ml_training_data(was_successful);

-- View for experiment summarization
CREATE OR REPLACE VIEW semantic.v_experiment_summary AS
SELECT 
    e.id,
    e.name,
    e.status,
    e.outcome,
    e.traffic_percent,
    e.started_at,
    e.ended_at,
    o.target_name,
    o.optimization_type,
    (e.metrics_json->>'improvement_pct')::numeric as improvement_pct,
    (e.metrics_json->>'p_value')::numeric as p_value,
    (e.metrics_json->>'significant')::boolean as significant
FROM semantic.aso_experiment e
JOIN semantic.aso_optimization o ON e.optimization_id = o.id;
