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
    region VARCHAR(50),
    
    -- Indexes for common queries
    INDEX idx_audit_log_incident_id (incident_id),
    INDEX idx_audit_log_user_id (user_id),
    INDEX idx_audit_log_action_type (action_type),
    INDEX idx_audit_log_created_at (created_at),
    INDEX idx_audit_log_status (status),
    
    -- Composite indexes for common filter patterns
    INDEX idx_audit_log_user_created (user_id, created_at DESC),
    INDEX idx_audit_log_action_created (action_type, created_at DESC),
    INDEX idx_audit_log_incident_created (incident_id, created_at DESC)
);

-- Add comment for documentation
COMMENT ON TABLE ops_audit_log IS 'Audit log entries for all action executions. Used for compliance, forensics, and operator analytics.';
COMMENT ON COLUMN ops_audit_log.parameters IS 'Redacted action parameters (sensitive fields masked)';
COMMENT ON COLUMN ops_audit_log.result IS 'Sanitized action result (secrets removed)';
COMMENT ON COLUMN ops_audit_log.region IS 'Region where action was executed (future multi-region support)';
