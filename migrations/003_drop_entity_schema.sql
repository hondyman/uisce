-- ============================================================================
-- MIGRATION: Drop legacy entity_schema table
-- ============================================================================
-- All business object schemas are now consolidated in business_objects table
-- with complete entity definitions stored in JSONB config column.
-- The entity_schema table is no longer needed after API migration.
-- Date: 2025-11-10

-- Drop the legacy entity_schema table
DROP TABLE IF EXISTS public.entity_schema CASCADE;

-- Confirm successful removal
SELECT 'entity_schema table dropped successfully' as status;
