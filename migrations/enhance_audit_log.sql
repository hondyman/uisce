-- Enhance audit log table with data quality and alerting

-- Add data quality column to audit_log
ALTER TABLE audit_log 
ADD COLUMN IF NOT EXISTS data_quality JSONB DEFAULT '{}';

-- Add version column if not exists
ALTER TABLE audit_log
ADD COLUMN IF NOT EXISTS version TEXT DEFAULT 'v1';

-- Create table for last hash tracking per tenant
CREATE TABLE IF NOT EXISTS tenant_last_hash (
    tenant_id TEXT PRIMARY KEY,
    last_hash TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create table for audit alerts
CREATE TABLE IF NOT EXISTS audit_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id TEXT NOT NULL,
    alert_type TEXT NOT NULL, -- critical, warning, info
    message TEXT NOT NULL,
    resolved BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_audit_log_data_quality ON audit_log USING GIN (data_quality);
CREATE INDEX IF NOT EXISTS idx_audit_alerts_tenant ON audit_alerts(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_alerts_unresolved ON audit_alerts(tenant_id) WHERE NOT resolved;

-- Function to check data quality SLA compliance
CREATE OR REPLACE FUNCTION check_audit_data_quality()
RETURNS TABLE(
    tenant_id TEXT,
    total_entries BIGINT,
    red_freshness_count BIGINT,
    high_null_rate_count BIGINT,
    sla_compliance_pct NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        a.tenant_id,
        COUNT(*) as total_entries,
        COUNT(*) FILTER (WHERE a.data_quality->>'freshness_status' = 'RED') as red_freshness_count,
        COUNT(*) FILTER (WHERE (a.data_quality->>'null_rate')::numeric > 0.10) as high_null_rate_count,
        ROUND(
            100.0 * COUNT(*) FILTER (WHERE a.data_quality->>'freshness_status' != 'RED') / NULLIF(COUNT(*), 0),
            2
        ) as sla_compliance_pct
    FROM audit_log a
    WHERE a.timestamp > NOW() - INTERVAL '30 days'
    GROUP BY a.tenant_id;
END;
$$ LANGUAGE plpgsql;

-- View for audit chain health status
CREATE OR REPLACE VIEW audit_chain_health AS
SELECT 
    a.tenant_id,
    COUNT(*) as total_entries,
    MAX(a.timestamp) as last_audit,
    COUNT(*) FILTER (WHERE a.data_quality->>'freshness_status' = 'GREEN') as green_count,
    COUNT(*) FILTER (WHERE a.data_quality->>'freshness_status' = 'AMBER') as amber_count,
    COUNT(*) FILTER (WHERE a.data_quality->>'freshness_status' = 'RED') as red_count,
    AVG((a.data_quality->>'null_rate')::numeric) as avg_null_rate,
    th.last_hash,
    th.updated_at as last_hash_update
FROM audit_log a
LEFT JOIN tenant_last_hash th ON a.tenant_id = th.tenant_id
WHERE a.timestamp > NOW() - INTERVAL '7 days'
GROUP BY a.tenant_id, th.last_hash, th.updated_at;

COMMENT ON VIEW audit_chain_health IS 'Aggregated health metrics for audit chains by tenant';
