-- migration: create bo_events table

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS bo_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    bo_type VARCHAR(100),
    bo_id UUID NOT NULL,
    changed_by UUID NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    field_name VARCHAR(200),
    old_value JSONB,
    new_value JSONB,
    bp_step VARCHAR(100),
    custom_data JSONB
);

CREATE INDEX IF NOT EXISTS idx_bo_events_bo ON bo_events(bo_type, bo_id);
CREATE INDEX IF NOT EXISTS idx_bo_events_time ON bo_events(changed_at);
