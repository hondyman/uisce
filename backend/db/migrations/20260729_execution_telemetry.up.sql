-- Migration: Query Execution Telemetry & Auditing Schema
-- Date: 2026-07-29
-- Purpose: Schema for tracking federated query execution engine telemetry, EXPLAIN plans, and tenant billing metrics.

CREATE SCHEMA IF NOT EXISTS security;

DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'execution_engine_enum') THEN
        CREATE TYPE execution_engine_enum AS ENUM ('POSTGRES_HOT', 'STARROCKS_FEDERATED', 'TRINO_COLD');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS security.query_execution_telemetry (
    telemetry_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    bo_id UUID NOT NULL,
    execution_engine execution_engine_enum NOT NULL,
    effective_time TIMESTAMP WITH TIME ZONE,
    execution_duration_ms INT NOT NULL,
    estimated_bytes_scanned BIGINT,
    query_plan_json JSONB,
    executed_sql TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for tenant-billing and compliance lookups
CREATE INDEX IF NOT EXISTS idx_telemetry_tenant_time ON security.query_execution_telemetry(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_telemetry_bo ON security.query_execution_telemetry(bo_id);
