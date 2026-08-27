-- Compliance Pre-Trade & Analytical Query Audit Ledger (SEC Rule 17a-4 compliant)

CREATE SCHEMA IF NOT EXISTS audit;

CREATE TABLE IF NOT EXISTS audit.analytical_query_execution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    query_hash CHAR(64) NOT NULL,
    complexity_score NUMERIC(5,2) NOT NULL,
    circuit_breaker_status VARCHAR(32) NOT NULL, -- 'ALLOWED', 'WARNING', 'FORBIDDEN'
    join_count INT NOT NULL DEFAULT 0,
    missing_partition BOOLEAN NOT NULL DEFAULT FALSE,
    cross_tier_federated BOOLEAN NOT NULL DEFAULT FALSE,
    execution_engine VARCHAR(64) NOT NULL, -- 'StarRocks', 'DataFusion', 'Postgres', 'ArrowVector'
    evaluation_time_us BIGINT NOT NULL,
    execution_duration_ms NUMERIC(10,3),
    scanned_bytes BIGINT DEFAULT 0,
    plan_json JSONB,
    previous_hash CHAR(64) NOT NULL DEFAULT '0000000000000000000000000000000000000000000000000000000000000000',
    record_hash CHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_query_audit_tenant_created ON audit.analytical_query_execution_logs (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_query_audit_hash ON audit.analytical_query_execution_logs (query_hash);
CREATE INDEX IF NOT EXISTS idx_query_audit_status ON audit.analytical_query_execution_logs (circuit_breaker_status);
