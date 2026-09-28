-- Restore parent_id for Studio STI subtypes (child BOs inherit the parent's
-- driving table / binding). 20261002_business_object_studio_engine recreated
-- business_objects without parent_id; subtype create + loadBOSubtypesAndFields
-- still expect it.

ALTER TABLE public.business_objects
  ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES public.business_objects(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_bo_parent_id
  ON public.business_objects (tenant_id, parent_id)
  WHERE parent_id IS NOT NULL;
