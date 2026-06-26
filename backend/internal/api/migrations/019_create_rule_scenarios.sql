-- Migration 019: Rule Scenarios and What-If Analysis
-- Creates tables for managing rule simulation scenarios and versions

CREATE TABLE IF NOT EXISTS rule_scenario (
    id           uuid primary key default gen_random_uuid(),
    tenant_id    uuid not null, -- references tenant(tenantid)
    base_rule_id uuid, -- references validationrule(ruleid)
    name         text not null,
    description  text,
    status       text not null default 'draft', -- draft|running|completed|archived
    created_by   text not null,
    created_at   timestamptz not null default now(),
    updated_at   timestamptz not null default now()
);

CREATE TABLE IF NOT EXISTS rule_scenario_version (
    id             uuid primary key default gen_random_uuid(),
    scenario_id    uuid not null references rule_scenario(id) on delete cascade,
    version        int  not null,
    rule_snapshot  jsonb not null, -- serialized TenantValidationRule
    created_at     timestamptz not null default now(),
    created_by     text not null,
    
    CONSTRAINT uq_scenario_version UNIQUE (scenario_id, version)
);

-- Add foreign key to rule_test_run to link simulations
ALTER TABLE rule_test_run
    ADD COLUMN IF NOT EXISTS scenario_version_id uuid references rule_scenario_version(id);
