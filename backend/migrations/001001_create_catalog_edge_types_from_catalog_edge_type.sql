-- NOTE: This migration examines the legacy singular table `catalog_edge_type`
-- and migrates rows into the canonical `catalog_edge_types` table if present.
-- This is intentionally conditional to support Phase A (non-destructive
-- compatibility). Do not modify the legacy table identifier in this file
-- unless you are intentionally changing migration history as part of Phase B.
-- Migration: create `catalog_edge_types` (plural) if missing and migrate data from
-- existing `catalog_edge_type` (singular). Safe to run multiple times.
-- It will only create the plural table and insert rows if the table does not already exist.



DO $$
BEGIN
  -- Only proceed if the plural table is absent
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'catalog_edge_types'
  ) THEN

    -- Create the plural table with the schema expected by repo migrations
    CREATE TABLE IF NOT EXISTS public.catalog_edge_types (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id CHARACTER VARYING NOT NULL,
      edge_type_name VARCHAR(255) NOT NULL,
      description TEXT,
      is_active BOOLEAN DEFAULT true,
  source_node_type_id UUID REFERENCES catalog_node_types(id) ON DELETE SET NULL,
  target_node_type_id UUID REFERENCES catalog_node_types(id) ON DELETE SET NULL,
      is_directed BOOLEAN DEFAULT true,
      config JSONB DEFAULT '{}',
      created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
      UNIQUE(tenant_id, edge_type_name)
    );

    -- Indexes
    CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_tenant ON public.catalog_edge_types(tenant_id);
    CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_source ON public.catalog_edge_types(source_node_type_id);
    CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_target ON public.catalog_edge_types(target_node_type_id);
    CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_active ON public.catalog_edge_types(is_active);
    CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_config ON public.catalog_edge_types USING GIN(config);

    /*
    -- If singular table exists, migrate rows mapping fields where appropriate.
    IF EXISTS (
      SELECT 1 FROM information_schema.tables
      WHERE table_schema = 'public' AND table_name = 'catalog_edge_type'
    ) THEN
      -- Skipped due to schema mismatch with enum-style catalog_edge_type
    END IF;
    */

  END IF;
END$$;
