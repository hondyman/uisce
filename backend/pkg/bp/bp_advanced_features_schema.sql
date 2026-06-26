-- ============================================
-- ADVANCED BP BRANCHING FEATURES (15 FEATURES)
-- ============================================
-- This extension adds support for:
-- 1. AI-Powered Predictive Routing
-- 2. Semantic Intent-Based Routing
-- 3. Multi-Dimensional Scoring Matrices
-- 4. Time-Series Predictive Branching
-- 5. Nested Parallel-Within-Conditional Branching (already supported)
-- 6. Context-Aware Adaptive Branching
-- 7. Smart Retry & Circuit Breaker Patterns
-- 8. Multi-Tenant Branch Isolation & Override
-- 9. Real-Time Branch Performance Analytics
-- 10. Collaborative Multi-Stakeholder Branching
-- 11. Geofencing & Location-Based Routing
-- 12. Blockchain-Verified Branch Execution
-- 13. Natural Language Query Interface
-- 14. Dynamic Resource-Aware Routing
-- 15. Explainable AI Branch Decisions

-- ============================================
-- FEATURE 1 & 2: AI MODELS & SEMANTIC ROUTING
-- ============================================

CREATE TABLE IF NOT EXISTS bp_ai_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Model identity
    model_id VARCHAR(255) NOT NULL UNIQUE,
    model_name VARCHAR(255),
    model_type VARCHAR(100), -- ml_classifier|semantic_classifier|time_series|scoring_matrix|custom
    version VARCHAR(50),
    description TEXT,
    
    -- Routing types supported
    use_cases TEXT[] DEFAULT '{}', -- fraud_detection|customer_intent|load_balancing|scoring|etc
    
    -- Model configuration
    model_endpoint VARCHAR(500),
    model_provider VARCHAR(100), -- openai|huggingface|custom_api|internal
    authentication_type VARCHAR(50), -- api_key|oauth|cert|none
    authentication_config JSONB,
    
    -- Performance & monitoring
    accuracy_threshold FLOAT DEFAULT 0.75,
    fallback_strategy VARCHAR(50), -- conservative|optimistic|random|next_model
    avg_latency_ms INT,
    success_rate FLOAT,
    total_predictions INT DEFAULT 0,
    failed_predictions INT DEFAULT 0,
    
    -- Features required for predictions
    input_features TEXT[] NOT NULL,
    output_format VARCHAR(100), -- classification|regression|probability_vector
    
    -- Auto-switching configuration
    auto_switch_enabled BOOLEAN DEFAULT FALSE,
    drift_detection_enabled BOOLEAN DEFAULT TRUE,
    min_accuracy_drop_threshold FLOAT DEFAULT 0.05,
    evaluation_window_hours INT DEFAULT 168,
    
    -- Explainability
    explainability_enabled BOOLEAN DEFAULT TRUE,
    explainability_method VARCHAR(50), -- shap|lime|counterfactual|attention
    log_feature_importance BOOLEAN DEFAULT TRUE,
    human_readable_reasoning BOOLEAN DEFAULT TRUE,
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_aimodel_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ai_models_tenant ON bp_ai_models(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ai_models_type ON bp_ai_models(model_type);
CREATE INDEX IF NOT EXISTS idx_ai_models_use_cases ON bp_ai_models USING GIN (use_cases);

-- ============================================
-- FEATURE 2: SEMANTIC INTENT VECTORS
-- ============================================

CREATE TABLE IF NOT EXISTS bp_semantic_intents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    
    -- Intent definition
    intent_id VARCHAR(255) NOT NULL,
    intent_label VARCHAR(255),
    intent_description TEXT,
    intent_vector FLOAT8[] NOT NULL, -- sentence-transformers embedding
    
    -- Semantic matching config
    semantic_model VARCHAR(100), -- sentence-transformers/all-MiniLM-L6-v2 | etc
    similarity_threshold FLOAT DEFAULT 0.75,
    keywords TEXT[],
    sentiment_threshold FLOAT,
    
    -- Branch mapping
    target_branch_id VARCHAR(255),
    
    -- Performance tracking
    match_count INT DEFAULT 0,
    avg_confidence FLOAT,
    false_positive_rate FLOAT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_intent_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_intent_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_semantic_intents_step ON bp_semantic_intents(step_id);
CREATE INDEX IF NOT EXISTS idx_semantic_intents_tenant ON bp_semantic_intents(tenant_id);

-- ============================================
-- FEATURE 3: SCORING MATRICES
-- ============================================

CREATE TABLE IF NOT EXISTS bp_scoring_matrices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    
    -- Matrix definition
    matrix_name VARCHAR(255),
    description TEXT,
    
    -- Dimensions (stored as JSONB)
    dimensions JSONB NOT NULL, -- [{name: "urgency", weight: 0.35, scoring_rules: [...]}, ...]
    
    -- Routing thresholds
    routing_thresholds JSONB NOT NULL, -- [{min_score: 8.0, branch_id: "...", label: "..."}, ...]
    
    -- Matrix performance
    evaluations_total INT DEFAULT 0,
    avg_score FLOAT,
    min_score FLOAT,
    max_score FLOAT,
    
    -- Auto-optimization
    auto_tune_enabled BOOLEAN DEFAULT FALSE,
    tuning_frequency_days INT DEFAULT 7,
    last_tuned_at TIMESTAMP,
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_matrix_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_matrix_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_scoring_matrices_step ON bp_scoring_matrices(step_id);
CREATE INDEX IF NOT EXISTS idx_scoring_matrices_tenant ON bp_scoring_matrices(tenant_id);

-- ============================================
-- FEATURE 4: TIME-SERIES FORECASTING
-- ============================================

CREATE TABLE IF NOT EXISTS bp_time_series_forecasts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    
    -- Forecast configuration
    forecast_model VARCHAR(100), -- arima_seasonal|prophet|lstm|xgboost
    lookback_window_days INT DEFAULT 90,
    prediction_horizon_hours INT DEFAULT 48,
    
    -- Features used in prediction
    features TEXT[] NOT NULL,
    
    -- Forecast data (rolling window)
    forecast_timestamp TIMESTAMP,
    predicted_queue_depth INT,
    predicted_approval_time_minutes INT,
    confidence_interval_lower FLOAT,
    confidence_interval_upper FLOAT,
    
    -- Branching mapping
    low_load_branch_id VARCHAR(255),
    high_load_branch_id VARCHAR(255),
    
    -- Model performance
    forecast_accuracy FLOAT,
    mae_mean_absolute_error FLOAT,
    rmse_root_mean_squared_error FLOAT,
    
    model_refresh_interval_hours INT DEFAULT 24,
    last_retrained_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_forecast_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_forecast_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_forecasts_step ON bp_time_series_forecasts(step_id);
CREATE INDEX IF NOT EXISTS idx_forecasts_tenant ON bp_time_series_forecasts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_forecasts_timestamp ON bp_time_series_forecasts(forecast_timestamp DESC);

-- ============================================
-- FEATURE 6: ADAPTIVE BRANCHING
-- ============================================

CREATE TABLE IF NOT EXISTS bp_adaptive_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    
    -- Trigger definition
    trigger_name VARCHAR(255),
    trigger_condition VARCHAR(500), -- "previous_step_duration > expected_duration * 1.5"
    trigger_type VARCHAR(50), -- duration|error_count|fraud_score|user_correction|system_load
    
    -- Action to take
    action_type VARCHAR(100), -- switch_to_branch|add_step|re_evaluate|modify_param
    action_config JSONB,
    
    -- Context awareness
    context_variables TEXT[] DEFAULT '{}', -- workflow_history|user_behavior|system_load
    learning_mode VARCHAR(50), -- offline|online|batch
    
    -- State management
    persist_across_branches BOOLEAN DEFAULT FALSE,
    reset_on_loop_back BOOLEAN DEFAULT TRUE,
    
    -- Performance tracking
    trigger_count INT DEFAULT 0,
    success_rate FLOAT,
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_trigger_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_trigger_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_adaptive_triggers_step ON bp_adaptive_triggers(step_id);
CREATE INDEX IF NOT EXISTS idx_adaptive_triggers_tenant ON bp_adaptive_triggers(tenant_id);

-- ============================================
-- FEATURE 7: CIRCUIT BREAKERS & RETRY POLICIES
-- ============================================

CREATE TABLE IF NOT EXISTS bp_resilience_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    
    -- Retry configuration
    retry_max_attempts INT DEFAULT 3,
    retry_initial_interval_seconds INT DEFAULT 5,
    retry_backoff_multiplier FLOAT DEFAULT 2.0,
    retry_max_interval_seconds INT DEFAULT 300,
    retry_on_errors TEXT[] DEFAULT '{}', -- timeout|service_unavailable|rate_limit
    
    -- Circuit breaker configuration
    circuit_breaker_enabled BOOLEAN DEFAULT TRUE,
    circuit_breaker_failure_threshold INT DEFAULT 5,
    circuit_breaker_reset_timeout_seconds INT DEFAULT 60,
    circuit_breaker_half_open_max_requests INT DEFAULT 3,
    circuit_breaker_fallback_branch_id VARCHAR(255),
    circuit_breaker_alert_channels TEXT[] DEFAULT '{}', -- slack|pagerduty|email
    
    -- Health checks
    health_check_enabled BOOLEAN DEFAULT TRUE,
    health_check_interval_seconds INT DEFAULT 30,
    health_check_endpoints JSONB, -- [{service: "...", health_url: "..."}, ...]
    
    -- Statistics
    total_retries INT DEFAULT 0,
    total_circuit_breaks INT DEFAULT 0,
    total_fallbacks INT DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_resilience_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_resilience_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_resilience_step ON bp_resilience_policies(step_id);
CREATE INDEX IF NOT EXISTS idx_resilience_tenant ON bp_resilience_policies(tenant_id);

-- ============================================
-- FEATURE 8: TENANT-SPECIFIC BRANCH OVERRIDES
-- ============================================

CREATE TABLE IF NOT EXISTS bp_tenant_branch_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    base_step_id UUID NOT NULL,
    
    -- Override configuration
    override_type VARCHAR(50), -- branch_modification|branch_addition|branch_deletion|parameter_override
    base_branch_id VARCHAR(255),
    override_branch_id VARCHAR(255),
    
    -- Modifications
    override_assignee_role VARCHAR(100),
    override_duration_hours INT,
    additional_steps TEXT[] DEFAULT '{}',
    removed_conditions JSONB,
    modified_conditions JSONB,
    
    -- Inheritance strategy
    inheritance_strategy VARCHAR(50), -- merge_with_override|replace|prepend|append
    
    -- Validation rules
    enforce_base_critical_paths BOOLEAN DEFAULT TRUE,
    allow_branch_deletion BOOLEAN DEFAULT FALSE,
    
    -- Tenant-specific branching rules
    custom_branches JSONB, -- additional branches specific to tenant
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_override_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_override_base_step FOREIGN KEY (base_step_id) REFERENCES bp_steps(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tenant_overrides_tenant ON bp_tenant_branch_overrides(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_overrides_base_step ON bp_tenant_branch_overrides(base_step_id);

-- ============================================
-- FEATURE 9: ADVANCED ANALYTICS & A/B TESTING
-- ============================================

CREATE TABLE IF NOT EXISTS bp_branch_analytics_extended (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    branch_id VARCHAR(255),
    
    -- Aggregated metrics (hourly/daily)
    metric_period TIMESTAMP, -- hourly, daily aggregation
    
    -- Branch distribution
    branch_selection_count INT DEFAULT 0,
    branch_completion_count INT DEFAULT 0,
    branch_abandonment_count INT DEFAULT 0,
    branch_timeout_count INT DEFAULT 0,
    
    -- Performance metrics
    avg_duration_ms INT,
    p50_duration_ms INT,
    p95_duration_ms INT,
    p99_duration_ms INT,
    std_dev_duration_ms FLOAT,
    
    -- Business metrics
    user_satisfaction_avg FLOAT,
    error_rate FLOAT,
    success_rate FLOAT,
    
    -- Trend analysis
    trend_direction VARCHAR(20), -- up|down|stable
    anomaly_score FLOAT, -- 0-1, 1 = highly anomalous
    anomaly_detected BOOLEAN DEFAULT FALSE,
    
    -- A/B test data
    ab_test_id UUID,
    variant_group VARCHAR(50), -- control|experiment
    conversion_rate FLOAT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_analytics_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_analytics_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_analytics_ext_step ON bp_branch_analytics_extended(step_id, metric_period DESC);
CREATE INDEX IF NOT EXISTS idx_analytics_ext_tenant ON bp_branch_analytics_extended(tenant_id);
CREATE INDEX IF NOT EXISTS idx_analytics_ext_anomaly ON bp_branch_analytics_extended(anomaly_detected);

-- ============================================
-- FEATURE 10: COLLABORATIVE VOTING
-- ============================================

CREATE TABLE IF NOT EXISTS bp_collaborative_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_instance_id UUID NOT NULL,
    step_id UUID NOT NULL,
    decision_id VARCHAR(255),
    
    -- Decision mechanism
    decision_mechanism VARCHAR(50), -- weighted_vote|consensus|majority|custom
    
    -- Stakeholders and votes
    stakeholders JSONB NOT NULL, -- [{role: "...", vote_weight: 0.5, required: true, vote: "approve|reject|abstain"}, ...]
    
    -- Voting rules
    approval_threshold FLOAT DEFAULT 0.7,
    quorum_requirement FLOAT DEFAULT 0.8,
    timeout_hours INT DEFAULT 48,
    on_timeout_action VARCHAR(100), -- escalate|auto_approve|auto_reject
    
    -- Voting state
    votes_received INT DEFAULT 0,
    votes_required INT,
    total_weight_received FLOAT DEFAULT 0.0,
    total_weight_required FLOAT,
    decision_outcome VARCHAR(50), -- approved|rejected|no_consensus|pending|timeout
    
    -- Results
    approved_by TEXT[] DEFAULT '{}',
    rejected_by TEXT[] DEFAULT '{}',
    abstained_by TEXT[] DEFAULT '{}',
    
    -- Branch routing
    approved_branch_id VARCHAR(255),
    rejected_branch_id VARCHAR(255),
    no_consensus_branch_id VARCHAR(255),
    
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    timeout_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_collab_workflow FOREIGN KEY (workflow_instance_id) REFERENCES bp_branch_executions(workflow_instance_id),
    CONSTRAINT fk_collab_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_collab_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_collab_decisions_workflow ON bp_collaborative_decisions(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_collab_decisions_tenant ON bp_collaborative_decisions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_collab_decisions_outcome ON bp_collaborative_decisions(decision_outcome);

-- ============================================
-- FEATURE 11: GEOFENCING & LOCATION ROUTING
-- ============================================

CREATE TABLE IF NOT EXISTS bp_geofence_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    
    -- Geofence definition
    geofence_name VARCHAR(255),
    geofence_type VARCHAR(50), -- polygon|country_list|coordinate_radius|address_proximity
    
    -- Geographic boundaries
    region_polygon_coords JSONB, -- [[lat,lng], [lat,lng], ...] for polygon type
    region_countries TEXT[], -- ISO country codes
    region_center_lat FLOAT,
    region_center_lng FLOAT,
    region_radius_km INT,
    
    -- Location sources
    location_sources TEXT[] DEFAULT '{}', -- ip_address|shipping_address|device_gps|user_location
    
    -- Branch routing
    target_branch_id VARCHAR(255),
    additional_steps TEXT[] DEFAULT '{}',
    
    -- Regional customization
    currency_conversion BOOLEAN DEFAULT FALSE,
    language_localization BOOLEAN DEFAULT FALSE,
    timezone VARCHAR(50),
    compliance_rules TEXT[], -- ccpa|gdpr|cra|prop65
    
    -- Distance-based routing (for logistics)
    distance_based_routing BOOLEAN DEFAULT FALSE,
    proximity_calculation VARCHAR(50), -- haversine|manhattan|euclidean
    distance_threshold_km INT,
    nearest_facility_branch_id VARCHAR(255),
    
    -- Performance tracking
    match_count INT DEFAULT 0,
    avg_accuracy FLOAT,
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_geofence_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_geofence_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_geofence_step ON bp_geofence_rules(step_id);
CREATE INDEX IF NOT EXISTS idx_geofence_tenant ON bp_geofence_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_geofence_countries ON bp_geofence_rules USING GIN (region_countries);

-- ============================================
-- FEATURE 12: BLOCKCHAIN AUDIT TRAIL
-- ============================================

CREATE TABLE IF NOT EXISTS bp_blockchain_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_instance_id UUID NOT NULL,
    
    -- Blockchain configuration
    blockchain_enabled BOOLEAN DEFAULT TRUE,
    network_type VARCHAR(50), -- hyperledger_fabric|ethereum|polygon|custom
    smart_contract_id VARCHAR(255),
    
    -- Event tracking
    event_type VARCHAR(100), -- branch_evaluation_start|branch_selection|branch_completion|condition_evaluation
    event_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    event_data JSONB,
    
    -- Cryptographic verification
    event_hash VARCHAR(256), -- SHA-256 hash of event
    parent_hash VARCHAR(256), -- Previous event hash (chain linking)
    signatures JSONB, -- [{signer: "...", signature: "...", timestamp: ...}, ...]
    required_signers TEXT[] DEFAULT '{system,approver}',
    
    -- Tamper detection
    tamper_detection_enabled BOOLEAN DEFAULT TRUE,
    verification_status VARCHAR(50), -- verified|unverified|tampered
    last_verified_at TIMESTAMP,
    verification_interval_hours INT DEFAULT 1,
    
    -- Compliance features
    gdpr_compliant BOOLEAN DEFAULT TRUE,
    sox_compliant BOOLEAN DEFAULT FALSE,
    iso_27001_audit_ready BOOLEAN DEFAULT TRUE,
    
    -- Data retention policies
    right_to_erasure_enabled BOOLEAN DEFAULT TRUE,
    anonymization_method VARCHAR(50), -- hash_with_salt|pseudonymization|differential_privacy
    expiration_date TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_blockchain_workflow FOREIGN KEY (workflow_instance_id) REFERENCES bp_branch_executions(workflow_instance_id),
    CONSTRAINT fk_blockchain_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_blockchain_workflow ON bp_blockchain_audit(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_blockchain_tenant ON bp_blockchain_audit(tenant_id);
CREATE INDEX IF NOT EXISTS idx_blockchain_event_type ON bp_blockchain_audit(event_type);
CREATE INDEX IF NOT EXISTS idx_blockchain_verification ON bp_blockchain_audit(verification_status);

-- ============================================
-- FEATURE 13: NL CONFIGURATION INTERFACE
-- ============================================

CREATE TABLE IF NOT EXISTS bp_nl_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    
    -- NL Query
    nl_query TEXT, -- "If the order is from a VIP customer and over $10k, send to the CFO"
    nl_query_embedding FLOAT8[], -- Embedding for semantic search
    
    -- Processing pipeline
    intent_extraction JSONB, -- Extracted intent
    entity_recognition JSONB, -- Extracted entities
    condition_synthesis JSONB, -- Generated conditions
    
    -- Validation
    field_validation_passed BOOLEAN,
    field_validation_errors TEXT[],
    validation_suggestions TEXT[],
    requires_human_approval BOOLEAN DEFAULT TRUE,
    human_approval_status VARCHAR(50), -- pending|approved|rejected
    
    -- Generated configuration
    generated_branching_config JSONB, -- The actual branching config created from NL
    
    -- Learning
    store_pattern BOOLEAN DEFAULT TRUE,
    pattern_category VARCHAR(100), -- routing|filtering|enrichment|custom
    similar_configs_found INT DEFAULT 0,
    
    -- LLM Configuration
    llm_model VARCHAR(100), -- gpt-4|gpt-3.5|claude|etc
    llm_temperature FLOAT DEFAULT 0.3,
    llm_max_tokens INT DEFAULT 1000,
    
    created_by UUID,
    reviewed_by UUID,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_nl_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_nl_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_nl_configs_step ON bp_nl_configurations(step_id);
CREATE INDEX IF NOT EXISTS idx_nl_configs_tenant ON bp_nl_configurations(tenant_id);

-- ============================================
-- FEATURE 14: DYNAMIC RESOURCE-AWARE ROUTING
-- ============================================

CREATE TABLE IF NOT EXISTS bp_resource_pools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    
    -- Resource pool definition
    pool_name VARCHAR(255),
    resource_type VARCHAR(100), -- approver_queue|api_rate_limit|compute_resource|custom
    resource_id VARCHAR(255),
    
    -- Monitoring configuration
    capacity_metric VARCHAR(100), -- pending_tasks|requests_per_minute|cpu_usage
    current_load_api VARCHAR(500), -- API endpoint for real-time load
    max_capacity INT,
    
    -- Routing strategy
    routing_strategy VARCHAR(50), -- least_loaded|round_robin|priority_based|affinity
    load_threshold_for_overflow FLOAT DEFAULT 0.8,
    
    -- Overflow configuration
    overflow_pool_id UUID,
    overflow_branch_id VARCHAR(255),
    
    -- Auto-scaling
    auto_scaling_enabled BOOLEAN DEFAULT TRUE,
    scale_up_threshold FLOAT DEFAULT 0.85,
    scale_down_threshold FLOAT DEFAULT 0.3,
    scale_up_increment INT DEFAULT 1,
    scale_down_decrement INT DEFAULT 1,
    cooldown_minutes INT DEFAULT 15,
    
    -- Performance tracking
    current_load INT,
    current_load_percent FLOAT,
    peak_load_recorded INT,
    avg_load_percent FLOAT,
    scale_events_total INT DEFAULT 0,
    
    last_scaled_at TIMESTAMP,
    last_load_check_at TIMESTAMP,
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_resource_step FOREIGN KEY (step_id) REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_resource_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_resource_pools_step ON bp_resource_pools(step_id);
CREATE INDEX IF NOT EXISTS idx_resource_pools_tenant ON bp_resource_pools(tenant_id);

-- ============================================
-- FEATURE 15: EXPLAINABLE AI DECISIONS
-- ============================================

CREATE TABLE IF NOT EXISTS bp_explainability_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    branch_execution_id UUID NOT NULL,
    
    -- Decision details
    selected_branch_id VARCHAR(255),
    decision_timestamp TIMESTAMP,
    
    -- Explanation configuration
    explanation_detail_level VARCHAR(50), -- summary|detailed|comprehensive
    explanation_methods TEXT[] DEFAULT '{shap,lime,counterfactual}',
    
    -- Explanation content
    feature_importance JSONB, -- [{feature: "...", importance: 0.45, direction: "positive|negative"}, ...]
    decision_path TEXT, -- Step-by-step explanation
    alternative_paths JSONB, -- [{branch: "...", score: 0.62, reason: "..."}, ...]
    
    -- Natural language summary
    natural_language_summary TEXT, -- Human-readable explanation
    
    -- Confidence & uncertainty
    decision_confidence FLOAT,
    uncertainty_estimate FLOAT,
    
    -- Counterfactual analysis
    counterfactual_enabled BOOLEAN DEFAULT TRUE,
    counterfactuals JSONB, -- [{modified_feature: "...", new_value: "...", result_branch: "..."}, ...]
    
    -- User feedback
    user_feedback_requested BOOLEAN DEFAULT FALSE,
    user_feedback_received VARCHAR(50), -- satisfactory|needs_improvement|incorrect
    user_feedback_comment TEXT,
    
    -- Audit trail inclusion
    include_in_audit_log BOOLEAN DEFAULT TRUE,
    show_in_workflow_ui BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_explain_execution FOREIGN KEY (branch_execution_id) REFERENCES bp_branch_executions(id) ON DELETE CASCADE,
    CONSTRAINT fk_explain_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_explainability_execution ON bp_explainability_records(branch_execution_id);
CREATE INDEX IF NOT EXISTS idx_explainability_tenant ON bp_explainability_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_explainability_timestamp ON bp_explainability_records(decision_timestamp DESC);

-- ============================================
-- GRANTS FOR ALL NEW TABLES
-- ============================================

GRANT SELECT, INSERT, UPDATE, DELETE ON bp_ai_models TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_semantic_intents TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_scoring_matrices TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_time_series_forecasts TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_adaptive_triggers TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_resilience_policies TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_tenant_branch_overrides TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_branch_analytics_extended TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_collaborative_decisions TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_geofence_rules TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_blockchain_audit TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_nl_configurations TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_resource_pools TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON bp_explainability_records TO app_user;
