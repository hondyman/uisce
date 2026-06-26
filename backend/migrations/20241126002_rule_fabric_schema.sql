-- =============================================================================
-- RULE FABRIC: Unified Metadata-Driven Rule Engine Schema
-- =============================================================================
-- A single, generic "Rule Fabric" that expresses all rule types 
-- (DQ, compliance, MDM, wash trades, values, rebalancing, etc.) as metadata 
-- over entities, events, and graphs, with pluggable evaluators and execution policies.
-- =============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- ENUMS
-- =============================================================================

-- Rule categories - extensible via custom type
CREATE TYPE rule_category AS ENUM (
    'data_quality',       -- DQ rules: not null, in range, unique, referential integrity
    'compliance',         -- Regulatory compliance: MiFID, Reg BI, KYC/AML
    'mdm',                -- Master Data Management: survivorship, merge, conflict resolution
    'wash_trade',         -- Trade pattern detection: wash sales, circular trading
    'values',             -- ESG/client values: exclusions, exposure limits
    'rebalancing',        -- Portfolio rebalancing: drift, TLH, CPPI
    'workflow',           -- Business process rules: approvals, routing
    'security',           -- Access control, ABAC policies
    'custom'              -- Custom/extensible rules
);

-- Primary context for rule evaluation
CREATE TYPE rule_context_type AS ENUM (
    'data_record',        -- Table row / entity instance
    'trade_event',        -- Trade, order, execution event
    'portfolio',          -- Portfolio-level context
    'client_profile',     -- Client/account context
    'mdm_group',          -- Master data match group
    'system_job',         -- Batch job / ETL context
    'relationship',       -- Entity relationship context
    'time_series',        -- Time-series / window context
    'aggregate'           -- Aggregated data context
);

-- Rule severity levels
CREATE TYPE rule_severity AS ENUM (
    'info',               -- Informational only
    'warning',            -- Warning, may proceed
    'error',              -- Error, should not proceed
    'hard_block',         -- Must block operation
    'soft_block',         -- Block unless overridden
    'quarantine'          -- Quarantine for review
);

-- Rule lifecycle status
CREATE TYPE rule_status AS ENUM (
    'draft',              -- Being authored
    'awaiting_approval',  -- Pending review
    'active',             -- Live and enforced
    'suspended',          -- Temporarily disabled
    'deprecated',         -- Marked for retirement
    'retired'             -- No longer in use
);

-- Execution enforcement modes
CREATE TYPE enforcement_mode AS ENUM (
    'hard_block',         -- Always block on violation
    'soft_block',         -- Block unless override
    'log_only',           -- Log but don't block
    'simulate',           -- Shadow mode, no action
    'disabled'            -- Completely disabled
);

-- Dependency relationship types
CREATE TYPE dependency_kind AS ENUM (
    'precondition',       -- Must pass before this rule runs
    'postcondition',      -- Run after this rule
    'mutually_exclusive', -- Cannot both be active
    'aggregates',         -- This rule aggregates results of dependent
    'overrides'           -- This rule can override dependent
);

-- =============================================================================
-- CORE TABLES
-- =============================================================================

-- Rule: Core metadata for any rule type
CREATE TABLE IF NOT EXISTS rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID,
    
    -- Identity
    rule_code VARCHAR(100) NOT NULL,  -- Human-readable code (e.g., "DQ_NOT_NULL_001")
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Classification
    category rule_category NOT NULL DEFAULT 'custom',
    primary_context rule_context_type NOT NULL DEFAULT 'data_record',
    severity rule_severity NOT NULL DEFAULT 'error',
    
    -- Scope definition (what this rule applies to)
    scope_entity VARCHAR(255),                    -- Primary entity name
    scope_fields TEXT[],                          -- Specific fields (optional)
    scope_event_types TEXT[],                     -- Event types for event-scoped rules
    scope_relationship_paths JSONB,               -- EntityPath definitions for cross-entity
    
    -- Lifecycle
    status rule_status NOT NULL DEFAULT 'draft',
    environment VARCHAR(20) NOT NULL DEFAULT 'dev', -- dev, test, staging, prod
    effective_from TIMESTAMPTZ,
    effective_to TIMESTAMPTZ,
    
    -- Governance
    owner_user_id UUID,
    created_by UUID,
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    
    -- Tagging and classification
    tags TEXT[] DEFAULT '{}',                     -- Free-form tags
    regulation_ids TEXT[] DEFAULT '{}',           -- e.g., ['MiFID-II', 'SOX']
    control_ids TEXT[] DEFAULT '{}',              -- Internal control references
    product_codes TEXT[] DEFAULT '{}',            -- Product applicability
    client_segments TEXT[] DEFAULT '{}',          -- Client segment applicability
    
    -- Feature flags
    feature_flags JSONB DEFAULT '{}',
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT rules_tenant_code_env_unique UNIQUE (tenant_id, rule_code, environment)
);

