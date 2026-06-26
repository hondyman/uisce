-- Migration: Initialize Cube.dev Semantic Term Type Definitions
-- Purpose: Create catalog_node_type and catalog_edge_type definitions for Dimensions, Measures, Hierarchies, Segments, Time
-- Date: 2026-01-04
-- Status: Production Ready

BEGIN;

-- Get first tenant (update this ID for specific tenant deployment)
WITH first_tenant AS (
    SELECT id FROM tenants ORDER BY created_at LIMIT 1
)

-- Insert semantic term node types (dimension, measure, time, hierarchy, segment)
INSERT INTO catalog_node_type (
    catalog_type_name,
    description,
    config,
    is_active,
    tenant_id
)
SELECT
    'semantic_term_dimension' as catalog_type_name,
    'Cube.js Dimension - An attribute related to a measure (e.g., country, user_id, product_name)' as description,
    '{
        "type_category": "semantic_term",
        "cube_type": "dimension",
        "required_properties": ["name", "sql", "type"],
        "optional_properties": ["title", "description", "public", "format", "meta", "order", "primary_key", "case", "granularities"],
        "properties_schema": {
            "name": {"type": "string", "description": "Unique identifier within cube", "required": true},
            "sql": {"type": "string", "description": "SQL expression, e.g., {CUBE}.column_name", "required": true},
            "type": {"type": "string", "enum": ["string", "number", "boolean", "time", "geo"], "required": true},
            "title": {"type": "string"},
            "description": {"type": "string"}
        },
        "example": {"name": "user_id", "sql": "{CUBE}.user_id", "type": "number", "title": "User ID"}
    }'::jsonb as config,
    true as is_active,
    (SELECT id FROM first_tenant) as tenant_id

UNION ALL SELECT
    'semantic_term_measure',
    'Cube.js Measure - A numeric aggregation for analysis (e.g., revenue, count, average)',
    '{
        "type_category": "semantic_term",
        "cube_type": "measure",
        "required_properties": ["name", "sql", "type"],
        "optional_properties": ["title", "description", "public", "format", "filters", "rolling_window", "time_shift", "drill_members"],
        "properties_schema": {
            "name": {"type": "string", "required": true},
            "sql": {"type": "string", "required": true},
            "type": {"type": "string", "enum": ["count", "count_distinct", "sum", "avg", "min", "max", "number"], "required": true},
            "title": {"type": "string"}
        },
        "example": {"name": "total_revenue", "sql": "SUM({amount})", "type": "sum", "title": "Total Revenue"}
    }'::jsonb,
    true,
    (SELECT id FROM first_tenant)

UNION ALL SELECT
    'semantic_term_time',
    'Cube.js Time Dimension - Temporal attribute with granularities (day, month, year, etc.)',
    '{
        "type_category": "semantic_term",
        "cube_type": "dimension",
        "sub_type": "time",
        "required_properties": ["name", "sql", "type"],
        "optional_properties": ["title", "description", "public", "order", "granularities", "time_shift"],
        "properties_schema": {
            "name": {"type": "string", "required": true},
            "sql": {"type": "string", "required": true},
            "type": {"type": "string", "const": "time", "required": true},
            "granularities": {"type": "array", "items": {"type": "string"}, "enum": ["second", "minute", "hour", "day", "week", "month", "quarter", "year"]}
        },
        "example": {"name": "created_at", "sql": "{CUBE}.created_at", "type": "time", "title": "Created At"}
    }'::jsonb,
    true,
    (SELECT id FROM first_tenant)

UNION ALL SELECT
    'semantic_term_hierarchy',
    'Cube.js Hierarchy - Groups dimensions for drill-down analysis (e.g., country → state → city)',
    '{
        "type_category": "semantic_term",
        "cube_type": "hierarchy",
        "required_properties": ["name", "levels"],
        "optional_properties": ["title", "description", "public"],
        "properties_schema": {
            "name": {"type": "string", "required": true},
            "levels": {"type": "array", "items": {"type": "string"}, "required": true},
            "title": {"type": "string"}
        },
        "example": {"name": "location_hierarchy", "title": "Location", "levels": ["country", "state", "city"]}
    }'::jsonb,
    true,
    (SELECT id FROM first_tenant)

