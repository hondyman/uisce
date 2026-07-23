-- Migration: 20260124_drop_semantic_lineage.sql
-- Description: Drop redundant lineage tables in semantic schema

DROP TABLE IF EXISTS semantic.lineage_edges CASCADE;
DROP TABLE IF EXISTS semantic.lineage_nodes CASCADE;

-- Also remove redundant indexes if they were created separately
DROP INDEX IF EXISTS semantic.idx_lineage_edges_from;
DROP INDEX IF EXISTS semantic.idx_lineage_edges_to;
DROP INDEX IF EXISTS semantic.idx_lineage_nodes_tenant;
