-- Add database_name column to tenants table to track which database a tenant uses
-- This is used by the provisioning workflow to know which database to clone from
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS database_name TEXT;

-- Create index for gold_copy lookups
CREATE INDEX IF NOT EXISTS idx_tenants_gold_copy ON tenants(gold_copy) WHERE gold_copy = true;

-- Create provisioning_jobs table to track tenant provisioning workflows
CREATE TABLE IF NOT EXISTS provisioning_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id TEXT NOT NULL,
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    instance_id UUID REFERENCES tenant_instance(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    UNIQUE(workflow_id)
);

CREATE INDEX IF NOT EXISTS idx_provisioning_jobs_status ON provisioning_jobs(status);
CREATE INDEX IF NOT EXISTS idx_provisioning_jobs_tenant_id ON provisioning_jobs(tenant_id);
