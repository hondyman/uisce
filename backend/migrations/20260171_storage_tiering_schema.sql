-- Storage Tiering Schema
-- Epic 22: Data Layer Intelligence - Phase 2

CREATE TABLE IF NOT EXISTS storage_tiering_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rules JSONB NOT NULL DEFAULT '[]', -- [{tableName, condition, targetTier, rationale, dataVolume, costSavings}]
    summary TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, migrating, completed, dismissed
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for tenant-specific plans
CREATE INDEX IF NOT EXISTS idx_storage_tiering_plans_tenant ON storage_tiering_plans(tenant_id);
CREATE INDEX IF NOT EXISTS idx_storage_tiering_plans_status ON storage_tiering_plans(status) WHERE status = 'pending';

-- Trigger for updated_at
DROP TRIGGER IF EXISTS tr_storage_tiering_plans_updated ON storage_tiering_plans;
CREATE TRIGGER tr_storage_tiering_plans_updated
    BEFORE UPDATE ON storage_tiering_plans
    FOR EACH ROW
    EXECUTE FUNCTION update_scheduler_updated_at();
