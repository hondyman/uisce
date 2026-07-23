-- Business Process Builder Schema
-- Complete persistence layer for BP Builder with multi-tenant support
-- Created: October 21, 2025

-- ============================================================================
-- TABLE: business_processes
-- Purpose: Store complete business process definitions
-- ============================================================================
CREATE TABLE IF NOT EXISTS business_processes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    process_name VARCHAR(255) NOT NULL,
    description TEXT,
    entity_type VARCHAR(100) NOT NULL,  -- e.g., 'Employee', 'Order', 'Invoice'
    status VARCHAR(50) DEFAULT 'draft',  -- 'draft', 'published', 'archived'
    is_active BOOLEAN DEFAULT false,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by VARCHAR(255),
    updated_at TIMESTAMP,
    total_duration_hours INTEGER,
    version_number INTEGER DEFAULT 1,
    CONSTRAINT fk_bp_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_bp_tenant_id ON business_processes(tenant_id);
CREATE INDEX idx_bp_entity_type ON business_processes(entity_type);
CREATE INDEX idx_bp_status ON business_processes(status);
CREATE INDEX idx_bp_created_at ON business_processes(created_at DESC);
CREATE INDEX idx_bp_is_active ON business_processes(is_active) WHERE is_active = true;

-- ============================================================================
-- TABLE: bp_steps
-- Purpose: Store individual workflow steps (ordered)
-- ============================================================================
CREATE TABLE IF NOT EXISTS bp_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_process_id UUID NOT NULL,
    step_order SMALLINT NOT NULL,
    step_type VARCHAR(50) NOT NULL,  -- 'data_entry', 'validate', 'approve', 'notify', 'integrate', 'condition'
    step_name VARCHAR(255) NOT NULL,
    description TEXT,
    duration_hours SMALLINT DEFAULT 24,
    status VARCHAR(50) DEFAULT 'pending',
    config JSONB NOT NULL,  -- Flexible config for step-specific data
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    CONSTRAINT fk_bp_process FOREIGN KEY (business_process_id) 
        REFERENCES business_processes(id) ON DELETE CASCADE,
    CONSTRAINT ck_step_type CHECK (step_type IN (
        'data_entry', 'validate', 'approve', 'notify', 'integrate', 'condition'
    )),
    CONSTRAINT ck_step_order_positive CHECK (step_order > 0)
);

CREATE INDEX idx_bp_steps_process_id ON bp_steps(business_process_id);
CREATE INDEX idx_bp_steps_order ON bp_steps(business_process_id, step_order);
CREATE INDEX idx_bp_steps_type ON bp_steps(step_type);

-- ============================================================================
-- TABLE: bp_step_validations
-- Purpose: Link validation rules to validate-type steps
-- ============================================================================
CREATE TABLE IF NOT EXISTS bp_step_validations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bp_step_id UUID NOT NULL,
    validation_rule_id UUID NOT NULL,
    is_required BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_bp_step FOREIGN KEY (bp_step_id) 
        REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT fk_validation_rule FOREIGN KEY (validation_rule_id)
        REFERENCES validation_rules(id) ON DELETE CASCADE,
    CONSTRAINT uc_bp_step_validation UNIQUE(bp_step_id, validation_rule_id)
);

CREATE INDEX idx_bp_step_val_step ON bp_step_validations(bp_step_id);
CREATE INDEX idx_bp_step_val_rule ON bp_step_validations(validation_rule_id);

-- ============================================================================
-- TABLE: bp_step_approvers
-- Purpose: Track approval assignments (role or user)
-- ============================================================================
CREATE TABLE IF NOT EXISTS bp_step_approvers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bp_step_id UUID NOT NULL,
    approver_type VARCHAR(50) NOT NULL,  -- 'role' or 'user'
    approver_value VARCHAR(255) NOT NULL,  -- role name or user email
    order_sequence SMALLINT DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_bp_step_approver FOREIGN KEY (bp_step_id)
        REFERENCES bp_steps(id) ON DELETE CASCADE,
    CONSTRAINT ck_approver_type CHECK (approver_type IN ('role', 'user'))
);

