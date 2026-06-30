-- Create audit_log table for Phase 2.4c: Audit Log Persistence
-- Tracks all action executions with full context for compliance and forensics

CREATE TABLE IF NOT EXISTS ops_audit_log (
    id UUID PRIMARY KEY,
    incident_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    user_role VARCHAR(255) NOT NULL,
    action_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,  -- success, failure, timeout
    parameters JSONB,
    result JSONB,
    error_msg TEXT,
    executed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_ms BIGINT,
    source_ip VARCHAR(45),  -- IPv4 or IPv6
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- For future multi-region support
    region VARCHAR(50)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_audit_log_incident_id ON ops_audit_log (incident_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON ops_audit_log (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_action_type ON ops_audit_log (action_type);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON ops_audit_log (created_at);
CREATE INDEX IF NOT EXISTS idx_audit_log_status ON ops_audit_log (status);

-- Composite indexes for common filter patterns
CREATE INDEX IF NOT EXISTS idx_audit_log_user_created ON ops_audit_log (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_action_created ON ops_audit_log (action_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_incident_created ON ops_audit_log (incident_id, created_at DESC);

-- Add comment for documentation
COMMENT ON TABLE ops_audit_log IS 'Audit log entries for all action executions. Used for compliance, forensics, and operator analytics.';
COMMENT ON COLUMN ops_audit_log.parameters IS 'Redacted action parameters (sensitive fields masked)';
COMMENT ON COLUMN ops_audit_log.result IS 'Sanitized action result (secrets removed)';
COMMENT ON COLUMN ops_audit_log.region IS 'Region where action was executed (future multi-region support)';
