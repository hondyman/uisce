-- 20260930_120_enterprise_mesh_foundations.up.sql
-- Bitemporal Compliance Replay, Autonomous Cost Governor & Schema Drift Sentinel

CREATE SCHEMA IF NOT EXISTS mesh_governance;

-- 1. Bitemporal Compliance Backtest Ledger
CREATE TABLE IF NOT EXISTS mesh_governance.bitemporal_compliance_runs (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    rule_id              UUID NOT NULL,
    rule_name            VARCHAR(255) NOT NULL,
    dynamic_denominator  VARCHAR(100) NOT NULL,
    as_of_start_date     DATE NOT NULL,
    as_of_end_date       DATE NOT NULL,
    days_evaluated       INTEGER NOT NULL,
    breaches_count       INTEGER NOT NULL DEFAULT 0,
    exceptions_count     INTEGER NOT NULL DEFAULT 0,
    merkle_root_hash     VARCHAR(64) NOT NULL,
    status               VARCHAR(50) NOT NULL DEFAULT 'COMPLETED',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compliance_runs_tenant 
ON mesh_governance.bitemporal_compliance_runs (tenant_id, created_at DESC);

-- 2. Materialized View Auto-Governor Recommendations
CREATE TABLE IF NOT EXISTS mesh_governance.materialized_view_recommendations (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    bo_id                UUID NOT NULL,
    suggested_view_name  VARCHAR(150) NOT NULL,
    target_engine        VARCHAR(50) NOT NULL DEFAULT 'StarRocks',
    query_frequency_30d  INTEGER NOT NULL DEFAULT 1,
    avg_runtime_ms       INTEGER NOT NULL DEFAULT 1500,
    projected_runtime_ms INTEGER NOT NULL DEFAULT 10,
    starrocks_ddl        TEXT NOT NULL,
    status               VARCHAR(50) NOT NULL DEFAULT 'PROPOSED', -- PROPOSED, APPLIED, DISMISSED
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Schema Drift Interceptor & Auto-Healing Queue
CREATE TABLE IF NOT EXISTS mesh_governance.schema_drift_events (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    source_table         VARCHAR(150) NOT NULL,
    event_type           VARCHAR(50) NOT NULL, -- COLUMN_DROPPED, COLUMN_ADDED, TYPE_ALTERED
    dropped_column       VARCHAR(100),
    replacement_column   VARCHAR(100),
    confidence_score     NUMERIC(4, 3) NOT NULL DEFAULT 0.950,
    affected_bo_key      VARCHAR(100) NOT NULL,
    affected_field_name  VARCHAR(100) NOT NULL,
    downstream_impact    JSONB NOT NULL DEFAULT '{}'::jsonb, -- {"reports": 4, "liveboards": 1, "apis": 1}
    status               VARCHAR(50) NOT NULL DEFAULT 'PENDING_APPROVAL', -- PENDING_APPROVAL, AUTO_HEALED, DISMISSED
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_drift_events_status 
ON mesh_governance.schema_drift_events (tenant_id, status);
