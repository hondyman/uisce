DROP VIEW IF EXISTS public.attribute_def_effective;

-- Restore prior effective view shape (without applies_to_types in logic).
CREATE VIEW public.attribute_def_effective AS
WITH tenant_ctx AS (
    SELECT COALESCE(
        NULLIF(current_setting('app.current_tenant', true), '')::uuid,
        '00000000-0000-0000-0000-000000000000'::uuid
    ) AS tenant_id,
    COALESCE(
        NULLIF(current_setting('app.shared_reference_tenant', true), '')::uuid,
        (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1),
        '00000000-0000-0000-0000-000000000001'::uuid
    ) AS core_tenant_id
),
shadowed AS (
    SELECT a.core_id
    FROM public.attribute_def a
    CROSS JOIN tenant_ctx t
    WHERE a.tenant_id = t.tenant_id
      AND a.is_shadow = true
      AND a.core_id IS NOT NULL
),
tenant_rows AS (
    SELECT a.*
    FROM public.attribute_def a
    CROSS JOIN tenant_ctx t
    WHERE a.tenant_id = t.tenant_id
      AND a.is_active
      AND NOT a.is_shadow
),
core_rows AS (
    SELECT a.*
    FROM public.attribute_def a
    CROSS JOIN tenant_ctx t
    WHERE a.tenant_id = t.core_tenant_id
      AND t.tenant_id IS DISTINCT FROM t.core_tenant_id
      AND a.is_active
      AND a.id NOT IN (SELECT core_id FROM shadowed)
      AND NOT EXISTS (
          SELECT 1 FROM tenant_rows tr WHERE tr.core_id = a.id
      )
)
SELECT *, 'CUSTOM'::text AS origin, 10 AS precedence_rank FROM tenant_rows
UNION ALL
SELECT *, 'CORE'::text AS origin, 100 AS precedence_rank FROM core_rows;

DROP INDEX IF EXISTS public.idx_attribute_def_applies_to_types;
ALTER TABLE public.attribute_def DROP COLUMN IF EXISTS applies_to_types;
