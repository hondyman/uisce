-- 002001_add_is_active_to_catalog.sql
-- Adds is_active flag to catalog_node and catalog_edge for gold-copy edge inheritance.
-- When a tenant edge has core_id pointing to a gold edge, it overrides the gold edge.
-- Edges that disappear from a scan are deactivated (is_active = false), not deleted.

DO $$
BEGIN
    -- Add is_active to catalog_node if not exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='is_active') THEN
        ALTER TABLE public.catalog_node ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;
    END IF;

    -- Add is_active to catalog_edge if not exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_edge' AND column_name='is_active') THEN
        ALTER TABLE public.catalog_edge ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true;
    END IF;

    -- Add is_active to each catalog_edge partition if not exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'catalog_edge_2026q1') THEN
        ALTER TABLE public.catalog_edge_2026q1 ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'catalog_edge_2026q2') THEN
        ALTER TABLE public.catalog_edge_2026q2 ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'catalog_edge_future') THEN
        ALTER TABLE public.catalog_edge_future ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;
    END IF;

    -- Backfill existing rows to true
    UPDATE public.catalog_node SET is_active = true WHERE is_active IS NULL;
    UPDATE public.catalog_edge SET is_active = true WHERE is_active IS NULL;
    UPDATE public.catalog_edge_2026q1 SET is_active = true WHERE is_active IS NULL;
    UPDATE public.catalog_edge_2026q2 SET is_active = true WHERE is_active IS NULL;
    UPDATE public.catalog_edge_future SET is_active = true WHERE is_active IS NULL;

    -- Create index for efficient active-edge queries
    CREATE INDEX IF NOT EXISTS idx_catalog_edge_active ON public.catalog_edge (tenant_datasource_id, is_active) WHERE is_active = true;

END $$;
