-- Enhanced BP Branching Schema - Enterprise-Grade Workflow Routing
-- Supports: Exclusive, Inclusive, Parallel, Weighted, ML-Powered, Event-Based Gateways
-- Plus: Nested branches, timeout routing, loop-back workflows, advanced join strategies

-- ============================================
-- 0. SETUP: ROLES AND USERS
-- ============================================

-- Create app_user role if it doesn't exist
DO $$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_user WHERE usename = 'app_user') THEN
    CREATE USER app_user WITH PASSWORD 'app_user_password';
  END IF;
END $$;

-- ============================================
-- 1. EXTENDED BP_STEPS TABLE
-- ============================================

-- Alter existing bp_steps table to add branching capabilities
ALTER TABLE bp_steps ADD COLUMN IF NOT EXISTS branching_config JSONB;
ALTER TABLE bp_steps ADD COLUMN IF NOT EXISTS join_config JSONB;
ALTER TABLE bp_steps ADD COLUMN IF NOT EXISTS execution_stats JSONB DEFAULT '{"total_executions": 0, "branch_distribution": {}}';

-- Create index for efficient branching queries
CREATE INDEX IF NOT EXISTS idx_bp_steps_branching ON bp_steps USING GIN (branching_config);
CREATE INDEX IF NOT EXISTS idx_bp_steps_process_order ON bp_steps (process_id, step_order);

