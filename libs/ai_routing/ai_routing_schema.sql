-- AI Routing Decision Storage and Feedback Tables

-- Store routing decisions for audit trail
CREATE TABLE IF NOT EXISTS routing_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_id VARCHAR(255) UNIQUE NOT NULL,
    workflow_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    selected_branch_id VARCHAR(255) NOT NULL,
    confidence FLOAT8 NOT NULL,
    reasoning JSONB NOT NULL DEFAULT '[]',
    model_scores JSONB NOT NULL DEFAULT '{}',
    execution_strategy VARCHAR(50) NOT NULL, -- immediate|delayed|conditional
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_routing_decisions_workflow ON routing_decisions(workflow_id);
CREATE INDEX idx_routing_decisions_tenant ON routing_decisions(tenant_id);
CREATE INDEX idx_routing_decisions_branch ON routing_decisions(selected_branch_id);
CREATE INDEX idx_routing_decisions_created ON routing_decisions(created_at);

-- Track workflow outcomes for RL training
CREATE TABLE IF NOT EXISTS workflow_outcomes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL,
    routing_decision_id VARCHAR(255) NOT NULL,
    tenant_id UUID NOT NULL,
    branch_id VARCHAR(255) NOT NULL,
    success BOOLEAN NOT NULL,
    completion_time_minutes FLOAT8,
    expected_time_minutes FLOAT8,
    customer_satisfaction_score FLOAT8,
    first_time_resolution BOOLEAN DEFAULT false,
    cost_incurred FLOAT8 DEFAULT 0,
    error_count INT DEFAULT 0,
    state_features TEXT, -- Encoded state for RL training
    rl_reward FLOAT8, -- Calculated reward for RL
    processed_for_training BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (routing_decision_id) REFERENCES routing_decisions(decision_id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_outcomes_workflow ON workflow_outcomes(workflow_id);
CREATE INDEX idx_workflow_outcomes_tenant ON workflow_outcomes(tenant_id);
CREATE INDEX idx_workflow_outcomes_decision ON workflow_outcomes(routing_decision_id);
CREATE INDEX idx_workflow_outcomes_training ON workflow_outcomes(processed_for_training);
CREATE INDEX idx_workflow_outcomes_created ON workflow_outcomes(created_at);

-- Store RL Q-table states for persistence
CREATE TABLE IF NOT EXISTS rl_q_table (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    state_key VARCHAR(512) NOT NULL,
    branch_id VARCHAR(255) NOT NULL,
    q_value FLOAT8 NOT NULL DEFAULT 0,
    episodes_visited INT DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, state_key, branch_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_rl_q_table_tenant ON rl_q_table(tenant_id);
CREATE INDEX idx_rl_q_table_state ON rl_q_table(state_key);

-- Model performance metrics
CREATE TABLE IF NOT EXISTS routing_model_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    accuracy FLOAT8,
    precision FLOAT8,
    recall FLOAT8,
    f1_score FLOAT8,
    avg_latency_ms FLOAT8,
    total_predictions INT DEFAULT 0,
    correct_predictions INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_routing_model_metrics_tenant ON routing_model_metrics(tenant_id);
CREATE INDEX idx_routing_model_metrics_model ON routing_model_metrics(model_name);

-- Daily aggregated routing statistics
CREATE TABLE IF NOT EXISTS routing_daily_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    stat_date DATE NOT NULL,
    total_routed INT DEFAULT 0,
    total_successful INT DEFAULT 0,
    avg_decision_time_ms FLOAT8,
    avg_confidence FLOAT8,
    avg_completion_time FLOAT8,
    avg_customer_satisfaction FLOAT8,
    total_cost FLOAT8 DEFAULT 0,
    model_agreement_rate FLOAT8,
    rl_epsilon FLOAT8,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, stat_date),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_routing_daily_stats_tenant ON routing_daily_stats(tenant_id);
CREATE INDEX idx_routing_daily_stats_date ON routing_daily_stats(stat_date);

-- A/B Testing for routing strategies
CREATE TABLE IF NOT EXISTS routing_ab_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    test_name VARCHAR(255) NOT NULL,
    control_strategy VARCHAR(100) NOT NULL,
    test_strategy VARCHAR(100) NOT NULL,
    test_start_date TIMESTAMP,
    test_end_date TIMESTAMP,
    control_sample_size INT,
    test_sample_size INT,
    control_success_rate FLOAT8,
    test_success_rate FLOAT8,
    statistical_significance FLOAT8,
    conclusion VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_routing_ab_tests_tenant ON routing_ab_tests(tenant_id);

-- Anomaly detection for routing
CREATE TABLE IF NOT EXISTS routing_anomalies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    anomaly_type VARCHAR(100), -- unusual_latency, low_confidence, branch_overload
    severity VARCHAR(50), -- low|medium|high|critical
    description TEXT,
    affected_branch_id VARCHAR(255),
    metric_value FLOAT8,
    normal_range_min FLOAT8,
    normal_range_max FLOAT8,
    resolved BOOLEAN DEFAULT false,
    resolution_notes TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_routing_anomalies_tenant ON routing_anomalies(tenant_id);
CREATE INDEX idx_routing_anomalies_resolved ON routing_anomalies(resolved);
CREATE INDEX idx_routing_anomalies_detected ON routing_anomalies(detected_at);

-- Feature Store for ML model inputs
CREATE TABLE IF NOT EXISTS routing_feature_store (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    feature_name VARCHAR(255) NOT NULL,
    feature_value FLOAT8,
    entity_id VARCHAR(255), -- customer, branch, etc
    entity_type VARCHAR(50),
    feature_timestamp TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_routing_feature_store_tenant ON routing_feature_store(tenant_id);
CREATE INDEX idx_routing_feature_store_feature ON routing_feature_store(feature_name);
CREATE INDEX idx_routing_feature_store_entity ON routing_feature_store(entity_id, entity_type);
