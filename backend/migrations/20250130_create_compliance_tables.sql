-- Create compliance policies table for versioned CUE rules
CREATE TABLE IF NOT EXISTS compliance_policies (
    policy_id SERIAL PRIMARY KEY,
    version_tag VARCHAR(50) NOT NULL,
    effective_start_date DATE NOT NULL,
    effective_end_date DATE,
    rule_type VARCHAR(50) NOT NULL, -- 'PRE_TRADE' or 'POST_TRADE'
    cue_content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    CONSTRAINT unique_version_type UNIQUE (version_tag, rule_type)
);

CREATE INDEX IF NOT EXISTS idx_compliance_policies_effective_dates 
    ON compliance_policies(effective_start_date, effective_end_date);
CREATE INDEX IF NOT EXISTS idx_compliance_policies_version 
    ON compliance_policies(version_tag);

-- Create compliance events table for real-time validation results
CREATE TABLE IF NOT EXISTS compliance_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trace_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(50) NOT NULL, -- 'PRE_TRADE' or 'POST_TRADE'
    status VARCHAR(50) NOT NULL, -- 'PASS' or 'FAIL'
    rule_version VARCHAR(50) NOT NULL,
    trade_data JSONB NOT NULL,
    error_details JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_compliance_events_trace_id ON compliance_events(trace_id);
CREATE INDEX IF NOT EXISTS idx_compliance_events_created_at ON compliance_events(created_at);
CREATE INDEX IF NOT EXISTS idx_compliance_events_status ON compliance_events(status);
CREATE INDEX IF NOT EXISTS idx_compliance_events_rule_version ON compliance_events(rule_version);

-- Create audit log table for comprehensive compliance trail
CREATE TABLE IF NOT EXISTS compliance_audit_log (
    audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID REFERENCES compliance_events(event_id),
    tenant_id UUID,
    user_id UUID,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100),
    entity_id VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_compliance_audit_log_event_id ON compliance_audit_log(event_id);
CREATE INDEX IF NOT EXISTS idx_compliance_audit_log_tenant_id ON compliance_audit_log(tenant_id);
CREATE INDEX IF NOT EXISTS idx_compliance_audit_log_created_at ON compliance_audit_log(created_at);

-- Insert initial 2025 and 2021 policy versions
-- These will be loaded from filesystem CUE files initially
INSERT INTO compliance_policies (version_tag, effective_start_date, effective_end_date, rule_type, cue_content, created_by)
VALUES 
    ('2025', '2025-01-01', NULL, 'PRE_TRADE', '-- Will be loaded from policy/2025/trade_compliance.cue', 'system'),
    ('2025', '2025-01-01', NULL, 'POST_TRADE', '-- Will be loaded from policy/2025/trade_compliance.cue', 'system'),
    ('2021', '2021-01-01', '2024-12-31', 'PRE_TRADE', '-- Will be loaded from policy/2021/trade_compliance.cue', 'system'),
    ('2021', '2021-01-01', '2024-12-31', 'POST_TRADE', '-- Will be loaded from policy/2021/trade_compliance.cue', 'system')
ON CONFLICT (version_tag, rule_type) DO NOTHING;