-- ============================================
-- 2. BRANCH EXECUTION HISTORY TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS bp_branch_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    workflow_instance_id UUID NOT NULL,
    step_id UUID NOT NULL,
    branch_id VARCHAR(255) NOT NULL,
    branch_label VARCHAR(255),
    
    -- Branch selection details
    selected_by VARCHAR(50) NOT NULL DEFAULT 'condition', -- condition|weight|ml_model|timeout|default|event
    priority_rank INT,
    
    -- Condition evaluation details
    condition_evaluation JSONB,
    matched_conditions TEXT[],
    evaluation_time_ms INT,
    
    -- ML model details (if ML-powered)
    ml_model_id VARCHAR(255),
    ml_model_score FLOAT,
    ml_model_confidence FLOAT,
    ml_features JSONB,
    
    -- Execution timing
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INT,
    status VARCHAR(20) DEFAULT 'active', -- active|completed|timeout|cancelled|failed
    
    -- Results and context
    result_data JSONB,
    next_step_id UUID,
    error_message TEXT,
    
    -- Join strategy tracking
    join_strategy VARCHAR(50),
    is_last_in_join BOOLEAN DEFAULT FALSE,
    join_results_aggregated JSONB,
    
    -- Nested branch tracking
    parent_branch_execution_id UUID,
    nesting_level INT DEFAULT 0,
    
    -- Loop-back tracking
    loop_iteration INT DEFAULT 0,
    loop_parent_execution_id UUID,
    
    -- Audit trail
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT uq_workflow_instance UNIQUE (workflow_instance_id)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_branch_executions_workflow ON bp_branch_executions(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_branch_executions_step ON bp_branch_executions(step_id);
CREATE INDEX IF NOT EXISTS idx_branch_executions_status ON bp_branch_executions(status, started_at);
CREATE INDEX IF NOT EXISTS idx_branch_executions_selected_by ON bp_branch_executions(selected_by);
CREATE INDEX IF NOT EXISTS idx_branch_executions_tenant ON bp_branch_executions(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_branch_executions_ml_model ON bp_branch_executions(ml_model_id);
CREATE INDEX IF NOT EXISTS idx_branch_executions_loop ON bp_branch_executions(loop_parent_execution_id);
CREATE INDEX IF NOT EXISTS idx_branch_executions_completed ON bp_branch_executions(completed_at) WHERE completed_at IS NOT NULL;

-- ============================================
-- 3. BRANCH PERFORMANCE METRICS TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS bp_branch_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    branch_id VARCHAR(255) NOT NULL,
    branch_label VARCHAR(255),
    
    -- Aggregated metrics
    total_executions INT DEFAULT 0,
    completed_count INT DEFAULT 0,
    timeout_count INT DEFAULT 0,
    failed_count INT DEFAULT 0,
    completion_rate FLOAT DEFAULT 0.0,
    
    -- Timing metrics
    avg_duration_ms INT,
    min_duration_ms INT,
    max_duration_ms INT,
    p95_duration_ms INT,
    p99_duration_ms INT,
    
    -- ML metrics (if applicable)
    avg_ml_score FLOAT,
    avg_ml_confidence FLOAT,
    ml_score_distribution JSONB,
    
    -- Business metrics
    success_rate FLOAT,
    error_rate FLOAT,
    avg_satisfaction_score FLOAT,
    
    -- Selection metrics
    selection_distribution JSONB, -- by selection_by type
    condition_accuracy FLOAT,
    
    -- Trend data
    daily_stats JSONB, -- {date: {count: N, avg_duration: N}}
    hourly_stats JSONB,
    
    -- Last updated
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_step_metrics FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_tenant_metrics FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_branch_metrics_step ON bp_branch_metrics(step_id, branch_id);
CREATE INDEX IF NOT EXISTS idx_branch_metrics_tenant ON bp_branch_metrics(tenant_id);
CREATE INDEX IF NOT EXISTS idx_branch_metrics_updated ON bp_branch_metrics(updated_at);

-- ============================================
-- 4. JOIN CONVERGENCE TRACKING TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS bp_join_convergences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_instance_id UUID NOT NULL,
    step_id UUID NOT NULL,
    join_id VARCHAR(100) NOT NULL,
    
    -- Join configuration
    join_strategy VARCHAR(50) NOT NULL, -- wait_all|first_complete|m_of_n|majority_vote
    required_branches INT,
    
    -- Convergence tracking
    completed_branches INT DEFAULT 0,
    completed_branch_ids TEXT[] DEFAULT '{}',
    aggregate_result JSONB,
    
    -- Timing
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    timeout_at TIMESTAMP,
    timed_out BOOLEAN DEFAULT FALSE,
    duration_ms INT,
    
    -- Status
    status VARCHAR(20) DEFAULT 'waiting', -- waiting|completed|timed_out|cancelled
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_join_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_join_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_join_convergences_workflow ON bp_join_convergences(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_join_convergences_status ON bp_join_convergences(status);
CREATE INDEX IF NOT EXISTS idx_join_convergences_timeout ON bp_join_convergences(timed_out);

-- ============================================
-- 5. ML MODEL CONFIGURATION TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS bp_ml_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Model metadata
    model_id VARCHAR(255) NOT NULL UNIQUE,
    model_name VARCHAR(255),
    description TEXT,
    version VARCHAR(50),
    
    -- Endpoint configuration
    model_endpoint VARCHAR(500),
    timeout_ms INT DEFAULT 5000,
    retry_count INT DEFAULT 2,
    
    -- Feature configuration
    input_features TEXT[] NOT NULL,
    feature_extraction_config JSONB,
    
    -- Model parameters
    confidence_threshold FLOAT DEFAULT 0.75,
    fallback_strategy VARCHAR(50) DEFAULT 'conservative', -- conservative|optimistic|random
    
    -- Performance tracking
    avg_latency_ms INT,
    success_rate FLOAT,
    last_used_at TIMESTAMP,
    total_predictions INT DEFAULT 0,
    failed_predictions INT DEFAULT 0,
    
    -- Monitoring
    alert_on_drift BOOLEAN DEFAULT TRUE,
    performance_threshold FLOAT DEFAULT 0.85,
    drift_detection_enabled BOOLEAN DEFAULT TRUE,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_ml_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bp_ml_models_tenant ON bp_ml_models(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bp_ml_models_active ON bp_ml_models(is_active);

-- ============================================
-- 6. WEIGHTED ROUTING A/B TEST TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS bp_ab_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    
    -- Experiment metadata
    test_name VARCHAR(255),
    description TEXT,
    control_branch_id VARCHAR(255),
    experiment_branch_id VARCHAR(255),
    
    -- Weights
    control_weight FLOAT DEFAULT 0.5,
    experiment_weight FLOAT DEFAULT 0.5,
    
    -- Randomization
    randomization_seed VARCHAR(100), -- customer_id|user_id|random|workflow_id
    
    -- Time range
    start_date TIMESTAMP,
    end_date TIMESTAMP,
    
    -- Success metrics
    success_metrics TEXT[] DEFAULT '{}',
    target_sample_size INT,
    
    -- Results
    control_sample_size INT DEFAULT 0,
    experiment_sample_size INT DEFAULT 0,
    control_success_rate FLOAT,
    experiment_success_rate FLOAT,
    statistical_significance FLOAT,
    winner VARCHAR(255),
    
    -- Status
    status VARCHAR(20) DEFAULT 'active', -- active|paused|completed|archived
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_ab_test_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_ab_test_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ab_tests_tenant ON bp_ab_tests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ab_tests_status ON bp_ab_tests(status);
CREATE INDEX IF NOT EXISTS idx_ab_tests_dates ON bp_ab_tests(start_date, end_date);

-- ============================================
-- 7. EVENT-BASED BRANCHING TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS bp_branch_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_instance_id UUID NOT NULL,
    step_id UUID NOT NULL,
    
    -- Event definition
    event_type VARCHAR(100) NOT NULL,
    event_payload JSONB,
    triggered_branch_id VARCHAR(255),
    
    -- Event lifecycle
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    triggered_at TIMESTAMP,
    timeout_at TIMESTAMP,
    
    -- Event status
    status VARCHAR(20) DEFAULT 'waiting', -- waiting|triggered|timeout|cancelled
    is_first_event BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_branch_events_workflow FOREIGN KEY (workflow_instance_id) REFERENCES bp_branch_executions(workflow_instance_id),
    CONSTRAINT fk_branch_events_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_branch_events_workflow ON bp_branch_events(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_branch_events_type ON bp_branch_events(event_type);
CREATE INDEX IF NOT EXISTS idx_branch_events_status ON bp_branch_events(status);

-- ============================================
-- 8. BRANCH ANOMALY DETECTION TABLE
-- ============================================

CREATE TABLE IF NOT EXISTS bp_branch_anomalies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    branch_id VARCHAR(255),
    
    -- Anomaly details
    anomaly_type VARCHAR(100), -- latency_spike|selection_bias|failure_rate|ml_drift|timeout_surge
    severity VARCHAR(20), -- low|medium|high|critical
    description TEXT,
    
    -- Metrics
    baseline_value FLOAT,
    actual_value FLOAT,
    deviation_percent FLOAT,
    
    -- Detection
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    investigation_status VARCHAR(20) DEFAULT 'open', -- open|investigating|resolved|false_alarm
    
    -- Metadata
    affected_executions INT,
    correlation_data JSONB,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_anomaly_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_anomaly_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_anomalies_tenant ON bp_branch_anomalies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_anomalies_severity ON bp_branch_anomalies(severity);
CREATE INDEX IF NOT EXISTS idx_anomalies_detected ON bp_branch_anomalies(detected_at);

-- ============================================
-- 9. MATERIALIZED VIEW: Branch Summary Metrics
-- ============================================

-- Note: This materialized view can be created after all base tables exist
-- For initial deployment, this may fail if bp_branch_executions is empty
-- That's OK - the view will work once data is inserted

CREATE MATERIALIZED VIEW IF NOT EXISTS bp_branch_summary_metrics AS
SELECT 
    s.id as step_id,
    s.step_name,
    s.process_id,
    be.branch_id,
    be.branch_label,
    COUNT(*) as execution_count,
    COUNT(CASE WHEN be.status = 'completed' THEN 1 END) as completed_count,
    COUNT(CASE WHEN be.status = 'timeout' THEN 1 END) as timeout_count,
    COUNT(CASE WHEN be.status = 'failed' THEN 1 END) as failed_count,
    ROUND(100.0 * COUNT(CASE WHEN be.status = 'completed' THEN 1 END) / 
        NULLIF(COUNT(*), 0), 2) as completion_rate,
    AVG(be.duration_ms) as avg_duration_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY be.duration_ms) as p95_duration_ms,
    MAX(be.duration_ms) as max_duration_ms,
    AVG(be.ml_model_score) as avg_ml_score,
    be.selected_by,
    COUNT(DISTINCT be.selected_by) as selection_methods_used,
    DATE_TRUNC('hour', be.created_at) as metric_hour
FROM bp_steps s
JOIN bp_branch_executions be ON s.id = be.step_id
GROUP BY s.id, s.step_name, s.process_id, be.branch_id, be.branch_label, 
         be.selected_by, DATE_TRUNC('hour', be.created_at);

CREATE INDEX IF NOT EXISTS idx_branch_summary_step ON bp_branch_summary_metrics(step_id);
CREATE INDEX IF NOT EXISTS idx_branch_summary_branch ON bp_branch_summary_metrics(branch_id);

-- ============================================
-- 10. GRANT PERMISSIONS
-- ============================================

GRANT SELECT, INSERT, UPDATE, DELETE ON bp_branch_executions TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_branch_metrics TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_join_convergences TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_ml_models TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_ab_tests TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_branch_events TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_branch_anomalies TO app_user;
GRANT SELECT ON bp_branch_summary_metrics TO app_user;

-- ============================================
-- 11. REFRESH MATERIALIZED VIEW
-- ============================================

-- Refresh the view (run periodically in production)
-- REFRESH MATERIALIZED VIEW CONCURRENTLY bp_branch_summary_metrics;
