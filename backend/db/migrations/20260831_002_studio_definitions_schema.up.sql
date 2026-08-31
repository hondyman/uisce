-- Page Studio and API Studio previously had no backend persistence at all:
-- the frontend clients (frontend/src/api/pageStudio.ts, apiStudio.ts) called
-- /api/page-studio/pages and /api/api-studio/endpoints with no matching
-- table or handler. This adds the minimal tenant-scoped storage for both,
-- modeled on bo_crud_handler.go's tenant_id-scoped write pattern.

CREATE TABLE IF NOT EXISTS page_definitions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    name          TEXT NOT NULL,
    slug          TEXT NOT NULL,
    description   TEXT,
    layout        JSONB NOT NULL DEFAULT '[]'::jsonb,
    components    JSONB NOT NULL DEFAULT '[]'::jsonb,
    data_sources  JSONB NOT NULL DEFAULT '[]'::jsonb,
    version       INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, slug)
);

CREATE TABLE IF NOT EXISTS api_definitions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    env                  TEXT NOT NULL DEFAULT 'default',
    name                 TEXT NOT NULL,
    path                 TEXT NOT NULL,
    method               TEXT NOT NULL DEFAULT 'GET',
    type                 TEXT NOT NULL DEFAULT 'rest',
    bo_name              TEXT,
    fields               JSONB NOT NULL DEFAULT '[]'::jsonb,
    filters              JSONB NOT NULL DEFAULT '{}'::jsonb,
    pagination           JSONB NOT NULL DEFAULT '{"type":"offset","default_limit":50}'::jsonb,
    auth_policy          TEXT,
    version              INTEGER NOT NULL DEFAULT 1,
    is_active            BOOLEAN NOT NULL DEFAULT true,
    status               TEXT NOT NULL DEFAULT 'active',
    semantic_version     TEXT NOT NULL DEFAULT 'v1',
    previous_version_id  UUID,
    owner_team           TEXT,
    deprecated_at        TIMESTAMPTZ,
    retired_at           TIMESTAMPTZ,
    request_schema_id    UUID,
    response_schema_id   UUID,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, env, path, method)
);

-- Endpoint definition saves are distinct from api_request/api_response
-- (which fire on actual runtime traffic to an already-saved endpoint).
INSERT INTO trigger_types (id, key, label, description, category)
VALUES
    (gen_random_uuid(), 'api_endpoint_save', 'API Endpoint Saved', 'Fires after an API Studio endpoint definition is created or updated.', 'api')
ON CONFLICT (key) DO NOTHING;
