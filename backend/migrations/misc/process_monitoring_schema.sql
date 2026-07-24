-- Add process_interventions table for tracking manual interventions
CREATE TABLE IF NOT EXISTS process_interventions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL, -- skip_step, reassign, cancel, retry
    step_name VARCHAR(255),
    new_assignee VARCHAR(255),
    reason TEXT NOT NULL,
    metadata JSONB DEFAULT '{}'::JSONB,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    executed_at TIMESTAMP,
    result TEXT,
    status VARCHAR(50) DEFAULT 'pending' -- pending, executed, failed
);

-- Indexes for fast querying
CREATE INDEX IF NOT EXISTS idx_interventions_workflow ON process_interventions(workflow_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_interventions_tenant ON process_interventions(tenant_id, datasource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_interventions_status ON process_interventions(status, created_at);

-- Add comments
COMMENT ON TABLE process_interventions IS 'Tracks manual interventions in running workflows';
COMMENT ON COLUMN process_interventions.action IS 'Type of intervention: skip_step, reassign, cancel, retry';
COMMENT ON COLUMN process_interventions.status IS 'Execution status: pending, executed, failed';
