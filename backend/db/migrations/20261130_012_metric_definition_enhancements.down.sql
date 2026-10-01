-- 20261130_012_metric_definition_enhancements.down.sql
DROP INDEX IF EXISTS data_explorer.idx_metric_definition_content_hash;
DROP INDEX IF EXISTS data_explorer.idx_metric_definition_catalog_term;

ALTER TABLE data_explorer.metric_definition
    DROP COLUMN IF EXISTS content_hash,
    DROP COLUMN IF EXISTS decomposable,
    DROP COLUMN IF EXISTS variables,
    DROP COLUMN IF EXISTS materialization_config,
    DROP COLUMN IF EXISTS catalog_term_id;
