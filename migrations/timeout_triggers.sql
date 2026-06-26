-- ============================================================================
-- WORKDAY STEP TIMEOUT TRIGGERS - DATABASE SCHEMA
-- ============================================================================
-- Manages workflow step timeouts with automatic escalation, notification, 
-- and logging actions. Runs every hour via Temporal worker.
-- ============================================================================

-- Primary table: Timeout trigger rules
CREATE TABLE IF NOT EXISTS workflow_timeout_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_name VARCHAR(50) NOT NULL,         -- "HireEmployee", "OrderApproval", etc
    step_name VARCHAR(50) NOT NULL,             -- "ManagerApproval", "CreditCheck", etc
    due_hours INT NOT NULL CHECK (due_hours > 0),  -- 48, 24, 72, etc
    
    -- Actions as JSONB array: [{"percent": 80, "type": "notify", ...}, {...}]
    -- percent: 80 (notify), 100 (escalate/cancel), etc
    -- type: "notify", "escalate", "log", "cancel"
    -- target: "assignee", "hr_director", "manager", "audit", etc
    -- message: Custom message for notification
    actions_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT valid_tenant_id CHECK (tenant_id != '00000000-0000-0000-0000-000000000000'::uuid)
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Fast lookup: workflow + step + tenant (most common query)
CREATE INDEX IF NOT EXISTS idx_timeout_triggers_workflow_step 
    ON workflow_timeout_triggers(tenant_id, workflow_name, step_name);

-- Check if trigger exists for workflow
CREATE INDEX IF NOT EXISTS idx_timeout_triggers_workflow 
    ON workflow_timeout_triggers(tenant_id, workflow_name);

-- Find active triggers only
CREATE INDEX IF NOT EXISTS idx_timeout_triggers_active 
    ON workflow_timeout_triggers(tenant_id, is_active) 
    WHERE is_active = TRUE;

-- ============================================================================
-- SAMPLE DATA - 5 REAL-WORLD SCENARIOS
-- ============================================================================

-- SCENARIO 1: Hire Employee Workflow - Manager Approval (48h)
-- Escalate to HR Director if manager doesn't approve in 48 hours
INSERT INTO workflow_timeout_triggers 
    (tenant_id, workflow_name, step_name, due_hours, actions_json, is_active)
VALUES (
    '910638ba-a459-4a3f-bb2d-78391b0595f6',  -- Default tenant (set to your tenant)
    'HireEmployee',
    'ManagerApproval',
    48,
    '[
        {
            "percent": 80,
            "type": "notify",
            "target": "assignee",
            "message": "Employee approval due in 10 hours - please review"
        },
        {
            "percent": 100,
            "type": "escalate",
            "target": "hr_director",
            "message": "Manager approval overdue - escalated to HR"
        },
        {
            "percent": 100,
            "type": "log",
            "target": "audit",
            "message": "HireEmployee approval timeout - escalated"
        }
    ]'::jsonb,
    TRUE
) ON CONFLICT DO NOTHING;

-- SCENARIO 2: Order Approval - Finance Approval (24h)
-- Notify finance at 20h, escalate at 24h
INSERT INTO workflow_timeout_triggers 
    (tenant_id, workflow_name, step_name, due_hours, actions_json, is_active)
VALUES (
    '910638ba-a459-4a3f-bb2d-78391b0595f6',
    'OrderApproval',
    'FinanceApproval',
    24,
    '[
        {
            "percent": 80,
            "type": "notify",
            "target": "assignee",
            "message": "Order approval needed - 5 hours remaining"
        },
        {
            "percent": 100,
            "type": "escalate",
            "target": "finance_manager",
            "message": "Finance approval overdue - escalated"
        }
    ]'::jsonb,
    TRUE
) ON CONFLICT DO NOTHING;

-- SCENARIO 3: Invoice Processing - Payment Setup (72h)
-- Long-running process with single escalation
INSERT INTO workflow_timeout_triggers 
    (tenant_id, workflow_name, step_name, due_hours, actions_json, is_active)
VALUES (
    '910638ba-a459-4a3f-bb2d-78391b0595f6',
    'InvoiceProcessing',
    'PaymentSetup',
    72,
    '[
        {
            "percent": 100,
            "type": "escalate",
            "target": "accounts_payable_manager",
            "message": "Invoice payment setup overdue"
        }
    ]'::jsonb,
    TRUE
) ON CONFLICT DO NOTHING;

-- SCENARIO 4: Product Launch - Pricing Review (12h - FAST!)
-- Quick turnaround with aggressive escalation
INSERT INTO workflow_timeout_triggers 
    (tenant_id, workflow_name, step_name, due_hours, actions_json, is_active)
VALUES (
    '910638ba-a459-4a3f-bb2d-78391b0595f6',
    'ProductLaunch',
    'PricingReview',
    12,
    '[
        {
            "percent": 60,
            "type": "notify",
            "target": "assignee",
            "message": "Pricing review needed - 5 hours remaining"
        },
        {
            "percent": 100,
            "type": "escalate",
            "target": "pricing_director",
            "message": "Pricing review overdue - escalated immediately"
        }
    ]'::jsonb,
    TRUE
) ON CONFLICT DO NOTHING;

