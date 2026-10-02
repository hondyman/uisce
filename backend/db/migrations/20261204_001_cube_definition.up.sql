-- 20261204_001_cube_definition
--
-- A cube is a published aggregation contract: a Business Object, an ordered
-- dimension surface, a governed metric set, and a declared set of physical
-- materialization grains. It sits beside data_explorer.metric_definition
-- rather than inside the BO, so many cubes can hang off one BO and the BO
-- stays clean. See docs/ARCHITECTURAL_DECISIONS.md ADR-011.
--
-- The load-bearing constraint is metric_ids: a cube references governed metrics
-- BY ID and never carries an ad-hoc SUM(col) of its own. That is what forces
-- cubes onto the metric layer, and it is why they inherit AST safety,
-- content-hash stability, and the `decomposable` flag for free.
--
-- The cube stores zero physical information: bo_id is a logical reference, not
-- a table or connection. That is what keeps cube definitions bundle-portable
-- and gold-copy friendly. Physical objects (mv_gold_* / mv_{tenant}_*) are
-- derived at deploy time and resolved by the router at query time.
CREATE TABLE IF NOT EXISTS data_explorer.cube_definition (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    -- Primary business object. Joins to other BOs use the existing relationship
    -- machinery (relatedBoIds), not raw join SQL.
    bo_id           VARCHAR(255) NOT NULL,

    -- Ordered dimension surface = the cube's axes. Each entry is
    -- {termNodeId, drillPath?}. Order is meaningful: it is the axis order the
    -- cube exposes.
    dimensions      JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- {termNodeId, defaultGrain}. Optional; a cube may be purely categorical.
    time_dimension  JSONB,

    -- Governed metric IDs ONLY. Never raw columns. JSONB array of strings.
    metric_ids      JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- The materialization plan: array of arrays, e.g.
    -- [["country","product","day"], ["country","day"]]. Each inner array is one
    -- physical grain. This is a plan, not a description: it is what the deploy
    -- step materializes.
    grains          JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- {strategy, refreshStrategy, refreshIntervalMinutes, partitionGrain,
    --  retentionDays, stalePolicy}. stalePolicy is "serve_with_flag"
    -- (default) or "force_raw_fallback" (compliance).
    materialization JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- SHA-256 over the canonical semantic content. Unchanged hash => deploy is
    -- a no-op, which is what makes redeploys idempotent and safe against
    -- concurrent applies.
    content_hash    VARCHAR(64),

    -- Core (gold copy) cubes are authored by the master tenant and adopted by
    -- client tenants via public.core_object_adoption with
    -- object_type = 'cube'.
    is_core         BOOLEAN NOT NULL DEFAULT false,
    status          VARCHAR(50) NOT NULL DEFAULT 'active',
    archived_at     TIMESTAMPTZ,
    created_by      VARCHAR(255),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT cube_definition_status_check
        CHECK (status IN ('active', 'deprecated', 'archived')),
    CONSTRAINT cube_definition_archived_check
        CHECK (status <> 'archived' OR archived_at IS NOT NULL),
    CONSTRAINT cube_definition_tenant_name_uniq UNIQUE (tenant_id, name)
);

-- Primary lookup: resolve a cube for a tenant/BO at query time.
CREATE INDEX IF NOT EXISTS idx_cube_definition_tenant_bo
    ON data_explorer.cube_definition (tenant_id, bo_id);

-- Same, restricted to cubes that are still usable. Mirrors the equivalent
-- partial index on metric_definition.
CREATE INDEX IF NOT EXISTS idx_cube_definition_active
    ON data_explorer.cube_definition (tenant_id, bo_id)
    WHERE status = 'active' AND archived_at IS NULL;

-- Deploy idempotency and gold-copy comparison.
CREATE INDEX IF NOT EXISTS idx_cube_definition_content_hash
    ON data_explorer.cube_definition (tenant_id, content_hash);

-- RLS: a cube is tenant-owned, but a core (gold) cube must be readable by the
-- tenant that adopts it. This mirrors page_fragments_tenant_gold_policy:
-- reads see own-tenant plus core rows, writes only ever own-tenant. Without the
-- core branch, an adopting tenant could not read the cube it adopted.
ALTER TABLE data_explorer.cube_definition ENABLE ROW LEVEL SECURITY;
ALTER TABLE data_explorer.cube_definition FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS cube_definition_tenant_policy ON data_explorer.cube_definition;
CREATE POLICY cube_definition_tenant_policy ON data_explorer.cube_definition
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
        OR (is_core = true AND tenant_id = uisce_get_gold_tenant())
    )
    WITH CHECK (tenant_id = uisce_get_current_tenant());

COMMENT ON TABLE data_explorer.cube_definition IS
    'Published aggregation contract: BO + dimension surface + governed metric IDs + materialization grains. See docs/ARCHITECTURAL_DECISIONS.md ADR-011.';
COMMENT ON COLUMN data_explorer.cube_definition.metric_ids IS
    'Governed metric_definition IDs only. Never raw columns; this constraint is what keeps cubes on the metric layer.';
COMMENT ON COLUMN data_explorer.cube_definition.grains IS
    'Declared materialization grains (array of arrays). The deploy step materializes exactly these.';
