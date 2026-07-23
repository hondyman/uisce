-- +goose Up
-- Persistent storage for semantic roles, governance metadata, and bundle assignments
CREATE TABLE IF NOT EXISTS semantic_roles (
    id uuid PRIMARY KEY,
    tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
    name text NOT NULL,
    normalized_name text NOT NULL,
    display_name text NOT NULL,
    description text,
    version text NOT NULL,
    status text NOT NULL,
    role_type text NOT NULL,
    owner text NOT NULL,
    scope text NOT NULL,
    tags jsonb NOT NULL DEFAULT '[]'::jsonb,
    attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    policies jsonb NOT NULL DEFAULT '[]'::jsonb,
    permissions jsonb NOT NULL DEFAULT '[]'::jsonb,
    attribute_constraints jsonb NOT NULL DEFAULT '[]'::jsonb,
    members jsonb NOT NULL DEFAULT '[]'::jsonb,
    bundle_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
    audit_trail jsonb NOT NULL DEFAULT '[]'::jsonb,
    audit_metadata jsonb NOT NULL,
    lifecycle jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, normalized_name)
);

CREATE INDEX IF NOT EXISTS idx_semantic_roles_tenant ON semantic_roles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_semantic_roles_status ON semantic_roles(status);
CREATE INDEX IF NOT EXISTS idx_semantic_roles_type ON semantic_roles(role_type);
CREATE INDEX IF NOT EXISTS idx_semantic_roles_owner ON semantic_roles(owner);

-- +goose Down
DROP TABLE IF EXISTS semantic_roles;
