-- Migration: Add core_id columns to business objects tables
-- Purpose: Enable Workday-style Core/Custom metadata separation
-- Each tenant's business object extensions link back to the gold copy source

-- ============================================================================
-- 1. Add core_id to business_objects
-- ============================================================================

ALTER TABLE public.business_objects 
ADD COLUMN IF NOT EXISTS core_id uuid;

-- Self-referential FK: links tenant BO to its gold copy source BO
-- ON DELETE SET NULL: if gold copy BO is deleted, tenant keeps a standalone copy
ALTER TABLE public.business_objects
DROP CONSTRAINT IF EXISTS business_objects_core_fk;

ALTER TABLE public.business_objects
ADD CONSTRAINT business_objects_core_fk 
    FOREIGN KEY (core_id) 
    REFERENCES public.business_objects(id) 
    ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS business_objects_core_id_idx 
    ON public.business_objects (core_id) 
    WHERE core_id IS NOT NULL;

-- ============================================================================
-- 2. Add core_id to bo_fields
-- ============================================================================

ALTER TABLE public.bo_fields 
ADD COLUMN IF NOT EXISTS core_id uuid;

ALTER TABLE public.bo_fields
DROP CONSTRAINT IF EXISTS bo_fields_core_fk;

ALTER TABLE public.bo_fields
ADD CONSTRAINT bo_fields_core_fk 
    FOREIGN KEY (core_id) 
    REFERENCES public.bo_fields(id) 
    ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS bo_fields_core_id_idx 
    ON public.bo_fields (core_id) 
    WHERE core_id IS NOT NULL;

-- ============================================================================
-- 3. Add core_id to bo_subtypes
-- ============================================================================

ALTER TABLE public.bo_subtypes 
ADD COLUMN IF NOT EXISTS core_id uuid;

ALTER TABLE public.bo_subtypes
DROP CONSTRAINT IF EXISTS bo_subtypes_core_fk;

ALTER TABLE public.bo_subtypes
ADD CONSTRAINT bo_subtypes_core_fk 
    FOREIGN KEY (core_id) 
    REFERENCES public.bo_subtypes(id) 
    ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS bo_subtypes_core_id_idx 
    ON public.bo_subtypes (core_id) 
    WHERE core_id IS NOT NULL;

-- ============================================================================
-- 4. Helper function to get gold copy tenant ID
-- ============================================================================

CREATE OR REPLACE FUNCTION public.get_gold_copy_tenant_id()
RETURNS uuid AS $$
DECLARE
    gcid uuid;
BEGIN
    SELECT id INTO gcid 
    FROM public.tenants 
    WHERE gold_copy = true 
    LIMIT 1;
    RETURN gcid;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- 5. View for composed business objects (optional - can be used as fallback)
-- ============================================================================

CREATE OR REPLACE VIEW public.business_objects_composed AS
WITH gold_copy AS (
    SELECT id, name, display_name, technical_name, description, icon, 
           config, category, parent_id, is_active, created_at, last_modified_at
    FROM public.business_objects
    WHERE tenant_id = public.get_gold_copy_tenant_id()
      AND is_core = true
)
SELECT 
    COALESCE(custom.id, core.id) as id,
    custom.tenant_id,
    COALESCE(custom.name, core.name) as name,
    COALESCE(custom.display_name, core.display_name) as display_name,
    COALESCE(custom.technical_name, core.technical_name) as technical_name,
    COALESCE(custom.description, core.description) as description,
    COALESCE(custom.icon, core.icon) as icon,
    COALESCE(custom.config, core.config) as config,
    COALESCE(custom.category, core.category) as category,
    COALESCE(custom.parent_id, core.parent_id) as parent_id,
    COALESCE(custom.is_active, core.is_active) as is_active,
    core.id as core_id,
    CASE WHEN custom.id IS NULL THEN true ELSE false END as is_pure_core,
    core.created_at as core_created_at,
    custom.created_at as custom_created_at,
    custom.last_modified_at
FROM gold_copy core
LEFT JOIN public.business_objects custom 
    ON custom.core_id = core.id 
    AND custom.tenant_id != public.get_gold_copy_tenant_id()
UNION ALL
-- Include tenant-only BOs (no core_id, not from gold copy)
SELECT 
    id, tenant_id, name, display_name, technical_name, description, icon,
    config, category, parent_id, is_active,
    NULL as core_id,
    false as is_pure_core,
    NULL as core_created_at,
    created_at as custom_created_at,
    last_modified_at
FROM public.business_objects
WHERE tenant_id != public.get_gold_copy_tenant_id()
  AND core_id IS NULL;

COMMENT ON VIEW public.business_objects_composed IS 
    'Workday-style composed view: Core BO definitions merged with tenant customizations';
