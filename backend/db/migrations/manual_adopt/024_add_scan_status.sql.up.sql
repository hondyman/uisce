-- Migration 024: Add Scan Status and Connection ID to tenant_product_datasource

ALTER TABLE public.tenant_product_datasource
ADD COLUMN IF NOT EXISTS connection_id UUID REFERENCES public.tenant_connections(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS last_scan_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS last_scan_status VARCHAR(50) DEFAULT 'pending', -- pending, running, success, failure
ADD COLUMN IF NOT EXISTS last_scan_message TEXT;

CREATE INDEX IF NOT EXISTS idx_tpd_connection_id ON public.tenant_product_datasource(connection_id);
