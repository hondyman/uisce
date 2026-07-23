-- Creates the business_entity table for a structured representation of entities and their subtypes.
-- This replaces the single JSON blob storage in the old entity_schema table.
CREATE TABLE public.business_entity (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    tenant_datasource_id uuid NOT NULL,
    parent_id uuid NULL,
    catalog_node_id uuid NULL,
    entity_key text NOT NULL,
    name text NOT NULL,
    is_core boolean DEFAULT false NOT NULL,
    business_name text NULL,
    technical_name text NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT business_entity_pk PRIMARY KEY (id),
    CONSTRAINT business_entity_parent_fk FOREIGN KEY (parent_id) REFERENCES public.business_entity(id) ON DELETE CASCADE,
    CONSTRAINT business_entity_catalog_node_fk FOREIGN KEY (catalog_node_id) REFERENCES public.catalog_node(id) ON DELETE SET NULL,
    CONSTRAINT business_entity_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
    CONSTRAINT business_entity_tenant_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE
);

CREATE INDEX business_entity_tenant_datasource_idx ON public.business_entity USING btree (tenant_id, tenant_datasource_id);
CREATE INDEX business_entity_parent_id_idx ON public.business_entity USING btree (parent_id);

-- It is recommended to run a script to migrate existing data from entity_schema to business_entity after this migration.
