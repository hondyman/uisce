-- Creates workflow_abac_policies table used by compliance-engine
-- Generated: 2025-10-31

CREATE TABLE IF NOT EXISTS workflow_abac_policies (
    id UUID PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    datasource_id VARCHAR(255) NOT NULL,
    workflow_type VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    resource_pattern VARCHAR(255) NOT NULL,
    subject_rules JSONB,
    environment_rules JSONB,
    risk_level VARCHAR(50),
    requires_approval BOOLEAN DEFAULT false,
    approval_roles JSONB,
    time_restrictions JSONB,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Optional indexes to help queries by tenant/datasource/workflow
CREATE INDEX IF NOT EXISTS idx_workflow_abac_policies_tenant_datasource ON workflow_abac_policies (tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_workflow_abac_policies_workflow_type ON workflow_abac_policies (workflow_type);
CREATE INDEX IF NOT EXISTS idx_workflow_abac_policies_enabled ON workflow_abac_policies (enabled);
