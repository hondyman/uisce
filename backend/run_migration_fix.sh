#!/bin/bash
# Run the business objects schema fix migration directly

psql -h localhost -U postgres -d postgres << 'EOF'
-- Fix business_objects schema mismatch
-- This migration renames entity_key to key and adds missing columns

-- Fix business_objects table
DO $$
BEGIN
    -- Rename entity_key to key if it exists
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='business_objects' AND column_name='entity_key'
    ) THEN
        ALTER TABLE public.business_objects RENAME COLUMN entity_key TO key;
        RAISE NOTICE 'Renamed entity_key to key in business_objects';
    END IF;
    
    -- Add parent_id if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='business_objects' AND column_name='parent_id'
    ) THEN
        ALTER TABLE public.business_objects ADD COLUMN parent_id uuid;
        ALTER TABLE public.business_objects ADD CONSTRAINT business_objects_parent_fk 
            FOREIGN KEY (parent_id) REFERENCES public.business_objects(id) ON DELETE CASCADE;
        RAISE NOTICE 'Added parent_id column to business_objects';
    END IF;
    
    -- Add config if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='business_objects' AND column_name='config'
    ) THEN
        ALTER TABLE public.business_objects ADD COLUMN config jsonb DEFAULT '{}'::jsonb;
        RAISE NOTICE 'Added config column to business_objects';
    END IF;
    
    -- Add is_active if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='business_objects' AND column_name='is_active'
    ) THEN
        ALTER TABLE public.business_objects ADD COLUMN is_active boolean DEFAULT true;
        RAISE NOTICE 'Added is_active column to business_objects';
    END IF;
END $$;

-- Fix bo_subtypes table
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='bo_subtypes' AND column_name='entity_key'
    ) THEN
        ALTER TABLE public.bo_subtypes RENAME COLUMN entity_key TO key;
        RAISE NOTICE 'Renamed entity_key to key in bo_subtypes';
    END IF;
END $$;

-- Fix bo_fields table
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='bo_fields' AND column_name='entity_key'
    ) THEN
        ALTER TABLE public.bo_fields RENAME COLUMN entity_key TO key;
        RAISE NOTICE 'Renamed entity_key to key in bo_fields';
    END IF;
END $$;

-- Drop and recreate indexes with correct column names
DROP INDEX IF EXISTS public.business_objects_entity_key_idx;
CREATE INDEX IF NOT EXISTS business_objects_key_idx ON public.business_objects (key);

DROP INDEX IF EXISTS public.bo_fields_entity_key_idx;
CREATE INDEX IF NOT EXISTS bo_fields_key_idx ON public.bo_fields (key);

-- Update unique constraint for business_objects
DO $$
BEGIN
    -- Drop old constraint if it exists
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'business_objects_unique' 
        AND conrelid = 'public.business_objects'::regclass
    ) THEN
        ALTER TABLE public.business_objects DROP CONSTRAINT business_objects_unique;
        RAISE NOTICE 'Dropped old business_objects_unique constraint';
    END IF;
    
    -- Add new constraint with correct column name
    ALTER TABLE public.business_objects ADD CONSTRAINT business_objects_unique UNIQUE (tenant_id, key);
    RAISE NOTICE 'Added new business_objects_unique constraint';
END $$;

-- Update unique constraint for bo_subtypes
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'bo_subtypes_unique' 
        AND conrelid = 'public.bo_subtypes'::regclass
    ) THEN
        ALTER TABLE public.bo_subtypes DROP CONSTRAINT bo_subtypes_unique;
        ALTER TABLE public.bo_subtypes ADD CONSTRAINT bo_subtypes_unique UNIQUE (tenant_id, business_object_id, key);
        RAISE NOTICE 'Updated bo_subtypes_unique constraint';
    END IF;
END $$;

SELECT 'Migration completed successfully!' AS status;
EOF
