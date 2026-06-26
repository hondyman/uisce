-- ============================================================================
-- Phase 1: Entity Relationship Discovery Schema
-- ============================================================================
-- Purpose: Add comprehensive entity relationship discovery support
-- - entity_attribute_column_mapping: Map entity attributes to actual database columns
-- - entity_relationship: Store discovered and applied relationships between entities
-- - Update metadata_columns: Add optional catalog_node_id for semantic linking
--
-- This migration enables:
-- 1. Tracing entity attributes through column mappings
-- 2. Discovering FK relationships between tables
-- 3. Maintaining semantic context via catalog_node_id
-- 4. Scoring relationship confidence
-- 5. Supporting multi-hop path discovery for self-service reporting
-- ============================================================================

BEGIN;

-- ============================================================================
-- 1. Update metadata_columns to add semantic context
-- ============================================================================
-- This allows columns to be linked to semantic terms (catalog_nodes) for meaning

ALTER TABLE public.metadata_columns
ADD COLUMN IF NOT EXISTS catalog_node_id uuid REFERENCES public.catalog_node(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_metadata_columns_catalog_node_id 
    ON public.metadata_columns USING btree (catalog_node_id);

COMMENT ON COLUMN public.metadata_columns.catalog_node_id IS
    'Foreign key to catalog_node - links this column to semantic definitions';

-- ============================================================================
-- 2. Create entity_attribute_column_mapping table
-- ============================================================================
-- Maps entity attributes (business concepts) to physical columns (database reality)

CREATE TABLE IF NOT EXISTS public.entity_attribute_column_mapping (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    tenant_datasource_id uuid NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    entity_attribute_id uuid NOT NULL REFERENCES public.entity_attribute(id) ON DELETE CASCADE,
    
    -- Physical column location
    table_name text NOT NULL,
    column_name text NOT NULL,
    
    -- Optional: Link to metadata_columns for richer context
    metadata_column_id uuid REFERENCES public.metadata_columns(id) ON DELETE SET NULL,
    
    -- Semantic context: what this mapping represents
    semantic_term_id uuid REFERENCES public.catalog_node(id) ON DELETE SET NULL,
    
    -- Quality metrics
    confidence numeric(3,2) DEFAULT 0.80 CHECK (confidence >= 0.0 AND confidence <= 1.0),
    is_primary_key boolean DEFAULT false,
    is_foreign_key boolean DEFAULT false,
    
    -- Audit
    created_at timestamp DEFAULT now(),
    created_by text,
    updated_at timestamp DEFAULT now(),
    updated_by text,
    
    -- Constraints
    CONSTRAINT entity_attr_col_mapping_unique 
        UNIQUE (tenant_datasource_id, entity_attribute_id, table_name, column_name),
    
    CONSTRAINT entity_attr_col_mapping_confidence_check 
        CHECK (confidence BETWEEN 0.0 AND 1.0)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_entity_attr_col_mapping_tenant_ds 
    ON public.entity_attribute_column_mapping(tenant_id, tenant_datasource_id);

CREATE INDEX IF NOT EXISTS idx_entity_attr_col_mapping_entity_attr_id 
    ON public.entity_attribute_column_mapping(entity_attribute_id);

CREATE INDEX IF NOT EXISTS idx_entity_attr_col_mapping_semantic_term 
    ON public.entity_attribute_column_mapping(semantic_term_id);

CREATE INDEX IF NOT EXISTS idx_entity_attr_col_mapping_metadata_col 
    ON public.entity_attribute_column_mapping(metadata_column_id);

CREATE INDEX IF NOT EXISTS idx_entity_attr_col_mapping_table_column 
    ON public.entity_attribute_column_mapping(tenant_datasource_id, table_name, column_name);

COMMENT ON TABLE public.entity_attribute_column_mapping IS
    'Maps business entity attributes to physical database columns, with semantic context';

COMMENT ON COLUMN public.entity_attribute_column_mapping.entity_attribute_id IS
    'References the business entity attribute (Customer, Order, Product, etc.)';

COMMENT ON COLUMN public.entity_attribute_column_mapping.table_name IS
    'Physical database table name containing the column';

COMMENT ON COLUMN public.entity_attribute_column_mapping.column_name IS
    'Physical column name in the table';

COMMENT ON COLUMN public.entity_attribute_column_mapping.semantic_term_id IS
    'Optional link to catalog_node for semantic meaning';

COMMENT ON COLUMN public.entity_attribute_column_mapping.confidence IS
    'Confidence score of this mapping (0.0-1.0)';

-- ============================================================================
-- 3. Create entity_relationship table
-- ============================================================================
-- Stores discovered and applied relationships between entities

CREATE TABLE IF NOT EXISTS public.entity_relationship (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    tenant_datasource_id uuid NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    
    -- Source and target entities
    source_entity_id uuid NOT NULL REFERENCES public.entity_attribute(id) ON DELETE CASCADE,
    target_entity_id uuid NOT NULL REFERENCES public.entity_attribute(id) ON DELETE CASCADE,
    
    -- Relationship characteristics
    relationship_type varchar(100) NOT NULL,  -- e.g., 'DIRECT_FK', 'SEMANTIC', 'MULTI_HOP'
    cardinality varchar(50),                  -- e.g., 'ONE_TO_ONE', 'ONE_TO_MANY', 'MANY_TO_MANY'
    hierarchy_depth int DEFAULT 1,            -- 1 = direct, 2+ = multi-hop
    
    -- FK constraint details
    fk_constraint text,                       -- e.g., 'orders.customer_id -> customers.id'
    source_column varchar(255),
    source_table varchar(255),
    target_column varchar(255),
    target_table varchar(255),
    
    -- Path for multi-hop relationships (JSON array of intermediate entities)
    relationship_path jsonb,
    
    -- Semantic and confidence metrics
    description text,
    confidence numeric(3,2) DEFAULT 0.80 CHECK (confidence >= 0.0 AND confidence <= 1.0),
    confidence_reason text,                   -- e.g., 'FK exists, semantic linked, naming match'
    
    -- User application status
    is_user_applied boolean DEFAULT false,
    user_applied_at timestamp,
    user_applied_by text,
    
    -- Metadata
    source_discovery_method varchar(100),     -- 'FK_SCAN', 'SEMANTIC_MATCH', 'PATTERN', etc.
    is_active boolean DEFAULT true,
    
    -- Audit
    created_at timestamp DEFAULT now(),
    created_by text,
    updated_at timestamp DEFAULT now(),
    updated_by text,
    
    -- Constraints
    CONSTRAINT entity_relationship_unique 
        UNIQUE (tenant_datasource_id, source_entity_id, target_entity_id, relationship_type),
    
    CONSTRAINT entity_relationship_no_self_loop 
        CHECK (source_entity_id != target_entity_id),
    
    CONSTRAINT entity_relationship_confidence_check 
        CHECK (confidence BETWEEN 0.0 AND 1.0),
    
    CONSTRAINT entity_relationship_valid_cardinality 
        CHECK (cardinality IN ('ONE_TO_ONE', 'ONE_TO_MANY', 'MANY_TO_ONE', 'MANY_TO_MANY', NULL))
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_entity_relationship_tenant_ds 
    ON public.entity_relationship(tenant_id, tenant_datasource_id);

CREATE INDEX IF NOT EXISTS idx_entity_relationship_source_entity 
    ON public.entity_relationship(source_entity_id, tenant_datasource_id);

CREATE INDEX IF NOT EXISTS idx_entity_relationship_target_entity 
    ON public.entity_relationship(target_entity_id, tenant_datasource_id);

CREATE INDEX IF NOT EXISTS idx_entity_relationship_type 
    ON public.entity_relationship(relationship_type, is_active);

CREATE INDEX IF NOT EXISTS idx_entity_relationship_confidence 
    ON public.entity_relationship(confidence DESC, is_active);

CREATE INDEX IF NOT EXISTS idx_entity_relationship_fk_constraint 
    ON public.entity_relationship(fk_constraint) WHERE fk_constraint IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_entity_relationship_hierarchy_depth 
    ON public.entity_relationship(hierarchy_depth) WHERE hierarchy_depth > 1;

CREATE INDEX IF NOT EXISTS idx_entity_relationship_user_applied 
    ON public.entity_relationship(is_user_applied) WHERE is_user_applied = true;

COMMENT ON TABLE public.entity_relationship IS
    'Stores discovered and user-applied relationships between business entities';

COMMENT ON COLUMN public.entity_relationship.source_entity_id IS
    'The starting entity (e.g., Customer)';

COMMENT ON COLUMN public.entity_relationship.target_entity_id IS
    'The related entity (e.g., Order)';

COMMENT ON COLUMN public.entity_relationship.relationship_type IS
    'Type of relationship: DIRECT_FK (1-hop), SEMANTIC (semantic term link), MULTI_HOP (N-hop path)';

COMMENT ON COLUMN public.entity_relationship.hierarchy_depth IS
    'How many hops: 1=direct FK, 2+=via intermediate entities';

COMMENT ON COLUMN public.entity_relationship.fk_constraint IS
    'Human-readable FK path, e.g., "orders.customer_id -> customers.id"';

COMMENT ON COLUMN public.entity_relationship.relationship_path IS
    'JSON array of intermediate entity IDs for multi-hop paths';

COMMENT ON COLUMN public.entity_relationship.confidence IS
    'Confidence score (0.0-1.0): 0.95=FK exists, 0.85+=semantic linked, 0.70+=pattern match';

COMMENT ON COLUMN public.entity_relationship.is_user_applied IS
    'True if user explicitly applied this relationship for their semantic layer';

-- ============================================================================
-- 4. Create relationship_suggestion_dismissal table (optional but recommended)
-- ============================================================================
-- Tracks which suggestions users have dismissed to avoid re-showing them

CREATE TABLE IF NOT EXISTS public.relationship_suggestion_dismissal (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    tenant_datasource_id uuid NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    entity_relationship_id uuid NOT NULL REFERENCES public.entity_relationship(id) ON DELETE CASCADE,
    
    dismissed_by text NOT NULL,
    dismissed_at timestamp DEFAULT now(),
    dismissal_reason text,
    
    -- Allow un-dismissal
    is_active boolean DEFAULT true,
    
    CONSTRAINT relationship_dismissal_unique 
        UNIQUE (tenant_datasource_id, entity_relationship_id),
    
    CONSTRAINT relationship_dismissal_active_check 
        CHECK (is_active IN (true, false))
);

CREATE INDEX IF NOT EXISTS idx_relationship_dismissal_tenant_ds 
    ON public.relationship_suggestion_dismissal(tenant_datasource_id);

CREATE INDEX IF NOT EXISTS idx_relationship_dismissal_entity_rel 
    ON public.relationship_suggestion_dismissal(entity_relationship_id);

CREATE INDEX IF NOT EXISTS idx_relationship_dismissal_active 
    ON public.relationship_suggestion_dismissal(is_active) WHERE is_active = true;

COMMENT ON TABLE public.relationship_suggestion_dismissal IS
    'Tracks dismissed relationship suggestions to avoid repeatedly recommending them';

-- ============================================================================
-- 5. Create view for easy relationship discovery queries
-- ============================================================================

CREATE OR REPLACE VIEW public.v_entity_relationships_with_context AS
SELECT 
    er.id,
    er.tenant_id,
    er.tenant_datasource_id,
    
    -- Source entity
    er.source_entity_id,
    source_ea.name as source_entity_name,
    source_ea.entity_key as source_entity_key,
    source_cn.node_name as source_semantic_name,
    
    -- Target entity
    er.target_entity_id,
    target_ea.name as target_entity_name,
    target_ea.entity_key as target_entity_key,
    target_cn.node_name as target_semantic_name,
    
    -- Relationship details
    er.relationship_type,
    er.cardinality,
    er.hierarchy_depth,
    er.fk_constraint,
    er.source_column,
    er.source_table,
    er.target_column,
    er.target_table,
    
    -- Quality metrics
    er.confidence,
    er.confidence_reason,
    er.source_discovery_method,
    
    -- Status
    er.is_user_applied,
    er.is_active,
    er.created_at,
    er.updated_at
    
FROM public.entity_relationship er
LEFT JOIN public.entity_attribute source_ea ON er.source_entity_id = source_ea.id
LEFT JOIN public.entity_attribute target_ea ON er.target_entity_id = target_ea.id
LEFT JOIN public.catalog_node source_cn ON source_ea.catalog_node_id = source_cn.id
LEFT JOIN public.catalog_node target_cn ON target_ea.catalog_node_id = target_cn.id;

COMMENT ON VIEW public.v_entity_relationships_with_context IS
    'View combining entity relationships with their semantic context';

-- ============================================================================
-- 6. Create utility functions for relationship management
-- ============================================================================

-- Function to calculate confidence score
CREATE OR REPLACE FUNCTION public.calculate_relationship_confidence(
    p_fk_exists boolean,
    p_semantic_linked boolean,
    p_naming_match boolean,
    p_column_type_match boolean
) RETURNS numeric AS $$
DECLARE
    v_confidence numeric := 0.0;
BEGIN
    -- Start with base score
    IF p_fk_exists THEN
        v_confidence := v_confidence + 0.95;  -- Strong indicator
    ELSIF p_semantic_linked THEN
        v_confidence := v_confidence + 0.85;  -- Good indicator
    ELSIF p_naming_match THEN
        v_confidence := v_confidence + 0.70;  -- Moderate indicator
    ELSE
        v_confidence := v_confidence + 0.50;  -- Weak base
    END IF;
    
    -- Boost if multiple signals align
    IF p_semantic_linked AND p_naming_match THEN
        v_confidence := LEAST(v_confidence + 0.05, 1.0);
    END IF;
    
    IF p_column_type_match AND p_fk_exists THEN
        v_confidence := LEAST(v_confidence + 0.05, 1.0);
    END IF;
    
    RETURN ROUND(v_confidence::numeric, 2);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION public.calculate_relationship_confidence IS
    'Calculates confidence score for entity relationship (0.0-1.0)';

-- ============================================================================
-- 7. Create audit trigger for entity_relationship changes
-- ============================================================================

CREATE OR REPLACE FUNCTION public.audit_entity_relationship_changes()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := now();
    
    -- Track who made the change
    IF NEW.is_user_applied AND OLD.is_user_applied IS DISTINCT FROM NEW.is_user_applied THEN
        NEW.user_applied_at := now();
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
DROP TRIGGER IF EXISTS trigger_audit_entity_relationship_changes 
    ON public.entity_relationship;

CREATE TRIGGER trigger_audit_entity_relationship_changes
BEFORE UPDATE ON public.entity_relationship
FOR EACH ROW
EXECUTE FUNCTION public.audit_entity_relationship_changes();

COMMENT ON TRIGGER trigger_audit_entity_relationship_changes ON public.entity_relationship IS
    'Automatically updates updated_at and tracks user application of relationships';

-- ============================================================================
-- 8. Grant permissions
-- ============================================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON public.entity_attribute_column_mapping TO postgres;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.entity_relationship TO postgres;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.relationship_suggestion_dismissal TO postgres;
GRANT SELECT ON public.v_entity_relationships_with_context TO postgres;
GRANT EXECUTE ON FUNCTION public.calculate_relationship_confidence TO postgres;

COMMIT;

-- ============================================================================
-- Schema Summary
-- ============================================================================
-- This migration adds three new tables:
--
-- 1. entity_attribute_column_mapping (many rows per entity)
--    - Traces business entities to physical database columns
--    - Includes confidence scores and semantic linking
--
-- 2. entity_relationship (many rows per datasource)
--    - Stores discovered and applied relationships
--    - Includes cardinality, FK paths, multi-hop info
--    - Tracks user application and confidence
--
-- 3. relationship_suggestion_dismissal (dismissals per tenant)
--    - Tracks which suggestions users have dismissed
--    - Prevents re-showing dismissed suggestions
--
-- Plus one view for easy query access with semantic context
--
-- These enable:
-- ✓ Auto-discovery of related entities via FK chains
-- ✓ Semantic meaning (what relationships are for)
-- ✓ Multi-hop paths (Customer → Order → Invoice)
-- ✓ Confidence scoring (reliability of relationships)
-- ✓ Self-service reporting (join entities automatically)
-- ✓ Multi-tenant isolation (all scoped by tenant_datasource_id)
-- ============================================================================
