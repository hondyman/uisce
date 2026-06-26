-- Audit Logs Table for Report Builder Phase 3
-- Created automatically by Docker Compose initialization

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    user_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity VARCHAR(500) NOT NULL,
    old_value JSONB,
    new_value JSONB,
    status VARCHAR(50) NOT NULL,
    error_msg TEXT,
    duration_ms BIGINT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs(entity);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_status ON audit_logs(status);

-- Partitioning by month (for large audit logs)
-- Optional: Enable for production systems with high volume
-- SELECT create_monthly_partitions('audit_logs', CURRENT_DATE);

-- Grant permissions
GRANT SELECT ON audit_logs TO postgres;
GRANT INSERT ON audit_logs TO postgres;
GRANT UPDATE ON audit_logs TO postgres;
