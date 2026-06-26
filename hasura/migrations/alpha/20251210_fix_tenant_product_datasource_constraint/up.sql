-- Drop the overly restrictive constraint that prevents multiple datasources of the same type
ALTER TABLE public.tenant_product_datasource 
DROP CONSTRAINT IF EXISTS tenant_product_datasource_uniq;

-- Add a more appropriate constraint: unique source_name per tenant_product
ALTER TABLE public.tenant_product_datasource 
ADD CONSTRAINT tenant_product_datasource_source_uniq 
UNIQUE (tenant_product_id, source_name);
