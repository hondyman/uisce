-- Drop the old entity_schema table (single JSON blob per datasource)
-- and replace it with a robust entity_attribute table that stores
-- each entity as its own row with proper relationships to semantic terms

-- Backup old schema data if it exists (optional, for data migration)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='entity_schema') THEN
        CREATE TABLE IF NOT EXISTS public.entity_schema_backup AS
        SELECT * FROM public.entity_schema;
    END IF;
END $$;

-- Drop old table if exists
DROP TABLE IF EXISTS public.entity_schema CASCADE;

-- Create new robust entity_attribute table
CREATE TABLE IF NOT EXISTS public.entity_attribute (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    tenant_datasource_id uuid NOT NULL,
    parent_id uuid,
    -- Link to the semantic term (catalog_node) - NOT a string name that can change
    catalog_node_id uuid,
    entity_key text NOT NULL,
    name text NOT NULL,
    is_core boolean DEFAULT false NOT NULL,
    business_name text,
    technical_name text,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT entity_attribute_pk PRIMARY KEY (id),
    CONSTRAINT entity_attribute_parent_fk FOREIGN KEY (parent_id) 
        REFERENCES public.entity_attribute(id) 
        ON DELETE CASCADE 
        ON UPDATE CASCADE 
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT entity_attribute_catalog_node_fk FOREIGN KEY (catalog_node_id) 
        REFERENCES public.catalog_node(id) 
        ON DELETE SET NULL 
        ON UPDATE CASCADE 
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT entity_attribute_tenant_fk FOREIGN KEY (tenant_id) 
        REFERENCES public.tenants(id) 
        ON DELETE CASCADE 
        ON UPDATE CASCADE 
        DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT entity_attribute_tenant_datasource_fk FOREIGN KEY (tenant_datasource_id) 
        REFERENCES public.tenant_product_datasource(id) 
        ON DELETE CASCADE 
        ON UPDATE CASCADE 
        DEFERRABLE INITIALLY DEFERRED,
    -- Ensure entity_key uniqueness within a datasource
    CONSTRAINT entity_attribute_key_datasource_unique UNIQUE (tenant_datasource_id, entity_key),
    -- Prevent self-parent-reference
    CONSTRAINT entity_attribute_no_self_parent CHECK (id != parent_id)
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS entity_attribute_tenant_datasource_idx 
    ON public.entity_attribute USING btree (tenant_id, tenant_datasource_id);

CREATE INDEX IF NOT EXISTS entity_attribute_parent_id_idx 
    ON public.entity_attribute USING btree (parent_id);

CREATE INDEX IF NOT EXISTS entity_attribute_catalog_node_id_idx 
    ON public.entity_attribute USING btree (catalog_node_id);

CREATE INDEX IF NOT EXISTS entity_attribute_entity_key_idx 
    ON public.entity_attribute USING btree (tenant_datasource_id, entity_key);

-- Create a view for backward compatibility (if needed)
CREATE OR REPLACE VIEW public.entity_attribute_hierarchy AS
    SELECT 
        ea.id,
        ea.tenant_id,
        ea.tenant_datasource_id,
        ea.parent_id,
        ea.catalog_node_id,
        ea.entity_key,
        ea.name,
        ea.is_core,
        ea.business_name,
        ea.technical_name,
        CASE WHEN ea.parent_id IS NULL THEN 'root' ELSE 'child' END AS hierarchy_level,
        (SELECT COUNT(*) FROM public.entity_attribute WHERE parent_id = ea.id) AS child_count,
        ea.created_at,
        ea.updated_at
    FROM public.entity_attribute ea;

COMMENT ON TABLE public.entity_attribute IS 
    'Robust entity storage with one row per entity, proper parent-child relationships, and semantic term (catalog_node) linking';

COMMENT ON COLUMN public.entity_attribute.catalog_node_id IS 
    'Foreign key to catalog_node table - ensures entities are linked to semantic definitions that cannot change';

COMMENT ON COLUMN public.entity_attribute.parent_id IS 
    'Self-referencing foreign key for entity hierarchy (subtypes linked to parent entities)';
