-- Migration: data_explorer saved queries CRUD
-- Date: 20260826

CREATE SCHEMA IF NOT EXISTS data_explorer;

CREATE TABLE IF NOT EXISTS data_explorer.saved_query (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    source_kind TEXT NOT NULL DEFAULT 'business_object',
    source_id TEXT NOT NULL,
    binding_id TEXT,
    query_state JSONB NOT NULL DEFAULT '{}',
    tags TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_de_saved_query_tenant_user
    ON data_explorer.saved_query(tenant_id, user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_de_saved_query_tenant_name
    ON data_explorer.saved_query(tenant_id, name);

ALTER TABLE data_explorer.saved_query ENABLE ROW LEVEL SECURITY;

CREATE POLICY de_saved_query_tenant_policy ON data_explorer.saved_query
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::uuid);

COMMENT ON TABLE data_explorer.saved_query IS 'Persisted saved queries from the data explorer query builder';
