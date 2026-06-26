-- bp-backend/db/migrations/001_create_workflow_tables.up.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE workflow_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    active_version_id UUID, -- Can be null if no version is active
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workflow_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    definition_id UUID NOT NULL REFERENCES workflow_definitions(id) ON DELETE CASCADE,
    version_tag VARCHAR(100) NOT NULL,
    definition_snapshot JSONB NOT NULL,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add a foreign key constraint from workflow_definitions to workflow_versions
ALTER TABLE workflow_definitions
ADD CONSTRAINT fk_active_version
FOREIGN KEY (active_version_id)
REFERENCES workflow_versions(id)
ON DELETE SET NULL;

CREATE TABLE workflow_nodes (
    id UUID PRIMARY KEY,
    version_id UUID NOT NULL REFERENCES workflow_versions(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    config_json JSONB,
    ui_metadata_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workflow_edges (
    id UUID PRIMARY KEY,
    version_id UUID NOT NULL REFERENCES workflow_versions(id) ON DELETE CASCADE,
    source_node_id UUID NOT NULL REFERENCES workflow_nodes(id) ON DELETE CASCADE,
    target_node_id UUID NOT NULL REFERENCES workflow_nodes(id) ON DELETE CASCADE,
    condition_expression VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflow_definitions_tenant_id ON workflow_definitions(tenant_id);
CREATE INDEX idx_workflow_versions_definition_id ON workflow_versions(definition_id);
CREATE UNIQUE INDEX idx_workflow_versions_published ON workflow_versions(definition_id) WHERE is_published = TRUE;
