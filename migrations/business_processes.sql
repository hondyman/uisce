-- ============================================================================
-- PHASE 6B: BUSINESS PROCESS FRAMEWORK - DATABASE SCHEMA
-- ============================================================================
-- Workday-style low-code business process definitions that orchestrate
-- multi-step workflows with validation, timeouts, assignments, and escalations.
--
-- A Business Process (BP) is a sequence of steps (e.g., Data Entry, Validation,
-- Approval, Notification) tied to a business object (e.g., Employee, Order).
--
-- Each step can include:
--   • Triggers: Validation rules (from Phase 6A)
--   • Actions: Save, notify, integrate, escalate
--   • Conditions: AND/OR logic for branching
--   • Assignees: Roles or users
--   • Timelines: Due dates with timeout triggers (Phase 6C integration)
--
-- This enables non-technical users to define complex workflows without coding.
-- ============================================================================

-- Business Process Definition
CREATE TABLE IF NOT EXISTS business_processes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Process metadata
    process_name VARCHAR(100) NOT NULL,        -- e.g., "HireEmployee", "OrderApproval"
    description TEXT,
    process_type VARCHAR(50),                  -- "hire", "purchase", "travel", etc.
    
    -- Process configuration
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    version INT NOT NULL DEFAULT 1,            -- Support versioning for BP changes
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    
    CONSTRAINT fk_bp_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id),
    CONSTRAINT unique_bp_name UNIQUE (tenant_id, process_name, version)
);

-- Business Process Steps (ordered sequence)
CREATE TABLE IF NOT EXISTS bp_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    process_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    
    -- Step sequence
    step_order INT NOT NULL,                   -- 1, 2, 3...
    
    -- Step definition
    step_type VARCHAR(50) NOT NULL,            -- data_entry, validate, approve, notify, integrate, compute
    step_name VARCHAR(100) NOT NULL,           -- "Manager Approval", "Data Validation", etc.
    description TEXT,
    
    -- Step configuration
    duration_hours INT,                        -- Timeout duration (integrates with Phase 6C)
    assignee_role VARCHAR(100),                -- Role assigned to this step (e.g., "manager", "finance_team")
    assignee_user VARCHAR(255),                -- Specific user (if not role-based)
    
    -- Trigger & validation integration
    trigger_ids UUID[] DEFAULT '{}',           -- Phase 6A triggers to run for this step
    
    -- Conditional logic (branching)
    condition_json JSONB,                      -- Optional branching: {"type":"if","field":"status","value":"approved","then_step":3,"else_step":4}
    
    -- Step actions (what happens in this step)
    action_config JSONB,                       -- Step-specific config: {"action":"send_email","to":"$assignee_email","template":"approval_request"}
    
    -- Output handling
    output_mapping JSONB,                      -- Map step output to process variables
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_bp_step_process FOREIGN KEY (process_id) REFERENCES business_processes(id) ON DELETE CASCADE,
    CONSTRAINT fk_bp_step_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id),
    CONSTRAINT unique_step_order UNIQUE (process_id, step_order)
);

-- Business Process Instances (execution records)
CREATE TABLE IF NOT EXISTS bp_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    process_id UUID NOT NULL,
    
    -- Instance data
    entity_id VARCHAR(255),                    -- e.g., employee_id, order_id (the business object being processed)
    entity_type VARCHAR(50),                   -- e.g., "employee", "order"
    
    -- Execution state
    current_step INT,                          -- Which step are we on (1, 2, 3...)
    status VARCHAR(50) NOT NULL,               -- pending, in_progress, completed, failed, paused
    
    -- Instance data
    instance_data JSONB,                       -- Variables/state: {"employee_name":"John","salary":100000,...}
    
    -- Timeline
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    current_step_started_at TIMESTAMP,
    current_step_due_at TIMESTAMP,             -- Due time for current step (duration_hours from start)
    
    -- Temporal workflow reference
    temporal_workflow_id VARCHAR(255),         -- Link to Temporal workflow execution
    temporal_run_id VARCHAR(255),
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    
    CONSTRAINT fk_bp_inst_process FOREIGN KEY (process_id) REFERENCES business_processes(id),
    CONSTRAINT fk_bp_inst_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id)
);

