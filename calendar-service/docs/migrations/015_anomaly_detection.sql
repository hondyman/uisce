-- ============================================================================
-- Migration 015: Anomaly Detection
-- ============================================================================
-- Purpose: Track anomalies, alerts, and automated remediation
-- Deploy: psql $DB_URL -f docs/migrations/015_anomaly_detection.sql
-- ============================================================================

-- Create anomaly detection table
CREATE TABLE IF NOT EXISTS calendar.anomalies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Anomaly details
    anomaly_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    description TEXT NOT NULL,
    
    -- Metrics at time of anomaly
    metrics JSONB NOT NULL,
    threshold_violated JSONB NOT NULL,
    
    -- Detection info
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    detection_method VARCHAR(50) NOT NULL, -- ml_based, threshold_based, rule_based
    confidence_score FLOAT NOT NULL,
    
    -- Resolution
    status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open', 'investigating', 'resolved', 'false_positive')),
    resolved_at TIMESTAMPTZ,
    resolved_by UUID REFERENCES public.users(id),
    resolution_notes TEXT,
    
    -- Automated actions
    auto_remediation_attempted BOOLEAN DEFAULT FALSE,
    auto_remediation_success BOOLEAN,
    auto_remediation_action TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create anomaly alerts table
CREATE TABLE IF NOT EXISTS calendar.anomaly_alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    anomaly_id UUID NOT NULL REFERENCES calendar.anomalies(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Alert details
    channel VARCHAR(50) NOT NULL CHECK (channel IN ('email', 'slack', 'pagerduty', 'sms', 'webhook')),
    recipient VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    
    -- Delivery status
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'delivered', 'failed')),
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    error_message TEXT,
    
    -- Engagement
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by VARCHAR(255),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create anomaly thresholds table
CREATE TABLE IF NOT EXISTS calendar.anomaly_thresholds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Threshold definition
    metric_name VARCHAR(100) NOT NULL,
    threshold_type VARCHAR(50) NOT NULL CHECK (threshold_type IN ('absolute', 'percentage', 'standard_deviation')),
    warning_threshold FLOAT NOT NULL,
    critical_threshold FLOAT NOT NULL,
    
    -- Scope
    scope_type VARCHAR(50) DEFAULT 'global' CHECK (scope_type IN ('global', 'tenant', 'user', 'calendar')),
    scope_id UUID,
    
    -- Time window
    time_window_minutes INT NOT NULL DEFAULT 5,
    
    -- Active status
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, metric_name, scope_type, scope_id)
);

-- Create anomaly baselines table (for ML-based detection)
CREATE TABLE IF NOT EXISTS calendar.anomaly_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    
    -- Baseline metrics
    metric_name VARCHAR(100) NOT NULL,
    baseline_value FLOAT NOT NULL,
    baseline_std_dev FLOAT NOT NULL,
    min_value FLOAT NOT NULL,
    max_value FLOAT NOT NULL,
    
    -- Time-based patterns
    hour_of_day INT,
    day_of_week INT,
    
    -- Sample info
    sample_count INT NOT NULL DEFAULT 0,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, metric_name, hour_of_day, day_of_week)
);

-- Indexes for anomaly queries
CREATE INDEX idx_anomalies_tenant_detected 
ON calendar.anomalies(tenant_id, detected_at DESC);

CREATE INDEX idx_anomalies_status 
ON calendar.anomalies(status, detected_at DESC);

CREATE INDEX idx_anomalies_severity 
ON calendar.anomalies(severity, detected_at DESC);

CREATE INDEX idx_anomaly_alerts_anomaly 
ON calendar.anomaly_alerts(anomaly_id);

CREATE INDEX idx_anomaly_alerts_status 
ON calendar.anomaly_alerts(status, created_at DESC);

CREATE INDEX idx_anomaly_thresholds_tenant 
ON calendar.anomaly_thresholds(tenant_id, is_active) 
WHERE is_active = TRUE;

CREATE INDEX idx_anomaly_baselines_tenant 
ON calendar.anomaly_baselines(tenant_id, metric_name);

-- Enable RLS
ALTER TABLE calendar.anomalies ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.anomaly_alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.anomaly_thresholds ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.anomaly_baselines ENABLE ROW LEVEL SECURITY;

CREATE POLICY anomalies_tenant_isolation 
ON calendar.anomalies
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

CREATE POLICY anomaly_alerts_tenant_isolation 
ON calendar.anomaly_alerts
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

CREATE POLICY anomaly_thresholds_tenant_isolation 
ON calendar.anomaly_thresholds
USING (tenant_id IS NULL OR tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

CREATE POLICY anomaly_baselines_tenant_isolation 
ON calendar.anomaly_baselines
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

-- View for active anomalies
CREATE OR REPLACE VIEW calendar.active_anomalies AS
SELECT 
    a.id,
    a.tenant_id,
    a.anomaly_type,
    a.severity,
    a.description,
    a.detected_at,
    a.status,
    a.confidence_score,
    a.auto_remediation_attempted,
    COUNT(aa.id) as alerts_sent
FROM calendar.anomalies a
LEFT JOIN calendar.anomaly_alerts aa ON a.id = aa.anomaly_id
WHERE a.status = 'open'
GROUP BY a.id, a.tenant_id, a.anomaly_type, a.severity, a.description, a.detected_at, a.status, a.confidence_score, a.auto_remediation_attempted
ORDER BY a.severity DESC, a.detected_at DESC;

-- View for anomaly statistics
CREATE OR REPLACE VIEW calendar.anomaly_stats AS
SELECT 
    DATE_TRUNC('day', detected_at) as date,
    anomaly_type,
    severity,
    COUNT(*) as total_anomalies,
    COUNT(*) FILTER (WHERE status = 'resolved') as resolved_anomalies,
    COUNT(*) FILTER (WHERE auto_remediation_attempted = TRUE) as auto_remediated,
    COUNT(*) FILTER (WHERE auto_remediation_success = TRUE) as auto_remediation_success,
    AVG(confidence_score) as avg_confidence,
    AVG(EXTRACT(EPOCH FROM (resolved_at - detected_at))) as avg_resolution_time_seconds
FROM calendar.anomalies
WHERE detected_at > NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('day', detected_at), anomaly_type, severity
ORDER BY date DESC, anomaly_type, severity;

-- Function to create anomaly
CREATE OR REPLACE FUNCTION calendar.create_anomaly(
    p_tenant_id UUID,
    p_anomaly_type VARCHAR,
    p_severity VARCHAR,
    p_description TEXT,
    p_metrics JSONB,
    p_threshold_violated JSONB,
    p_detection_method VARCHAR,
    p_confidence_score FLOAT
) RETURNS UUID AS $$
DECLARE
    v_anomaly_id UUID;
BEGIN
    INSERT INTO calendar.anomalies (
        tenant_id, anomaly_type, severity, description,
        metrics, threshold_violated, detection_method, confidence_score
    ) VALUES (
        p_tenant_id, p_anomaly_type, p_severity, p_description,
        p_metrics, p_threshold_violated, p_detection_method, p_confidence_score
    ) RETURNING id INTO v_anomaly_id;
    
    RETURN v_anomaly_id;
END;
$$ LANGUAGE plpgsql;

-- Comment columns
COMMENT ON COLUMN calendar.anomalies.auto_remediation_attempted IS 'Whether automated remediation was attempted';
COMMENT ON COLUMN calendar.anomaly_thresholds.time_window_minutes IS 'Time window for threshold evaluation';
COMMENT ON COLUMN calendar.anomaly_baselines.baseline_std_dev IS 'Standard deviation for ML-based anomaly detection';
