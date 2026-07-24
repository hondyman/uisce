-- Enhance Step Definition
ALTER TABLE business_process_step
    ADD COLUMN approval_chain JSONB,       -- Multi-level approval logic
    ADD COLUMN routing_rules JSONB,        -- Conditional routing logic
    ADD COLUMN delay_expr TEXT,            -- Starlark expression for delay (e.g. "hours(2)")
    ADD COLUMN sla_expr TEXT,              -- Starlark expression for SLA (e.g. "days(3)")
    ADD COLUMN integration_config JSONB;   -- Config for type='integration'

-- Enhance Participants
ALTER TABLE business_process_step_participant
    ADD COLUMN include_condition TEXT,     -- Starlark condition to include user
    ADD COLUMN exclude_initiator BOOLEAN NOT NULL DEFAULT FALSE;

-- Detailed Execution Logging (Process Level)
CREATE TABLE business_process_execution (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    bp_def_id    UUID NOT NULL REFERENCES business_process_definition(id),
    bp_run_id    TEXT NOT NULL,            -- Temporal Workflow ID
    entity       TEXT NOT NULL,
    entity_id    TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'running', -- running|completed|failed|cancelled
    started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    initiated_by TEXT
);

-- Detailed Execution Logging (Step Level)
CREATE TABLE business_process_step_execution (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bp_exec_id         UUID NOT NULL REFERENCES business_process_execution(id) ON DELETE CASCADE,
    step_id            UUID NOT NULL REFERENCES business_process_step(id),
    step_key           TEXT NOT NULL,
    status             TEXT NOT NULL DEFAULT 'pending', -- pending|running|completed|skipped|failed
    started_at         TIMESTAMPTZ,
    completed_at       TIMESTAMPTZ,
    actor              TEXT,               -- Who acted (if task/approval)
    routing_info       JSONB,              -- Details on how it was routed
    validation_results JSONB               -- Rule pass/fail details
);
