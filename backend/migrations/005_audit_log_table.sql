-- Phase 3: Audit Log Table for Rules Engine
-- Tracks all mutations and governance actions

CREATE TABLE IF NOT EXISTS edm.audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    actor_id UUID NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource_id UUID NOT NULL,
    resource_type VARCHAR(50),
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for audit queries
CREATE INDEX idx_audit_log_tenant ON edm.audit_log(tenant_id, created_at DESC);
CREATE INDEX idx_audit_log_actor ON edm.audit_log(actor_id, created_at DESC);
CREATE INDEX idx_audit_log_resource ON edm.audit_log(resource_id);
CREATE INDEX idx_audit_log_action ON edm.audit_log(action);

-- Add comment explaining table
COMMENT ON TABLE edm.audit_log IS 'Immutable audit log tracking all rule mutations, approvals, and promotions for compliance and debugging';

-- Row-level security
ALTER TABLE edm.audit_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY audit_log_tenant_isolation ON edm.audit_log
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
