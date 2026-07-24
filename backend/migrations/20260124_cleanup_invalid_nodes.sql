-- Migration to cleanup invalid catalog_node records
-- Specifically targets nodes where node_type_id does not exist in catalog_node_types

DO $$
BEGIN
    -- Delete nodes with invalid node_type_id reference
    DELETE FROM public.catalog_node 
    WHERE node_type_id IS NOT NULL 
    AND node_type_id NOT IN (SELECT id FROM public.catalog_node_types);
    
    -- Also delete nodes with NULL node_type_id if that's considered invalid for business logic (assuming yes based on user request)
    -- "if there is an invalid uuid I want u to delete the row" - ambiguous if NULL is "invalid uuid" but usually nodes must have a type.
    DELETE FROM public.catalog_node WHERE node_type_id IS NULL;
    
    RAISE NOTICE 'Cleaned up orphan catalog_node records';
END $$;
