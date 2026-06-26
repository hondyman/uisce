-- Migration: 20260157_page_studio.sql
-- Goal: Support Page Studio with core/overlay inheritance and visibility

CREATE SCHEMA IF NOT EXISTS semantic;

-- Core pages (Gold Copy)
CREATE TABLE IF NOT EXISTS semantic.pages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    env text NOT NULL, -- production | sandbox
    tenant_id text, -- NULL for core
    name text NOT NULL,
    slug text NOT NULL UNIQUE,
    description text,
    layout jsonb NOT NULL, -- layout tree (rows/cols/tabs)
    components jsonb NOT NULL, -- component definitions
    data_bindings jsonb NOT NULL, -- bindings to active APIs
    visibility jsonb NOT NULL, -- roles/entitlements
    version int NOT NULL DEFAULT 1,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    created_by text
);

-- Tenant Overlays
CREATE TABLE IF NOT EXISTS semantic.page_overlays (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id uuid NOT NULL REFERENCES semantic.pages(id),
    env text NOT NULL,
    tenant_id text NOT NULL,
    overrides jsonb NOT NULL, -- delta-based overrides
    version int NOT NULL DEFAULT 1,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    created_by text,
    UNIQUE(parent_id, tenant_id, env)
);

-- SLOs for Pages
-- We use the existing semantic_slos but add page scope support
-- No new table needed, but we ensure indices exist
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'semantic' AND table_name = 'semantic_slos') THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_semantic_slos_page ON semantic.semantic_slos(scope_id) WHERE scope_type = ''page''';
  ELSIF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'semantic_slos') THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_semantic_slos_page ON public.semantic_slos(scope_id) WHERE scope_type = ''page''';
  ELSE
    RAISE NOTICE 'semantic_slos table not found in semantic or public schemas; skipping index creation';
  END IF;
END
$do$;
