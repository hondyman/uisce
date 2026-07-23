-- Migration to create catalog_validation_rules table for validation rule management
-- This table stores data quality and business logic validation rules

CREATE TABLE IF NOT EXISTS catalog_validation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rule_name VARCHAR(255) NOT NULL,
    rule_type VARCHAR(50) NOT NULL CHECK (rule_type IN ('field_format', 'cardinality', 'uniqueness', 'referential_integrity', 'business_logic')),
    description TEXT,
    target_entity VARCHAR(255) NOT NULL,
    condition_json JSONB NOT NULL DEFAULT '{}',
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('error', 'warning', 'info')) DEFAULT 'error',
    is_active BOOLEAN DEFAULT true,
    created_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, rule_name)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_validation_rules_tenant ON catalog_validation_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_validation_rules_type ON catalog_validation_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_validation_rules_entity ON catalog_validation_rules(target_entity);
CREATE INDEX IF NOT EXISTS idx_validation_rules_severity ON catalog_validation_rules(severity);
CREATE INDEX IF NOT EXISTS idx_validation_rules_active ON catalog_validation_rules(is_active);
CREATE INDEX IF NOT EXISTS idx_validation_rules_condition ON catalog_validation_rules USING GIN(condition_json);
CREATE INDEX IF NOT EXISTS idx_validation_rules_created ON catalog_validation_rules(created_at DESC);

-- Comments
COMMENT ON TABLE catalog_validation_rules IS 'Stores validation rules for data quality, business logic, and referential integrity checks';
COMMENT ON COLUMN catalog_validation_rules.tenant_id IS 'Tenant scope for multi-tenancy';
COMMENT ON COLUMN catalog_validation_rules.rule_name IS 'Human-readable name of the validation rule';
COMMENT ON COLUMN catalog_validation_rules.rule_type IS 'Type of validation: field_format, cardinality, uniqueness, referential_integrity, business_logic';
COMMENT ON COLUMN catalog_validation_rules.target_entity IS 'The node type or edge type this rule applies to';
COMMENT ON COLUMN catalog_validation_rules.condition_json IS 'JSON structure defining the validation condition parameters';
COMMENT ON COLUMN catalog_validation_rules.severity IS 'Impact level: error (block), warning (alert), info (log)';
COMMENT ON COLUMN catalog_validation_rules.is_active IS 'Enable/disable rule without deletion';
COMMENT ON COLUMN catalog_validation_rules.created_by IS 'User ID who created the rule';

-- Audit table for validation rule changes
CREATE TABLE IF NOT EXISTS catalog_validation_rules_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES catalog_validation_rules(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL CHECK (action IN ('CREATE', 'UPDATE', 'DELETE')),
    old_values JSONB,
    new_values JSONB,
    changed_by UUID,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_validation_rules_audit_rule ON catalog_validation_rules_audit(rule_id);
CREATE INDEX IF NOT EXISTS idx_validation_rules_audit_tenant ON catalog_validation_rules_audit(tenant_id);
CREATE INDEX IF NOT EXISTS idx_validation_rules_audit_action ON catalog_validation_rules_audit(action);

COMMENT ON TABLE catalog_validation_rules_audit IS 'Audit trail for validation rule changes';
COMMENT ON COLUMN catalog_validation_rules_audit.action IS 'Type of change: CREATE, UPDATE, DELETE';
