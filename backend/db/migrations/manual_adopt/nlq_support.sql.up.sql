-- Migration: Add NLQ and Calculation DAG support
-- This migration adds:
-- 1. pgvector extension for semantic search
-- 2. embedding column to catalog_node
-- 3. Functions for building calculation DAGs with metadata

-- Enable pgvector extension for semantic search
CREATE EXTENSION IF NOT EXISTS vector;

-- Add embedding column to catalog_node for semantic search
ALTER TABLE public.catalog_node
ADD COLUMN IF NOT EXISTS embedding vector(768);

-- Create index for fast similarity searches (using cosine distance)
CREATE INDEX IF NOT EXISTS idx_catalog_node_embedding 
ON public.catalog_node USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- Alternatively, use HNSW for better accuracy (uncomment if preferred):
-- CREATE INDEX IF NOT EXISTS idx_catalog_node_embedding_hnsw
-- ON public.catalog_node USING hnsw (embedding vector_cosine_ops);

-- Function: get_calc_dag
-- Builds a calculation DAG (dependency graph) for a given entity
CREATE OR REPLACE FUNCTION get_calc_dag(start_path text, tenant uuid)
RETURNS jsonb AS $$
WITH RECURSIVE dag AS (
    -- Start from the calculation node
    SELECT
        n.id AS node_id,
        n.node_name,
        n.qualified_path,
        n.node_type_id,
        e.target_node_id,
        e.relationship_type
    FROM catalog_node n
    LEFT JOIN catalog_edge e ON n.id = e.source_node_id
    WHERE n.qualified_path = start_path
      AND n.tenant_id = tenant

    UNION ALL

    -- Walk dependencies recursively
    SELECT
        n.id,
        n.node_name,
        n.qualified_path,
        n.node_type_id,
        e.target_node_id,
        e.relationship_type
    FROM dag d
    JOIN catalog_edge e ON d.target_node_id = e.source_node_id
    JOIN catalog_node n ON e.source_node_id = n.id
    WHERE n.tenant_id = tenant
)
SELECT jsonb_build_object(
    'nodes', jsonb_agg(
        DISTINCT jsonb_build_object(
            'id', node_id,
            'name', node_name,
            'path', qualified_path,
            'type', node_type_id
        )
    ),
    'edges', jsonb_agg(
        DISTINCT jsonb_build_object(
            'source', node_id,
            'target', target_node_id,
            'relationship', relationship_type
        )
    )
)
FROM dag;
$$ LANGUAGE sql;

-- Function: get_calc_dag_with_metadata
-- Enhanced version that includes lineage, data quality contracts, and SLA metadata
CREATE OR REPLACE FUNCTION get_calc_dag_with_metadata(start_path text, tenant uuid)
RETURNS jsonb AS $$
WITH RECURSIVE dag AS (
    -- Start from the calculation node
    SELECT
        n.id AS node_id,
        n.node_name,
        n.qualified_path,
        n.node_type_id,
        n.description,
        n.lineage,
        n.data_quality_contract,
        n.sla,
        n.properties,
        e.target_node_id,
        e.relationship_type,
        e.properties AS edge_properties
    FROM catalog_node n
    LEFT JOIN catalog_edge e ON n.id = e.source_node_id
    WHERE n.qualified_path = start_path
      AND n.tenant_id = tenant

    UNION ALL

    -- Walk dependencies recursively
    SELECT
        n.id,
        n.node_name,
        n.qualified_path,
        n.node_type_id,
        n.description,
        n.lineage,
        n.data_quality_contract,
        n.sla,
        n.properties,
        e.target_node_id,
        e.relationship_type,
        e.properties
    FROM dag d
    JOIN catalog_edge e ON d.target_node_id = e.source_node_id
    JOIN catalog_node n ON e.source_node_id = n.id
    WHERE n.tenant_id = tenant
)
SELECT jsonb_build_object(
    'nodes', (
        SELECT jsonb_agg(
            DISTINCT jsonb_build_object(
                'id', node_id,
                'name', node_name,
                'path', qualified_path,
                'type', node_type_id,
                'description', description,
                'lineage', lineage,
                'data_quality_contract', data_quality_contract,
                'sla', sla,
                'properties', properties
            )
        )
        FROM dag
        WHERE node_id IS NOT NULL
    ),
    'edges', (
        SELECT jsonb_agg(
            DISTINCT jsonb_build_object(
                'source', node_id,
                'target', target_node_id,
                'relationship', relationship_type,
                'properties', edge_properties
            )
        )
        FROM dag
        WHERE target_node_id IS NOT NULL
    )
);
$$ LANGUAGE sql;

-- Function: resolve_node
-- Maps aliases or partial names to canonical qualified_path
CREATE OR REPLACE FUNCTION resolve_node(ref text, tenant uuid)
RETURNS uuid AS $$
  SELECT id FROM catalog_node
  WHERE qualified_path = ref AND tenant_id = tenant
  LIMIT 1;
$$ LANGUAGE sql;

-- Add comment documentation
COMMENT ON FUNCTION get_calc_dag(text, uuid) IS 
'Builds a calculation DAG showing all dependencies for a given entity path';

COMMENT ON FUNCTION get_calc_dag_with_metadata(text, uuid) IS 
'Builds an enriched calculation DAG with lineage, data quality contracts, SLA, and properties';

COMMENT ON FUNCTION resolve_node(text, uuid) IS 
'Resolves a reference (qualified path) to a catalog node ID for a given tenant';

COMMENT ON COLUMN catalog_node.embedding IS 
'Vector embedding for semantic search using pgvector (768-dimensional for text-embedding-004)';
