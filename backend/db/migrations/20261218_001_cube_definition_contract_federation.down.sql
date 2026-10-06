-- 20261218_001_cube_definition_contract_federation (down)

DROP INDEX IF EXISTS data_explorer.idx_cube_definition_tenant_name_version;

ALTER TABLE data_explorer.cube_definition
    DROP CONSTRAINT IF EXISTS cube_definition_contract_version_check;

ALTER TABLE data_explorer.cube_definition
    DROP COLUMN IF EXISTS federation;

ALTER TABLE data_explorer.cube_definition
    DROP COLUMN IF EXISTS contract_version;
