-- Migration: 20260729000008_mcp_tool_registry.up.sql
-- Phase 8: MCP tool registry with role-based access control
-- Idempotent: creates table only if not exists

CREATE TABLE IF NOT EXISTS mcp_tool_registry (
    id              UUID        NOT NULL DEFAULT gen_random_uuid(),
    tool_name       TEXT        NOT NULL,
    display_name    TEXT,
    description     TEXT,
    allowed_roles   TEXT[]      NOT NULL DEFAULT '{}',
    is_active       BOOLEAN     NOT NULL DEFAULT true,
    tenant_id       UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT mcp_tool_registry_pkey PRIMARY KEY (id),
    CONSTRAINT mcp_tool_registry_tool_name_tenant_key UNIQUE (tool_name, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_mcp_tool_registry_tool_name ON mcp_tool_registry(tool_name);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_registry_roles ON mcp_tool_registry USING gin(allowed_roles);

-- Seed Gold Copy tenant tools
INSERT INTO mcp_tool_registry (tool_name, display_name, description, allowed_roles, tenant_id)
VALUES
    ('create_business_object', 'Create Business Object', 'Creates a new business object definition', ARRAY['DATA_ENGINEER', 'SYSTEM_ADMIN'], '00000000-0000-0000-0000-000000000000'),
    ('update_business_object', 'Update Business Object', 'Updates an existing business object', ARRAY['DATA_ENGINEER', 'SYSTEM_ADMIN'], '00000000-0000-0000-0000-000000000000'),
    ('delete_business_object', 'Delete Business Object', 'Deletes a business object', ARRAY['SYSTEM_ADMIN'], '00000000-0000-0000-0000-000000000000'),
    ('execute_ai_rule', 'Execute AI Rule', 'Executes an AI-generated rule', ARRAY['TRADER', 'PORTFOLIO_MANAGER', 'DATA_ENGINEER'], '00000000-0000-0000-0000-000000000000'),
    ('query_semantic_layer', 'Query Semantic Layer', 'Queries the semantic layer', ARRAY['TRADER', 'PORTFOLIO_MANAGER', 'COMPLIANCE_OFFICER', 'DATA_STEWARD', 'DATA_ENGINEER', 'BUSINESS_ANALYST'], '00000000-0000-0000-0000-000000000000'),
    ('manage_personalization', 'Manage Personalization', 'Updates user personalization profiles', ARRAY['TRADER', 'PORTFOLIO_MANAGER', 'BUSINESS_ANALYST'], '00000000-0000-0000-0000-000000000000')
ON CONFLICT (tool_name, tenant_id) DO NOTHING;
