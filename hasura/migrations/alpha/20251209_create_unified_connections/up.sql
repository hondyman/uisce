CREATE TABLE IF NOT EXISTS public.connections (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    name varchar(255) NOT NULL,
    type varchar(50) NOT NULL,
    host varchar(255),
    port integer,
    database varchar(255),
    schema varchar(255),
    username varchar(255),
    password varchar(255),
    base_url varchar(255),
    api_key varchar(255),
    metadata jsonb DEFAULT '{}'::jsonb,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT connections_pkey PRIMARY KEY (id),
    CONSTRAINT connections_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TRIGGER update_connections_updated_at BEFORE UPDATE ON public.connections FOR EACH ROW EXECUTE FUNCTION public.update_timestamp();

ALTER TABLE public.tenant_product_datasource ADD COLUMN IF NOT EXISTS connection_id uuid;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tenant_product_datasource_connection_fk') THEN
        ALTER TABLE public.tenant_product_datasource 
        ADD CONSTRAINT tenant_product_datasource_connection_fk 
        FOREIGN KEY (connection_id) REFERENCES public.connections(id) 
        ON UPDATE RESTRICT ON DELETE RESTRICT;
    END IF;
END $$;
