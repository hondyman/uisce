-- ============================================================================
-- WEALTH TRANSFER PLATFORM: METADATA CONFIGURATION
-- ============================================================================
-- Migration: 022_wealth_transfer_metadata.sql
-- Purpose: Configurable templates, validation rules, and workflow definitions
--          for metadata-first estate planning (Workday-style configuration)
-- ============================================================================

-- ============================================================================
-- STRATEGY TEMPLATES (Reusable Estate Planning Configurations)
-- ============================================================================

CREATE TABLE strategy_templates (
    template_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Template Identification
    strategy_type strategy_type_enum NOT NULL,
    template_name TEXT NOT NULL,
    template_description TEXT,
    template_version VARCHAR(20) DEFAULT '1.0',
    
    -- Configuration (metadata-driven)
    configuration JSONB NOT NULL,
    /* Example for SLAT template:
    {
        "funding_calculation": {
            "method": "PERCENTAGE_OF_EXEMPTION",
            "percentage": 80,
            "max_percentage_of_liquid": 40
        },
        "trust_terms": {
            "grantor_spouse": "NON_DONOR",
            "beneficiaries": ["SPOUSE", "DESCENDANTS"],
            "distribution_standard": "HEMS",
            "spendthrift_clause": true
        },
        "tax_benefits": {
            "removes_growth_from_estate": true,
            "allows_spousal_access": true,
            "uses_exemption": true
        },
        "requirements": {
            "married": true,
            "min_liquid_assets": 1000000,
            "min_networth": 5000000
        }
    }
    */
    
    -- Eligibility Rules
    min_networth DECIMAL(15,2),
    max_networth DECIMAL(15,2),
    min_age INTEGER,
    max_age INTEGER,
    requires_married BOOLEAN DEFAULT FALSE,
    requires_children BOOLEAN DEFAULT FALSE,
    requires_business_interest BOOLEAN DEFAULT FALSE,
    
    -- Complexity & Cost
   default_complexity_score INTEGER CHECK (default_complexity_score BETWEEN 1 AND 10),
    typical_implementation_weeks INTEGER,
    typical_implementation_cost DECIMAL(12,2),
    typical_annual_maintenance_cost DECIMAL(12,2),
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    
    CONSTRAINT unique_strategy_template UNIQUE(strategy_type, template_version)
);

CREATE INDEX idx_strategy_template_type ON strategy_templates(strategy_type) WHERE is_active = TRUE;
CREATE INDEX idx_strategy_template_config ON strategy_templates USING GIN(configuration);

COMMENT ON TABLE strategy_templates IS 'Configurable estate planning strategy templates - metadata-first design';

-- Seed core strategy templates
INSERT INTO strategy_templates (strategy_type, template_name, template_description, min_networth, default_complexity_score, typical_implementation_weeks, configuration) VALUES

('ANNUAL_GIFTING', 'Annual Exclusion Gifting', 'Maximize use of annual gift tax exclusion', 1000000, 2, 1, '{
    "gifting_rules": {
        "per_recipient_annual": "CURRENT_EXCLUSION",
        "spousal_split_default": true,
        "track_crummey_notices": false
    },
    "tax_benefits": {
        "estate_reduction": true,
        "no_gift_tax": true,
        "removes_future_growth": true
    },
    "requirements": {
        "min_recipients": 1
    }
}'::jsonb),

('SLAT', 'Spousal Lifetime Access Trust', 'Irrevocable trust for spouse benefit using lifetime exemption', 5000000, 6, 8, '{
    "funding_calculation": {"method": "PERCENTAGE_OF_EXEMPTION", "percentage": 80},
    "trust_terms": {
        "grantor_spouse": "NON_DONOR",
        "beneficiaries": ["SPOUSE", "DESCENDANTS"],
        "distribution_standard": "HEMS"
    },
    "requirements": {"married": true, "min_liquid_assets": 2000000}
}'::jsonb),

