-- Migration: Add cascading deletes to catalog tables
-- Created: 2026-01-23

-- Add foreign key constraints with ON DELETE CASCADE for catalog_edge
ALTER TABLE public.catalog_edge DROP CONSTRAINT IF EXISTS fk_catalog_edge_source_node;
ALTER TABLE public.catalog_edge 
ADD CONSTRAINT fk_catalog_edge_source_node 
FOREIGN KEY (source_node_id) 
REFERENCES public.catalog_node(id) 
ON DELETE CASCADE;

ALTER TABLE public.catalog_edge DROP CONSTRAINT IF EXISTS fk_catalog_edge_target_node;
ALTER TABLE public.catalog_edge 
ADD CONSTRAINT fk_catalog_edge_target_node 
FOREIGN KEY (target_node_id) 
REFERENCES public.catalog_node(id) 
ON DELETE CASCADE;

-- Add foreign key constraints with ON DELETE CASCADE for semantic_mapping_suggestions
ALTER TABLE public.semantic_mapping_suggestions DROP CONSTRAINT IF EXISTS fk_semantic_mapping_suggestions_column;
ALTER TABLE public.semantic_mapping_suggestions 
ADD CONSTRAINT fk_semantic_mapping_suggestions_column 
FOREIGN KEY (database_column_node_id) 
REFERENCES public.catalog_node(id) 
ON DELETE CASCADE;

ALTER TABLE public.semantic_mapping_suggestions DROP CONSTRAINT IF EXISTS fk_semantic_mapping_suggestions_term;
ALTER TABLE public.semantic_mapping_suggestions 
ADD CONSTRAINT fk_semantic_mapping_suggestions_term 
FOREIGN KEY (semantic_term_node_id) 
REFERENCES public.catalog_node(id) 
ON DELETE CASCADE;
