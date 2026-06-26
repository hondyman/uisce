-- Fix catalog_edge schema to match handler expectations
ALTER TABLE catalog_edge ADD COLUMN IF NOT EXISTS properties JSONB DEFAULT '[]'::jsonb;
ALTER TABLE catalog_edge ADD COLUMN IF NOT EXISTS source_node_id UUID;
ALTER TABLE catalog_edge ADD COLUMN IF NOT EXISTS target_node_id UUID;
ALTER TABLE catalog_edge ADD COLUMN IF NOT EXISTS edge_type_id UUID;
ALTER TABLE catalog_edge ADD COLUMN IF NOT EXISTS tenant_datasource_id TEXT;

-- Backfill data for legacy rows (best effort)
UPDATE catalog_edge SET source_node_id = source_id WHERE source_node_id IS NULL AND source_id IS NOT NULL;
UPDATE catalog_edge SET target_node_id = target_id WHERE target_node_id IS NULL AND target_id IS NOT NULL;
UPDATE catalog_edge SET tenant_datasource_id = datasource_id WHERE tenant_datasource_id IS NULL AND datasource_id IS NOT NULL;

-- We cannot easily backfill edge_type_id from edge_type code without complex join, and this is verification only.
-- New edges will have correct IDs.
