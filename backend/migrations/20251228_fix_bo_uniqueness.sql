-- Fix business_objects uniqueness to account for datasource scope
-- Date: 2025-12-28

BEGIN;

-- 1) Ensure datasource_id column exists (nullable for global BOs)
ALTER TABLE public.business_objects 
ADD COLUMN IF NOT EXISTS datasource_id UUID;

-- Optionally add FK if tenant_product_datasource exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_schema = 'public' AND table_name = 'tenant_product_datasource'
    ) THEN
        -- Add FK only if not already present
        IF NOT EXISTS (
            SELECT 1
            FROM information_schema.table_constraints tc
            WHERE tc.table_schema = 'public'
              AND tc.table_name = 'business_objects'
              AND tc.constraint_type = 'FOREIGN KEY'
              AND tc.constraint_name = 'bo_datasource_fk'
        ) THEN
            ALTER TABLE public.business_objects 
            ADD CONSTRAINT bo_datasource_fk FOREIGN KEY (datasource_id)
            REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

-- 2) Drop old uniqueness constraints based on name or key without datasource
ALTER TABLE public.business_objects 
DROP CONSTRAINT IF EXISTS business_objects_name_tenant_id_key;

ALTER TABLE public.business_objects 
DROP CONSTRAINT IF EXISTS business_objects_unique;

-- Drop prior unique indexes if they exist
DROP INDEX IF EXISTS idx_bo_key_global;
DROP INDEX IF EXISTS idx_bo_key_datasource;

-- 3) Create new natural key uniqueness that respects datasource scope
-- Global BOs (datasource_id IS NULL): unique on (tenant_id, key)
CREATE UNIQUE INDEX IF NOT EXISTS idx_bo_natural_global
ON public.business_objects(tenant_id, key)
WHERE datasource_id IS NULL;

-- Datasource-scoped BOs (datasource_id IS NOT NULL): unique on (tenant_id, key, datasource_id)
CREATE UNIQUE INDEX IF NOT EXISTS idx_bo_natural_scoped
ON public.business_objects(tenant_id, key, datasource_id)
WHERE datasource_id IS NOT NULL;

COMMIT;
