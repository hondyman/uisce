-- Migration: 20260160_api_lifecycle.sql
-- Goal: Add versioning and lifecycle tracking to API endpoints

DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'api_endpoints') THEN
    -- Add lifecycle columns
    ALTER TABLE api_endpoints ADD COLUMN IF NOT EXISTS status text DEFAULT 'active'; -- active, deprecated, retired
    ALTER TABLE api_endpoints ADD COLUMN IF NOT EXISTS semantic_version text DEFAULT 'v1';
    ALTER TABLE api_endpoints ADD COLUMN IF NOT EXISTS previous_version_id uuid REFERENCES api_endpoints(id);
    ALTER TABLE api_endpoints ADD COLUMN IF NOT EXISTS owner_team text;
    ALTER TABLE api_endpoints ADD COLUMN IF NOT EXISTS deprecated_at timestamp with time zone;
    ALTER TABLE api_endpoints ADD COLUMN IF NOT EXISTS retired_at timestamp with time zone;

    -- Index for status-based filtering (active endpoints)
    CREATE INDEX IF NOT EXISTS idx_api_endpoints_status ON api_endpoints(status);

    -- Optional: Track request/response schema objects if not already handled by 'fields'
    ALTER TABLE api_endpoints ADD COLUMN IF NOT EXISTS request_schema_id text;
    ALTER TABLE api_endpoints ADD COLUMN IF NOT EXISTS response_schema_id text;
  ELSE
    RAISE NOTICE 'api_endpoints table not found; skipping API lifecycle changes';
  END IF;
END
$do$;