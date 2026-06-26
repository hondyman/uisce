-- Add dynamic rule support to participants
ALTER TABLE business_process_step_participant ADD COLUMN rule_id TEXT; -- Optional: if set, use rule to resolve users
ALTER TABLE business_process_step_participant ALTER COLUMN role_key DROP NOT NULL; -- Can be null if rule_id is set

-- Inbox Tasks (User items to complete)
CREATE TABLE business_process_task (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    bp_run_id     TEXT NOT NULL,          -- Temporal Workflow ID
    step_id       UUID NOT NULL REFERENCES business_process_step(id),
    status        TEXT NOT NULL DEFAULT 'pending', -- pending|completed|reassigned|cancelled
    assignee_id   TEXT,                   -- User ID if directly assigned
    assignee_role TEXT,                   -- Role key if group assigned
    due_date      TIMESTAMPTZ,
    data_payload  JSONB,                  -- Context needed to complete task
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ
);

-- Audit / History
CREATE TABLE business_process_event (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    bp_run_id   TEXT NOT NULL,
    step_key    TEXT,
    event_type  TEXT NOT NULL, -- step_start|step_complete|rule_failure|task_assigned
    details     JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
