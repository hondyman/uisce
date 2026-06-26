-- ============================================================================
-- MIGRATION: Add technical_name column to business_objects for subtype keys
-- ============================================================================
-- Adds technical_name column to store normalized keys (retail_customer, individual_investor, etc.)
-- This ensures subtypes can be properly keyed in the API response without hardcoding
-- Date: 2025-11-10

ALTER TABLE public.business_objects 
ADD COLUMN technical_name text;

-- Update technical_name for parent objects (from config if available)
UPDATE public.business_objects bo
SET technical_name = COALESCE(
    config->>'technical_name',
    lower(replace(replace(name, ' ', '_'), '&', 'and'))
)
WHERE parent_id IS NULL AND tenant_id = (SELECT id FROM tenants LIMIT 1);

-- Update technical_name for subtypes based on pattern
UPDATE public.business_objects bo
SET technical_name = lower(replace(replace(name, ' ', '_'), '&', 'and'))
WHERE parent_id IS NOT NULL AND tenant_id = (SELECT id FROM tenants LIMIT 1);

-- Create index for lookups
CREATE INDEX idx_business_objects_technical_name ON public.business_objects(technical_name);

-- Verify the updates
SELECT name, technical_name, parent_id FROM public.business_objects 
WHERE parent_id IS NOT NULL 
ORDER BY name;
