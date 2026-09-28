DROP INDEX IF EXISTS public.idx_bo_parent_id;

ALTER TABLE public.business_objects
  DROP COLUMN IF EXISTS parent_id;
