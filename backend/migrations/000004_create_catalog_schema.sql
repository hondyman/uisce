-- 000032_improved_catalog_schema.up.sql
-- Improved Workday-inspired Business Object Linking Schema
-- Includes integrity constraints, audit triggers, and performance indexes

-- catalog_node: one row per BO/table
DROP TABLE IF EXISTS catalog_node CASCADE;
DROP TABLE IF EXISTS catalog_edge CASCADE;
DROP TABLE IF EXISTS relationship_suggestion_audit CASCADE;

CREATE TABLE IF NOT EXISTS catalog_node (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    datasource_id TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('table', 'view', 'bo')),  -- Enforced kinds
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- CREATE index IF NOT EXISTS for node lookups by tenant, datasource, and name
CREATE INDEX IF NOT EXISTS idx_node_scope_name ON catalog_node(tenant_id, datasource_id, name);

-- NOTE: This migration defines a small controlled vocabulary used by older
-- implementations. It creates `catalog_edge_type` (singular) for historical
-- compatibility. Newer code uses `catalog_edge_types` (plural). This file is
-- intentionally left as the historical source for the minimal vocabulary and
-- should not be altered without a migration plan.
-- catalog_edge_type: minimal controlled vocabulary
CREATE TABLE IF NOT EXISTS catalog_edge_type (
    code TEXT PRIMARY KEY,
    label TEXT NOT NULL
);

-- Insert default edge types (idempotent)
INSERT INTO catalog_edge_type (code, label) VALUES
    ('REFERENCE', 'Reference'),
    ('COMPOSITION', 'Composition'),
    ('ASSOCIATION', 'Association'),
    ('FOREIGN_KEY', 'Foreign Key')
ON CONFLICT (code) DO NOTHING;

-- catalog_edge: typed, directional edges
CREATE TABLE IF NOT EXISTS catalog_edge (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    datasource_id TEXT NOT NULL,
    source_id UUID NOT NULL REFERENCES catalog_node(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES catalog_node(id) ON DELETE CASCADE,
    edge_type TEXT NOT NULL REFERENCES catalog_edge_type(code),
    cardinality TEXT CHECK (cardinality IN ('1:1', '1:N', 'N:1', 'N:N')),  -- Enforced cardinalities
    fk_table TEXT,
    fk_column TEXT,
    pk_table TEXT,
    pk_column TEXT,
    confidence NUMERIC DEFAULT 0 CHECK (confidence BETWEEN 0 AND 1),  -- Normalized confidence
    suggested BOOLEAN DEFAULT FALSE,
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Unique constraint to prevent duplicate edges
ALTER TABLE catalog_edge ADD CONSTRAINT unique_edge UNIQUE (tenant_id, datasource_id, source_id, target_id, edge_type);

-- Create indexes for edge queries
CREATE INDEX IF NOT EXISTS idx_edge_scope_src ON catalog_edge(tenant_id, datasource_id, source_id);
CREATE INDEX IF NOT EXISTS idx_edge_scope_tgt ON catalog_edge(tenant_id, datasource_id, target_id);
CREATE INDEX IF NOT EXISTS idx_edge_scope_type ON catalog_edge(tenant_id, datasource_id, edge_type);

-- audit: accepted/dismissed suggestions
CREATE TABLE IF NOT EXISTS relationship_suggestion_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    datasource_id TEXT NOT NULL,
    entity TEXT NOT NULL,
    target_entity TEXT NOT NULL,
    fk_column TEXT,
    edge_type TEXT NOT NULL,
    cardinality TEXT,
    confidence NUMERIC,
    action TEXT NOT NULL CHECK (action IN ('accepted', 'dismissed')),
    reason TEXT,
    acted_by TEXT,
    acted_at TIMESTAMPTZ DEFAULT now()
);

-- CREATE index IF NOT EXISTS for audit queries
CREATE INDEX IF NOT EXISTS idx_audit_scope_action ON relationship_suggestion_audit(tenant_id, datasource_id, action);

-- Trigger for auto-auditing suggested edges
CREATE OR REPLACE FUNCTION audit_suggested_edge() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.suggested THEN
        INSERT INTO relationship_suggestion_audit (
            tenant_id, datasource_id, entity, target_entity, fk_column, 
            edge_type, cardinality, confidence, action, acted_by
        )
        VALUES (
            NEW.tenant_id, 
            NEW.datasource_id, 
            (SELECT name FROM catalog_node WHERE id = NEW.source_id),
            (SELECT name FROM catalog_node WHERE id = NEW.target_id), 
            NEW.fk_column, 
            NEW.edge_type, 
            NEW.cardinality, 
            NEW.confidence, 
            'accepted', 
            NEW.created_by
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger after insert on catalog_edge
DROP TRIGGER IF EXISTS trg_audit_edge ON catalog_edge;
CREATE TRIGGER trg_audit_edge AFTER INSERT ON catalog_edge FOR EACH ROW EXECUTE PROCEDURE audit_suggested_edge();
