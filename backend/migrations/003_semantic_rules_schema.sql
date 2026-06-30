-- Phase 3: Semantic Rules Engine Schema
-- Tables for rules, versions, approvals, and governance

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Rules table - main rule definitions
CREATE TABLE IF NOT EXISTS edm.rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    business_object VARCHAR(255) NOT NULL,
    name VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'draft', -- draft, testing, staging, production
    current_version INT NOT NULL DEFAULT 1,
    default_action VARCHAR(255),
    created_by UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_by UUID,
    
    -- Constraints
    CONSTRAINT rules_status_check CHECK (status IN ('draft', 'testing', 'staging', 'production')),
    CONSTRAINT rules_version_check CHECK (current_version > 0)
);

-- Index for common queries
CREATE INDEX IF NOT EXISTS idx_rules_tenant_business ON edm.rules(tenant_id, business_object);
CREATE INDEX IF NOT EXISTS idx_rules_status ON edm.rules(status);
CREATE INDEX IF NOT EXISTS idx_rules_created_by ON edm.rules(created_by);
CREATE INDEX IF NOT EXISTS idx_rules_updated_at ON edm.rules(updated_at DESC);

-- Row-level security
ALTER TABLE edm.rules ENABLE ROW LEVEL SECURITY;
CREATE POLICY rules_tenant_isolation ON edm.rules
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Rule steps table - individual priority conditions
CREATE TABLE IF NOT EXISTS edm.rule_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL,
    version INT NOT NULL,
    priority INT NOT NULL,
    semantic_term VARCHAR(255) NOT NULL,
    operator VARCHAR(50) NOT NULL,
    value TEXT NOT NULL,
    confidence INT NOT NULL DEFAULT 100,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT rule_steps_rule_fk FOREIGN KEY (rule_id) REFERENCES edm.rules(id) ON DELETE CASCADE,
    CONSTRAINT rule_steps_priority_check CHECK (priority >= 1),
    CONSTRAINT rule_steps_confidence_check CHECK (confidence >= 0 AND confidence <= 100),
    CONSTRAINT rule_steps_operator_check CHECK (operator IN (
        'equals', 'contains', 'starts_with', 'in_list',
        'after', 'before', 'between',
        'greater_than', 'less_than'
    ))
);

-- Index for rule steps
CREATE INDEX IF NOT EXISTS idx_rule_steps_rule_version ON edm.rule_steps(rule_id, version);
CREATE INDEX IF NOT EXISTS idx_rule_steps_priority ON edm.rule_steps(priority);

-- Rule versions table - version history
CREATE TABLE IF NOT EXISTS edm.rule_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL,
    version INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    promoted_at TIMESTAMP,
    promoted_by UUID,
    source_version INT,
    release_notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT rule_versions_rule_fk FOREIGN KEY (rule_id) REFERENCES edm.rules(id) ON DELETE CASCADE,
    CONSTRAINT rule_versions_unique UNIQUE (rule_id, version),
    CONSTRAINT rule_versions_version_check CHECK (version > 0)
);

-- Index for versions
CREATE INDEX IF NOT EXISTS idx_rule_versions_rule_status ON edm.rule_versions(rule_id, status);
CREATE INDEX IF NOT EXISTS idx_rule_versions_promoted_at ON edm.rule_versions(promoted_at DESC);

-- Approval records table - governance workflow
CREATE TABLE IF NOT EXISTS edm.rule_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL,
    version INT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, approved, rejected
    promotion_stage VARCHAR(50), -- testing, staging, production
    role VARCHAR(100) NOT NULL, -- data_steward, compliance_officer, business_owner
    approver_id UUID,
    approved_at TIMESTAMP,
    rejection_reason TEXT,
    comments TEXT,
    required_for_promotion BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT rule_approvals_rule_fk FOREIGN KEY (rule_id) REFERENCES edm.rules(id) ON DELETE CASCADE,
    CONSTRAINT rule_approvals_status_check CHECK (status IN ('pending', 'approved', 'rejected')),
    CONSTRAINT rule_approvals_role_check CHECK (role IN (
        'data_steward', 'compliance_officer', 'business_owner'
    ))
);

