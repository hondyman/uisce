-- Migration: Add Workflow Timeout Triggers
-- Date: 2025-10-20
-- Purpose: Enable automatic escalation, notification, and logging for overdue workflow steps

-- Timeout Triggers Table
CREATE TABLE IF NOT EXISTS workflow_timeout_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_name VARCHAR(100) NOT NULL,           -- "HireEmployee"
    step_name VARCHAR(100) NOT NULL,               -- "ManagerApproval"
    due_hours INT NOT NULL,                        -- 48 hours
    trigger_percentages JSONB DEFAULT '[80, 100]',-- Warning at 80%, Escalate at 100%
    actions_json JSONB NOT NULL,                   -- [{"percent": 80, "type": "notify", ...}, ...]
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, workflow_name, step_name)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_timeout_triggers_workflow 
ON workflow_timeout_triggers(tenant_id, workflow_name, step_name);

CREATE INDEX IF NOT EXISTS idx_timeout_triggers_active 
ON workflow_timeout_triggers(tenant_id, is_active);

-- Sample Data: HireEmployee Manager Approval (48h timeout)
INSERT INTO workflow_timeout_triggers (
    tenant_id,
    workflow_name,
    step_name,
    due_hours,
    trigger_percentages,
    actions_json
) VALUES (
    '910638ba-a459-4a3f-bb2d-78391b0595f6',
    'HireEmployee',
    'ManagerApproval',
    48,
    '[80, 100]'::jsonb,
    '[
        {
            "percent": 80,
            "type": "notify",
            "target": "assignee",
            "message": "Approval due in 8 hours - please review"
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
            "message": "Timeout: Step exceeded due date"
        }
    ]'::jsonb
) ON CONFLICT DO NOTHING;

-- Sample Data: Order Credit Approval (24h timeout)
INSERT INTO workflow_timeout_triggers (
    tenant_id,
    workflow_name,
    step_name,
    due_hours,
    trigger_percentages,
    actions_json
) VALUES (
    '910638ba-a459-4a3f-bb2d-78391b0595f6',
    'OrderApproval',
    'CreditApproval',
    24,
    '[80, 100]'::jsonb,
    '[
        {
            "percent": 80,
            "type": "notify",
            "target": "assignee",
            "message": "Credit approval needed - due in 4 hours"
        },
        {
            "percent": 100,
            "type": "escalate",
            "target": "finance_director",
            "message": "Credit approval overdue - escalated"
        }
    ]'::jsonb
) ON CONFLICT DO NOTHING;

-- Sample Data: Invoice Payment (72h timeout)
INSERT INTO workflow_timeout_triggers (
    tenant_id,
    workflow_name,
    step_name,
    due_hours,
    trigger_percentages,
    actions_json
) VALUES (
    '910638ba-a459-4a3f-bb2d-78391b0595f6',
    'InvoiceProcessing',
    'PaymentApproval',
    72,
    '[80, 100]'::jsonb,
    '[
        {
            "percent": 100,
            "type": "escalate",
            "target": "accounting_manager",
            "message": "Payment approval overdue - escalated"
        },
        {
            "percent": 100,
            "type": "log",
            "target": "audit",
            "message": "Invoice payment timeout"
        }
    ]'::jsonb
) ON CONFLICT DO NOTHING;

-- Comments
COMMENT ON TABLE workflow_timeout_triggers IS 'Defines timeout rules for workflow steps - escalation, notification, and audit logging';
COMMENT ON COLUMN workflow_timeout_triggers.due_hours IS 'Number of hours before step times out';
COMMENT ON COLUMN workflow_timeout_triggers.actions_json IS 'Array of actions to execute at trigger percentages (80%, 100%)';
