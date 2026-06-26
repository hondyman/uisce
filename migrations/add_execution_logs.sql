-- Create execution_logs table for immutable audit trail of all executions
CREATE TABLE IF NOT EXISTS execution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL, -- 'calculation', 'workflow', 'event'
    status VARCHAR(50) NOT NULL, -- 'started', 'completed', 'failed'
    engine VARCHAR(50) NOT NULL, -- 'internal', 'cube', 'spark', 'temporal'
    payload JSONB, -- Input arguments / context
    result JSONB, -- Output result or error details
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    user_id UUID, -- Optional, if triggered by user
    tenant_id UUID, -- Optional, for multi-tenancy
    error_message TEXT,
    
    -- Metadata for easier querying
    calculation_id UUID, -- Link to specific calculation definition if applicable
    workflow_id VARCHAR(255), -- Link to Temporal workflow ID if applicable
    run_id VARCHAR(255), -- Link to Temporal run ID if applicable
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_execution_logs_event_type ON execution_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_execution_logs_status ON execution_logs(status);
CREATE INDEX IF NOT EXISTS idx_execution_logs_engine ON execution_logs(engine);
CREATE INDEX IF NOT EXISTS idx_execution_logs_started_at ON execution_logs(started_at);
CREATE INDEX IF NOT EXISTS idx_execution_logs_user_id ON execution_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_execution_logs_tenant_id ON execution_logs(tenant_id);
