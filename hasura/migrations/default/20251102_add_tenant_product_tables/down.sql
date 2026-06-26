-- Rollback: Remove tenant/product/datasource tables

DROP TABLE IF EXISTS public.tenant_product_datasource CASCADE;
DROP TABLE IF EXISTS public.tenant_product CASCADE;
DROP TABLE IF EXISTS public.tenant_instance CASCADE;
DROP TABLE IF EXISTS public.product CASCADE;
DROP TABLE IF EXISTS public.alpha_product CASCADE;
DROP TABLE IF EXISTS public.alpha_datasource CASCADE;
