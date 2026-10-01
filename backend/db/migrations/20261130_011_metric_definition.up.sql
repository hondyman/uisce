CREATE TABLE IF NOT EXISTS data_explorer.metric_definition (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    bo_id VARCHAR(255) NOT NULL,
    expression JSONB NOT NULL,
    grain_allowlist JSONB NOT NULL DEFAULT '[]'::jsonb,
    format_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    tags TEXT[] DEFAULT '{}',
    is_core BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    archived_at TIMESTAMPTZ,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_metric_definition_tenant ON data_explorer.metric_definition (tenant_id, bo_id);
CREATE INDEX IF NOT EXISTS idx_metric_definition_status ON data_explorer.metric_definition (tenant_id, status) WHERE archived_at IS NULL;
