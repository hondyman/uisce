#!/usr/bin/env bash
# Backfill script (manual) to populate `region` in iceberg.audit.semantic_snapshots.
# This script is an example; adapt to your cluster (Trino/Presto/Starburst) or run via your data platform.

set -euo pipefail

cat <<'SQL'
-- Example (Trino / Starburst): Create a new table with the region column populated, then swap atomically.
CREATE TABLE audit.semantic_snapshots_new AS
SELECT
  ss.*,
  COALESCE(n.region, 'unknown') AS region
FROM iceberg.audit.semantic_snapshots ss
LEFT JOIN (
  SELECT properties->>'snapshot_id' AS snapshot_id, properties->>'region' AS region
  FROM postgres.public.catalog_node
  WHERE properties->>'snapshot_id' IS NOT NULL
) n ON n.snapshot_id = ss.snapshot_id;

-- Validate counts / sanity checks here.
-- Then (depending on your platform) rename and drop old table or perform an atomic swap.
SQL

echo "NOTE: This is a template. Run the SQL above in your Trino/Starburst/SQL client with proper permissions and platform-specific swap steps."