('GRAT', 'Grantor Retained Annuity Trust', 'Transfer appreciating assets with minimal gift tax', 10000000, 7, 12, '{
    "term_options": [2, 3, 5, 7, 10],
    "annuity_calculation": {"method": "7520_RATE_PLUS", "basis_points": 0},
    "funding_targets": {"high_growth_assets": true, "volatile_markets": false},
    "requirements": {"min_appreciating_assets": 5000000, "good_health": true}
}'::jsonb),

('ILIT', 'Irrevocable Life Insurance Trust', 'Keep life insurance proceeds out of estate', 2000000, 5, 6, '{
    "insurance_requirements": {"min_death_benefit": 1000000},
    "trust_terms": {"crummey_powers": true, "withdrawal_window_days": 30},
    "funding": {"annual_premium_gifts": true, "trustee_purchases_policy": true},
    "requirements": {"min_life_insurance": 1000000, "insurable_interest": true}
}'::jsonb),

('DYNASTY_TRUST', 'Dynasty Trust', 'Multi-generational wealth transfer using GST exemption', 15000000, 8, 16, '{
    "duration": {"perpetual_if_allowed": true, "fallback_years": 999},
    "funding": {"use_gst_exemption": true, "funding_amount_method": "MAX_EXEMPTION"},
    "beneficiaries": {"all_descendants": true, "per_stirpes": true},
    "tax_benefits": {"no_estate_tax_any_generation": true, "no_gst_tax": true},
    "requirements": {"min_networth": 15000000, "has_grandchildren_or_planned": true}
}'::jsonb);

-- ============================================================================
-- VALIDATION RULES (Configurable Constraints)
-- ============================================================================

CREATE TABLE estate_planning_validation_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Rule Identification
    rule_name TEXT NOT NULL UNIQUE,
    rule_description TEXT,
    rule_category VARCHAR(50), -- 'ELIGIBILITY', 'COMPLIANCE', 'BEST_PRACTICE', 'TAX_LAW'
    
    -- Rule Logic (metadata-driven)
    rule_expression JSONB NOT NULL,
    /* Example:
    {
        "condition": "AND",
        "rules": [
            {"field": "networth", "operator": ">=", "value": 5000000},
            {"field": "marital_status", "operator": "==", "value": "MARRIED"},
            {"field": "liquid_assets", "operator": ">=", "value": 2000000}
        ]
    }
    */
    
    -- Severity & Messaging
    severity VARCHAR(20) NOT NULL, -- 'ERROR', 'WARNING', 'INFO'
    error_message_template TEXT,
    remediation_guidance TEXT,
    
    -- Applicability
    applies_to_strategies TEXT[], -- Array of strategy_type_enum values
    applies_to_jurisdictions TEXT[], -- Array of jurisdiction codes
    
    -- Lifecycle
    effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
    expiration_date DATE,
    is_active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_validation_rule_category ON estate_planning_validation_rules(rule_category) WHERE is_active = TRUE;
CREATE INDEX idx_validation_rule_strategy ON estate_planning_validation_rules USING GIN(applies_to_strategies);

COMMENT ON TABLE estate_planning_validation_rules IS 'Configurable validation rules for estate planning eligibility and compliance';

-- Seed validation rules
INSERT INTO estate_planning_validation_rules (rule_name, rule_description, rule_category, severity, rule_expression, error_message_template, applies_to_strategies) VALUES

('SLAT_REQUIRES_MARRIAGE', 'SLAT requires married couple', 'ELIGIBILITY', 'ERROR', '{
    "condition": "AND",
    "rules": [{"field": "marital_status", "operator": "==", "value": "MARRIED"}]
}'::jsonb, 'Spousal Lifetime Access Trust requires the grantor to be married', ARRAY['SLAT']),

('GRAT_MINIMUM_ASSETS', 'GRAT requires substantial appreciating assets', 'ELIGIBILITY', 'ERROR', '{
    "condition": "AND", 
    "rules": [
        {"field": "networth", "operator": ">=", "value": 10000000},
        {"field": "appreciating_assets", "operator": ">=", "value": 5000000}
    ]
}'::jsonb, 'GRAT strategy requires at least $5M in appreciating assets and $10M net worth', ARRAY['GRAT']),

