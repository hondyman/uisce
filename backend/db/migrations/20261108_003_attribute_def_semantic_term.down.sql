DROP INDEX IF EXISTS public.idx_attribute_def_semantic_term;
ALTER TABLE public.attribute_def DROP COLUMN IF EXISTS semantic_term_id;
