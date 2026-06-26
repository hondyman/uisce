-- Migration: Ensure edge_type_id exists in catalog_edge immediately
-- Date: 2026-01-24
-- Purpose: Handlers expect 'edge_type_id' column, but migration 20260126 is future dated.
-- This migration ensures the schema is correct now.

ALTER TABLE catalog_edge ADD COLUMN IF NOT EXISTS properties JSONB DEFAULT '[]'::jsonb;
ALTER TABLE catalog_edge ADD COLUMN IF NOT EXISTS source_node_id UUID;
ALTER TABLE catalog_edge ADD COLUMN IF NOT EXISTS target_node_id UUID;
ALTER TABLE catalog_edge ADD COLUMN IF NOT EXISTS edge_type_id UUID DEFAULT gen_random_uuid();
ALTER TABLE catalog_edge ADD COLUMN IF NOT EXISTS tenant_datasource_id TEXT DEFAULT '';

-- Backfill data for legacy rows if possible, but safely check if columns exist
DO $$
BEGIN
    -- Check for source_id
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'catalog_edge' AND column_name = 'source_id') THEN
        EXECUTE 'UPDATE catalog_edge SET source_node_id = source_id WHERE source_node_id IS NULL AND source_id IS NOT NULL';
    END IF;

    -- Check for target_id
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'catalog_edge' AND column_name = 'target_id') THEN
        EXECUTE 'UPDATE catalog_edge SET target_node_id = target_id WHERE target_node_id IS NULL AND target_id IS NOT NULL';
    END IF;
    
    -- Check for datasource_id
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'catalog_edge' AND column_name = 'datasource_id') THEN
         EXECUTE 'UPDATE catalog_edge SET tenant_datasource_id = datasource_id WHERE tenant_datasource_id IS NULL AND datasource_id IS NOT NULL AND datasource_id != ''''';
    END IF;
END $$;
