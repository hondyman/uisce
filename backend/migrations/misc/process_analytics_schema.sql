-- Process Analytics Tables for Predictive Process Optimization
-- This migration creates tables to store workflow execution metrics and optimization recommendations

-- Table to store detailed metrics for each workflow step execution
CREATE TABLE IF NOT EXISTS process_execution_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id VARCHAR(255) NOT NULL,
    workflow_type VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    step_type VARCHAR(100) NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    duration INTERVAL,
    status VARCHAR(50) NOT NULL CHECK (status IN ('running', 'completed', 'failed', 'timeout')),
    error_message TEXT,
    resource_usage JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_process_execution_metrics_workflow_id ON process_execution_metrics(workflow_id);
CREATE INDEX IF NOT EXISTS idx_process_execution_metrics_workflow_type ON process_execution_metrics(workflow_type);
CREATE INDEX IF NOT EXISTS idx_process_execution_metrics_tenant_id ON process_execution_metrics(tenant_id);
CREATE INDEX IF NOT EXISTS idx_process_execution_metrics_step_name ON process_execution_metrics(step_name);
CREATE INDEX IF NOT EXISTS idx_process_execution_metrics_status ON process_execution_metrics(status);
CREATE INDEX IF NOT EXISTS idx_process_execution_metrics_created_at ON process_execution_metrics(created_at);
CREATE INDEX IF NOT EXISTS idx_process_execution_metrics_tenant_workflow ON process_execution_metrics(tenant_id, workflow_type, created_at);

-- Unique constraint to prevent duplicate metrics for same workflow step
CREATE UNIQUE INDEX IF NOT EXISTS idx_process_execution_metrics_unique 
    ON process_execution_metrics(workflow_id, step_name);

-- Table to store identified bottlenecks
CREATE TABLE IF NOT EXISTS process_bottleneck_analysis (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_type VARCHAR(255) NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    bottleneck_type VARCHAR(50) NOT NULL CHECK (bottleneck_type IN ('duration', 'failure_rate', 'resource_contention')),
    severity DECIMAL(3,2) NOT NULL CHECK (severity >= 0 AND severity <= 1),
    avg_duration INTERVAL NOT NULL,
    failure_rate DECIMAL(5,4) NOT NULL CHECK (failure_rate >= 0 AND failure_rate <= 1),
    recommendation TEXT NOT NULL,
    confidence DECIMAL(3,2) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_analyzed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for bottleneck analysis
CREATE INDEX IF NOT EXISTS idx_process_bottleneck_analysis_tenant_workflow ON process_bottleneck_analysis(tenant_id, workflow_type);
CREATE INDEX IF NOT EXISTS idx_process_bottleneck_analysis_severity ON process_bottleneck_analysis(severity DESC);
CREATE INDEX IF NOT EXISTS idx_process_bottleneck_analysis_detected_at ON process_bottleneck_analysis(detected_at DESC);

-- Add unique constraint for bottleneck analysis (prevent duplicates)
CREATE UNIQUE INDEX IF NOT EXISTS idx_process_bottleneck_unique 
    ON process_bottleneck_analysis(workflow_type, step_name, tenant_id, bottleneck_type);

-- Table to store AI-generated optimization recommendations
CREATE TABLE IF NOT EXISTS process_optimization_recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_type VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL CHECK (priority IN ('high', 'medium', 'low')),
    expected_impact DECIMAL(3,2) NOT NULL CHECK (expected_impact >= 0 AND expected_impact <= 1),
    implementation JSONB DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'implemented', 'rejected', 'in_progress')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    implemented_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for optimization recommendations
CREATE INDEX IF NOT EXISTS idx_process_optimization_recommendations_tenant ON process_optimization_recommendations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_process_optimization_recommendations_priority ON process_optimization_recommendations(priority, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_process_optimization_recommendations_status ON process_optimization_recommendations(status, created_at DESC);

-- Table to store process performance baselines for comparison
CREATE TABLE IF NOT EXISTS process_performance_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_type VARCHAR(255) NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    baseline_value DECIMAL(15,6) NOT NULL,
    standard_deviation DECIMAL(15,6),
    sample_size INTEGER NOT NULL,
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    valid_until TIMESTAMP WITH TIME ZONE,
    UNIQUE(workflow_type, step_name, tenant_id, metric_name)
);

-- Indexes for performance baselines
CREATE INDEX IF NOT EXISTS idx_process_performance_baselines_tenant_workflow ON process_performance_baselines(tenant_id, workflow_type);
CREATE INDEX IF NOT EXISTS idx_process_performance_baselines_valid_until ON process_performance_baselines(valid_until);

-- Function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for automatic timestamp updates
CREATE TRIGGER update_process_execution_metrics_updated_at
    BEFORE UPDATE ON process_execution_metrics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_process_optimization_recommendations_updated_at
    BEFORE UPDATE ON process_optimization_recommendations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to calculate step duration when end_time is updated
CREATE OR REPLACE FUNCTION calculate_step_duration()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.end_time IS NOT NULL AND OLD.end_time IS NULL THEN
        NEW.duration = NEW.end_time - OLD.start_time;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically calculate duration
CREATE TRIGGER calculate_process_execution_metrics_duration
    BEFORE UPDATE ON process_execution_metrics
    FOR EACH ROW EXECUTE FUNCTION calculate_step_duration();

-- Comments for documentation
COMMENT ON TABLE process_execution_metrics IS 'Stores detailed execution metrics for each workflow step to enable process mining and optimization';
COMMENT ON TABLE process_bottleneck_analysis IS 'Stores identified performance bottlenecks with severity scores and recommendations';
COMMENT ON TABLE process_optimization_recommendations IS 'Stores AI-generated optimization recommendations with implementation details';
COMMENT ON TABLE process_performance_baselines IS 'Stores statistical baselines for process performance metrics';