UNION ALL SELECT
    'semantic_term_segment',
    'Cube.js Segment - Pre-calculated filter or cohort condition applied to measures',
    '{
        "type_category": "semantic_term",
        "cube_type": "segment",
        "required_properties": ["name", "sql"],
        "optional_properties": ["title", "description", "public"],
        "properties_schema": {
            "name": {"type": "string", "required": true},
            "sql": {"type": "string", "required": true},
            "title": {"type": "string"}
        },
        "example": {"name": "premium_users", "sql": "{status} = ''premium''", "title": "Premium Users"}
    }'::jsonb,
    true,
    (SELECT id FROM first_tenant)

ON CONFLICT DO NOTHING;

COMMIT;

-- ============================================================================
-- Insert Edge Types (Relationships between semantic term types)
-- ============================================================================

BEGIN;

WITH first_tenant AS (
    SELECT id FROM tenants ORDER BY created_at LIMIT 1
),

node_types AS (
    SELECT
        (SELECT id FROM first_tenant) as tenant_id,
        (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term_dimension' LIMIT 1) as dimension_id,
        (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term_measure' LIMIT 1) as measure_id,
        (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term_hierarchy' LIMIT 1) as hierarchy_id,
        (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term_segment' LIMIT 1) as segment_id,
        (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term_time' LIMIT 1) as time_id
)

INSERT INTO catalog_edge_type (
    edge_type_name,
    description,
    subject_node_type_id,
    object_node_type_id,
    properties,
    is_active,
    tenant_id
)
SELECT 
    'hierarchy_contains_dimension' as edge_type_name,
    'Hierarchy contains dimensions for drill-down analysis' as description,
    hierarchy_id,
    dimension_id,
    '{"relationship_type": "composition", "direction": "one_to_many", "description": "Hierarchies are composed of ordered dimension levels"}'::jsonb as properties,
    true as is_active,
    tenant_id
FROM node_types
WHERE hierarchy_id IS NOT NULL AND dimension_id IS NOT NULL

UNION ALL SELECT 
    'measure_aggregates_dimension',
    'Measure aggregates across dimensions',
    measure_id,
    dimension_id,
    '{"relationship_type": "uses", "direction": "many_to_many", "description": "Measures can aggregate over multiple dimensions"}'::jsonb,
    true,
    tenant_id
FROM node_types
WHERE measure_id IS NOT NULL AND dimension_id IS NOT NULL

UNION ALL SELECT 
    'segment_filters_measure',
    'Segment filters are applied to measures',
    segment_id,
    measure_id,
    '{"relationship_type": "filter", "direction": "many_to_many", "description": "Segments provide pre-calculated filters for measures"}'::jsonb,
    true,
    tenant_id
FROM node_types
WHERE segment_id IS NOT NULL AND measure_id IS NOT NULL

UNION ALL SELECT 
    'dimension_references_time',
    'Dimension can reference time for temporal context',
    dimension_id,
    time_id,
    '{"relationship_type": "reference", "direction": "many_to_one", "description": "Dimensions can reference time for temporal context"}'::jsonb,
    true,
    tenant_id
FROM node_types
WHERE dimension_id IS NOT NULL AND time_id IS NOT NULL

UNION ALL SELECT 
    'measure_uses_time',
    'Measure aggregates over time dimension',
    measure_id,
    time_id,
    '{"relationship_type": "uses", "direction": "many_to_many", "description": "Measures aggregate over time dimensions for temporal analysis"}'::jsonb,
    true,
    tenant_id
FROM node_types
WHERE measure_id IS NOT NULL AND time_id IS NOT NULL

UNION ALL SELECT 
    'hierarchy_organizes_time',
    'Hierarchy includes time dimensions for temporal drill-down',
    hierarchy_id,
    time_id,
    '{"relationship_type": "composition", "direction": "one_to_many", "description": "Time hierarchies organize temporal data by granularity"}'::jsonb,
    true,
    tenant_id
FROM node_types
WHERE hierarchy_id IS NOT NULL AND time_id IS NOT NULL

ON CONFLICT DO NOTHING;

COMMIT;

-- ============================================================================
-- VERIFICATION QUERIES (Run after migration to confirm success)
-- ============================================================================
-- SELECT COUNT(*) FROM catalog_node_type WHERE catalog_type_name LIKE 'semantic_term_%';
-- Expected: 5

-- SELECT edge_type_name, is_active FROM catalog_edge_type 
-- WHERE edge_type_name IN ('hierarchy_contains_dimension', 'measure_aggregates_dimension', 'segment_filters_measure', 'dimension_references_time', 'measure_uses_time', 'hierarchy_organizes_time')
-- ORDER BY edge_type_name;
-- Expected: 6 rows, all with is_active = true
