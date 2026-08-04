-- 20260805000001_marketplace_unified_schema.down.sql
-- Reverses: added columns, new tables, new indexes, Gold Copy role seeds.

BEGIN;

-- Un-seed Gold Copy marketplace roles (we don't know if they existed before).
DELETE FROM bp_roles WHERE role_key IN ('marketplace_publisher','marketplace_admin','marketplace_auditor');

DROP TABLE IF EXISTS marketplace_idempotency;
DROP TABLE IF EXISTS installed_marketplace_artifacts;

-- Restore marketplace_artifacts to pre-extension shape (drop added columns only).
ALTER TABLE marketplace_artifacts
    DROP COLUMN IF EXISTS canonical_sha256,
    DROP COLUMN IF EXISTS min_platform_version,
    DROP COLUMN IF EXISTS artifact_version,
    DROP COLUMN IF EXISTS is_latest;

DROP INDEX IF EXISTS idx_artifacts_listing_version;
DROP INDEX IF EXISTS idx_artifacts_latest;

-- Remove added listing columns (PostgreSQL cannot RESTORE a column to NULL DEFAULT
-- if it already has data, so we just drop the constraints/columns we added).
ALTER TABLE marketplace_listings
    DROP COLUMN IF EXISTS kind,
    DROP COLUMN IF EXISTS billing_period,
    DROP COLUMN IF EXISTS publisher_payout_account,
    DROP COLUMN IF EXISTS publisher_actor_id,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS version_count;

DROP INDEX IF EXISTS idx_listings_kind;
DROP INDEX IF EXISTS idx_listings_status;
DROP INDEX IF EXISTS idx_listings_publisher;
DROP INDEX IF EXISTS idx_listings_installs;
DROP INDEX IF EXISTS idx_listings_browse;

COMMIT;