-- RuleLogic: Versioned rule logic (condition + actions)
CREATE TABLE IF NOT EXISTS rule_logic (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    
    -- Versioning
    version INT NOT NULL DEFAULT 1,
    version_label VARCHAR(50),                    -- e.g., "v1.0.0", "hotfix-1"
    
    -- Condition definition (AdvancedConditionBuilder JSON)
    condition_json JSONB NOT NULL DEFAULT '{"type": "group", "operator": "AND", "conditions": []}',
    
    -- Actions to take when rule triggers
    actions_json JSONB NOT NULL DEFAULT '[]',
    -- Actions schema: [{
    --   "type": "reject_row" | "block_trade" | "route_to_queue" | "override_field" | "emit_event" | etc.,
    --   "params": { queue_name, field_name, override_source, event_type, escalation_policy, etc. },
    --   "order": 1
    -- }]
    
    -- Scoring formula (optional, for prioritization)
    scoring_formula TEXT,                         -- CEL expression for score calculation
    
    -- Pre-computed operator hints for optimization
    operator_hints JSONB DEFAULT '{}',            -- Operator types used, index hints
    
    -- Approval tracking
    is_approved BOOLEAN DEFAULT FALSE,
    approved_by UUID,
    approved_at TIMESTAMPTZ,
    approval_notes TEXT,
    
    -- Change tracking
    change_reason TEXT,
    changed_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT rule_logic_version_unique UNIQUE (rule_id, version)
);

-- RuleDependency: Relationships between rules
CREATE TABLE IF NOT EXISTS rule_dependencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    rule_id UUID NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    depends_on_rule_id UUID NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    
    kind dependency_kind NOT NULL DEFAULT 'precondition',
    
    -- Execution control
    stop_on_failure BOOLEAN DEFAULT TRUE,         -- Stop if dependency fails
    propagate_result BOOLEAN DEFAULT FALSE,       -- Pass dependency result to this rule
    
    -- Ordering
    execution_order INT DEFAULT 0,
    
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT rule_dependencies_unique UNIQUE (rule_id, depends_on_rule_id, kind),
    CONSTRAINT rule_dependencies_no_self CHECK (rule_id != depends_on_rule_id)
);

-- RuleExecutionPolicy: Channel-specific enforcement configuration
CREATE TABLE IF NOT EXISTS rule_execution_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID,
    
    -- Policy identity
    policy_code VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Scope
    channel VARCHAR(100) NOT NULL,                -- e.g., "etl_batch", "realtime_trade_api", "ui_form_save"
    category rule_category,                       -- NULL = applies to all categories
    
    -- Enforcement configuration
    enforcement enforcement_mode NOT NULL DEFAULT 'hard_block',
    max_severity rule_severity,                   -- Only enforce up to this severity
    
    -- Override configuration
    allow_override BOOLEAN DEFAULT FALSE,
    override_requires_approval BOOLEAN DEFAULT TRUE,
    override_approval_roles TEXT[] DEFAULT '{}',
    
    -- Timing
    timeout_ms INT DEFAULT 5000,                  -- Evaluation timeout
    async_allowed BOOLEAN DEFAULT FALSE,          -- Can evaluate asynchronously
    
    -- Event configuration
    emit_events BOOLEAN DEFAULT TRUE,
    event_topic VARCHAR(255),                     -- Topic for violation events
    
    -- Feature flags
    is_active BOOLEAN DEFAULT TRUE,
    environment VARCHAR(20) NOT NULL DEFAULT 'dev',
    
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT rule_exec_policy_unique UNIQUE (tenant_id, policy_code, channel, environment)
);