-- Index for approvals
CREATE INDEX IF NOT EXISTS idx_rule_approvals_rule_version ON edm.rule_approvals(rule_id, version);
CREATE INDEX IF NOT EXISTS idx_rule_approvals_status ON edm.rule_approvals(status);
CREATE INDEX IF NOT EXISTS idx_rule_approvals_approver ON edm.rule_approvals(approver_id);
CREATE INDEX IF NOT EXISTS idx_rule_approvals_created ON edm.rule_approvals(created_at DESC);

-- Row-level security for approvals
ALTER TABLE edm.rule_approvals ENABLE ROW LEVEL SECURITY;

-- Approval workflow requirements - defines who approves what
CREATE TABLE IF NOT EXISTS edm.approval_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_object VARCHAR(255) NOT NULL,
    promotion_stage VARCHAR(50) NOT NULL, -- testing, staging, production
    required_role VARCHAR(100) NOT NULL,
    sequence_order INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT approval_workflows_stage_check CHECK (promotion_stage IN ('testing', 'staging', 'production')),
    CONSTRAINT approval_workflows_unique UNIQUE (business_object, promotion_stage, required_role),
    CONSTRAINT approval_workflows_order_check CHECK (sequence_order >= 1)
);

-- Semantic terms catalog - business-friendly rule dimensions
-- If table already exists but columns are missing from an earlier run, add them
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables
                   WHERE table_schema = 'edm' AND table_name = 'semantic_terms') THEN
        CREATE TABLE IF NOT EXISTS edm.semantic_terms (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            business_object VARCHAR(255) NOT NULL,
            name VARCHAR(255) NOT NULL,
            data_type VARCHAR(50) NOT NULL, -- string, boolean, date, number
            business_definition TEXT NOT NULL,
            source_field VARCHAR(255),
            sample_values TEXT[], -- Array of example values
            governance_status VARCHAR(50) DEFAULT 'approved', -- approved, draft, deprecated
            category VARCHAR(100) NOT NULL, -- identification, classification, data_quality, business_impact
            created_by UUID NOT NULL,
            created_at TIMESTAMP NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
            
            -- Constraints
            CONSTRAINT semantic_terms_data_type_check CHECK (data_type IN ('string', 'boolean', 'date', 'number')),
            CONSTRAINT semantic_terms_category_check CHECK (category IN (
                'identification', 'classification', 'data_quality', 'business_impact'
            ))
        );
    ELSE
        ALTER TABLE edm.semantic_terms
            ADD COLUMN IF NOT EXISTS business_object VARCHAR(255),
            ADD COLUMN IF NOT EXISTS category VARCHAR(100),
            ADD COLUMN IF NOT EXISTS governance_status VARCHAR(50) DEFAULT 'approved';
    END IF;
END$$;

-- Index for semantic terms
CREATE INDEX IF NOT EXISTS idx_semantic_terms_business_object ON edm.semantic_terms(business_object);
CREATE INDEX IF NOT EXISTS idx_semantic_terms_category ON edm.semantic_terms(category);
CREATE INDEX IF NOT EXISTS idx_semantic_terms_governance_status ON edm.semantic_terms(governance_status);

-- Rule execution audit trail - simulation/execution history
-- drop old table if present to avoid mismatched column types
DROP TABLE IF EXISTS edm.rule_execution_history;
CREATE TABLE IF NOT EXISTS edm.rule_execution_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL,
    version INT NOT NULL,
    execution_type VARCHAR(50) NOT NULL, -- simulation, scheduled, manual
    input_data JSONB NOT NULL,
    output_data JSONB NOT NULL,
    matched_steps TEXT,
    winning_rule_step UUID,
    execution_duration_ms INT,
    status VARCHAR(50) NOT NULL DEFAULT 'success', -- success, error
    error_message TEXT,
    executed_by UUID,
    executed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT rule_exec_rule_fk FOREIGN KEY (rule_id) REFERENCES edm.rules(id) ON DELETE CASCADE,
    CONSTRAINT rule_exec_type_check CHECK (execution_type IN ('simulation', 'scheduled', 'manual'))
);

