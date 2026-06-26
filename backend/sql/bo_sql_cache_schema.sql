-- dedicated table for caching resolved BO SQL
CREATE SCHEMA IF NOT EXISTS semantic;

CREATE TABLE IF NOT EXISTS semantic.bo_sql_cache (
    business_object_id uuid NOT NULL,
    dialect text NOT NULL,
    version_hash text NOT NULL,
    sql text NOT NULL,
    created_at timestamptz DEFAULT now(),
    PRIMARY KEY (business_object_id, dialect)
);

-- Index for faster invalidation by BO ID
CREATE INDEX IF NOT EXISTS idx_bo_sql_cache_business_object_id ON semantic.bo_sql_cache(business_object_id);

-- Index for lookup by version hash (e.g. for content-addressable optimizations or debugging)
CREATE INDEX IF NOT EXISTS idx_bo_sql_cache_version_hash ON semantic.bo_sql_cache(version_hash);
