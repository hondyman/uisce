-- Migration: Channel Telemetry & Billing Audit Schema
-- Date: 2026-07-30
-- Purpose: Support channel auditing (REST, JDBC_PGWIRE, MCP_AI, GRAPHQL) and compute/billing metrics.

CREATE SCHEMA IF NOT EXISTS security;

DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'request_channel_enum') THEN
        CREATE TYPE request_channel_enum AS ENUM ('REST_API', 'JDBC_PGWIRE', 'MCP_AI', 'GRAPHQL', 'UI_DASHBOARD');
    END IF;
END $$;

ALTER TABLE security.query_execution_telemetry 
ADD COLUMN IF NOT EXISTS channel request_channel_enum DEFAULT 'REST_API',
ADD COLUMN IF NOT EXISTS client_ip VARCHAR(45),
ADD COLUMN IF NOT EXISTS compute_units_billed NUMERIC(12, 4) DEFAULT 1.0000,
ADD COLUMN IF NOT EXISTS estimated_cost_usd NUMERIC(10, 6) DEFAULT 0.000100;

CREATE INDEX IF NOT EXISTS idx_telemetry_channel_tenant ON security.query_execution_telemetry(tenant_id, channel, created_at DESC);
