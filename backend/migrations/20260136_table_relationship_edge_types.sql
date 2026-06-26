-- Migration: 20260136_table_relationship_edge_types.sql
-- Purpose: Create edge types for TABLE_RELATES_TO_TABLE and BO_RELATES_TO_BO relationships
-- These enable profile-driven relationship discovery and semantic relationship inheritance

-- ============================================================================
-- Step 1: Ensure required node types exist
-- ============================================================================

-- Ensure 'physical_table' node type exists (may already exist as 'table')
INSERT INTO catalog_node_type (
    id, tenant_id, catalog_type_name, config, is_active
)
SELECT 
    gen_random_uuid(),
    t.id,
    'physical_table',
    jsonb_build_object(
        'label', 'Physical Table',
        'icon', 'table_chart',
        'description', 'Physical database table for relationship tracking'
    ),
    true
FROM tenants t
WHERE NOT EXISTS (
    SELECT 1 FROM catalog_node_type cnt 
    WHERE cnt.tenant_id = t.id 
    AND cnt.catalog_type_name IN ('physical_table', 'table')
);

-- Ensure 'business_object' node type exists
INSERT INTO catalog_node_type (
    id, tenant_id, catalog_type_name, config, is_active
)
SELECT 
    gen_random_uuid(),
    t.id,
    'business_object',
    jsonb_build_object(
        'label', 'Business Object',
        'icon', 'business',
        'description', 'Semantic business object for relationship tracking'
    ),
    true
FROM tenants t
WHERE NOT EXISTS (
    SELECT 1 FROM catalog_node_type cnt 
    WHERE cnt.tenant_id = t.id 
    AND cnt.catalog_type_name = 'business_object'
);

-- ============================================================================
-- Step 2: Create TABLE_RELATES_TO_TABLE edge type
-- ============================================================================

INSERT INTO catalog_edge_types (
    id, 
    tenant_id, 
    edge_type_name, 
    description,
    source_node_type_id, 
    target_node_type_id, 
    config,
    is_active
)
SELECT 
    gen_random_uuid(),
    t.id,
    'TABLE_RELATES_TO_TABLE',
    'Physical relationship between database tables. Stores join conditions, cardinality, and profile-derived metadata.',
    src.id,
    tgt.id,
    jsonb_build_object(
        'properties_schema', jsonb_build_object(
            'join_condition', 'string',
            'join_type', jsonb_build_object('type', 'enum', 'values', ARRAY['inner', 'left', 'right', 'full']),
            'cardinality', jsonb_build_object('type', 'enum', 'values', ARRAY['1:1', '1:M', 'M:1', 'M:M', 'unknown']),
            'confidence', jsonb_build_object('type', 'number', 'min', 0, 'max', 1),
            'origin', jsonb_build_object('type', 'enum', 'values', ARRAY['manual', 'inferred', 'imported']),
            'lookup_candidate', 'boolean',
            'profile', jsonb_build_object(
                'type', 'object',
                'properties', jsonb_build_object(
                    'left_distinct', 'integer',
                    'right_distinct', 'integer',
                    'left_row_count', 'integer',
                    'right_row_count', 'integer',
                    'join_selectivity', 'number',
                    'left_unique', 'boolean',
                    'right_unique', 'boolean'
                )
            ),
            'notes', 'string'
        ),
        'ui', jsonb_build_object(
            'label', 'Table Relationship',
            'icon', 'link',
            'color', '#3f51b5'
        )
    ),
    true
FROM tenants t
CROSS JOIN LATERAL (
    SELECT id FROM catalog_node_type 
    WHERE tenant_id = t.id 
    AND catalog_type_name IN ('physical_table', 'table')
    LIMIT 1
) src
CROSS JOIN LATERAL (
    SELECT id FROM catalog_node_type 
    WHERE tenant_id = t.id 
    AND catalog_type_name IN ('physical_table', 'table')
    LIMIT 1
) tgt
WHERE NOT EXISTS (
    SELECT 1 FROM catalog_edge_types cet 
    WHERE cet.tenant_id = t.id 
    AND cet.edge_type_name = 'TABLE_RELATES_TO_TABLE'
);

-- ============================================================================
-- Step 3: Create BO_RELATES_TO_BO edge type
-- ============================================================================

INSERT INTO catalog_edge_types (
    id, 
    tenant_id, 
    edge_type_name, 
    description,
    source_node_type_id, 
    target_node_type_id, 
    config,
    is_active
)
SELECT 
    gen_random_uuid(),
    t.id,
    'BO_RELATES_TO_BO',
    'Semantic relationship between Business Objects. Inherited from physical table relationships with UI hints.',
    src.id,
    tgt.id,
    jsonb_build_object(
        'properties_schema', jsonb_build_object(
            'relationship_type', jsonb_build_object('type', 'enum', 'values', ARRAY['1:1', '1:M', 'M:1', 'M:M']),
            'join_path', jsonb_build_object(
                'type', 'array',
                'items', jsonb_build_object(
                    'type', 'object',
                    'properties', jsonb_build_object(
                        'table', 'string',
                        'alias', 'string',
                        'column', 'string'
                    )
                )
            ),
            'via_tables', jsonb_build_object('type', 'array', 'items', 'string'),
            'lookup', 'boolean',
            'ui_role', jsonb_build_object('type', 'enum', 'values', ARRAY['lookup', 'detail', 'child_collection', 'association']),
            'description', 'string',
            'origin', jsonb_build_object('type', 'enum', 'values', ARRAY['manual', 'inferred']),
            'confidence', jsonb_build_object('type', 'number', 'min', 0, 'max', 1)
        ),
        'ui', jsonb_build_object(
            'label', 'BO Relationship',
            'icon', 'account_tree',
            'color', '#4caf50'
        )
    ),
    true
FROM tenants t
CROSS JOIN LATERAL (
    SELECT id FROM catalog_node_type 
    WHERE tenant_id = t.id 
    AND catalog_type_name = 'business_object'
    LIMIT 1
) src
CROSS JOIN LATERAL (
    SELECT id FROM catalog_node_type 
    WHERE tenant_id = t.id 
    AND catalog_type_name = 'business_object'
    LIMIT 1
) tgt
WHERE NOT EXISTS (
    SELECT 1 FROM catalog_edge_types cet 
    WHERE cet.tenant_id = t.id 
    AND cet.edge_type_name = 'BO_RELATES_TO_BO'
);

-- ============================================================================
-- Step 4: Create indexes for efficient relationship queries
-- ============================================================================

-- Index for finding all relationships from a source table (no subquery in predicate)
CREATE INDEX IF NOT EXISTS idx_catalog_edge_table_relationships
ON catalog_edge (source_node_id, edge_type_id);

-- Index for finding all relationships from a source BO (no subquery in predicate)
CREATE INDEX IF NOT EXISTS idx_catalog_edge_bo_relationships
ON catalog_edge (source_node_id, edge_type_id);

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE catalog_edge_types IS 
'Extended to support TABLE_RELATES_TO_TABLE for physical relationships and BO_RELATES_TO_BO for semantic relationships.';
