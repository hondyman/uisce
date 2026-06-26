-- Add region to semantic pre-aggregations and audit snapshots
-- Run with: make migrate (project migration runner will pick this up)

-- 1) Add region column to semantic_pre_aggregations_v2 (Postgres)
ALTER TABLE IF EXISTS semantic_pre_aggregations_v2
  ADD COLUMN IF NOT EXISTS region TEXT;

CREATE INDEX IF NOT EXISTS idx_semantic_pre_aggs_region ON semantic_pre_aggregations_v2(region);

-- Backfill plan (manual script run recommended):
-- If pre-aggregations were stored in catalog_node properties, run:
-- UPDATE semantic_pre_aggregations_v2 sp
-- SET region = coalesce((
--     SELECT n.properties->>'region' FROM catalog_node n
--     WHERE n.node_name = sp.name AND (n.tenant_id::text = sp.tenant_id::text OR n.tenant_id IS NULL)
--     LIMIT 1
-- ), sp.region);

-- 2) Add region column to Iceberg audit semantic_snapshots
-- Note: Altering Iceberg tables depends on the engine. For Trino/Starburst/ICEBERG you can use:
-- ALTER TABLE iceberg.audit.semantic_snapshots ADD COLUMN IF NOT EXISTS region VARCHAR;
-- Depending on your environment you may also want to re-partition by region; that's an operational task and not performed here.

-- Backfill for snapshots (example Trino SQL):
-- -- This reads region from catalog_node properties which the catalog ingestion worker already populated when ingesting snapshots
-- INSERT INTO iceberg.audit.semantic_snapshots /* or UPDATE via CTAS/INSERT-OVERWRITE depending on engine support */
-- SELECT ss.*, coalesce(n.properties->>'region', 'unknown') AS region
-- FROM iceberg.audit.semantic_snapshots ss
-- LEFT JOIN (
--   SELECT (properties->>'snapshot_id') AS snapshot_id, properties->>'region' AS region
--   FROM public.catalog_node
--   WHERE properties->>'snapshot_id' IS NOT NULL
-- ) n ON n.snapshot_id = ss.snapshot_id;

-- NOTE: Because Iceberg/Trino semantics vary across installations, perform the snapshot backfill using your platform's recommended method (CTAS with new column, then swap).

-- 3) Migration verification queries (Postgres)
-- SELECT COUNT(*) FROM semantic_pre_aggregations_v2 WHERE region IS NULL;
-- (Expect 0 after backfill if all pre-aggregations have region)

-- 4) Rollout plan
-- - Run this migration (adds nullable region columns)
-- - Backfill semantic_pre_aggregations_v2 using the UPDATE shown above
-- - Backfill iceberg.audit.semantic_snapshots using your Trino/CTAS procedure
-- - After verification (and acceptance tests), change columns to NOT NULL in a follow-up migration if desired
