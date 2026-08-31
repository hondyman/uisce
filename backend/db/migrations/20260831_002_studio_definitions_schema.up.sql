-- Page Studio previously had no backend persistence at all: the frontend
-- client (frontend/src/api/pageStudio.ts) called /api/page-studio/pages
-- with no matching table or handler. This adds the minimal tenant-scoped
-- storage for it, modeled on bo_crud_handler.go's tenant_id-scoped write
-- pattern.
--
-- API Studio endpoint definitions are NOT added here: backend/internal/apistudio
-- already has a fully-built (but previously unwired) Repository/Service
-- targeting the existing semantic.api_endpoints table
-- (see migrations/20260156_api_studio.sql, 20260158_fix_api_studio_schema.sql,
-- 20260160_api_lifecycle.sql) — apistudio_handler.go wires that in instead
-- of introducing a second, competing table.

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

-- Endpoint definition saves are distinct from api_request/api_response
-- (which fire on actual runtime traffic to an already-saved endpoint).
INSERT INTO trigger_types (id, key, label, description, category)
VALUES
    (gen_random_uuid(), 'api_endpoint_save', 'API Endpoint Saved', 'Fires after an API Studio endpoint definition is created or updated.', 'api')
ON CONFLICT (key) DO NOTHING;
