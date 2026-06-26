-- =============================================================================
-- Migration: Validation Rules Tables for Workday-like BP Validation
-- =============================================================================
-- This migration creates tables for the low-code validation engine that
-- supports Business Process (BP) validation with AND/OR/NOT logic,
-- configurable actions (route to queue, notify), and tenant isolation.

-- Main validation rules table
CREATE TABLE IF NOT EXISTS bp_validations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    bp_name VARCHAR(100) NOT NULL,
    step_name VARCHAR(100) NOT NULL,
    condition_json JSONB NOT NULL,
    action_on_success VARCHAR(200),
    action_on_failure VARCHAR(200),
    error_message TEXT NOT NULL,
    priority INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for fast lookup
CREATE INDEX IF NOT EXISTS idx_bp_validations_tenant ON bp_validations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bp_validations_lookup ON bp_validations(tenant_id, bp_name, step_name);
CREATE INDEX IF NOT EXISTS idx_bp_validations_enabled ON bp_validations(tenant_id, enabled);

-- Audit/execution history table
CREATE TABLE IF NOT EXISTS bp_validation_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rule_id UUID NOT NULL REFERENCES bp_validations(id) ON DELETE CASCADE,
    bp_name VARCHAR(100) NOT NULL,
    step_name VARCHAR(100) NOT NULL,
    input_data JSONB NOT NULL,
    result_passed BOOLEAN NOT NULL,
    error_message TEXT,
    action_taken VARCHAR(200),
    execution_time_ms INTEGER,
    executed_by UUID,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for audit table
CREATE INDEX IF NOT EXISTS idx_bp_validation_executions_tenant ON bp_validation_executions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bp_validation_executions_rule ON bp_validation_executions(rule_id);
CREATE INDEX IF NOT EXISTS idx_bp_validation_executions_bp ON bp_validation_executions(tenant_id, bp_name, step_name);
CREATE INDEX IF NOT EXISTS idx_bp_validation_executions_time ON bp_validation_executions(executed_at DESC);

-- Update trigger for bp_validations
CREATE OR REPLACE FUNCTION update_bp_validations_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER bp_validations_timestamp_trigger
BEFORE UPDATE ON bp_validations
FOR EACH ROW
EXECUTE FUNCTION update_bp_validations_timestamp();

-- Sample rule configurations for marital status validation
-- INSERT INTO bp_validations (
--     tenant_id, bp_name, step_name, condition_json, action_on_success, action_on_failure, error_message, priority, enabled
-- ) VALUES (
--     '00000000-0000-0000-0000-000000000000'::uuid,
--     'ChangeMaritalStatus',
--     'Submit',
--     '{"and": [{"field": "marital_status", "operator": "=", "value": "married"}, {"field": "age", "operator": ">=", "value": 18}]}'::jsonb,
--     'route:hr_updates.queue',
--     'route:validation_errors.queue',
--     'Age must be at least 18 for married status',
--     1,
--     TRUE
-- );

-- Sample rule for email validation
-- INSERT INTO bp_validations (
--     tenant_id, bp_name, step_name, condition_json, action_on_success, action_on_failure, error_message, priority, enabled
-- ) VALUES (
--     '00000000-0000-0000-0000-000000000000'::uuid,
--     'ChangeContactInfo',
--     'Submit',
--     '{"and": [{"field": "email", "operator": "regex", "value": "^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$"}]}'::jsonb,
--     NULL,
--     'route:validation_errors.queue',
--     'Invalid email format',
--     1,
--     TRUE
-- );
