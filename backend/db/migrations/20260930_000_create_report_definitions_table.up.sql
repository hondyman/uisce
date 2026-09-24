-- 20260930_000_create_report_definitions_table.up.sql
-- Creates the report_definitions table referenced by the reporting repository
-- and seeded by 20260930_002_seed_core_report_definitions.up.sql

CREATE TABLE IF NOT EXISTS report_definitions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL,
    tenant_datasource_id     UUID,
    report_key              TEXT NOT NULL,
    display_name            TEXT NOT NULL,
    description             TEXT,
    category                TEXT,
    tags                    JSONB DEFAULT '[]'::jsonb,
    report_type             TEXT,
    output_formats          JSONB DEFAULT '["pdf","html","excel"]'::jsonb,
    definition              JSONB NOT NULL DEFAULT '{}'::jsonb,
    parameters_schema       JSONB DEFAULT '[]'::jsonb,
    semantic_cube_id        UUID,
    semantic_query          JSONB,
    version                 INTEGER DEFAULT 1,
    is_current             BOOLEAN DEFAULT TRUE,
    previous_version_id     UUID,
    is_core                BOOLEAN DEFAULT FALSE,
    base_report_id          UUID,
    status                 TEXT DEFAULT 'draft',
    published_at           TIMESTAMPTZ,
    published_by           UUID,
    created_by             TEXT,
    created_at             TIMESTAMPTZ DEFAULT NOW(),
    updated_at             TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_report_definitions_tenant_datasource_key_version
    ON report_definitions (tenant_id, tenant_datasource_id, report_key, version);
CREATE INDEX IF NOT EXISTS idx_report_definitions_tenant_id
    ON report_definitions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_report_definitions_is_current
    ON report_definitions (is_current) WHERE is_current = TRUE;
CREATE INDEX IF NOT EXISTS idx_report_definitions_category
    ON report_definitions (category);
CREATE INDEX IF NOT EXISTS idx_report_definitions_status
    ON report_definitions (status);

ALTER TABLE report_definitions ENABLE ROW LEVEL SECURITY;

CREATE POLICY report_definitions_tenant_isolation_policy ON report_definitions
    USING (((tenant_id)::text = current_setting('uisce.current_tenant'::text, true)));
