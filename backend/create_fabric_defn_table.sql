-- Create the fabric_defn table if it doesn't exist
CREATE TABLE IF NOT EXISTS public.fabric_defn (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id uuid NOT NULL,
    tenant_datasource_id uuid NOT NULL,
    model_key text NOT NULL,
    version integer NOT NULL DEFAULT 1,
    status text NOT NULL DEFAULT 'draft',
    title text,
    description text,
    source_config jsonb,
    resolved_config jsonb,
    created_by uuid,
    is_current boolean NOT NULL DEFAULT true,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_fabric_defn_tenant_datasource_id ON public.fabric_defn(tenant_datasource_id);
CREATE INDEX IF NOT EXISTS idx_fabric_defn_model_key ON public.fabric_defn(model_key);
CREATE INDEX IF NOT EXISTS idx_fabric_defn_is_current ON public.fabric_defn(is_current);
CREATE INDEX IF NOT EXISTS idx_fabric_defn_tenant_id ON public.fabric_defn(tenant_id);
