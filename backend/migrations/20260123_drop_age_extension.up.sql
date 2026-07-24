-- Drop Apache AGE extension and all related objects
-- This migration removes AGE in favor of using the existing catalog_node and catalog_edge relational tables

-- Drop the AGE graph if it exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM ag_catalog.ag_graph WHERE name = 'semantic_lineage') THEN
        PERFORM ag_catalog.drop_graph('semantic_lineage', true);
    END IF;
EXCEPTION
    WHEN undefined_function THEN
        -- AGE not loaded, continue
        NULL;
    WHEN undefined_table THEN
        -- ag_catalog doesn't exist, continue
        NULL;
END $$;

-- Drop the AGE extension
DROP EXTENSION IF EXISTS age CASCADE;

-- Remove ag_catalog from search path if present
-- Note: This only affects current session, users will need to update their postgresql.conf if needed
RESET search_path;

-- Add comment explaining the change
COMMENT ON DATABASE alpha IS 'Using catalog_node and catalog_edge for lineage instead of Apache AGE';
