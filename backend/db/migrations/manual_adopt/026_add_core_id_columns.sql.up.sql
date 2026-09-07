-- Migration: Add core_id columns for Gold Copy cloning
-- This allows non-gold-copy instances to reference their "parent" gold copy items

-- Add core_id to tenant_instance
ALTER TABLE public.tenant_instance 
ADD COLUMN IF NOT EXISTS core_id UUID REFERENCES public.tenant_instance(id) ON DELETE SET NULL;

-- Add core_id to tenant_product
ALTER TABLE public.tenant_product 
ADD COLUMN IF NOT EXISTS core_id UUID REFERENCES public.tenant_product(id) ON DELETE SET NULL;

-- Add core_id to tenant_product_datasource
ALTER TABLE public.tenant_product_datasource 
ADD COLUMN IF NOT EXISTS core_id UUID REFERENCES public.tenant_product_datasource(id) ON DELETE SET NULL;

-- Add core_id to connections
ALTER TABLE public.connections 
ADD COLUMN IF NOT EXISTS core_id UUID REFERENCES public.connections(id) ON DELETE SET NULL;

-- Add indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_tenant_instance_core_id ON public.tenant_instance(core_id) WHERE core_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tenant_product_core_id ON public.tenant_product(core_id) WHERE core_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tenant_product_datasource_core_id ON public.tenant_product_datasource(core_id) WHERE core_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_connections_core_id ON public.connections(core_id) WHERE core_id IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN public.tenant_instance.core_id IS 'Reference to the Gold Copy instance this was cloned from';
COMMENT ON COLUMN public.tenant_product.core_id IS 'Reference to the Gold Copy product this was cloned from';
COMMENT ON COLUMN public.tenant_product_datasource.core_id IS 'Reference to the Gold Copy datasource this was cloned from';
COMMENT ON COLUMN public.connections.core_id IS 'Reference to the Gold Copy connection this was cloned from';
