-- ============================================================================
-- REPO 4: THE UISCE GLOBAL INTELLIGENCE STORE & AI BENCHMARKING SCHEMA
-- Target: PostgreSQL (100.84.50.65 / alpha) & StarRocks FE (port 9030)
-- ============================================================================

-- 1. PostgreSQL Schema for Repo 4 Intelligence Metadata
CREATE TABLE IF NOT EXISTS repo4_anonymized_audit_events (
    anonymized_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hashed_user_id VARCHAR(64) NOT NULL,
    industry_vertical VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    event_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS repo4_ai_insights (
    insight_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    anonymized_id UUID REFERENCES repo4_anonymized_audit_events(anonymized_id) ON DELETE CASCADE,
    sentiment_score NUMERIC(3, 2) NOT NULL, -- -1.0 to 1.0
    risk_level VARCHAR(20) NOT NULL, -- LOW, MEDIUM, HIGH, CRITICAL
    keywords TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS repo4_ai_alerts (
    alert_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_hash VARCHAR(64) NOT NULL,
    industry_vertical VARCHAR(50) NOT NULL,
    anomaly_type VARCHAR(100) NOT NULL,
    z_score NUMERIC(5, 2) NOT NULL,
    summary TEXT NOT NULL,
    acknowledged BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Seed initial AI alerts
INSERT INTO repo4_ai_alerts (tenant_hash, industry_vertical, anomaly_type, z_score, summary)
VALUES 
  ('a7f3c9e1208b', 'finance', 'RBAC_ROLE_EXPLOSION', 4.52, 'Tenant assigned 482 roles within a 1-hour window. Industry mean is 12.'),
  ('b9e4d1f8803a', 'healthcare', 'OFF_HOURS_ACCESS_SPIKE', 3.87, 'Unusual surge in portfolio access queries between 02:00-04:00 UTC.')
ON CONFLICT DO NOTHING;

-- 2. StarRocks SQL: External Catalog & Intelligence Views
CREATE DATABASE IF NOT EXISTS iceberg_lakehouse.repo4_intelligence;

USE iceberg_lakehouse.repo4_intelligence;

CREATE OR REPLACE VIEW v_global_industry_benchmarks AS
SELECT 
    industry_vertical,
    COUNT(*) as total_events,
    ROUND(AVG(CASE WHEN action = 'UPDATE' THEN 1 ELSE 0 END) * 100, 2) as role_churn_rate_pct,
    COUNT(DISTINCT hashed_user_id) as active_users_count
FROM iceberg_lakehouse.repo4_intelligence.fact_anonymized_audit_events
GROUP BY industry_vertical;
