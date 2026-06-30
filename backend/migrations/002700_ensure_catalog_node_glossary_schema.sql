-- 002700_ensure_catalog_node_glossary_schema.sql
-- Ensure catalog_node table has all required columns for glossary/business term storage

DO $$
DECLARE
    constraint_exists BOOLEAN;
BEGIN
    -- Check if catalog_node table exists, if not create it
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'catalog_node') THEN
        CREATE TABLE IF NOT EXISTS public.catalog_node (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            node_name TEXT NOT NULL,
            description TEXT,
            node_type_id UUID,
            tenant_id TEXT NOT NULL,
            tenant_datasource_id TEXT,
            properties JSONB DEFAULT '{}'::jsonb,
            qualified_path TEXT NOT NULL,
            parent_type_id UUID,
            config JSONB,
            core_id TEXT,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW()
        );
        CREATE INDEX IF NOT EXISTS idx_catalog_node_tenant ON public.catalog_node(tenant_id);
        CREATE INDEX IF NOT EXISTS idx_catalog_node_qualified_path ON public.catalog_node(qualified_path);
    ELSE
        -- Table exists, drop the problematic unique constraint if it exists
        BEGIN
            ALTER TABLE public.catalog_node DROP CONSTRAINT IF EXISTS catalog_node_unique;
        EXCEPTION WHEN OTHERS THEN
            -- Constraint doesn't exist, that's fine
        END;
        
        -- Add missing columns if they don't exist
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='node_name') THEN
            ALTER TABLE public.catalog_node ADD COLUMN node_name TEXT;
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='node_type_id') THEN
            ALTER TABLE public.catalog_node ADD COLUMN node_type_id UUID;
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='tenant_datasource_id') THEN
            ALTER TABLE public.catalog_node ADD COLUMN tenant_datasource_id TEXT;
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='properties') THEN
            ALTER TABLE public.catalog_node ADD COLUMN properties JSONB DEFAULT '{}'::jsonb;
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='qualified_path') THEN
            ALTER TABLE public.catalog_node ADD COLUMN qualified_path TEXT;
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='parent_type_id') THEN
            ALTER TABLE public.catalog_node ADD COLUMN parent_type_id UUID;
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='config') THEN
            ALTER TABLE public.catalog_node ADD COLUMN config JSONB;
        END IF;
        
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='core_id') THEN
            ALTER TABLE public.catalog_node ADD COLUMN core_id TEXT;
        END IF;
    END IF;
END $$;
