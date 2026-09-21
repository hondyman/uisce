CREATE SCHEMA IF NOT EXISTS catalog_ai;

-- 1. Configured AI target destinations per tenant
CREATE TABLE IF NOT EXISTS catalog_ai.ai_bridge_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    vendor_type VARCHAR(64) NOT NULL, -- 'SNOWFLAKE_CORTEX', 'DATABRICKS_GENIE', 'CLAUDE_MCP', 'COPILOT_MCP', 'OPENAI_ASSISTANT'
    target_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Connection parameters stored as encrypted JSON (Rule 7: Vaulted BYOK)
    credentials_vaulted JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    -- Configuration metadata (stage names, catalog IDs, warehouse/SQL warehouse)
    config_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    
    sync_frequency VARCHAR(32) NOT NULL DEFAULT 'MANUAL', -- 'MANUAL', 'ON_PUBLISH', 'HOURLY', 'DAILY'
    last_sync_at TIMESTAMPTZ,
    last_sync_status VARCHAR(32), -- 'SUCCESS', 'FAILED', 'PENDING'
    last_sync_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(tenant_id, vendor_type, target_name)
);

-- 2. Audit ledger for semantic pushes and external agent tool calls
CREATE TABLE IF NOT EXISTS catalog_ai.ai_bridge_sync_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    target_id UUID REFERENCES catalog_ai.ai_bridge_targets(id) ON DELETE SET NULL,
    vendor_type VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL, -- 'EXPORT_PREVIEW', 'STAGE_PUSH', 'MCP_TOOL_QUERY'
    payload_hash VARCHAR(64) NOT NULL,
    artifact_payload TEXT NOT NULL,
    status VARCHAR(32) NOT NULL, -- 'SUCCESS', 'ERROR'
    http_status INT,
    response_body TEXT,
    execution_time_ms INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_ai_bridge_targets_tenant ON catalog_ai.ai_bridge_targets(tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_ai_bridge_sync_logs_lookup ON catalog_ai.ai_bridge_sync_logs(tenant_id, target_id, created_at DESC);
