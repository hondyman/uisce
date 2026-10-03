-- 20261204_001_cube_definition (down)
--
-- Drops the cube definition table. The indexes and the RLS policy are owned by
-- the table and go with it, so only the table itself needs an explicit DROP.
--
-- This is destructive: any deployed cube materializations (mv_gold_* or
-- mv_{tenant}_*) are NOT dropped here, because physical objects are derived
-- state and are reconciled separately. A dropped cube stops being routed to;
-- the orphaned physical object is harmless until pruned.
DROP TABLE IF EXISTS data_explorer.cube_definition;
