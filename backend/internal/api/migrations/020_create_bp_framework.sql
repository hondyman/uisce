-- Business process definition (per tenant, per entity/type)
CREATE TABLE business_process_definition (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL, -- references tenant(id) implicit
    key         TEXT NOT NULL, -- e.g. "account_opening"
    version     INT  NOT NULL,
    name        TEXT NOT NULL,
    entity      TEXT NOT NULL, -- e.g. "Account"
    status      TEXT NOT NULL DEFAULT 'draft', -- draft|in_review|approved|deployed|deprecated
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  TEXT,
    UNIQUE (tenant_id, key, version)
);

-- Steps in the business process
CREATE TABLE business_process_step (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bp_def_id     UUID NOT NULL REFERENCES business_process_definition(id) ON DELETE CASCADE,
    seq           INT  NOT NULL,             -- order within process
    step_key      TEXT NOT NULL,             -- stable key within BP (e.g. "approval_mgr")
    type          TEXT NOT NULL,             -- task|approval|validation|notification|subprocess|wait_signal
    activity_name TEXT,                      -- Temporal activity or child workflow name
    signal_name   TEXT,                      -- for wait_signal
    description   TEXT,
    pre_validation_rule_ids  TEXT[] NOT NULL DEFAULT '{}', -- array of rule IDs
    post_validation_rule_ids TEXT[] NOT NULL DEFAULT '{}',
    condition_expr TEXT,                     -- optional Starlark or expression (run to decide if step executes)
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Step Participants (e.g. "Manager", "Compliance_Officer")
CREATE TABLE business_process_step_participant (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    step_id   UUID NOT NULL REFERENCES business_process_step(id) ON DELETE CASCADE,
    role_key  TEXT NOT NULL,                 -- e.g. "Manager", "Compliance", "Ops"
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