-- RuleActionType: Registry of available action types per category
CREATE TABLE IF NOT EXISTS rule_action_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Identity
    action_type VARCHAR(100) NOT NULL,            -- e.g., "reject_row", "block_trade"
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Applicability
    categories rule_category[] NOT NULL,          -- Which categories can use this action
    contexts rule_context_type[] NOT NULL,        -- Which contexts support this action
    
    -- Parameter schema
    params_schema JSONB NOT NULL DEFAULT '{}',    -- JSON Schema for params
    
    -- Execution
    handler_service VARCHAR(100),                 -- Service that handles this action
    is_blocking BOOLEAN DEFAULT FALSE,            -- Does this action block the operation?
    is_async BOOLEAN DEFAULT FALSE,               -- Can be executed asynchronously?
    
    -- UI hints
    icon VARCHAR(50),
    color VARCHAR(20),
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT rule_action_types_unique UNIQUE (action_type)
);

-- =============================================================================
-- EVALUATION & RESULTS
-- =============================================================================

-- RuleEvaluationResult: Stores results of rule evaluations
CREATE TABLE IF NOT EXISTS rule_evaluation_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Rule reference
    rule_id UUID NOT NULL REFERENCES rules(id),
    rule_logic_id UUID REFERENCES rule_logic(id),
    rule_version INT,
    
    -- Evaluation context
    channel VARCHAR(100),
    context_type rule_context_type,
    context_id VARCHAR(255),                      -- ID of the evaluated entity/event
    context_snapshot JSONB,                       -- Snapshot of evaluated data
    
    -- Result
    status VARCHAR(20) NOT NULL,                  -- passed, failed, not_applicable, error
    severity rule_severity,
    
    -- Details
    details JSONB DEFAULT '{}',                   -- Operator values, distances, matched paths
    -- details schema: {
    --   "operand_values": { "field1": value1, "field2": value2 },
    --   "distance_to_threshold": 0.15,
    --   "matched_paths": ["account.client.kyc_status"],
    --   "failure_reasons": ["Field X is null", "Value Y out of range"]
    -- }
    
    -- Computed score (for prioritization)
    score DECIMAL(10, 4),
    
    -- Actions taken/suggested
    actions_executed JSONB DEFAULT '[]',
    actions_suggested JSONB DEFAULT '[]',
    
    -- Timing
    evaluation_time_ms INT,
    evaluated_at TIMESTAMPTZ DEFAULT NOW(),
    evaluated_by UUID,
    
    -- Override tracking
    was_overridden BOOLEAN DEFAULT FALSE,
    override_reason TEXT,
    override_by UUID,
    override_at TIMESTAMPTZ,
    
    -- Indexing
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- RuleViolation: Denormalized violation events for dashboards/reporting
CREATE TABLE IF NOT EXISTS rule_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    evaluation_result_id UUID REFERENCES rule_evaluation_results(id),
    rule_id UUID NOT NULL REFERENCES rules(id),
    
    -- Violation details
    violation_code VARCHAR(100),
    category rule_category NOT NULL,
    severity rule_severity NOT NULL,
    
    -- Context
    channel VARCHAR(100),
    entity_type VARCHAR(255),
    entity_id VARCHAR(255),
    
    -- Description
    title VARCHAR(500),
    description TEXT,
    
    -- Status
    status VARCHAR(50) DEFAULT 'open',            -- open, acknowledged, resolved, dismissed
    resolution_notes TEXT,
    resolved_by UUID,
    resolved_at TIMESTAMPTZ,
    
    -- Metadata for filtering/reporting
    regulation_ids TEXT[],
    tags TEXT[],
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- SIMULATION & TESTING
-- =============================================================================

