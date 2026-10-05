-- 20261218_001_cube_definition_contract_federation
--
-- CUBE-0.1: contract versioning + declared federation plan on cube_definition.
--
-- contract_version bumps when the published surface breaks (grain change,
-- dimension/metric removal). Consumers pin cube_id + contract_version.
--
-- federation is the declared multi-BO join plan (sources + keys). Transform
-- keys reference deterministic terms/calcs (transform_term_id), never
-- metric_definition aggregates. Empty object = single-BO cube.
--
-- See docs plan: end-to-end cube tables (StarRocks-native federation).

ALTER TABLE data_explorer.cube_definition
    ADD COLUMN IF NOT EXISTS contract_version INTEGER NOT NULL DEFAULT 1;

ALTER TABLE data_explorer.cube_definition
    ADD COLUMN IF NOT EXISTS federation JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE data_explorer.cube_definition
    DROP CONSTRAINT IF EXISTS cube_definition_contract_version_check;
ALTER TABLE data_explorer.cube_definition
    ADD CONSTRAINT cube_definition_contract_version_check
        CHECK (contract_version >= 1);

COMMENT ON COLUMN data_explorer.cube_definition.contract_version IS
    'Published contract generation. Bump on breaking surface changes; consumers pin this with cube_id.';
COMMENT ON COLUMN data_explorer.cube_definition.federation IS
    'Declared federation: {sources[], joins[] with left_term_ids/right_term_ids/transform_term_id}. Empty = single-BO.';

-- Lookup by tenant + name + version (adopted consumers / versioned publish).
CREATE INDEX IF NOT EXISTS idx_cube_definition_tenant_name_version
    ON data_explorer.cube_definition (tenant_id, name, contract_version)
    WHERE status = 'active' AND archived_at IS NULL;
