ALTER TABLE public.tenant_product_datasource DROP CONSTRAINT IF EXISTS tenant_product_datasource_connection_fk;
ALTER TABLE public.tenant_product_datasource DROP COLUMN IF EXISTS connection_id;
DROP TABLE IF EXISTS public.connections;
