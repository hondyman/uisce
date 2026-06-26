-- ============================================================================
-- Migration 014: Sync Optimization
-- ============================================================================
-- Purpose: Track sync optimization recommendations and cost savings
-- Deploy: psql $DB_URL -f docs/migrations/014_sync_optimization.sql
-- ============================================================================

-- Add optimization tracking to sync_jobs
ALTER TABLE calendar.sync_jobs 
ADD COLUMN IF NOT EXISTS optimal_scheduled_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS actual_scheduled_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS batch_size INT,
ADD COLUMN IF NOT EXISTS resource_profile VARCHAR(50),
ADD COLUMN IF NOT EXISTS predicted_duration_seconds FLOAT,
ADD COLUMN IF NOT EXISTS actual_duration_seconds FLOAT,
ADD COLUMN IF NOT EXISTS cost_estimate_cents INT,
ADD COLUMN IF NOT EXISTS optimization_score FLOAT;

-- Create sync optimization recommendations table
CREATE TABLE IF NOT EXISTS calendar.sync_optimization_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    calendar_id UUID NOT NULL REFERENCES calendar.calendars(id) ON DELETE CASCADE,
    user_id UUID REFERENCES public.users(id) ON DELETE SET NULL,
    
    -- Optimization recommendations
    recommended_sync_time TIMESTAMPTZ NOT NULL,
    recommended_batch_size INT NOT NULL,
    recommended_resource_profile VARCHAR(50) NOT NULL,
    predicted_duration_seconds FLOAT NOT NULL,
    predicted_cost_cents INT NOT NULL,
    
    -- ML model info
    model_version VARCHAR(50) NOT NULL,
    confidence_score FLOAT NOT NULL,
    features_used JSONB NOT NULL,
    
    -- Tracking
    accepted BOOLEAN DEFAULT FALSE,
    accepted_at TIMESTAMPTZ,
    actual_duration_seconds FLOAT,
    actual_cost_cents INT,
    savings_cents INT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_batch_size CHECK (recommended_batch_size > 0 AND recommended_batch_size <= 1000),
    CONSTRAINT chk_confidence CHECK (confidence_score >= 0 AND confidence_score <= 1)
);

-- Create sync cost tracking table
CREATE TABLE IF NOT EXISTS calendar.sync_cost_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    sync_job_id UUID REFERENCES calendar.sync_jobs(id) ON DELETE SET NULL,
    
    -- Cost components
    api_calls INT NOT NULL DEFAULT 0,
    api_call_cost_cents INT NOT NULL DEFAULT 0,
    compute_time_seconds FLOAT NOT NULL DEFAULT 0,
    compute_cost_cents INT NOT NULL DEFAULT 0,
    storage_mb FLOAT NOT NULL DEFAULT 0,
    storage_cost_cents INT NOT NULL DEFAULT 0,
    data_transfer_mb FLOAT NOT NULL DEFAULT 0,
    data_transfer_cost_cents INT NOT NULL DEFAULT 0,
    
    -- Total cost
    total_cost_cents INT NOT NULL DEFAULT 0,
    
    -- Timestamps
    sync_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create sync performance baseline table
CREATE TABLE IF NOT EXISTS calendar.sync_performance_baseline (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    calendar_id UUID REFERENCES calendar.calendars(id) ON DELETE CASCADE,
    
    -- Time-based patterns
    hour_of_day INT NOT NULL,
    day_of_week INT NOT NULL,
    
    -- Performance metrics
    avg_duration_seconds FLOAT NOT NULL,
    avg_success_rate FLOAT NOT NULL,
    avg_events_processed INT NOT NULL,
    avg_api_calls INT NOT NULL,
    avg_cost_cents INT NOT NULL,
    
    -- Sample size
    sample_count INT NOT NULL DEFAULT 0,
    
    -- Last updated
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, calendar_id, hour_of_day, day_of_week)
);

