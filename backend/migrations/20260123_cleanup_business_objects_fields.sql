-- Cleanup Business Objects Fields
-- 1. Drop the redundant 'fields' column
-- 2. Remove 'fields' key from 'config' JSONB column

-- Drop the column if it exists
ALTER TABLE public.business_objects DROP COLUMN IF EXISTS fields;

-- Remove 'fields' key from config
UPDATE public.business_objects 
SET config = config - 'fields' 
WHERE config ? 'fields';
