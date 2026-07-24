-- Audit Trail Tables for Compliance
-- These tables provide comprehensive audit logging for regulatory compliance

-- Main audit events table
CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'medium',
    user_id VARCHAR(255),
    tenant_id VARCHAR(255),
    session_id VARCHAR(255),
    resource_id VARCHAR(255),
    resource_type VARCHAR(100),
    action VARCHAR(100),
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    details JSONB,
    old_values JSONB,
    new_values JSONB,
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    compliance_flags TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_user_id ON audit_events(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_id ON audit_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_severity ON audit_events(severity);
CREATE INDEX IF NOT EXISTS idx_audit_events_resource_type ON audit_events(resource_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_resource_id ON audit_events(resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_ip_address ON audit_events(ip_address);
CREATE INDEX IF NOT EXISTS idx_audit_events_success ON audit_events(success);

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_audit_events_user_time ON audit_events(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_time ON audit_events(tenant_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_type_time ON audit_events(event_type, timestamp DESC);

-- Audit event summaries table for reporting
CREATE TABLE IF NOT EXISTS audit_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    summary_date DATE NOT NULL,
    tenant_id VARCHAR(255),
    total_events BIGINT NOT NULL DEFAULT 0,
    events_by_type JSONB,
    events_by_severity JSONB,
    events_by_user JSONB,
    critical_events BIGINT NOT NULL DEFAULT 0,
    compliance_violations BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(summary_date, tenant_id)
);

-- Compliance reports table
CREATE TABLE IF NOT EXISTS compliance_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_type VARCHAR(100) NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    generated_by VARCHAR(255) NOT NULL,
    summary JSONB,
    violations JSONB,
    recommendations TEXT[],
    status VARCHAR(50) NOT NULL DEFAULT 'generated',
    file_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit retention policies table
CREATE TABLE IF NOT EXISTS audit_retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL,
    retention_days INTEGER NOT NULL,
    archive_after_days INTEGER,
    delete_after_days INTEGER,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_type)
);

-- Audit alerts configuration table
CREATE TABLE IF NOT EXISTS audit_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    conditions JSONB,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- User sessions table for session tracking
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255),
    session_id VARCHAR(255) NOT NULL UNIQUE,
    ip_address INET,
    user_agent TEXT,
    login_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    logout_time TIMESTAMPTZ,
    last_activity TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for user sessions
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_session_id ON user_sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_active ON user_sessions(is_active) WHERE is_active = true;

-- Data access log table for detailed data access tracking
CREATE TABLE IF NOT EXISTS data_access_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255),
    session_id VARCHAR(255),
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    query_parameters JSONB,
    accessed_fields TEXT[],
    record_count INTEGER,
    access_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT
);

-- Indexes for data access log
CREATE INDEX IF NOT EXISTS idx_data_access_log_user_time ON data_access_log(user_id, access_time DESC);
CREATE INDEX IF NOT EXISTS idx_data_access_log_resource ON data_access_log(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_data_access_log_time ON data_access_log(access_time DESC);

-- Insert default retention policies
INSERT INTO audit_retention_policies (event_type, retention_days, archive_after_days, delete_after_days)
VALUES
    ('login', 365, 90, 365),
    ('logout', 365, 90, 365),
    ('data_access', 2555, 365, 2555), -- 7 years for data access
    ('data_modify', 2555, 365, 2555),
    ('calculation_run', 1825, 365, 1825), -- 5 years for calculations
    ('config_change', 2555, 365, 2555),
    ('policy_violation', 2555, 365, 2555),
    ('system_start', 365, 90, 365),
    ('system_stop', 365, 90, 365)
ON CONFLICT (event_type) DO NOTHING;

-- Insert default audit alerts
INSERT INTO audit_alerts (name, description, event_type, severity, conditions, enabled)
VALUES
    ('Multiple Failed Logins', 'Alert when user has multiple failed login attempts', 'login_failed', 'high',
     '{"threshold": 5, "time_window_minutes": 15}', true),
    ('Policy Violations', 'Alert on policy violations', 'policy_violation', 'critical',
     '{"immediate_alert": true}', true),
    ('Unauthorized Data Access', 'Alert on unauthorized data access attempts', 'access_denied', 'high',
     '{"immediate_alert": true}', true),
    ('Configuration Changes', 'Alert on system configuration changes', 'config_change', 'medium',
     '{"immediate_alert": true}', true)
ON CONFLICT DO NOTHING;