-- BP Step Executions (detailed logs of each step execution)
CREATE TABLE IF NOT EXISTS bp_step_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    bp_instance_id UUID NOT NULL,
    bp_step_id UUID NOT NULL,
    
    -- Execution details
    step_number INT,                           -- Which step (1, 2, 3...)
    assignee_role VARCHAR(100),
    assignee_user VARCHAR(255),
    status VARCHAR(50),                        -- pending, in_progress, completed, failed, escalated
    
    -- Input/Output
    input_data JSONB,                          -- Data passed to step
    output_data JSONB,                         -- Data produced by step
    
    -- Timeline
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    due_at TIMESTAMP,
    escalated_at TIMESTAMP,                    -- When escalation occurred (if any)
    
    -- Result
    result VARCHAR(20),                        -- pass, fail, manual_override, skipped
    error_message TEXT,
    approval_decision VARCHAR(20),             -- approved, rejected, needs_revision
    
    -- Audit
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_step_exec_instance FOREIGN KEY (bp_instance_id) REFERENCES bp_instances(id) ON DELETE CASCADE,
    CONSTRAINT fk_step_exec_step FOREIGN KEY (bp_step_id) REFERENCES bp_steps(id),
    CONSTRAINT fk_step_exec_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id)
);

-- BP Audit Log (compliance/debugging)
CREATE TABLE IF NOT EXISTS bp_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    bp_instance_id UUID NOT NULL,
    
    -- Event
    event_type VARCHAR(50),                    -- step_started, step_completed, escalated, completed, failed
    step_number INT,
    message TEXT,
    
    -- Actor
    actor VARCHAR(255),                        -- User or system that performed action
    action VARCHAR(50),                        -- approved, rejected, escalated, reassigned
    
    -- Details
    details JSONB,
    
    -- Timing
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_audit_instance FOREIGN KEY (bp_instance_id) REFERENCES bp_instances(id) ON DELETE CASCADE,
    CONSTRAINT fk_audit_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id)
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_business_processes_tenant
    ON business_processes(tenant_id, is_active);

CREATE INDEX IF NOT EXISTS idx_bp_steps_process
    ON bp_steps(process_id, step_order);

