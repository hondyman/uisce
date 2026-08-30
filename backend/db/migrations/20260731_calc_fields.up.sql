-- Migration: calc_fields table for UI-based calculated fields
-- Date: 2026-07-31
-- Purpose: Table for storing calculated fields created via the UI, replacing the previously auto-managed calc_fields.

CREATE TABLE IF NOT EXISTS public.calc_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    object_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    sql_expr TEXT NOT NULL,
    data_type VARCHAR(50) NOT NULL DEFAULT 'number',
    is_measure BOOLEAN DEFAULT false,
    realtime BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_calc_fields_tenant ON public.calc_fields(tenant_id);
CREATE INDEX IF NOT EXISTS idx_calc_fields_object ON public.calc_fields(object_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_calc_fields_tenant_object_name ON public.calc_fields(tenant_id, object_id, name);

ALTER TABLE public.calc_fields ENABLE ROW LEVEL SECURITY;

CREATE POLICY "tenant_isolation_calc_fields"
    ON public.calc_fields
    FOR ALL
    USING (((tenant_id)::text = current_setting('uisce.current_tenant'::text, true)));
