-- Phase 3.5: Regional Metrics & SLA Tracking
-- Track per-region performance metrics, health scores, and SLA compliance
-- Provides observability for multi-region operations on PostgreSQL

-- CREATE TABLE IF NOT EXISTS for regional metrics
CREATE TABLE IF NOT EXISTS regional_metrics (
    id UUID PRIMARY KEY,
    region VARCHAR(50) NOT NULL,
    error_rate FLOAT NOT NULL DEFAULT 0.0,
    p50_latency_ms INTEGER NOT NULL DEFAULT 0,
    p95_latency_ms INTEGER NOT NULL DEFAULT 0,
    p99_latency_ms INTEGER NOT NULL DEFAULT 0,
    availability_pct FLOAT NOT NULL DEFAULT 100.0,
    request_count BIGINT NOT NULL DEFAULT 0,
    incident_count INTEGER NOT NULL DEFAULT 0,
    components JSONB DEFAULT '{}'::jsonb,  -- Composite scores: CPU, memory, etc.
    computed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(region, computed_at)
);

-- Create indexes for regional metrics
CREATE INDEX IF NOT EXISTS idx_regional_metrics_region ON regional_metrics(region);
CREATE INDEX IF NOT EXISTS idx_regional_metrics_region_computed_at ON regional_metrics(region, computed_at DESC);
CREATE INDEX IF NOT EXISTS idx_regional_metrics_created_at ON regional_metrics(created_at DESC);

-- CREATE TABLE IF NOT EXISTS for regional health scores
CREATE TABLE IF NOT EXISTS regional_health (
    id UUID PRIMARY KEY,
    region VARCHAR(50) NOT NULL UNIQUE,
    health_score INTEGER NOT NULL DEFAULT 50 CHECK (health_score >= 0 AND health_score <= 100),
    status VARCHAR(20) NOT NULL DEFAULT 'degraded' CHECK (status IN ('healthy', 'degraded', 'critical')),
    computed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for regional health
CREATE INDEX IF NOT EXISTS idx_regional_health_region ON regional_health(region);
CREATE INDEX IF NOT EXISTS idx_regional_health_status ON regional_health(status);
CREATE INDEX IF NOT EXISTS idx_regional_health_updated_at ON regional_health(updated_at DESC);

-- CREATE TABLE IF NOT EXISTS for Regional SLA definitions
CREATE TABLE IF NOT EXISTS regional_sla (
    id UUID PRIMARY KEY,
    region VARCHAR(50) NOT NULL UNIQUE,
    availability_sla_pct FLOAT NOT NULL DEFAULT 99.9 CHECK (availability_sla_pct > 0 AND availability_sla_pct <= 100),
    p95_latency_sla_ms INTEGER NOT NULL DEFAULT 500 CHECK (p95_latency_sla_ms > 0),
    error_rate_sla_pct FLOAT NOT NULL DEFAULT 1.0 CHECK (error_rate_sla_pct >= 0 AND error_rate_sla_pct < 50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for regional SLA
CREATE INDEX IF NOT EXISTS idx_regional_sla_region ON regional_sla(region);

-- CREATE TABLE IF NOT EXISTS for tracking SLA compliance
CREATE TABLE IF NOT EXISTS regional_sla_status (
    id UUID PRIMARY KEY,
    region VARCHAR(50) NOT NULL,
    sla_id UUID NOT NULL REFERENCES regional_sla(id) ON DELETE CASCADE,
    availability_met BOOLEAN NOT NULL DEFAULT false,
    latency_met BOOLEAN NOT NULL DEFAULT false,
    error_rate_met BOOLEAN NOT NULL DEFAULT false,
    compliance_pct FLOAT NOT NULL DEFAULT 0.0 CHECK (compliance_pct >= 0 AND compliance_pct <= 100),
    checked_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for SLA status
CREATE INDEX IF NOT EXISTS idx_regional_sla_status_region ON regional_sla_status(region);
CREATE INDEX IF NOT EXISTS idx_regional_sla_status_sla_id ON regional_sla_status(sla_id);
CREATE INDEX IF NOT EXISTS idx_regional_sla_status_region_checked_at ON regional_sla_status(region, checked_at DESC);

-- Add comments
COMMENT ON TABLE regional_metrics IS 'Per-region performance metrics computed from monitoring data';
COMMENT ON TABLE regional_health IS 'Per-region health scores derived from metrics and incidents';
COMMENT ON TABLE regional_sla IS 'Service level agreements defined per geographic region';
COMMENT ON TABLE regional_sla_status IS 'Historical tracking of regional SLA compliance';

COMMENT ON COLUMN regional_metrics.components IS 'JSON object with component scores: {cpu: 85.5, memory: 90.2, disk: 75.0}';
COMMENT ON COLUMN regional_health.health_score IS 'Composite health score 0-100 based on metrics';
COMMENT ON COLUMN regional_health.status IS 'Categorical status: healthy (≥80), degraded (50-79), critical (<50)';