-- Index for execution history
CREATE INDEX IF NOT EXISTS idx_rule_execution_rule_version ON edm.rule_execution_history(rule_id, version);
CREATE INDEX IF NOT EXISTS idx_rule_execution_executed_at ON edm.rule_execution_history(executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_rule_execution_type ON edm.rule_execution_history(execution_type);

-- Upsert semantic terms for calendar business object
INSERT INTO edm.semantic_terms (business_object, name, data_type, business_definition, source_field, sample_values, governance_status, category, created_by, created_at)
VALUES 
    (
        'calendar',
        'CalendarDate',
        'date',
        'The trading date being evaluated for business calendar classification',
        'calendar_date',
        ARRAY['2026-02-20', '2026-02-23', '2026-02-24'],
        'approved',
        'identification',
        '00000000-0000-0000-0000-000000000001'::uuid,
        NOW()
    ),
    (
        'calendar',
        'IsBusinessDay',
        'boolean',
        'Indicates whether the date is a business day (not a weekend or holiday)',
        'is_business_day',
        ARRAY['true', 'false'],
        'approved',
        'classification',
        '00000000-0000-0000-0000-000000000001'::uuid,
        NOW()
    ),
    (
        'calendar',
        'RegionCode',
        'string',
        'Geographic region code (GB, US, JP, etc.) for region-specific holidays',
        'region_code',
        ARRAY['GB', 'US', 'JP', 'EU'],
        'approved',
        'classification',
        '00000000-0000-0000-0000-000000000001'::uuid,
        NOW()
    ),
    (
        'calendar',
        'HolidayName',
        'string',
        'Name of the holiday if date is a holiday',
        'holiday_name',
        ARRAY['Christmas', 'New Year', 'Easter', 'Thanksgiving'],
        'approved',
        'classification',
        '00000000-0000-0000-0000-000000000001'::uuid,
        NOW()
    ),
    (
        'calendar',
        'SourceSystem',
        'string',
        'Source system that provided the calendar data (Nager.Date, OpenHolidays, etc.)',
        'source_system',
        ARRAY['nager_date', 'open_holidays', 'workalendar'],
        'approved',
        'data_quality',
        '00000000-0000-0000-0000-000000000001'::uuid,
        NOW()
    ),
    (
        'calendar',
        'ConfidenceScore',
        'number',
        'Confidence level (0-100) of the business day classification',
        'confidence_score',
        ARRAY['95', '87', '72'],
        'approved',
        'data_quality',
        '00000000-0000-0000-0000-000000000001'::uuid,
        NOW()
    ),
    (
        'calendar',
        'TradingImpact',
        'boolean',
        'Indicates if date impacts trading operations',
        'trading_impact',
        ARRAY['true', 'false'],
        'draft',
        'business_impact',
        '00000000-0000-0000-0000-000000000001'::uuid,
        NOW()
    )
ON CONFLICT DO NOTHING;

-- Insert default approval workflow for calendar business object
INSERT INTO edm.approval_workflows (business_object, promotion_stage, required_role, sequence_order)
VALUES 
    ('calendar', 'testing', 'data_steward', 1),
    ('calendar', 'staging', 'compliance_officer', 1),
    ('calendar', 'production', 'business_owner', 1)
ON CONFLICT DO NOTHING;

-- Create audit log trigger for rules
CREATE OR REPLACE FUNCTION edm.log_rule_change()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO edm.rule_execution_history (
        rule_id, version, execution_type, input_data, output_data,
        status, executed_by, executed_at
    ) VALUES (
        NEW.id, NEW.current_version, 'manual',
        jsonb_build_object('old_status', OLD.status),
        jsonb_build_object('new_status', NEW.status),
        'success', NEW.updated_by, NOW()
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Don't create trigger yet - uncomment when needed
-- DROP TRIGGER IF EXISTS rule_change_audit ON edm;
CREATE TRIGGER rule_change_audit AFTER UPDATE ON edm.rules
-- FOR EACH ROW EXECUTE FUNCTION edm.log_rule_change();

-- Grant permissions
-- ensure application role exists before granting privileges
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_role') THEN
        CREATE ROLE app_role NOLOGIN;
    END IF;
END$$;

GRANT SELECT, INSERT, UPDATE, DELETE ON edm.rules TO app_role;
GRANT SELECT, INSERT, UPDATE, DELETE ON edm.rule_steps TO app_role;
GRANT SELECT, INSERT ON edm.rule_versions TO app_role;
GRANT SELECT, INSERT, UPDATE ON edm.rule_approvals TO app_role;
GRANT SELECT ON edm.approval_workflows TO app_role;
GRANT SELECT ON edm.semantic_terms TO app_role;
GRANT SELECT, INSERT ON edm.rule_execution_history TO app_role;
