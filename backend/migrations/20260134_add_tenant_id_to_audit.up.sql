-- +goose Up
ALTER TABLE workflow_audit_log ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_audit_tenant_id ON workflow_audit_log(tenant_id);

-- +goose Down
ALTER TABLE workflow_audit_log DROP COLUMN IF EXISTS tenant_id;