CREATE INDEX idx_bp_approvers_step ON bp_step_approvers(bp_step_id);
CREATE INDEX idx_bp_approvers_type ON bp_step_approvers(approver_type);

-- ============================================================================
-- TABLE: bp_executions
-- Purpose: Track workflow instances and their progress
-- ============================================================================
CREATE TABLE IF NOT EXISTS bp_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    business_process_id UUID NOT NULL,
    workflow_id VARCHAR(255),  -- Temporal workflow ID
    entity_id UUID NOT NULL,  -- The entity being processed (Employee, Order, etc.)
    initiated_by VARCHAR(255) NOT NULL,
    initiated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    execution_status VARCHAR(50) DEFAULT 'running',  -- 'running', 'completed', 'failed', 'paused'
    current_step_order SMALLINT,
    total_duration_minutes INTEGER,
    error_message TEXT,
    metadata JSONB,  -- Additional execution context
    CONSTRAINT fk_bp_exec_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_bp_exec_process FOREIGN KEY (business_process_id)
        REFERENCES business_processes(id) ON DELETE CASCADE
);

CREATE INDEX idx_bp_exec_tenant_id ON bp_executions(tenant_id);
CREATE INDEX idx_bp_exec_process_id ON bp_executions(business_process_id);
CREATE INDEX idx_bp_exec_workflow_id ON bp_executions(workflow_id);
CREATE INDEX idx_bp_exec_status ON bp_executions(execution_status);
CREATE INDEX idx_bp_exec_initiated_at ON bp_executions(initiated_at DESC);

-- ============================================================================
-- TABLE: bp_execution_steps
-- Purpose: Track individual step executions within a workflow
-- ============================================================================
CREATE TABLE IF NOT EXISTS bp_execution_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bp_execution_id UUID NOT NULL,
    bp_step_id UUID NOT NULL,
    step_order SMALLINT NOT NULL,
    step_status VARCHAR(50) DEFAULT 'pending',  -- 'pending', 'in_progress', 'completed', 'failed', 'skipped'
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_minutes INTEGER,
    result_data JSONB,  -- Step execution results
    error_message TEXT,
    assigned_to VARCHAR(255),  -- User assigned to this step
    assigned_at TIMESTAMP,
    completed_by VARCHAR(255),
    CONSTRAINT fk_bp_exec_step_exec FOREIGN KEY (bp_execution_id)
        REFERENCES bp_executions(id) ON DELETE CASCADE,
    CONSTRAINT fk_bp_exec_step_def FOREIGN KEY (bp_step_id)
        REFERENCES bp_steps(id) ON DELETE RESTRICT
);

CREATE INDEX idx_bp_exec_steps_exec ON bp_execution_steps(bp_execution_id);
CREATE INDEX idx_bp_exec_steps_status ON bp_execution_steps(step_status);
CREATE INDEX idx_bp_exec_steps_assigned ON bp_execution_steps(assigned_to) WHERE assigned_to IS NOT NULL;

-- ============================================================================
-- TABLE: bp_audit_trail
-- Purpose: Complete audit trail for compliance
-- ============================================================================
CREATE TABLE IF NOT EXISTS bp_audit_trail (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    business_process_id UUID,
    bp_execution_id UUID,
    action_type VARCHAR(100) NOT NULL,  -- 'created', 'modified', 'executed', 'approved', 'rejected'
    actor_email VARCHAR(255) NOT NULL,
    actor_role VARCHAR(100),
    action_details JSONB,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    CONSTRAINT fk_audit_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_audit_process FOREIGN KEY (business_process_id)
        REFERENCES business_processes(id) ON DELETE SET NULL,
    CONSTRAINT fk_audit_execution FOREIGN KEY (bp_execution_id)
        REFERENCES bp_executions(id) ON DELETE SET NULL
);

CREATE INDEX idx_audit_tenant_id ON bp_audit_trail(tenant_id);
CREATE INDEX idx_audit_action_type ON bp_audit_trail(action_type);
CREATE INDEX idx_audit_timestamp ON bp_audit_trail(timestamp DESC);
CREATE INDEX idx_audit_actor ON bp_audit_trail(actor_email);