CREATE INDEX IF NOT EXISTS idx_bp_instances_lookup
    ON bp_instances(tenant_id, process_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_bp_instances_entity
    ON bp_instances(tenant_id, entity_type, entity_id);

CREATE INDEX IF NOT EXISTS idx_bp_step_executions_instance
    ON bp_step_executions(bp_instance_id, step_number);

CREATE INDEX IF NOT EXISTS idx_bp_step_executions_status
    ON bp_step_executions(tenant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_bp_audit_log_instance
    ON bp_audit_log(bp_instance_id, created_at DESC);

-- ============================================================================
-- HELPER VIEWS
-- ============================================================================

CREATE OR REPLACE VIEW v_active_bp_instances AS
SELECT 
    bi.id,
    bi.tenant_id,
    bp.process_name,
    bi.entity_id,
    bi.entity_type,
    bi.status,
    bs.step_name as current_step_name,
    bi.current_step_started_at,
    bi.current_step_due_at,
    CASE 
        WHEN bi.current_step_due_at < NOW() AND bi.status = 'in_progress' THEN 'OVERDUE'
        WHEN bi.current_step_due_at > NOW() AND bi.status = 'in_progress' THEN 'ON_TIME'
        ELSE 'N/A'
    END as timeline_status,
    bi.created_at,
    bi.updated_at
FROM bp_instances bi
JOIN business_processes bp ON bp.id = bi.process_id
LEFT JOIN bp_steps bs ON bs.process_id = bp.id AND bs.step_order = bi.current_step
WHERE bi.status IN ('pending', 'in_progress');

CREATE OR REPLACE VIEW v_bp_completion_metrics AS
SELECT 
    tenant_id,
    process_id,
    COUNT(*) as total_instances,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_count,
    ROUND(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END)::numeric / COUNT(*) * 100, 2) as completion_rate,
    AVG(EXTRACT(EPOCH FROM (completed_at - created_at)) / 3600) as avg_hours_to_complete,
    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count
FROM bp_instances
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY tenant_id, process_id;

-- ============================================================================
-- SAMPLE DATA: HireEmployee Business Process
-- ============================================================================

-- Create HireEmployee BP
INSERT INTO business_processes (tenant_id, process_name, description, process_type, is_active, created_by)
SELECT 
    tenant_id,
    'HireEmployee',
    'Complete hiring workflow from application to onboarding',
    'hire',
    true,
    'system'
FROM tenants LIMIT 1;

-- Get the BP ID for the sample
WITH new_bp AS (
    SELECT id, tenant_id FROM business_processes 
    WHERE process_name = 'HireEmployee' LIMIT 1
)

-- Step 1: Data Entry
INSERT INTO bp_steps (process_id, tenant_id, step_order, step_type, step_name, duration_hours, assignee_role, trigger_ids, action_config, created_by)
SELECT 
    new_bp.id, new_bp.tenant_id, 1, 'data_entry', 'Candidate Information', 0, NULL,
    ARRAY[]::UUID[],
    jsonb_build_object('action', 'collect', 'fields', ARRAY['name', 'email', 'position', 'salary']),
    'system'
FROM new_bp
ON CONFLICT DO NOTHING;

-- Step 2: Validation
INSERT INTO bp_steps (process_id, tenant_id, step_order, step_type, step_name, duration_hours, assignee_role, trigger_ids, action_config, created_by)
SELECT 
    new_bp.id, new_bp.tenant_id, 2, 'validate', 'Background Check & Validation', 24, 'hr_team',
    ARRAY['rule-candidate-validation'::UUID],
    jsonb_build_object('action', 'validate', 'type', 'background_check'),
    'system'
FROM new_bp
ON CONFLICT DO NOTHING;

-- Step 3: Manager Approval (48h timeout)
INSERT INTO bp_steps (process_id, tenant_id, step_order, step_type, step_name, duration_hours, assignee_role, trigger_ids, action_config, created_by)
SELECT 
    new_bp.id, new_bp.tenant_id, 3, 'approve', 'Manager Approval', 48, 'manager',
    ARRAY['rule-manager-approval'::UUID],
    jsonb_build_object('action', 'send_approval_request', 'template', 'manager_approval_email', 'escalation', jsonb_build_object('after_hours', 24, 'escalate_to', 'director')),
    'system'
FROM new_bp
ON CONFLICT DO NOTHING;

-- Step 4: HR Final Action
INSERT INTO bp_steps (process_id, tenant_id, step_order, step_type, step_name, duration_hours, assignee_role, trigger_ids, action_config, created_by)
SELECT 
    new_bp.id, new_bp.tenant_id, 4, 'notify', 'HR Final Action & Offer', 0, 'hr_director',
    ARRAY[]::UUID[],
    jsonb_build_object('action', 'send_offer', 'notification', 'send_offer_letter'),
    'system'
FROM new_bp
ON CONFLICT DO NOTHING;

-- ============================================================================
-- QUERIES FOR COMMON OPERATIONS
-- ============================================================================

-- View all BPs for a tenant:
-- SELECT * FROM business_processes WHERE tenant_id = '...' AND is_active = true;

-- View all steps in a BP:
-- SELECT * FROM bp_steps WHERE process_id = '...' ORDER BY step_order;

-- Start a BP instance:
-- INSERT INTO bp_instances (tenant_id, process_id, entity_id, entity_type, current_step, status, instance_data)
-- VALUES ('tenant-123', 'process-id', 'emp-456', 'employee', 1, 'pending', '{"name":"John","salary":100000}');

-- Get active BP instances for an entity:
-- SELECT * FROM bp_instances WHERE tenant_id = '...' AND entity_type = 'employee' AND entity_id = '...' AND status IN ('pending', 'in_progress');

-- Update BP instance status:
-- UPDATE bp_instances SET status = 'in_progress', current_step = 2 WHERE id = '...';

-- Log step execution:
-- INSERT INTO bp_step_executions (tenant_id, bp_instance_id, bp_step_id, step_number, status, started_at)
-- VALUES ('tenant-123', 'instance-id', 'step-id', 2, 'started', NOW());
