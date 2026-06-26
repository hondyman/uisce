-- 20251215_001_adapt_catalog.sql
-- Adapt catalog tables for Metadata-First CQRS Architecture
-- Adds 'properties' column to catalog_edge to store validation rules, join conditions, etc.

DO $$
BEGIN
    -- Add properties column to catalog_edge if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_edge' AND column_name='properties') THEN
        ALTER TABLE public.catalog_edge ADD COLUMN properties JSONB DEFAULT '{}'::jsonb;
    END IF;

    -- Ensure we have indexes for the new JSONB column to support performant querying if needed later
    -- (GIN index on jsonb)
    CREATE INDEX IF NOT EXISTS idx_catalog_edge_properties ON public.catalog_edge USING GIN (properties);

END $$;
