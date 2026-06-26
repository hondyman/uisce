-- Migration: Migrate ag_catalog to public
-- Date: 2026-01-22

-- 1. Ensure public.bo_fields has all necessary columns from ag_catalog.bo_fields
DO $$ 
BEGIN
    -- tenant_id
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'tenant_id') THEN
        ALTER TABLE public.bo_fields ADD COLUMN tenant_id UUID;
    END IF;

    -- bo_id (business object id)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'bo_id') THEN
        ALTER TABLE public.bo_fields ADD COLUMN bo_id UUID;
    END IF;

    -- key
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'key') THEN
        ALTER TABLE public.bo_fields ADD COLUMN key VARCHAR(255);
    END IF;

    -- technical_name
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'technical_name') THEN
        ALTER TABLE public.bo_fields ADD COLUMN technical_name VARCHAR(255);
    END IF;

    -- subtype_id
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'subtype_id') THEN
        ALTER TABLE public.bo_fields ADD COLUMN subtype_id UUID;
    END IF;

    -- is_subtype_only
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'is_subtype_only') THEN
        ALTER TABLE public.bo_fields ADD COLUMN is_subtype_only BOOLEAN DEFAULT FALSE;
    END IF;

    -- is_core
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'is_core') THEN
        ALTER TABLE public.bo_fields ADD COLUMN is_core BOOLEAN DEFAULT FALSE;
    END IF;

    -- description
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'description') THEN
        ALTER TABLE public.bo_fields ADD COLUMN description TEXT;
    END IF;
    
    -- display_label
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'display_label') THEN
        ALTER TABLE public.bo_fields ADD COLUMN display_label VARCHAR(255);
    END IF;

    -- field_type
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'field_type') THEN
        ALTER TABLE public.bo_fields ADD COLUMN field_type VARCHAR(100);
    END IF;

    -- is_required
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'is_required') THEN
        ALTER TABLE public.bo_fields ADD COLUMN is_required BOOLEAN DEFAULT FALSE;
    END IF;

    -- is_system_field
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'is_system_field') THEN
        ALTER TABLE public.bo_fields ADD COLUMN is_system_field BOOLEAN DEFAULT FALSE;
    END IF;

    -- display_order
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'display_order') THEN
        ALTER TABLE public.bo_fields ADD COLUMN display_order INT;
    END IF;

    -- semantic_term_id
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'semantic_term_id') THEN
        ALTER TABLE public.bo_fields ADD COLUMN semantic_term_id UUID;
    END IF;

    -- timestamps
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'created_at') THEN
        ALTER TABLE public.bo_fields ADD COLUMN created_at TIMESTAMP WITH TIME ZONE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'updated_at') THEN
        ALTER TABLE public.bo_fields ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE;
    END IF;

    -- name (if missing - though field_name exists, some code uses name)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'bo_fields' AND column_name = 'name') THEN
        ALTER TABLE public.bo_fields ADD COLUMN name VARCHAR(255);
    END IF;
END $$;

-- 2. Data Migration (Optional/Safety)
-- If there's data in ag_catalog.bo_fields, move it to public.bo_fields
-- Note: We map columns appropriately
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'ag_catalog' AND table_name = 'bo_fields') THEN
    INSERT INTO public.bo_fields (
      id, tenant_id, bo_id, subtype_id, key, name, display_label, technical_name, 
      field_type, is_core, is_subtype_only, is_required, is_system_field, description, 
      display_order, semantic_term_id, created_at, updated_at
    )
    SELECT 
      id, tenant_id, business_object_id, subtype_id, key, name, COALESCE(display_name, name), technical_name,
      type, COALESCE(is_core, false), COALESCE(is_subtype_only, false), COALESCE(is_required, false), COALESCE(is_system, false), description,
      sequence, semantic_term_id, created_at, last_modified_at
    FROM ag_catalog.bo_fields
    ON CONFLICT (id) DO UPDATE SET
      tenant_id = EXCLUDED.tenant_id,
      bo_id = EXCLUDED.bo_id,
      subtype_id = EXCLUDED.subtype_id,
      key = EXCLUDED.key,
      name = EXCLUDED.name,
      display_label = EXCLUDED.display_label,
      technical_name = EXCLUDED.technical_name,
      field_type = EXCLUDED.field_type,
      is_core = EXCLUDED.is_core,
      is_subtype_only = EXCLUDED.is_subtype_only,
      is_required = EXCLUDED.is_required,
      is_system_field = EXCLUDED.is_system_field,
      description = EXCLUDED.description,
      display_order = EXCLUDED.display_order,
      semantic_term_id = EXCLUDED.semantic_term_id,
      updated_at = EXCLUDED.updated_at;
  ELSE
    RAISE NOTICE 'No ag_catalog.bo_fields present - skipping data migration';
  END IF;
END
$do$;

-- 3. Drop ag_catalog schema and all its tables
DROP SCHEMA IF EXISTS ag_catalog CASCADE;

-- 4. Re-enable AGE if needed (optional, AGE usually resides in public or its own schema, 
-- but let's ensure it's not broken if it was relying on ag_catalog)
-- CREATE EXTENSION IF NOT EXISTS age;
