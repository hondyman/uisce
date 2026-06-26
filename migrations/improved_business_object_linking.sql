-- Improved SQL Schema for Workday-Inspired Business Object Linking
-- Alter existing tables to add new columns for structured relationships

-- Insert edge types if not exist (using existing predicate, description columns)
INSERT INTO catalog_edge_types (edge_type_name, description, source_node_type_id, target_node_type_id, tenant_id)
SELECT 'REFERENCE', 'Reference', '49a50271-ae58-4d3e-ae1c-2f5b89d89192', '49a50271-ae58-4d3e-ae1c-2f5b89d89192', '870361a8-87e2-4171-95ad-0473cc93791e'
WHERE NOT EXISTS (SELECT 1 FROM catalog_edge_types WHERE edge_type_name = 'REFERENCE');

INSERT INTO catalog_edge_types (edge_type_name, description, source_node_type_id, target_node_type_id, tenant_id)
SELECT 'COMPOSITION', 'Composition', '49a50271-ae58-4d3e-ae1c-2f5b89d89192', '49a50271-ae58-4d3e-ae1c-2f5b89d89192', '870361a8-87e2-4171-95ad-0473cc93791e'
WHERE NOT EXISTS (SELECT 1 FROM catalog_edge_types WHERE edge_type_name = 'COMPOSITION');

INSERT INTO catalog_edge_types (edge_type_name, description, source_node_type_id, target_node_type_id, tenant_id)
SELECT 'ASSOCIATION', 'Association', '49a50271-ae58-4d3e-ae1c-2f5b89d89192', '49a50271-ae58-4d3e-ae1c-2f5b89d89192', '870361a8-87e2-4171-95ad-0473cc93791e'
WHERE NOT EXISTS (SELECT 1 FROM catalog_edge_types WHERE edge_type_name = 'ASSOCIATION');

INSERT INTO catalog_edge_types (edge_type_name, description, source_node_type_id, target_node_type_id, tenant_id)
SELECT 'FOREIGN_KEY', 'Foreign Key', '49a50271-ae58-4d3e-ae1c-2f5b89d89192', '49a50271-ae58-4d3e-ae1c-2f5b89d89192', '870361a8-87e2-4171-95ad-0473cc93791e'
WHERE NOT EXISTS (SELECT 1 FROM catalog_edge_types WHERE edge_type_name = 'FOREIGN_KEY');

-- catalog_edge already has the new columns, so skip ALTER

-- Create relationship_suggestion_audit table
CREATE TABLE IF NOT EXISTS relationship_suggestion_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- changed to UUID to match tenants.id
    datasource_id UUID NOT NULL,  -- assuming tenant_datasource_id is this
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

-- Unique constraint on catalog_edge already exists, skip

-- Trigger for auto-auditing suggested edges
CREATE OR REPLACE FUNCTION audit_suggested_edge() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.suggested THEN
    INSERT INTO relationship_suggestion_audit (tenant_id, datasource_id, entity, target_entity, fk_column, edge_type, cardinality, confidence, action, acted_by)
    VALUES (NEW.tenant_id, NEW.tenant_datasource_id, (SELECT node_name FROM catalog_node WHERE id = NEW.source_node_id), (SELECT node_name FROM catalog_node WHERE id = NEW.target_node_id), NEW.fk_column, (SELECT edge_type_name FROM catalog_edge_types WHERE id = NEW.edge_type_id), NEW.cardinality, NEW.confidence, 'accepted', NEW.created_by);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_audit_edge ON catalog_edge;
CREATE TRIGGER trg_audit_edge AFTER INSERT ON catalog_edge FOR EACH ROW EXECUTE PROCEDURE audit_suggested_edge();

-- Indexes (add if not exist)
CREATE INDEX IF NOT EXISTS idx_node_scope_name ON catalog_node(tenant_id, tenant_datasource_id, node_name);
CREATE INDEX IF NOT EXISTS idx_edge_scope_src ON catalog_edge(tenant_datasource_id, source_node_id);
CREATE INDEX IF NOT EXISTS idx_edge_scope_tgt ON catalog_edge(tenant_datasource_id, target_node_id);
CREATE INDEX IF NOT EXISTS idx_edge_scope_type ON catalog_edge(tenant_datasource_id, edge_type_id);