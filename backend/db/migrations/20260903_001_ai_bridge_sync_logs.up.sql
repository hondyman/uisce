-- backend/db/migrations/20260903_001_ai_bridge_sync_logs.up.sql
-- Creates catalog_ai.ai_bridge_targets and catalog_ai.ai_bridge_sync_logs,
-- which 20260903_002_ai_bridge_ledger_integrity.up.sql depends on. Must
-- sort before that migration so the schema/tables exist when it runs.

CREATE SCHEMA IF NOT EXISTS catalog_ai;

CREATE TABLE IF NOT EXISTS catalog_ai.ai_bridge_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    vendor_type VARCHAR(64) NOT NULL,
    target_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    credentials_vaulted JSONB NOT NULL DEFAULT '{}',
    config_payload JSONB NOT NULL DEFAULT '{}',
    sync_frequency VARCHAR(32) NOT NULL DEFAULT 'MANUAL',
    last_sync_at TIMESTAMPTZ,
    last_sync_status VARCHAR(32),
    last_sync_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    credentials_rotated_at TIMESTAMPTZ,
    UNIQUE (tenant_id, vendor_type, target_name)
);

CREATE INDEX IF NOT EXISTS idx_ai_bridge_targets_tenant
ON catalog_ai.ai_bridge_targets (tenant_id, is_active);

CREATE TABLE IF NOT EXISTS catalog_ai.ai_bridge_sync_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    target_id UUID REFERENCES catalog_ai.ai_bridge_targets(id) ON DELETE SET NULL,
    vendor_type VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL,
    payload_hash VARCHAR(64) NOT NULL,
    artifact_payload TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    http_status INT,
    response_body TEXT,
    execution_time_ms INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_ai_bridge_sync_logs_lookup
ON catalog_ai.ai_bridge_sync_logs (tenant_id, target_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ai_bridge_sync_logs_tenant_created
ON catalog_ai.ai_bridge_sync_logs (tenant_id, created_at);
