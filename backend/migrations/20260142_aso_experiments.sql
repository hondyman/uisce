-- Up Migration

CREATE TABLE IF NOT EXISTS aso.experiments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    env text NOT NULL,
    tenant_id uuid, -- nullable for global experiments
    name text NOT NULL,
    optimization_id uuid NOT NULL, -- link back to optimization
    control_changeset_id uuid NOT NULL,
    treatment_changeset_id uuid NOT NULL,
    traffic_split_control double precision NOT NULL DEFAULT 0.5,
    traffic_split_treatment double precision NOT NULL DEFAULT 0.5,
    status text NOT NULL CHECK (status IN ('created', 'running', 'stopped', 'completed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    started_at timestamptz,
    stopped_at timestamptz,
    created_by text
);

CREATE TABLE IF NOT EXISTS aso.experiment_metrics (
    experiment_id uuid NOT NULL REFERENCES aso.experiments(id),
    variant text NOT NULL CHECK (variant IN ('control', 'treatment')),
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    queries int NOT NULL DEFAULT 0,
    avg_latency_ms double precision NOT NULL DEFAULT 0,
    p95_latency_ms double precision NOT NULL DEFAULT 0,
    error_count int NOT NULL DEFAULT 0,
    correctness_mismatches int NOT NULL DEFAULT 0,
    PRIMARY KEY (experiment_id, variant, window_start)
);

CREATE INDEX IF NOT EXISTS idx_experiments_status ON aso.experiments(status);
CREATE INDEX IF NOT EXISTS idx_experiments_opt_id ON aso.experiments(optimization_id);

-- Down Migration
-- DROP TABLE aso.experiment_metrics;
-- DROP TABLE aso.experiments;
