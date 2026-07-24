-- Fix invalid cache invalidation trigger function
CREATE OR REPLACE FUNCTION invalidate_semantic_cube_cache()
RETURNS TRIGGER AS $$
DECLARE
    target_cube_id UUID;
    t_id UUID;
    c_name TEXT;
BEGIN
    -- Determine cube ID based on table
    IF TG_TABLE_NAME = 'semantic_cubes_v2' THEN
        target_cube_id := COALESCE(NEW.id, OLD.id);
    ELSE
        target_cube_id := COALESCE(NEW.cube_id, OLD.cube_id);
    END IF;

    -- Get tenant_id and name from the cube table
    -- This handles cases where child tables (dimensions/measures) don't have tenant_id
    SELECT tenant_id, name INTO t_id, c_name
    FROM semantic_cubes_v2
    WHERE id = target_cube_id;

    -- Invalidate cache if cube found
    IF FOUND THEN
        DELETE FROM semantic_cube_cache 
        WHERE tenant_id = t_id AND cube_name = c_name;
    END IF;
    
    RETURN NULL; -- Return NULL for AFTER triggers (or NEW for BEFORE)
END;
$$ LANGUAGE plpgsql;
