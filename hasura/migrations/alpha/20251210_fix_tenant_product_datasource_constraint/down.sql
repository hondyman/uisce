-- Revert to the old constraint (for rollback purposes)
ALTER TABLE public.tenant_product_datasource 
DROP CONSTRAINT IF EXISTS tenant_product_datasource_source_uniq;

ALTER TABLE public.tenant_product_datasource 
ADD CONSTRAINT tenant_product_datasource_uniq 
UNIQUE (tenant_product_id, alpha_datasource_id);