-- SCENARIO 5: Employee Termination - HR Final Review (96h)
-- Multi-day compliance process
INSERT INTO workflow_timeout_triggers 
    (tenant_id, workflow_name, step_name, due_hours, actions_json, is_active)
VALUES (
    '910638ba-a459-4a3f-bb2d-78391b0595f6',
    'EmployeeTermination',
    'HRFinalReview',
    96,
    '[
        {
            "percent": 80,
            "type": "notify",
            "target": "assignee",
            "message": "Final termination review required"
        },
        {
            "percent": 100,
            "type": "escalate",
            "target": "hr_director",
            "message": "Termination final review overdue"
        },
        {
            "percent": 100,
            "type": "log",
            "target": "audit",
            "message": "Termination review timeout - compliance flag"
        }
    ]'::jsonb,
    TRUE
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- AUDIT TABLE (Optional but recommended)
-- ============================================================================
-- Tracks all timeout trigger executions for compliance/debugging

CREATE TABLE IF NOT EXISTS workflow_timeout_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_id VARCHAR(100) NOT NULL,         -- Actual workflow instance ID
    workflow_name VARCHAR(50) NOT NULL,        -- Template name
    step_name VARCHAR(50) NOT NULL,
    trigger_id UUID REFERENCES workflow_timeout_triggers(id),
    action_type VARCHAR(20) NOT NULL,          -- "notify", "escalate", "log", "cancel"
    target VARCHAR(50),                        -- Who was notified/escalated to
    status VARCHAR(20) DEFAULT 'pending',      -- "pending", "completed", "failed"
    error_message TEXT,
    executed_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT valid_tenant_id CHECK (tenant_id != '00000000-0000-0000-0000-000000000000'::uuid)
);

-- Index for querying timeout events
CREATE INDEX IF NOT EXISTS idx_timeout_events_workflow 
    ON workflow_timeout_events(tenant_id, workflow_id);

CREATE INDEX IF NOT EXISTS idx_timeout_events_executed 
    ON workflow_timeout_events(tenant_id, executed_at DESC);

-- ============================================================================
-- UTILITY VIEWS
-- ============================================================================

-- View: All active timeout triggers
CREATE OR REPLACE VIEW v_active_timeout_triggers AS
SELECT 
    id, 
    tenant_id, 
    workflow_name, 
    step_name, 
    due_hours,
    actions_json,
    created_at
FROM workflow_timeout_triggers
WHERE is_active = TRUE
ORDER BY workflow_name, step_name;

-- View: Recently executed timeout events
CREATE OR REPLACE VIEW v_recent_timeout_events AS
SELECT 
    id,
    workflow_name,
    step_name,
    action_type,
    target,
    status,
    executed_at,
    error_message
FROM workflow_timeout_events
WHERE executed_at > NOW() - INTERVAL '24 hours'
ORDER BY executed_at DESC;

-- ============================================================================
-- GRANTS (Adjust tenant_id to your actual tenant)
-- ============================================================================

-- Allow temporal worker to read and write
-- GRANT SELECT, INSERT ON workflow_timeout_triggers TO temporal_worker;
-- GRANT SELECT, INSERT ON workflow_timeout_events TO temporal_worker;

-- Allow API service to CRUD triggers
-- GRANT SELECT, INSERT, UPDATE, DELETE ON workflow_timeout_triggers TO api_service;

-- ============================================================================
-- EXAMPLE QUERIES FOR DEVELOPMENT
-- ============================================================================

-- List all timeout triggers for a workflow
-- SELECT * FROM workflow_timeout_triggers 
-- WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
--   AND workflow_name = 'HireEmployee'
--   AND is_active = TRUE;

-- Get triggers for a specific step
-- SELECT * FROM workflow_timeout_triggers 
-- WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
--   AND workflow_name = 'HireEmployee'
--   AND step_name = 'ManagerApproval';

-- Check recent timeout executions
-- SELECT * FROM v_recent_timeout_events WHERE workflow_name = 'HireEmployee';

-- Find workflow steps that should trigger timeout (elapsed > due_hours)
-- SELECT 
--     wf.id,
--     wf.workflow,
--     wf.step,
--     wf.step_start,
--     EXTRACT(HOUR FROM NOW() - wf.step_start) as elapsed_hours,
--     tt.due_hours,
--     tt.actions_json
-- FROM workflow_instances wf
-- JOIN workflow_timeout_triggers tt 
--     ON wf.workflow = tt.workflow_name 
--     AND wf.step = tt.step_name
-- WHERE wf.step_start < NOW() - (tt.due_hours * INTERVAL '1 hour')
--   AND wf.status = 'pending'
--   AND tt.is_active = TRUE;

-- ============================================================================
-- END OF TIMEOUT TRIGGERS SCHEMA
-- ============================================================================
