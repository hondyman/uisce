CREATE SCHEMA IF NOT EXISTS migration;

CREATE TABLE IF NOT EXISTS migration.parity_results (
    id BIGSERIAL PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    query_id TEXT NOT NULL,
    status TEXT NOT NULL,
    max_delta DOUBLE PRECISION NOT NULL,
    diff TEXT,
    legacy_hash TEXT NOT NULL,
    cube_hash TEXT NOT NULL,
    metadata JSONB,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS parity_results_tenant_query_idx
    ON migration.parity_results (tenant_id, query_id);

CREATE INDEX IF NOT EXISTS parity_results_observed_at_idx
    ON migration.parity_results (observed_at DESC);
