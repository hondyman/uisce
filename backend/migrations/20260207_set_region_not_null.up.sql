-- Ensure region column is non-null for semantic pre-aggregations and provide instructions for snapshots
-- Run only after backfill verification described in previous migration

-- 1) Safety check: fail if any pre-aggregations still have NULL region
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM semantic_pre_aggregations_v2 WHERE region IS NULL) THEN
        RAISE EXCEPTION 'semantic_pre_aggregations_v2 contains rows with NULL region; run backfill before applying NOT NULL';
    END IF;
END $$;

-- 2) Alter column to NOT NULL in Postgres
ALTER TABLE IF EXISTS semantic_pre_aggregations_v2
  ALTER COLUMN region SET NOT NULL;

-- Ensure index exists (already added previously, but keep idempotent here)
CREATE INDEX IF NOT EXISTS idx_semantic_pre_aggs_region ON semantic_pre_aggregations_v2(region);

-- 3) Snapshot (Iceberg) guidance (manual ops required)
-- NOTE: Iceberg/Trino/Starburst installations differ. Perform the following on your data platform:
--   a) Backfill the region column for all rows (see backend/scripts/backfill_snapshot_region.sh template)
--   b) Create a new table or CTAS that includes the region column as NOT NULL
--   c) Validate row counts and sanity checks
--   d) Swap the tables or use your platform's atomic rename/swap method
-- Example Trino/Starburst step (template only):
-- CREATE TABLE IF NOT EXISTS audit.semantic_snapshots_new AS
-- SELECT ss.snapshot_id, ss.semantic_term_id, ss.version, ss.timestamp, ss.definition,
--        ss.business_term_id, ss.tenant_id, COALESCE(n.region, 'unknown') AS region,
--        ss.compliance, ss.lineage, ss.metadata, ss._ingest_ts, ss._source_service, ss._schema_version
-- FROM iceberg.audit.semantic_snapshots ss
-- LEFT JOIN (SELECT properties->>'snapshot_id' AS snapshot_id, properties->>'region' AS region FROM postgres.public.catalog_node WHERE properties->>'snapshot_id' IS NOT NULL) n
--   ON n.snapshot_id = ss.snapshot_id;
-- After validation, rename/swap and then (if supported) add NOT NULL constraint via your engine's ALTER TABLE semantics.

-- 4) Verification queries (Postgres)
-- SELECT COUNT(*) FROM semantic_pre_aggregations_v2 WHERE region IS NULL; -- should be 0

-- 5) Rollback plan: If this migration fails due to nulls, backfill and re-run. If you must rollback, there is no guaranteed automatic rollback for Iceberg swaps; plan accordingly.
