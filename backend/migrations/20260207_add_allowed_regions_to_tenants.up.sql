-- Add a JSONB column to store per-tenant allowed regions
ALTER TABLE public.tenants
ADD COLUMN IF NOT EXISTS allowed_regions JSONB;

-- Optionally backfill by copying metadata->'allowed_regions' if present
UPDATE public.tenants
SET allowed_regions = (CASE WHEN metadata ? 'allowed_regions' THEN metadata->'allowed_regions' ELSE NULL END)
WHERE allowed_regions IS NULL AND (metadata ? 'allowed_regions');

-- Add index to speed lookups by tenant and to allow querying membership
CREATE INDEX IF NOT EXISTS idx_tenants_allowed_regions ON public.tenants USING GIN (allowed_regions jsonb_path_ops);

-- Helpful comment
COMMENT ON COLUMN public.tenants.allowed_regions IS 'Array of allowed regions (e.g. ["eu-west","us-east"])';