-- ============================================================================
-- TABLE: bp_notifications_log
-- Purpose: Track sent notifications
-- ============================================================================
CREATE TABLE IF NOT EXISTS bp_notifications_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bp_execution_step_id UUID NOT NULL,
    notification_type VARCHAR(50),  -- 'email', 'sms', 'webhook'
    recipient VARCHAR(255) NOT NULL,
    subject VARCHAR(255),
    body TEXT,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    delivery_status VARCHAR(50),  -- 'sent', 'failed', 'bounced'
    delivery_details JSONB,
    CONSTRAINT fk_notif_exec_step FOREIGN KEY (bp_execution_step_id)
        REFERENCES bp_execution_steps(id) ON DELETE SET NULL
);

CREATE INDEX idx_notif_exec_step ON bp_notifications_log(bp_execution_step_id);
CREATE INDEX idx_notif_status ON bp_notifications_log(delivery_status);

-- ============================================================================
-- GRANTS for application user
-- ============================================================================
DO $$
BEGIN
    -- Grant table privileges if tables exist
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'business_processes') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON business_processes TO app_user;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'bp_steps') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON bp_steps TO app_user;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'bp_step_validations') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON bp_step_validations TO app_user;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'bp_step_approvers') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON bp_step_approvers TO app_user;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'bp_executions') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON bp_executions TO app_user;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'bp_execution_steps') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON bp_execution_steps TO app_user;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'bp_audit_trail') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON bp_audit_trail TO app_user;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'bp_notifications_log') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON bp_notifications_log TO app_user;
    END IF;

    -- Grant sequence privileges if sequences exist
    IF EXISTS (SELECT 1 FROM pg_class WHERE relkind = 'S' AND relname = 'business_processes_id_seq') THEN
        GRANT USAGE, SELECT ON SEQUENCE business_processes_id_seq TO app_user;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_class WHERE relkind = 'S' AND relname = 'bp_steps_id_seq') THEN
        GRANT USAGE, SELECT ON SEQUENCE bp_steps_id_seq TO app_user;
    END IF;
END$$;

-- ============================================================================
-- COMMENTS for documentation
-- ============================================================================
COMMENT ON TABLE business_processes IS 'Stores complete BP definitions with metadata';
COMMENT ON TABLE bp_steps IS 'Individual workflow steps with flexible JSONB config';
COMMENT ON TABLE bp_executions IS 'Workflow instances with Temporal integration';
COMMENT ON TABLE bp_execution_steps IS 'Step-level execution tracking';
COMMENT ON TABLE bp_audit_trail IS 'Complete audit trail for compliance';
COMMENT ON TABLE bp_notifications_log IS 'Notification delivery tracking';

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'business_processes' AND column_name = 'config') THEN
        EXECUTE 'COMMENT ON COLUMN business_processes.config IS ''Flexible JSONB for custom BP properties'';';
    ELSE
        RAISE NOTICE 'Skipping comment: business_processes.config not present';
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bp_steps' AND column_name = 'config') THEN
        EXECUTE 'COMMENT ON COLUMN bp_steps.config IS ''Step-specific config: validation rules, approvers, templates, etc.'';';
    ELSE
        RAISE NOTICE 'Skipping comment: bp_steps.config not present';
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bp_executions' AND column_name = 'workflow_id') THEN
        EXECUTE 'COMMENT ON COLUMN bp_executions.workflow_id IS ''Temporal Workflow ID for tracking execution'';';
    ELSE
        RAISE NOTICE 'Skipping comment: bp_executions.workflow_id not present';
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bp_audit_trail' AND column_name = 'action_details') THEN
        EXECUTE 'COMMENT ON COLUMN bp_audit_trail.action_details IS ''JSON details of the action taken'';';
    ELSE
        RAISE NOTICE 'Skipping comment: bp_audit_trail.action_details not present';
    END IF;
END$$;
