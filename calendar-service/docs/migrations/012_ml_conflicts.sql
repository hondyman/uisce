-- ============================================================================
-- Migration 012: ML Conflict Resolution
-- ============================================================================
-- Purpose: Track ML recommendations and auto-resolution
-- ============================================================================

-- Add ML recommendation tracking to sync_conflicts
ALTER TABLE calendar.sync_conflicts 
ADD COLUMN IF NOT EXISTS ml_recommendation VARCHAR(50),
ADD COLUMN IF NOT EXISTS ml_confidence FLOAT,
ADD COLUMN IF NOT EXISTS ml_reasoning TEXT,
ADD COLUMN IF NOT EXISTS ml_model_version VARCHAR(50),
ADD COLUMN IF NOT EXISTS auto_resolved BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS user_overrode_ml BOOLEAN DEFAULT FALSE;

-- Create ML model versions table
CREATE TABLE IF NOT EXISTS calendar.ml_model_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_name VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL,
    s3_path VARCHAR(500) NOT NULL,
    accuracy FLOAT,
    trained_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deployed_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT FALSE,
    metadata JSONB DEFAULT '{}'::jsonb,
    
    UNIQUE(model_name, version)
);

-- Create ML predictions log table
CREATE TABLE IF NOT EXISTS calendar.ml_predictions_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_name VARCHAR(100) NOT NULL,
    model_version VARCHAR(50) NOT NULL,
    input_features JSONB NOT NULL,
    prediction VARCHAR(100) NOT NULL,
    confidence FLOAT NOT NULL,
    actual_outcome VARCHAR(100), -- For tracking accuracy
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id UUID REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES public.users(id) ON DELETE CASCADE
);

-- Indexes for ML queries
CREATE INDEX idx_sync_conflicts_ml_recommendation 
ON calendar.sync_conflicts(ml_recommendation, auto_resolved);

CREATE INDEX idx_ml_predictions_log_model 
ON calendar.ml_predictions_log(model_name, created_at DESC);

CREATE INDEX idx_ml_predictions_log_tenant 
ON calendar.ml_predictions_log(tenant_id, created_at DESC);

CREATE INDEX idx_ml_model_versions_active 
ON calendar.ml_model_versions(model_name, is_active) 
WHERE is_active = TRUE;

-- Enable RLS
ALTER TABLE calendar.ml_model_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar.ml_predictions_log ENABLE ROW LEVEL SECURITY;

CREATE POLICY ml_model_versions_tenant_isolation 
ON calendar.ml_model_versions
USING (true); -- Model versions are global

CREATE POLICY ml_predictions_log_tenant_isolation 
ON calendar.ml_predictions_log
USING (tenant_id = NULLIF(current_setting('request.tenant_id', TRUE), '')::UUID);

-- View for ML accuracy tracking
CREATE OR REPLACE VIEW calendar.ml_accuracy_stats AS
SELECT 
    model_name,
    model_version,
    COUNT(*) as total_predictions,
    COUNT(*) FILTER (WHERE prediction = actual_outcome) as correct_predictions,
    COUNT(*) FILTER (WHERE prediction = actual_outcome) * 100.0 / COUNT(*) as accuracy_percent,
    AVG(confidence) as avg_confidence,
    DATE_TRUNC('day', created_at) as prediction_date
FROM calendar.ml_predictions_log
WHERE actual_outcome IS NOT NULL
GROUP BY model_name, model_version, DATE_TRUNC('day', created_at)
ORDER BY prediction_date DESC;

-- Comment columns
COMMENT ON COLUMN calendar.sync_conflicts.ml_recommendation IS 'ML recommended resolution strategy';
COMMENT ON COLUMN calendar.sync_conflicts.ml_confidence IS 'ML confidence score (0-1)';
COMMENT ON COLUMN calendar.sync_conflicts.auto_resolved IS 'Whether conflict was auto-resolved by ML';
COMMENT ON COLUMN calendar.sync_conflicts.user_overrode_ml IS 'Whether user overrode ML recommendation';
