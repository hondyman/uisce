-- Create API Endpoint table
CREATE TABLE IF NOT EXISTS semantic.api_endpoints (
    id uuid PRIMARY KEY,
    env text NOT NULL,
    tenant_id text NOT NULL,
    name text NOT NULL,                 -- e.g. "PositionsAPI"
    path text NOT NULL,                 -- e.g. "/positions"
    method text NOT NULL DEFAULT 'GET', -- GET | POST | PUT | DELETE
    type text NOT NULL DEFAULT 'rest',   -- rest | graphql
    bo_name text NOT NULL,              -- backing Business Object
    fields jsonb NOT NULL,              -- array of strings: which BO fields are exposed
    filters jsonb NOT NULL DEFAULT '[]'::jsonb, -- configuration of allowed filters
    pagination jsonb NOT NULL DEFAULT '{"type": "offset", "default_limit": 100}'::jsonb,
    auth_policy text,                   -- reference to entitlement / auth policy
    version int NOT NULL DEFAULT 1,
    status text NOT NULL DEFAULT 'active', -- active | deprecated | retired
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

-- Create API Catalog table
CREATE TABLE IF NOT EXISTS semantic.api_catalogs (
    id uuid PRIMARY KEY,
    env text NOT NULL,
    tenant_id text NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamptz DEFAULT now(),
    created_by text
);

-- Associate Endpoints with Catalogs
CREATE TABLE IF NOT EXISTS semantic.api_catalog_entries (
    id uuid PRIMARY KEY,
    catalog_id uuid REFERENCES semantic.api_catalogs(id) ON DELETE CASCADE,
    endpoint_id uuid REFERENCES semantic.api_endpoints(id) ON DELETE CASCADE,
    path_override text,
    auth_policy_override text,
    rate_limit jsonb NOT NULL DEFAULT '{"qps": 10, "burst": 20}'::jsonb,
    enabled boolean DEFAULT true,
    UNIQUE(catalog_id, endpoint_id)
);

-- API Specific Tests
CREATE TABLE IF NOT EXISTS semantic.api_tests (
    id uuid PRIMARY KEY,
    env text NOT NULL,
    tenant_id text NOT NULL,
    endpoint_id uuid REFERENCES semantic.api_endpoints(id) ON DELETE CASCADE,
    name text NOT NULL,
    type text NOT NULL,          -- contract | latency | pii | regression
    definition jsonb NOT NULL,
    created_at timestamptz DEFAULT now(),
    created_by text,
    enabled boolean DEFAULT true
);

-- Test Results for API tests
CREATE TABLE IF NOT EXISTS semantic.api_test_runs (
    id uuid PRIMARY KEY,
    api_test_id uuid REFERENCES semantic.api_tests(id) ON DELETE CASCADE,
    env text NOT NULL,
    tenant_id text NOT NULL,
    status text NOT NULL,        -- pending | running | passed | failed
    started_at timestamptz,
    finished_at timestamptz,
    result jsonb NOT NULL,
    logs text[]
);
