-- Per-entity-type discriminator for attribute definitions (e.g. account_type_cd).
-- [] = applies to all types; ["RETAIL","SMA"] = only those types.

ALTER TABLE public.attribute_def
    ADD COLUMN IF NOT EXISTS applies_to_types jsonb NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_attribute_def_applies_to_types
    ON public.attribute_def USING gin (applies_to_types);

COMMENT ON COLUMN public.attribute_def.applies_to_types IS
    'Empty array = all subtypes. Otherwise JSON string array of type codes (e.g. account_type_cd values).';

-- Platform core tenant (zero UUID) is always readable alongside gold/shared-ref.
DROP POLICY IF EXISTS attr_def_read ON public.attribute_def;
CREATE POLICY attr_def_read ON public.attribute_def
    AS PERMISSIVE FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR tenant_id = '00000000-0000-0000-0000-000000000000'::uuid
        OR tenant_id = COALESCE(
            NULLIF(current_setting('app.shared_reference_tenant', true), '')::uuid,
            (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1),
            '00000000-0000-0000-0000-000000000001'::uuid
        )
    );

-- Recreate effective view: CORE includes platform zero UUID + gold/shared when distinct.
DROP VIEW IF EXISTS public.attribute_def_effective;
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
    ) AS shared_tenant_id
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
      AND a.tenant_id <> '00000000-0000-0000-0000-000000000000'::uuid
      AND a.is_active
      AND NOT a.is_shadow
),
core_rows AS (
    SELECT a.*
    FROM public.attribute_def a
    CROSS JOIN tenant_ctx t
    WHERE a.is_active
      AND (
          a.tenant_id = '00000000-0000-0000-0000-000000000000'::uuid
          OR (
              a.tenant_id = t.shared_tenant_id
              AND t.tenant_id IS DISTINCT FROM t.shared_tenant_id
          )
      )
      AND a.id NOT IN (SELECT core_id FROM shadowed WHERE core_id IS NOT NULL)
      AND NOT EXISTS (
          SELECT 1 FROM tenant_rows tr WHERE tr.core_id = a.id
      )
)
SELECT *, 'CUSTOM'::text AS origin, 10 AS precedence_rank FROM tenant_rows
UNION ALL
SELECT *, 'CORE'::text AS origin, 100 AS precedence_rank FROM core_rows;
