-- Migration: 010_compat_views.sql
-- Creates a compatibility VIEW named 'business_objects' (plural) that maps to the
-- real 'business_object' (singular) table in the alpha DB.
-- The Go service currently queries/inserts 'business_objects' (plural) which doesn't exist.
-- 'legacy_business_objects' already exists as a BASE TABLE and is left as-is.

-- Ensure ENTITY bo_type exists (required NOT NULL FK in business_object)
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM public.bo_type WHERE type_name = 'ENTITY') THEN
    INSERT INTO public.bo_type (bo_type_id, type_name, description, created_at)
    VALUES (gen_random_uuid(), 'ENTITY', 'Standard Entity BO', now());
  END IF;
END $$;

-- Drop old view if it exists (safe re-run)
DROP VIEW IF EXISTS public.business_objects CASCADE;

-- business_objects compatibility view (SELECT)
CREATE OR REPLACE VIEW public.business_objects AS
SELECT
  bo_id                            AS id,
  tenant_id,
  bo_key                           AS key,
  bo_name                          AS name,
  bo_name                          AS display_name,
  ''                               AS technical_name,
  COALESCE(description, '')        AS description,
  ''                               AS icon,
  ''                               AS category,
  COALESCE(is_core, false)         AS is_core,
  COALESCE(is_active, true)        AS is_active,
  NULL::uuid                       AS parent_id,
  core_reference_bo_id             AS core_id,
  false                            AS enable_history,
  'EXPLICIT_RANGE'                 AS history_mode,
  '{}'::jsonb                      AS config,
  created_by,
  updated_by                       AS last_modified_by,
  created_at,
  updated_at                       AS last_modified_at,
  NULL::uuid                       AS datasource_id,
  NULL::uuid                       AS driver_table_id,
  ''                               AS driver_table_name,
  ''                               AS clone_parent_key,
  ''                               AS clone_parent_display_name,
  ''                               AS clones_from,
  0                                AS instance_count
FROM public.business_object;

-- INSTEAD OF INSERT rule so Go service INSERT INTO business_objects ... still works
CREATE OR REPLACE RULE business_objects_insert AS ON INSERT TO public.business_objects
DO INSTEAD
  INSERT INTO public.business_object (
    bo_id, tenant_id, bo_key, bo_name,
    description, is_core, is_active,
    core_reference_bo_id,
    created_by, updated_by, created_at, updated_at,
    status, bo_type_id, model_id
  ) VALUES (
    COALESCE(NULLIF(NEW.id::text, '')::uuid, gen_random_uuid()),
    NEW.tenant_id::uuid,
    COALESCE(NULLIF(NEW.key, ''), NULLIF(NEW.name, 'unnamed')),
    NEW.name,
    NULLIF(NEW.description, ''),
    COALESCE(NEW.is_core, false),
    COALESCE(NEW.is_active, true),
    NEW.core_id::uuid,
    NULLIF(NEW.created_by, ''),
    NULLIF(NEW.last_modified_by, ''),
    COALESCE(NEW.created_at, now()),
    COALESCE(NEW.last_modified_at, now()),
    'DRAFT',
    (SELECT bo_type_id FROM public.bo_type WHERE type_name = 'ENTITY' LIMIT 1),
    'default'
  );

-- INSTEAD OF UPDATE rule
CREATE OR REPLACE RULE business_objects_update AS ON UPDATE TO public.business_objects
DO INSTEAD
  UPDATE public.business_object SET
    bo_name    = NEW.name,
    description = NULLIF(NEW.description, ''),
    is_active  = COALESCE(NEW.is_active, OLD.is_active),
    is_core    = COALESCE(NEW.is_core, OLD.is_core),
    updated_by  = NULLIF(NEW.last_modified_by, ''),
    updated_at  = COALESCE(NEW.last_modified_at, now())
  WHERE bo_id = OLD.id::uuid
    AND tenant_id = OLD.tenant_id::uuid;

-- INSTEAD OF DELETE rule
CREATE OR REPLACE RULE business_objects_delete AS ON DELETE TO public.business_objects
DO INSTEAD
  UPDATE public.business_object
  SET status = 'DELETED', is_active = false, updated_at = now()
  WHERE bo_id = OLD.id::uuid
    AND tenant_id = OLD.tenant_id::uuid;

COMMENT ON VIEW public.business_objects IS
  'Compatibility view: maps legacy plural "business_objects" calls to real "business_object" table. '
  'Remove once Go service is updated to use business_object directly.';

-- Verify
SELECT count(*) AS business_objects_count FROM public.business_objects;
