-- migration: create event_configs table for event routing configuration

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS event_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    bo_type VARCHAR(50) NOT NULL,
    field_name VARCHAR(200),
    filter_json JSONB DEFAULT '{}',
    route_queue VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_event_configs_tenant ON event_configs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_event_configs_type ON event_configs(event_type, bo_type);
