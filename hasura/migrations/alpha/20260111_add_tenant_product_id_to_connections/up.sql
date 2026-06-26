ALTER TABLE public.connections ADD COLUMN IF NOT EXISTS tenant_product_id uuid;

DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'connections_tenant_product_id_fkey') THEN
        ALTER TABLE public.connections ADD CONSTRAINT connections_tenant_product_id_fkey FOREIGN KEY (tenant_product_id) REFERENCES public.tenant_product(id) ON UPDATE RESTRICT ON DELETE SET NULL;
    END IF;
END $$;
