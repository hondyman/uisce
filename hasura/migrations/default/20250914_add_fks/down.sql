-- Migration rollback: remove foreign keys added in 20250914_add_fks
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_fabric_defn_tenant'
  ) THEN
    ALTER TABLE public.fabric_defn
      DROP CONSTRAINT fk_fabric_defn_tenant;
  END IF;

  IF EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_fabric_defn_tenant_datasource'
  ) THEN
    ALTER TABLE public.fabric_defn
      DROP CONSTRAINT fk_fabric_defn_tenant_datasource;
  END IF;
END$$;