('DYNASTY_TRUST_GST_CHECK', 'Dynasty trust requires available GST exemption', 'COMPLIANCE', 'WARNING', '{
    "condition": "AND",
    "rules": [{"field": "gst_exemption_remaining", "operator": ">", "value": 0}]
}'::jsonb, 'No GST exemption remaining. Dynasty trust will incur 40% GST tax.', ARRAY['DYNASTY_TRUST' ]),

('EXCESSIVE_GIFTING_WARNING', 'Annual gifts exceed prudent levels', 'BEST_PRACTICE', 'WARNING', '{
    "condition": "AND",
    "rules": [{"field": "annual_gifts_pct_of_networth", "operator": ">", "value": 10}]
}'::jsonb, 'Annual gifts exceed 10% of net worth. Consider liquidity needs.', ARRAY['ANNUAL_GIFTING']),

('RECIPROCAL_TRUST_DOCTRINE', 'Warning about reciprocal trust issues with SLAT', 'COMPLIANCE', 'WARNING', '{
    "condition": "AND",
    "rules": [
        {"field": "has_multiple_slats", "operator": "==", "value": true},
        {"field": "trust_terms_substantially_similar", "operator": "==", "value": true}
    ]
}'::jsonb, 'Multiple SLATs with similar terms may trigger reciprocal trust doctrine. Differentiate trust terms.', ARRAY['SLAT']);

-- ============================================================================
-- WORKFLOW DEFINITIONS (Temporal Workflow Configuration)
-- ============================================================================

