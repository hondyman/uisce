-- Human Tasks Table for Inbox
CREATE TABLE IF NOT EXISTS human_tasks (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workflow_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    
    status TEXT NOT NULL CHECK (status IN ('Pending', 'Approved', 'Rejected', 'Cancelled')),
    assignee_group TEXT, -- e.g., 'compliance', 'manager'
    assignee_user TEXT,
    
    payload JSONB, -- Context for the approver
    decision_payload JSONB, -- Result of the decision (comments, etc.)
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_human_tasks_assignee_group ON human_tasks (assignee_group, status);
CREATE INDEX idx_human_tasks_workflow_id ON human_tasks (workflow_id);
