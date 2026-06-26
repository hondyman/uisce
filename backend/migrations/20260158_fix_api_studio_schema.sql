-- Ensure schema exists
CREATE SCHEMA IF NOT EXISTS semantic;

-- Ensure table exists with all required columns (for fresh install or if 20260156 failed completely)
CREATE TABLE IF NOT EXISTS semantic.api_endpoints (
    id uuid PRIMARY KEY,
    env text NOT NULL,
    tenant_id text NOT NULL,
    name text NOT NULL,
    path text NOT NULL,
    method text NOT NULL DEFAULT 'GET',
    type text NOT NULL DEFAULT 'rest',
    bo_name text NOT NULL,
    fields jsonb NOT NULL,
    filters jsonb NOT NULL DEFAULT '[]'::jsonb,
    pagination jsonb NOT NULL DEFAULT '{"type": "offset", "default_limit": 100}'::jsonb,
    auth_policy text,
    version int NOT NULL DEFAULT 1,
    status text NOT NULL DEFAULT 'active',
    semantic_version text NOT NULL DEFAULT '1.0.0',
    previous_version_id uuid,
    owner_team text,
    deprecated_at timestamptz,
    retired_at timestamptz,
    request_schema_id text,
    response_schema_id text,
    created_at timestamptz DEFAULT now(),
    created_by text,
    UNIQUE(env, tenant_id, path, method)
);

-- Idempotent column additions (for upgrades where table exists but lacks columns)
ALTER TABLE semantic.api_endpoints ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'active';
ALTER TABLE semantic.api_endpoints ADD COLUMN IF NOT EXISTS semantic_version text NOT NULL DEFAULT '1.0.0';
ALTER TABLE semantic.api_endpoints ADD COLUMN IF NOT EXISTS previous_version_id uuid;
ALTER TABLE semantic.api_endpoints ADD COLUMN IF NOT EXISTS owner_team text;
ALTER TABLE semantic.api_endpoints ADD COLUMN IF NOT EXISTS deprecated_at timestamptz;
ALTER TABLE semantic.api_endpoints ADD COLUMN IF NOT EXISTS retired_at timestamptz;
ALTER TABLE semantic.api_endpoints ADD COLUMN IF NOT EXISTS request_schema_id text;
ALTER TABLE semantic.api_endpoints ADD COLUMN IF NOT EXISTS response_schema_id text;
