-- Migration: 20260155_crs_schema.sql
-- Description: Schema for Change Review System (Lineage, Versioning, ChangeSets, Tests)

-- 1. Lineage Graph Tables
CREATE TABLE IF NOT EXISTS semantic.lineage_nodes (
    id TEXT PRIMARY KEY,          -- e.g. 'bo:Positions:prod'
    type TEXT NOT NULL,           -- bo | bo_field | preagg | table | column | entitlement | aso_opt | tenant | changeset
    env TEXT NOT NULL,
    tenant_id TEXT,
    name TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS semantic.lineage_edges (
    from_id TEXT NOT NULL REFERENCES semantic.lineage_nodes(id) ON DELETE CASCADE,
    to_id TEXT NOT NULL REFERENCES semantic.lineage_nodes(id) ON DELETE CASCADE,
    type TEXT NOT NULL,           -- depends_on | derived_from | governed_by | optimized_by | belongs_to | overrides | included_in
    env TEXT NOT NULL,
    tenant_id TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (from_id, to_id, type)
);

CREATE INDEX IF NOT EXISTS idx_lineage_edges_from ON semantic.lineage_edges(from_id);
CREATE INDEX IF NOT EXISTS idx_lineage_edges_to ON semantic.lineage_edges(to_id);

-- 2. Semantic Versioning Tables
CREATE TABLE IF NOT EXISTS semantic.objects (
    id TEXT,              -- logical id, e.g. 'bo:Positions'
    version INT,
    env TEXT,
    tenant_id TEXT,
    type TEXT,            -- bo | preagg | entitlement | policy | aso_policy
    payload JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    created_by TEXT,
    PRIMARY KEY (id, version)
);

CREATE TABLE IF NOT EXISTS semantic.heads (
    id TEXT PRIMARY KEY,
    env TEXT,
    tenant_id TEXT,
    type TEXT,
    current_version INT
);

-- 3. ChangeSets + Items
CREATE TABLE IF NOT EXISTS semantic.change_sets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env TEXT NOT NULL,
    tenant_id TEXT,
    author TEXT NOT NULL,
    status TEXT NOT NULL, -- draft | in_review | approved | rejected | promoted
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS semantic.change_set_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    change_set_id UUID NOT NULL REFERENCES semantic.change_sets(id) ON DELETE CASCADE,
    object_id TEXT NOT NULL,   -- e.g. 'bo:Positions'
    object_type TEXT NOT NULL, -- bo | preagg | entitlement | calc
    old_version INT,
    new_version INT,
    payload JSONB NOT NULL
);

-- 4. Semantic Tests
CREATE TABLE IF NOT EXISTS semantic.tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env TEXT,
    tenant_id TEXT,
    scope_type TEXT,      -- bo | preagg | entitlement | calc
    scope_id TEXT,        -- e.g. 'bo:Positions'
    name TEXT,
    type TEXT,            -- contract | entitlement | regression | calc
    definition JSONB,     -- test-specific config
    created_at TIMESTAMPTZ DEFAULT now(),
    created_by TEXT,
    enabled BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS semantic.test_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_id UUID REFERENCES semantic.tests(id) ON DELETE CASCADE,
    env TEXT,
    tenant_id TEXT,
    status TEXT,          -- pending | running | passed | failed
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    result JSONB
);

-- 5. Change Review Artifacts
CREATE TABLE IF NOT EXISTS semantic.change_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    change_set_id UUID NOT NULL REFERENCES semantic.change_sets(id) ON DELETE CASCADE,
    lineage_impact JSONB,
    semantic_diff JSONB,
    test_results JSONB,
    aso_impact JSONB,
    reviewer TEXT,
    status TEXT NOT NULL, -- pending | approved | rejected
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_objects_id_version ON semantic.objects(id, version);
CREATE INDEX IF NOT EXISTS idx_change_set_items_cs_id ON semantic.change_set_items(change_set_id);
CREATE INDEX IF NOT EXISTS idx_tests_scope ON semantic.tests(scope_type, scope_id);
