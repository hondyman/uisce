-- Migration: Create Data Pipeline Definitions Table
CREATE TABLE IF NOT EXISTS data_pipeline_definitions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    mode TEXT NOT NULL DEFAULT 'business_object',
    target_entity TEXT,
    dag_json JSONB NOT NULL DEFAULT '{"nodes":[],"edges":[]}'::jsonb,
    concurrency INT NOT NULL DEFAULT 8,
    batch_size INT NOT NULL DEFAULT 2000,
    error_policy TEXT NOT NULL DEFAULT 'skip_and_log',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_modified_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_data_pipeline_tenant ON data_pipeline_definitions(tenant_id, is_active);