-- RuleSimulation: Track simulation runs for impact analysis
CREATE TABLE IF NOT EXISTS rule_simulations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Simulation identity
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Rules being simulated
    rule_ids UUID[] NOT NULL,
    rule_logic_versions JSONB,                    -- {rule_id: version} mapping
    
    -- Data scope
    data_source_query TEXT,                       -- Query to get test data
    data_snapshot_id UUID,                        -- Reference to data snapshot
    sample_size INT,
    
    -- Results summary
    status VARCHAR(50) DEFAULT 'pending',         -- pending, running, completed, failed
    total_records INT,
    records_passed INT,
    records_failed INT,
    records_not_applicable INT,
    
    -- Impact metrics
    impact_summary JSONB DEFAULT '{}',
    -- impact_summary schema: {
    --   "by_severity": { "error": 150, "warning": 42 },
    --   "by_entity": { "Client": 100, "Account": 92 },
    --   "top_violations": [{ "rule_id": "...", "count": 50 }],
    --   "affected_clients": 25,
    --   "estimated_blocked_trades": 12
    -- }
    
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- INDEXES
-- =============================================================================

-- Rules indexes
CREATE INDEX IF NOT EXISTS idx_rules_tenant ON rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rules_tenant_category ON rules(tenant_id, category);
CREATE INDEX IF NOT EXISTS idx_rules_tenant_status ON rules(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_rules_tenant_env ON rules(tenant_id, environment);
CREATE INDEX IF NOT EXISTS idx_rules_scope_entity ON rules(scope_entity);
CREATE INDEX IF NOT EXISTS idx_rules_tags ON rules USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_rules_regulation ON rules USING GIN(regulation_ids);

-- Rule logic indexes
CREATE INDEX IF NOT EXISTS idx_rule_logic_rule ON rule_logic(rule_id);
CREATE INDEX IF NOT EXISTS idx_rule_logic_approved ON rule_logic(rule_id, is_approved);

-- Dependencies indexes
CREATE INDEX IF NOT EXISTS idx_rule_deps_rule ON rule_dependencies(rule_id);
CREATE INDEX IF NOT EXISTS idx_rule_deps_depends_on ON rule_dependencies(depends_on_rule_id);

-- Execution policy indexes
CREATE INDEX IF NOT EXISTS idx_exec_policy_tenant ON rule_execution_policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_exec_policy_channel ON rule_execution_policies(channel);
CREATE INDEX IF NOT EXISTS idx_exec_policy_category ON rule_execution_policies(category);

-- Evaluation results indexes
CREATE INDEX IF NOT EXISTS idx_eval_results_tenant ON rule_evaluation_results(tenant_id);
CREATE INDEX IF NOT EXISTS idx_eval_results_rule ON rule_evaluation_results(rule_id);
CREATE INDEX IF NOT EXISTS idx_eval_results_status ON rule_evaluation_results(status);
CREATE INDEX IF NOT EXISTS idx_eval_results_context ON rule_evaluation_results(context_type, context_id);
CREATE INDEX IF NOT EXISTS idx_eval_results_date ON rule_evaluation_results(evaluated_at);

-- Violations indexes
CREATE INDEX IF NOT EXISTS idx_violations_tenant ON rule_violations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_violations_rule ON rule_violations(rule_id);
CREATE INDEX IF NOT EXISTS idx_violations_status ON rule_violations(status);
CREATE INDEX IF NOT EXISTS idx_violations_severity ON rule_violations(severity);
CREATE INDEX IF NOT EXISTS idx_violations_entity ON rule_violations(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_violations_date ON rule_violations(created_at);

-- =============================================================================
-- SEED DATA: Action Types
-- =============================================================================

INSERT INTO rule_action_types (action_type, display_name, description, categories, contexts, params_schema, is_blocking, icon) VALUES
-- Data Quality actions
('reject_row', 'Reject Row', 'Reject the entire row/record', ARRAY['data_quality']::rule_category[], ARRAY['data_record']::rule_context_type[], '{"type": "object", "properties": {"reason": {"type": "string"}}}', true, 'block'),
('quarantine_row', 'Quarantine Row', 'Move row to quarantine for review', ARRAY['data_quality']::rule_category[], ARRAY['data_record']::rule_context_type[], '{"type": "object", "properties": {"queue_name": {"type": "string"}}}', true, 'pause_circle'),
('mask_value', 'Mask Value', 'Mask sensitive field value', ARRAY['data_quality', 'security']::rule_category[], ARRAY['data_record']::rule_context_type[], '{"type": "object", "properties": {"field": {"type": "string"}, "mask_pattern": {"type": "string"}}}', false, 'visibility_off'),
('default_value', 'Set Default Value', 'Apply a default value to field', ARRAY['data_quality', 'mdm']::rule_category[], ARRAY['data_record']::rule_context_type[], '{"type": "object", "properties": {"field": {"type": "string"}, "default": {"type": "any"}}}', false, 'edit'),
('flag_for_review', 'Flag for Review', 'Flag record for manual review', ARRAY['data_quality', 'compliance', 'mdm']::rule_category[], ARRAY['data_record', 'mdm_group']::rule_context_type[], '{"type": "object", "properties": {"queue_name": {"type": "string"}, "priority": {"type": "string"}}}', false, 'flag'),

-- Compliance actions
('block_trade', 'Block Trade', 'Block the trade from execution', ARRAY['compliance', 'wash_trade', 'values']::rule_category[], ARRAY['trade_event']::rule_context_type[], '{"type": "object", "properties": {"reason_code": {"type": "string"}}}', true, 'block'),
('require_approval', 'Require Approval', 'Route to approval workflow', ARRAY['compliance', 'values']::rule_category[], ARRAY['trade_event', 'data_record']::rule_context_type[], '{"type": "object", "properties": {"approver_role": {"type": "string"}, "sla_hours": {"type": "number"}}}', true, 'approval'),
('record_exception', 'Record Exception', 'Log compliance exception', ARRAY['compliance']::rule_category[], ARRAY['trade_event', 'data_record', 'client_profile']::rule_context_type[], '{"type": "object", "properties": {"exception_type": {"type": "string"}}}', false, 'note_add'),
('escalate_case', 'Escalate Case', 'Create or escalate a case', ARRAY['compliance', 'wash_trade']::rule_category[], ARRAY['trade_event', 'client_profile']::rule_context_type[], '{"type": "object", "properties": {"case_type": {"type": "string"}, "priority": {"type": "string"}}}', false, 'priority_high'),

-- MDM actions
('select_survivor_value', 'Select Survivor Value', 'Choose winning value in merge', ARRAY['mdm']::rule_category[], ARRAY['mdm_group', 'data_record']::rule_context_type[], '{"type": "object", "properties": {"source_priority": {"type": "array"}, "field": {"type": "string"}}}', false, 'merge_type'),
('prevent_merge', 'Prevent Merge', 'Block automatic merge of records', ARRAY['mdm']::rule_category[], ARRAY['mdm_group']::rule_context_type[], '{"type": "object", "properties": {"reason": {"type": "string"}}}', true, 'call_split'),
('flag_conflict', 'Flag Conflict', 'Mark conflicting values for review', ARRAY['mdm']::rule_category[], ARRAY['mdm_group']::rule_context_type[], '{"type": "object", "properties": {"fields": {"type": "array"}}}', false, 'warning'),

-- Wash trade actions
('tag_suspicious', 'Tag Suspicious', 'Tag pattern as suspicious', ARRAY['wash_trade']::rule_category[], ARRAY['trade_event']::rule_context_type[], '{"type": "object", "properties": {"pattern_type": {"type": "string"}, "confidence": {"type": "number"}}}', false, 'local_fire_department'),
('route_to_risk_queue', 'Route to Risk Queue', 'Send to risk team review', ARRAY['wash_trade', 'compliance']::rule_category[], ARRAY['trade_event']::rule_context_type[], '{"type": "object", "properties": {"queue_name": {"type": "string"}, "urgency": {"type": "string"}}}', false, 'move_to_inbox'),

-- Values/ESG actions
('exclude_instrument', 'Exclude Instrument', 'Mark instrument as excluded', ARRAY['values']::rule_category[], ARRAY['portfolio', 'trade_event']::rule_context_type[], '{"type": "object", "properties": {"reason": {"type": "string"}, "theme": {"type": "string"}}}', true, 'remove_circle'),
('cap_exposure', 'Cap Exposure', 'Limit exposure to threshold', ARRAY['values', 'rebalancing']::rule_category[], ARRAY['portfolio']::rule_context_type[], '{"type": "object", "properties": {"max_percent": {"type": "number"}, "scope": {"type": "string"}}}', false, 'trending_down'),
('notify_client', 'Notify Client', 'Send notification to client', ARRAY['values', 'compliance']::rule_category[], ARRAY['portfolio', 'client_profile']::rule_context_type[], '{"type": "object", "properties": {"template_id": {"type": "string"}, "channel": {"type": "string"}}}', false, 'notifications'),
('attach_explanation', 'Attach Explanation', 'Add explanation to holding/trade', ARRAY['values']::rule_category[], ARRAY['portfolio', 'trade_event']::rule_context_type[], '{"type": "object", "properties": {"explanation_type": {"type": "string"}}}', false, 'description'),

-- Rebalancing actions
('trigger_rebalance', 'Trigger Rebalance', 'Initiate portfolio rebalance', ARRAY['rebalancing']::rule_category[], ARRAY['portfolio']::rule_context_type[], '{"type": "object", "properties": {"strategy": {"type": "string"}, "urgency": {"type": "string"}}}', false, 'autorenew'),
('generate_trades', 'Generate Trades', 'Generate rebalancing trades', ARRAY['rebalancing']::rule_category[], ARRAY['portfolio']::rule_context_type[], '{"type": "object", "properties": {"constraints": {"type": "object"}}}', false, 'list_alt'),
('tlh_harvest', 'Tax-Loss Harvest', 'Execute tax-loss harvesting', ARRAY['rebalancing']::rule_category[], ARRAY['portfolio']::rule_context_type[], '{"type": "object", "properties": {"min_loss": {"type": "number"}, "substitute_rules": {"type": "object"}}}', false, 'savings'),

-- Generic actions
('emit_event', 'Emit Event', 'Publish event to message bus', ARRAY['data_quality', 'compliance', 'mdm', 'wash_trade', 'values', 'rebalancing', 'workflow', 'security', 'custom']::rule_category[], ARRAY['data_record', 'trade_event', 'portfolio', 'client_profile', 'mdm_group', 'system_job']::rule_context_type[], '{"type": "object", "properties": {"event_type": {"type": "string"}, "topic": {"type": "string"}, "payload_template": {"type": "object"}}}', false, 'send'),
('log_only', 'Log Only', 'Log violation without action', ARRAY['data_quality', 'compliance', 'mdm', 'wash_trade', 'values', 'rebalancing', 'workflow', 'security', 'custom']::rule_category[], ARRAY['data_record', 'trade_event', 'portfolio', 'client_profile', 'mdm_group', 'system_job']::rule_context_type[], '{"type": "object", "properties": {"log_level": {"type": "string"}}}', false, 'note')
ON CONFLICT (action_type) DO NOTHING;

-- =============================================================================
-- FUNCTIONS
-- =============================================================================

-- Function to get active rule logic for a rule
CREATE OR REPLACE FUNCTION get_active_rule_logic(
    p_rule_id UUID,
    p_environment VARCHAR DEFAULT 'prod'
) RETURNS TABLE (
    rule_logic_id UUID,
    version INT,
    condition_json JSONB,
    actions_json JSONB,
    scoring_formula TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        rl.id,
        rl.version,
        rl.condition_json,
        rl.actions_json,
        rl.scoring_formula
    FROM rule_logic rl
    JOIN rules r ON r.id = rl.rule_id
    WHERE rl.rule_id = p_rule_id
      AND rl.is_approved = TRUE
      AND r.environment = p_environment
    ORDER BY rl.version DESC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Function to get all active rules for evaluation
CREATE OR REPLACE FUNCTION get_rules_for_evaluation(
    p_tenant_id UUID,
    p_category rule_category DEFAULT NULL,
    p_context_type rule_context_type DEFAULT NULL,
    p_entity VARCHAR DEFAULT NULL,
    p_channel VARCHAR DEFAULT NULL,
    p_environment VARCHAR DEFAULT 'prod'
) RETURNS TABLE (
    rule_id UUID,
    rule_code VARCHAR,
    rule_name VARCHAR,
    category rule_category,
    severity rule_severity,
    condition_json JSONB,
    actions_json JSONB,
    scoring_formula TEXT,
    enforcement enforcement_mode,
    timeout_ms INT
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT
        r.id,
        r.rule_code,
        r.name,
        r.category,
        r.severity,
        rl.condition_json,
        rl.actions_json,
        rl.scoring_formula,
        COALESCE(ep.enforcement, 'hard_block'::enforcement_mode),
        COALESCE(ep.timeout_ms, 5000)
    FROM rules r
    JOIN rule_logic rl ON r.id = rl.rule_id
    LEFT JOIN rule_execution_policies ep ON (
        ep.tenant_id = r.tenant_id 
        AND ep.environment = r.environment
        AND ep.is_active = TRUE
        AND (ep.category IS NULL OR ep.category = r.category)
        AND (ep.channel IS NULL OR ep.channel = p_channel)
    )
    WHERE r.tenant_id = p_tenant_id
      AND r.status = 'active'
      AND r.environment = p_environment
      AND rl.is_approved = TRUE
      AND (p_category IS NULL OR r.category = p_category)
      AND (p_context_type IS NULL OR r.primary_context = p_context_type)
      AND (p_entity IS NULL OR r.scope_entity = p_entity)
      AND (r.effective_from IS NULL OR r.effective_from <= NOW())
      AND (r.effective_to IS NULL OR r.effective_to >= NOW())
    ORDER BY r.category, r.rule_code;
END;
$$ LANGUAGE plpgsql;

-- Trigger function for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers
DROP TRIGGER IF EXISTS update_rules_updated_at ON rules;
CREATE TRIGGER update_rules_updated_at
    BEFORE UPDATE ON rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_exec_policy_updated_at ON rule_execution_policies;
CREATE TRIGGER update_exec_policy_updated_at
    BEFORE UPDATE ON rule_execution_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_violations_updated_at ON rule_violations;
CREATE TRIGGER update_violations_updated_at
    BEFORE UPDATE ON rule_violations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE rules IS 'Core rule definitions - domain-agnostic metadata for all rule types';
COMMENT ON TABLE rule_logic IS 'Versioned rule logic with conditions and actions';
COMMENT ON TABLE rule_dependencies IS 'Relationships between rules (preconditions, exclusions, etc.)';
COMMENT ON TABLE rule_execution_policies IS 'Channel-specific enforcement configuration';
COMMENT ON TABLE rule_action_types IS 'Registry of available action types per category';
COMMENT ON TABLE rule_evaluation_results IS 'Results of rule evaluations for audit trail';
COMMENT ON TABLE rule_violations IS 'Denormalized violation events for dashboards';
COMMENT ON TABLE rule_simulations IS 'Simulation runs for impact analysis';

COMMENT ON TYPE rule_category IS 'Categories of rules: DQ, compliance, MDM, wash trades, values, rebalancing, etc.';
COMMENT ON TYPE rule_context_type IS 'Primary context for rule evaluation: data_record, trade_event, portfolio, etc.';
COMMENT ON TYPE rule_severity IS 'Severity levels: info, warning, error, hard_block, soft_block, quarantine';
COMMENT ON TYPE rule_status IS 'Rule lifecycle: draft, awaiting_approval, active, deprecated, retired';
COMMENT ON TYPE enforcement_mode IS 'Enforcement modes: hard_block, soft_block, log_only, simulate, disabled';
