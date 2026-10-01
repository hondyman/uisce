-- 20261130_012_metric_definition_enhancements.up.sql
ALTER TABLE data_explorer.metric_definition
    ADD COLUMN IF NOT EXISTS catalog_term_id UUID,
    ADD COLUMN IF NOT EXISTS materialization_config JSONB NOT NULL DEFAULT '{"strategy": "on_the_fly"}'::jsonb,
    ADD COLUMN IF NOT EXISTS variables JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS decomposable BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS content_hash VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_metric_definition_catalog_term 
    ON data_explorer.metric_definition (tenant_id, catalog_term_id);

CREATE INDEX IF NOT EXISTS idx_metric_definition_content_hash 
    ON data_explorer.metric_definition (tenant_id, content_hash);
