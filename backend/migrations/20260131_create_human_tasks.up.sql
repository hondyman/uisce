CREATE TABLE IF NOT EXISTS human_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id VARCHAR(255) NOT NULL,
    run_id VARCHAR(255) NOT NULL,
    task_token TEXT NOT NULL, -- Temporal Task Token to resume execution
    view_definition_id UUID,
    title VARCHAR(255),
    input_context JSONB DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- PENDING, COMPLETED, CANCELED
    result JSONB,
    assigned_to VARCHAR(255), -- Optional: User ID or Role
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_human_tasks_status ON human_tasks(status);
CREATE INDEX idx_human_tasks_workflow ON human_tasks(workflow_id, run_id);
