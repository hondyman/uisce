ALTER TABLE public.tenant_product_datasource ADD COLUMN IF NOT EXISTS last_scan_at timestamptz NULL;
ALTER TABLE public.tenant_product_datasource ADD COLUMN IF NOT EXISTS last_scan_status varchar(50) NULL;
-- Ensure connection_id exists as well
ALTER TABLE public.tenant_product_datasource ADD COLUMN IF NOT EXISTS connection_id uuid NULL;
