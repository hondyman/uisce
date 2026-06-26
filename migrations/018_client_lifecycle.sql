-- Migration: 018_client_lifecycle.sql
-- Description: Creates tables for Life Events and Wealth Graph Entities.

-- 1. Life Events (Polymorphic Event Store)
CREATE TABLE IF NOT EXISTS life_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL, -- In a real app, this would reference a clients table
    event_type VARCHAR(50) NOT NULL, -- e.g., 'IPO', 'MARRIAGE', 'LIQUIDITY_EVENT'
    event_date DATE NOT NULL,
    status VARCHAR(20) DEFAULT 'ACTIVE', -- ACTIVE, SCENARIO, ARCHIVED
    
    -- The Polymorphic Payload
    -- Validated against JSON Schema defined in the Object Definition Service
    attributes JSONB NOT NULL, 
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast JSON querying
CREATE INDEX IF NOT EXISTS idx_life_events_attributes ON life_events USING GIN (attributes);
CREATE INDEX IF NOT EXISTS idx_life_events_client_id ON life_events (client_id);

-- 2. Wealth Graph Nodes (Entities)
CREATE TABLE IF NOT EXISTS entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(20) NOT NULL, -- 'INDIVIDUAL', 'TRUST', 'CORPORATION'
    display_name TEXT NOT NULL,
    attributes JSONB, -- Stores Tax ID, DOB, Trust Date, etc.
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_entities_attributes ON entities USING GIN (attributes);

-- 3. Wealth Graph Edges (Relationships)
CREATE TABLE IF NOT EXISTS relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    to_entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) NOT NULL, -- 'GRANTOR', 'TRUSTEE', 'BENEFICIARY', 'OWNER'
    ownership_percentage DECIMAL(5,4), -- e.g., 0.5000 for 50%
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT unique_rel UNIQUE (from_entity_id, to_entity_id, relationship_type)
);

CREATE INDEX IF NOT EXISTS idx_relationships_from ON relationships (from_entity_id);
CREATE INDEX IF NOT EXISTS idx_relationships_to ON relationships (to_entity_id);
