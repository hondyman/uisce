-- Rollback for 20260920_002b_drop_semantic_cube_cache.up.sql
-- Re-creates the cache table, function, and three triggers.
-- Authoritative function body saved to: /tmp/trigger_fn_backups.sql

BEGIN;

-- Table must exist before the function (the function DELETE references it)
CREATE TABLE IF NOT EXISTS public.semantic_cube_cache (
    tenant_id  UUID        NOT NULL,
    cube_name  TEXT        NOT NULL,
    payload    JSONB       NOT NULL,
    cached_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, cube_name)
);

-- Function: invalidate_semantic_cube_cache
-- tgtype 29 = ROW AFTER INSERT OR UPDATE OR DELETE
CREATE OR REPLACE FUNCTION public.invalidate_semantic_cube_cache()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
DECLARE
    target_cube_id UUID;
    t_id UUID;
    c_name TEXT;
BEGIN
    IF TG_TABLE_NAME = 'semantic_cubes_v2' THEN
        target_cube_id := COALESCE(NEW.id, OLD.id);
    ELSE
        target_cube_id := COALESCE(NEW.cube_id, OLD.cube_id);
    END IF;
    SELECT tenant_id, name INTO t_id, c_name
    FROM semantic_cubes_v2
    WHERE id = target_cube_id;
    IF FOUND THEN
        DELETE FROM semantic_cube_cache
        WHERE tenant_id = t_id AND cube_name = c_name;
    END IF;
    RETURN NULL;
END;
$function$;

-- Re-attach the three triggers
CREATE TRIGGER semantic_cubes_v2_cache_invalidate
    AFTER INSERT OR UPDATE OR DELETE ON public.semantic_cubes_v2
    FOR EACH ROW EXECUTE FUNCTION public.invalidate_semantic_cube_cache();

CREATE TRIGGER semantic_dimensions_v2_cache_invalidate
    AFTER INSERT OR UPDATE OR DELETE ON public.semantic_dimensions_v2
    FOR EACH ROW EXECUTE FUNCTION public.invalidate_semantic_cube_cache();

CREATE TRIGGER semantic_measures_v2_cache_invalidate
    AFTER INSERT OR UPDATE OR DELETE ON public.semantic_measures_v2
    FOR EACH ROW EXECUTE FUNCTION public.invalidate_semantic_cube_cache();

COMMIT;
