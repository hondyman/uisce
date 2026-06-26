-- Migration: Add role and semantic_term_id to bo_fields
-- These fields enable linking BO fields to semantic terms and defining their roles (Dimension, Measure, etc.)

ALTER TABLE public.bo_fields 
ADD COLUMN IF NOT EXISTS role varchar(50),
ADD COLUMN IF NOT EXISTS semantic_term_id uuid;

-- Add index for semantic_term_id
CREATE INDEX IF NOT EXISTS idx_bo_fields_semantic_term ON public.bo_fields(semantic_term_id);

COMMENT ON COLUMN public.bo_fields.role IS 'The role of the field (DIMENSION, MEASURE, VALIDITY_START, etc.)';
COMMENT ON COLUMN public.bo_fields.semantic_term_id IS 'Link to the semantic term in catalog_node';
