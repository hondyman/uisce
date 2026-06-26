-- Add datasource_id to business_objects and update natural key
-- Date: 2026-01-06

-- 1. Add datasource_id column
ALTER TABLE public.business_objects 
ADD COLUMN IF NOT EXISTS datasource_id UUID REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE;

-- 2. Drop old unique constraint
ALTER TABLE public.business_objects 
DROP CONSTRAINT IF EXISTS business_objects_unique;

-- 3. Create new partial unique indexes for natural key
-- Case 1: Global Business Objects (datasource_id IS NULL)
CREATE UNIQUE INDEX IF NOT EXISTS idx_bo_key_global 
ON public.business_objects(tenant_id, key) 
WHERE datasource_id IS NULL;

-- Case 2: Datasource-scoped Business Objects (datasource_id IS NOT NULL)
CREATE UNIQUE INDEX IF NOT EXISTS idx_bo_key_datasource 
ON public.business_objects(tenant_id, key, datasource_id) 
WHERE datasource_id IS NOT NULL;

-- 4. Update Instances to respect new integrity (optional, but good practice)
-- Ensure instances' datasource matches BO's datasource if BO is scoped
-- (This is complex to enforce with simple constraints if BO can be global. 
--  If BO is global, Instance can have any datasource. 
--  If BO is scoped, Instance MUST match that boolean logic. 
--  We won't add a trigger now but adding an index for lookups is good.)
CREATE INDEX IF NOT EXISTS idx_business_objects_datasource ON public.business_objects(datasource_id);
