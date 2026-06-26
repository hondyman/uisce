-- Migration: add missing foreign keys expected by Hasura
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_fabric_defn_tenant'
  ) THEN
    ALTER TABLE public.fabric_defn
      ADD CONSTRAINT fk_fabric_defn_tenant
      FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_fabric_defn_tenant_datasource'
  ) THEN
    ALTER TABLE public.fabric_defn
      ADD CONSTRAINT fk_fabric_defn_tenant_datasource
      FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE;
  END IF;
END$$;
