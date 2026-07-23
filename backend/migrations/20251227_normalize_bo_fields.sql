-- Migration: Normalize business_objects.fields JSONB to bo_fields table
-- 
-- This migration:
-- 1. Extracts fields from the JSONB 'fields' column in business_objects
-- 2. Inserts them as individual rows in bo_fields table
-- 3. Drops the 'fields' column from business_objects
-- 4. Updates bo_fields.sequence based on original array position
--
-- Run this AFTER migration 000029 (bo_fields table creation)

-- ============================================================================
-- Step 1: Create a temporary table to hold extracted fields
-- ============================================================================
CREATE TEMP TABLE temp_extracted_fields AS
SELECT 
  bo.id as business_object_id,
  bo.tenant_id,
  (field->>'key')::text as key,
  (field->>'name')::text as name,
  (field->>'display_name')::text as display_name,
  (field->>'technical_name')::text as technical_name,
  (field->>'type')::text as type,
  (field->>'is_core')::boolean as is_core,
  (field->>'is_required')::boolean as is_required,
  (field->>'is_system')::boolean as is_system,
  (field->>'description')::text as description,
  (field->>'reference_entity')::text as reference_entity,
  (field->>'sequence')::integer as sequence,
  (field->>'created_at')::timestamptz as created_at,
  (field->>'created_by')::uuid as created_by,
  (field->>'last_modified_at')::timestamptz as last_modified_at,
  (field->>'last_modified_by')::uuid as last_modified_by,
  row_number() OVER (PARTITION BY bo.id ORDER BY (field->>'sequence')::integer, (field->>'key')) as row_num
FROM public.business_objects bo,
     jsonb_array_elements(COALESCE(bo.config->'fields', '[]'::jsonb)) as field
WHERE bo.config->'fields' IS NOT NULL 
  AND bo.config->'fields' != '[]'::jsonb
  AND bo.tenant_id IN (SELECT id FROM public.tenants);

-- ============================================================================
-- Step 2: Insert extracted fields into bo_fields table (compatible with variant schemas)
-- ============================================================================
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'key') THEN
    INSERT INTO public.bo_fields (
      tenant_id,
      business_object_id,
      subtype_id,
      key,
      name,
      display_name,
      technical_name,
      type,
      is_core,
      is_required,
      is_system,
      description,
      reference_entity,
      sequence,
      created_at,
      created_by,
      last_modified_at,
      last_modified_by
    )
    SELECT 
      tenant_id,
      business_object_id,
      NULL, -- these fields are at the BO level, not subtype level
      COALESCE(key, 'field_' || row_num::text),
      COALESCE(name, 'Field ' || row_num::text),
      COALESCE(display_name, 'Field ' || row_num::text),
      COALESCE(technical_name, key),
      COALESCE(type, 'text'), -- default type
      COALESCE(is_core, false),
      COALESCE(is_required, false),
      COALESCE(is_system, false),
      description,
      reference_entity,
      COALESCE(sequence, row_num),
      COALESCE(created_at, now()),
      created_by,
      COALESCE(last_modified_at, now()),
      last_modified_by
    FROM temp_extracted_fields
    ON CONFLICT DO NOTHING;
  ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'entity_key') THEN
    INSERT INTO public.bo_fields (
      tenant_id,
      business_object_id,
      subtype_id,
      entity_key,
      name,
      display_name,
      technical_name,
      type,
      is_core,
      is_required,
      is_system,
      description,
      reference_entity,
      sequence,
      created_at,
      created_by,
      last_modified_at,
      last_modified_by
    )
    SELECT 
      tenant_id,
      business_object_id,
      NULL, -- these fields are at the BO level, not subtype level
      COALESCE(key, 'field_' || row_num::text),
      COALESCE(name, 'Field ' || row_num::text),
      COALESCE(display_name, 'Field ' || row_num::text),
      COALESCE(technical_name, key),
      COALESCE(type, 'text'), -- default type
      COALESCE(is_core, false),
      COALESCE(is_required, false),
      COALESCE(is_system, false),
      description,
      reference_entity,
      COALESCE(sequence, row_num),
      COALESCE(created_at, now()),
      created_by,
      COALESCE(last_modified_at, now()),
      last_modified_by
    FROM temp_extracted_fields
    ON CONFLICT DO NOTHING;
  ELSE
    -- As a last resort, insert minimal fields where possible
    INSERT INTO public.bo_fields (
      tenant_id,
      business_object_id,
      subtype_id,
      name,
      display_name,
      technical_name,
      type,
      is_core,
      is_required,
      is_system,
      description,
      reference_entity,
      sequence,
      created_at,
      created_by,
      last_modified_at,
      last_modified_by
    )
    SELECT 
      tenant_id,
      business_object_id,
      NULL,
      COALESCE(name, 'Field ' || row_num::text),
      COALESCE(display_name, 'Field ' || row_num::text),
      COALESCE(technical_name, key),
      COALESCE(type, 'text'),
      COALESCE(is_core, false),
      COALESCE(is_required, false),
      COALESCE(is_system, false),
      description,
      reference_entity,
      COALESCE(sequence, row_num),
      COALESCE(created_at, now()),
      created_by,
      COALESCE(last_modified_at, now()),
      last_modified_by
    FROM temp_extracted_fields
    ON CONFLICT DO NOTHING;
  END IF;
END$$;
-- ============================================================================
-- Step 3: Drop the fields column from business_objects
-- ============================================================================
ALTER TABLE public.business_objects DROP COLUMN IF EXISTS fields CASCADE;

-- ============================================================================
-- Step 4: Add key column to business_objects (if not already present)
--         to match the simplified model
-- ============================================================================
-- Step 4 skipped: entity_key already exists
-- ALTER TABLE public.business_objects ADD COLUMN IF NOT EXISTS entity_key varchar(255) UNIQUE;

-- ============================================================================
-- Verification Queries (run these manually to verify migration)
-- ============================================================================

-- Check how many BOs have fields
-- SELECT COUNT(*) as bo_count FROM public.business_objects;

-- Check how many fields were migrated
-- SELECT COUNT(*) as field_count FROM public.bo_fields;

-- Check fields for a specific BO
-- SELECT * FROM public.bo_fields WHERE business_object_id = $1 ORDER BY sequence;

-- Check for any BOs missing fields (if they originally had fields)
-- SELECT bo.id, bo.name, COUNT(bf.id) as field_count
-- FROM public.business_objects bo
-- LEFT JOIN public.bo_fields bf ON bf.business_object_id = bo.id
-- GROUP BY bo.id, bo.name
-- HAVING bo.fields IS NOT NULL;

