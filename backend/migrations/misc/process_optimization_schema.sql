-- AI-Powered Process Optimization Schema
-- Tracks optimization suggestions and applied changes

-- Optimization suggestions table
CREATE TABLE IF NOT EXISTS process_optimization_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_type VARCHAR(255) NOT NULL,
    suggestion_type VARCHAR(50) NOT NULL, -- parallel_execution, reorder_steps, remove_step, sla_adjustment, resource_allocation
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    confidence_score DECIMAL(5,2) NOT NULL CHECK (confidence_score >= 0 AND confidence_score <= 100),
    expected_improvement TEXT,
    impact_metrics JSONB DEFAULT '{}'::JSONB,
    target_steps TEXT[] DEFAULT ARRAY[]::TEXT[],
    action_details JSONB DEFAULT '{}'::JSONB,
    based_on_executions INT DEFAULT 0,
    status VARCHAR(50) DEFAULT 'pending', -- pending, applied, dismissed, testing
    priority VARCHAR(20) DEFAULT 'medium', -- critical, high, medium, low
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL
);

-- Applied optimizations tracking
CREATE TABLE IF NOT EXISTS applied_optimizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    suggestion_id UUID REFERENCES process_optimization_suggestions(id),
    workflow_type VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP DEFAULT NOW(),
    applied_by VARCHAR(255) NOT NULL,
    before_metrics JSONB DEFAULT '{}'::JSONB,
    after_metrics JSONB DEFAULT '{}'::JSONB,
    actual_improvement DECIMAL(10,2) DEFAULT 0,
    rollback_available BOOLEAN DEFAULT true,
    rollback_at TIMESTAMP,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL
);

-- Auto-tune configuration
CREATE TABLE IF NOT EXISTS auto_tune_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    enabled BOOLEAN DEFAULT false,
    confidence_threshold DECIMAL(5,2) DEFAULT 80.0,
    auto_apply_types TEXT[] DEFAULT ARRAY['sla_adjustment']::TEXT[],
    notification_email VARCHAR(255),
    last_run TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, datasource_id)
);

-- Indexes for fast querying
CREATE INDEX IF NOT EXISTS idx_optimization_suggestions_workflow 
    ON process_optimization_suggestions(workflow_type, tenant_id, datasource_id);
    
CREATE INDEX IF NOT EXISTS idx_optimization_suggestions_status 
    ON process_optimization_suggestions(status, priority, created_at DESC);
    
CREATE INDEX IF NOT EXISTS idx_optimization_suggestions_tenant 
    ON process_optimization_suggestions(tenant_id, datasource_id, created_at DESC);

-- Unique constraint to prevent duplicate pending suggestions
CREATE UNIQUE INDEX IF NOT EXISTS idx_optimization_suggestions_unique 
    ON process_optimization_suggestions(workflow_type, suggestion_type, tenant_id, datasource_id)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_applied_optimizations_workflow 
    ON applied_optimizations(workflow_type, tenant_id, datasource_id);
    
CREATE INDEX IF NOT EXISTS idx_applied_optimizations_tenant 
    ON applied_optimizations(tenant_id, datasource_id, applied_at DESC);

CREATE INDEX IF NOT EXISTS idx_auto_tune_tenant 
    ON auto_tune_config(tenant_id, datasource_id);

-- Triggers
CREATE OR REPLACE FUNCTION update_optimization_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_optimization_timestamp
BEFORE UPDATE ON process_optimization_suggestions
FOR EACH ROW
EXECUTE FUNCTION update_optimization_timestamp();

CREATE TRIGGER trigger_update_autotune_timestamp
BEFORE UPDATE ON auto_tune_config
FOR EACH ROW
EXECUTE FUNCTION update_optimization_timestamp();

-- Comments
COMMENT ON TABLE process_optimization_suggestions IS 'AI-generated optimization suggestions for workflows';
COMMENT ON TABLE applied_optimizations IS 'History of applied optimizations with before/after metrics';
COMMENT ON TABLE auto_tune_config IS 'Configuration for automatic optimization application';
COMMENT ON COLUMN process_optimization_suggestions.confidence_score IS 'ML confidence score 0-100';
COMMENT ON COLUMN process_optimization_suggestions.suggestion_type IS 'Type: parallel_execution, reorder_steps, remove_step, sla_adjustment, resource_allocation';
COMMENT ON COLUMN applied_optimizations.actual_improvement IS 'Measured improvement percentage after applying';
