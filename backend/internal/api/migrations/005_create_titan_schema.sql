-- Titan Framework: Universal Definition Schema

-- 1. Object Definitions (Metadata for Objects)
CREATE TABLE IF NOT EXISTS object_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL, -- e.g., "Trade", "Position"
    description TEXT,
    fields_json JSONB NOT NULL, -- Schema definition (e.g., list of fields, types, validations)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- 2. Workflow Definitions (Metadata for Processes)
CREATE TABLE IF NOT EXISTS workflow_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    object_type TEXT NOT NULL, -- Links to object_definitions.name
    event TEXT NOT NULL, -- e.g., "Submit", "Approve"
    steps_json JSONB NOT NULL, -- DSL definition (Steps, Transitions)
    version INT DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. Bi-Temporal Ledger (Position Versions)
-- Requires btree_gist extension for EXCLUDE constraints with UUIDs if not already enabled
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS position_versions (
    id UUID DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL, -- The permanent ID of the Position
    tenant_id UUID NOT NULL,
    
    -- The Business Data (Schema-less or structured, here we use columns for core IBOR)
    asset_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    quantity DECIMAL(20, 8) NOT NULL,
    custodian TEXT,
    attributes JSONB, -- Flexible attributes based on object_definition
    
    -- Bi-Temporal Dimensions
    valid_time tstzrange NOT NULL, -- When is this true in the real world?
    system_time tstzrange NOT NULL DEFAULT tstzrange(now(), null), -- When did we know it?
    
    -- Constraint: No two versions of the same entity can overlap in valid_time
    -- This ensures that for any given point in valid_time, there is only one "truth"
    EXCLUDE USING GIST (entity_id WITH =, valid_time WITH &&)
);

-- Indexes for Time Travel Queries
CREATE INDEX idx_positions_entity_valid ON position_versions USING GIST (entity_id, valid_time);
CREATE INDEX idx_positions_system_time ON position_versions USING GIST (system_time);
