-- Drop redundant triggers identified by pg_trigger audit 2026-09-20
-- Phase B: unused semantic cube cache invalidation
--
-- semantic_cube_cache: 0 live rows, 0 index scans, no FK references, no views.
-- The three triggers all call invalidate_semantic_cube_cache() which DELETEs from
-- that empty table on every I/U/D to semantic_cubes_v2/dimensions_v2/measures_v2.
--
-- Verified: no Go code references semantic_cube_cache (grep ran clean).
-- If any reader is later discovered, the down migration restores everything.

BEGIN;

-- Drop triggers first (function depends on them being gone before we can drop the function)
DROP TRIGGER IF EXISTS semantic_cubes_v2_cache_invalidate
    ON public.semantic_cubes_v2;
DROP TRIGGER IF EXISTS semantic_dimensions_v2_cache_invalidate
    ON public.semantic_dimensions_v2;
DROP TRIGGER IF EXISTS semantic_measures_v2_cache_invalidate
    ON public.semantic_measures_v2;
DROP FUNCTION  IF EXISTS public.invalidate_semantic_cube_cache();
-- Drop the table last (the function references it; must be gone first)
DROP TABLE     IF EXISTS public.semantic_cube_cache;

COMMIT;