CREATE TABLE workflow_definitions (
    workflow_def_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Workflow Identification
    workflow_name TEXT NOT NULL UNIQUE,
    workflow_type VARCHAR(100) NOT NULL, -- Temporal workflow type name
    workflow_description TEXT,
    
    -- Trigger Configuration
    trigger_type VARCHAR(50) NOT NULL, -- 'MANUAL', 'SCHEDULED', 'EVENT', 'LIFECYCLE'
    trigger_config JSONB,
    /* Example for scheduled review:
    {
        "schedule": "ANNUAL",
        "specific_date": "01-15", // January 15
        "advance_notice_days": 60,
        "conditions": {
            "estate_plan_status": "IMPLEMENTED",
            "days_since_last_review": 365
        }
    }
    */
    
    -- Workflow Parameters
    default_parameters JSONB,
    parameter_schema JSONB, -- JSON Schema for validation
    
    -- Execution Settings
    task_queue TEXT DEFAULT 'wealth-transfer',
    timeout_seconds INTEGER DEFAULT 3600,
    retry_policy JSONB,
    
    -- Lifecycle
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_workflow_def_type ON workflow_definitions(workflow_type) WHERE is_active = TRUE;
CREATE INDEX idx_workflow_def_trigger ON workflow_definitions(trigger_type) WHERE is_active = TRUE;

COMMENT ON TABLE workflow_definitions IS 'Temporal workflow configuration - metadata-driven orchestration';

-- Seed workflow definitions
INSERT INTO workflow_definitions (workflow_name, workflow_type, workflow_description, trigger_type, trigger_config, default_parameters) VALUES

('Estate Plan Generation', 'EstatePlanGenerationWorkflow', 'Generate comprehensive estate planning scenarios', 'MANUAL', '{}'::jsonb, '{
    "max_scenarios": 15,
    "include_ml_optimization": true,
    "generate_narratives": true
}'::jsonb),

('Annual Plan Review', 'AnnualPlanReviewWorkflow', 'Scheduled annual review of existing estate plans', 'SCHEDULED', '{
    "schedule": "ANNUAL",
    "specific_date": "01-15",
    "advance_notice_days": 60
}'::jsonb, '{
    "recalculate_taxes": true,
    "check_law_changes": true,
    "compare_to_baseline": true
}'::jsonb),

('Gift Tax Filing', 'GiftTaxFilingWorkflow', 'Automated Form 709 preparation and filing', 'LIFECYCLE', '{
    "trigger_event": "ANNUAL_GIFTS_EXCEED_EXCLUSION",
    "filing_deadline": "04-15",
    "reminder_days_before": [90, 60, 30, 14, 7]
}'::jsonb, '{
    "include_prior_year_gifts": true,
    "calculate_gst_allocation": true,
    "generate_pdf": true
}'::jsonb),

('Trust Compliance Monitoring', 'TrustComplianceWorkflow', 'Monitor trust compliance and filing requirements', 'SCHEDULED', '{
    "schedule": "QUARTERLY",
    "check_tax_filings": true,
    "check_distribution_rules": true
}'::jsonb, '{
    "alert_on_violations": true,
    "generate_compliance_report": true
}'::jsonb);

-- ============================================================================
-- ML MODEL REGISTRY
-- ============================================================================

CREATE TABLE ml_model_registry (
    model_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Model Identification
    model_name TEXT NOT NULL,
    model_version VARCHAR(20) NOT NULL,
    model_type VARCHAR(50) NOT NULL, -- 'RECOMMENDER', 'OPTIMIZER', 'SCORER', 'CLASSIFIER'
    
    -- Model Metadata
    algorithm TEXT, -- 'RandomForest', 'GradientBoosting', 'NeuralNetwork'
    framework TEXT, -- 'sklearn', 'tensorflow', 'pytorch'
    
    -- Training Information
    training_date TIMESTAMPTZ NOT NULL,
    training_dataset_size INTEGER,
    training_features TEXT[],
    feature_importance JSONB,
    
    -- Performance Metrics
    validation_accuracy DECIMAL(5,4),
    test_accuracy DECIMAL(5,4),
    precision_score DECIMAL(5,4),
    recall_score DECIMAL(5,4),
    f1_score DECIMAL(5,4),
    
    -- Model Artifacts
    model_file_path TEXT NOT NULL,
    model_file_checksum TEXT,
    preprocessing_pipeline_path TEXT,
    
    -- Hyperparameters
    hyperparameters JSONB,
    
    -- Deployment
    deployed BOOLEAN DEFAULT FALSE,
    deployment_date TIMESTAMPTZ,
    deployment_environment VARCHAR(20), -- 'PRODUCTION', 'STAGING', 'DEVELOPMENT'
    
    -- Monitoring
    prediction_count INTEGER DEFAULT 0,
    average_prediction_time_ms DECIMAL(8,2),
    last_prediction_at TIMESTAMPTZ,
    
    -- Lifecycle
    is_active BOOLEAN DEFAULT TRUE,
    deprecated_date TIMESTAMPTZ,
    replaced_by_model_id UUID REFERENCES ml_model_registry(model_id),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_model_version UNIQUE(model_name, model_version)
);

CREATE INDEX idx_ml_model_deployed ON ml_model_registry(deployed, deployment_date DESC) WHERE is_active = TRUE;
CREATE INDEX idx_ml_model_type ON ml_model_registry(model_type, model_version DESC);

COMMENT ON TABLE ml_model_registry IS 'ML model versioning and metadata for estate planning AI';

-- Seed initial model registry (placeholders)
INSERT INTO ml_model_registry (model_name, model_version, model_type, algorithm, framework, training_date, model_file_path, deployed) VALUES
('EstratePlanRecommender', 'v1.0', 'RECOMMENDER', 'RandomForest', 'sklearn', NOW(), '/models/recommender_v1.0.pkl', FALSE),
('ScenarioOptimizer', 'v1.0', 'OPTIMIZER', 'SLSQP', 'scipy', NOW(), '/models/optimizer_v1.0.pkl', FALSE),
('ConfidenceScorer', 'v1.0', 'SCORER', 'GradientBoosting', 'sklearn', NOW(), '/models/confidence_v1.0.pkl', FALSE);

-- ============================================================================
-- TAX LAW CHANGE TRACKING
-- ============================================================================

CREATE TABLE tax_law_changes (
    change_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Change Details
    jurisdiction_code VARCHAR(10) NOT NULL,
    change_title TEXT NOT NULL,
    change_description TEXT,
    change_category VARCHAR(50), -- 'EXEMPTION', 'RATE', 'DEDUCTION', 'PROCEDURE'
    
    -- Dates
    announced_date DATE,
    effective_date DATE NOT NULL,
    sunset_date DATE,
    
    -- Impact Assessment
    estimated_families_affected INTEGER,
    average_tax_impact DECIMAL(15,2),
    requires_plan_update BOOLEAN DEFAULT TRUE,
    
    -- Documentation
    law_citation TEXT,
    irs_guidance_url TEXT,
    internal_memo_id UUID,
    
    -- Status
    status VARCHAR(20) DEFAULT 'PENDING', -- 'PENDING', 'ENACTED', 'PROPOSED', 'DEFEATED'
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tax_law_change_jurisdiction ON tax_law_changes(jurisdiction_code, effective_date DESC);
CREATE INDEX idx_tax_law_change_status ON tax_law_changes(status, effective_date);

COMMENT ON TABLE tax_law_changes IS 'Tracking of tax law changes for proactive plan updates';

-- Seed known upcoming change (2026 exemption sunset)
INSERT INTO tax_law_changes (jurisdiction_code, change_title, change_description, change_category, effective_date, sunset_date, estimated_families_affected, requires_plan_update, status) VALUES
('US-FEDERAL', '2026 Estate Tax Exemption Sunset', 'Federal estate tax exemption reverts from ~$14M to ~$7M per person', 'EXEMPTION', '2026-01-01', NULL, 500000, TRUE, 'ENACTED');

-- ============================================================================
-- CONFIGURATION AUDIT LOG
-- ============================================================================

CREATE TABLE configuration_audit_log (
    audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- What Changed
    table_name TEXT NOT NULL,
    record_id UUID NOT NULL,
    operation VARCHAR(10) NOT NULL, -- 'INSERT', 'UPDATE', 'DELETE'
    
    -- Changes
    old_values JSONB,
    new_values JSONB,
    changed_fields TEXT[],
    
    -- Who & When
    changed_by UUID,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Context
    change_reason TEXT,
    approval_required BOOLEAN DEFAULT FALSE,
    approved_by UUID,
    approved_at TIMESTAMPTZ
);

CREATE INDEX idx_config_audit_table ON configuration_audit_log(table_name, record_id);
CREATE INDEX idx_config_audit_time ON configuration_audit_log(changed_at DESC);

COMMENT ON TABLE configuration_audit_log IS 'Audit trail for all metadata configuration changes';

-- Trigger function for configuration auditing
CREATE OR REPLACE FUNCTION audit_configuration_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Only audit configuration tables
    IF TG_TABLE_NAME IN ('strategy_templates', 'estate_planning_validation_rules', 
                          'workflow_definitions', 'tax_jurisdictions') THEN
        
        IF TG_OP = 'DELETE' THEN
            INSERT INTO configuration_audit_log(table_name, record_id, operation, old_values)
            VALUES (TG_TABLE_NAME, OLD.template_id, 'DELETE', to_jsonb(OLD));
            RETURN OLD;
        ELSIF TG_OP = 'UPDATE' THEN
            INSERT INTO configuration_audit_log(table_name, record_id, operation, old_values, new_values)
            VALUES (TG_TABLE_NAME, NEW.template_id, 'UPDATE', to_jsonb(OLD), to_jsonb(NEW));
            RETURN NEW;
        ELSIF TG_OP = 'INSERT' THEN
            INSERT INTO configuration_audit_log(table_name, record_id, operation, new_values)
            VALUES (TG_TABLE_NAME, NEW.template_id, 'INSERT', to_jsonb(NEW));
            RETURN NEW;
        END IF;
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Apply audit triggers to configuration tables
CREATE TRIGGER audit_strategy_templates
AFTER INSERT OR UPDATE OR DELETE ON strategy_templates
FOR EACH ROW EXECUTE FUNCTION audit_configuration_change();

CREATE TRIGGER audit_validation_rules
AFTER INSERT OR UPDATE OR DELETE ON estate_planning_validation_rules
FOR EACH ROW EXECUTE FUNCTION audit_configuration_change();

CREATE TRIGGER audit_workflow_definitions
AFTER INSERT OR UPDATE OR DELETE ON workflow_definitions
FOR EACH ROW EXECUTE FUNCTION audit_configuration_change();

-- Migration complete
COMMENT ON SCHEMA public IS 'Wealth Transfer Platform v1.0 - Metadata configuration deployed';