-- Indexes for optimization queries
CREATE INDEX idx_sync_optimization_recommendations_tenant 
ON calendar.sync_optimization_recommendations(tenant_id, created_at DESC);

CREATE INDEX idx_sync_optimization_recommendations_accepted 
ON calendar.sync_optimization_recommendations(tenant_id, accepted) 
WHERE accepted = FALSE;

CREATE INDEX idx_sync_cost_tracking_date 
ON calendar.sync_cost_tracking(tenant_id, sync_date DESC);

CREATE INDEX idx_sync_performance_baseline_time 
ON calendar.sync_performance_baseline(tenant_id, hour_of_day, day_of_week);

-- Enable RLS
ALTER TABLE calendar.sync_optimization_recommendations ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.sync_cost_tracking ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.sync_performance_baseline ENABLE ROW LEVEL SECURITY;

CREATE POLICY sync_optimization_tenant_isolation 
ON calendar.sync_optimization_recommendations
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

CREATE POLICY sync_cost_tracking_tenant_isolation 
ON calendar.sync_cost_tracking
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

CREATE POLICY sync_performance_baseline_tenant_isolation 
ON calendar.sync_performance_baseline
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

-- View for cost savings tracking
CREATE OR REPLACE VIEW calendar.sync_cost_savings AS
SELECT 
    DATE_TRUNC('day', sor.created_at) as date,
    sor.tenant_id,
    COUNT(*) as recommendations_made,
    COUNT(*) FILTER (WHERE sor.accepted = TRUE) as recommendations_accepted,
    SUM(sor.savings_cents) FILTER (WHERE sor.accepted = TRUE) as total_savings_cents,
    AVG(sor.savings_cents) FILTER (WHERE sor.accepted = TRUE) as avg_savings_per_sync,
    AVG(sor.confidence_score) as avg_confidence_score
FROM calendar.sync_optimization_recommendations sor
WHERE sor.created_at > NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('day', sor.created_at), sor.tenant_id
ORDER BY date DESC;

-- View for sync performance by time
CREATE OR REPLACE VIEW calendar.sync_performance_by_hour AS
SELECT 
    hour_of_day,
    day_of_week,
    AVG(avg_duration_seconds) as avg_duration,
    AVG(avg_success_rate) as avg_success_rate,
    AVG(avg_cost_cents) as avg_cost,
    SUM(sample_count) as total_samples
FROM calendar.sync_performance_baseline
GROUP BY hour_of_day, day_of_week
ORDER BY hour_of_day, day_of_week;

-- Function to calculate sync cost
CREATE OR REPLACE FUNCTION calendar.calculate_sync_cost(
    p_api_calls INT,
    p_compute_seconds FLOAT,
    p_storage_mb FLOAT,
    p_transfer_mb FLOAT
) RETURNS INT AS $$
DECLARE
    v_total_cents INT;
BEGIN
    -- Cost rates (adjust based on your actual cloud provider)
    -- API calls: $0.0001 per call
    -- Compute: $0.00001667 per second (Lambda)
    -- Storage: $0.023 per GB-month
    -- Data transfer: $0.09 per GB
    
    v_total_cents := 
        (p_api_calls * 0.01) + -- API calls in cents
        (p_compute_seconds * 0.001667) + -- Compute in cents
        (p_storage_mb * 0.000023) + -- Storage in cents
        (p_transfer_mb * 0.009); -- Transfer in cents
    
    RETURN v_total_cents;
END;
$$ LANGUAGE plpgsql;

-- Comment columns
COMMENT ON COLUMN calendar.sync_jobs.optimal_scheduled_at IS 'ML-recommended optimal sync time';
COMMENT ON COLUMN calendar.sync_jobs.optimization_score IS 'How well optimization matched prediction (0-1)';
COMMENT ON COLUMN calendar.sync_optimization_recommendations.savings_cents IS 'Cost savings from following recommendation';
