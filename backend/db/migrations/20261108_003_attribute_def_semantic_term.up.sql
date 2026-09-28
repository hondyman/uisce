-- Link attribute definitions to semantic terms (catalog_node of type semantic_term).
-- BO fields that reference the same term_node_id resolve through JSON_PATH
-- into custom_attributes->>'field_cd'.

ALTER TABLE public.attribute_def
    ADD COLUMN IF NOT EXISTS semantic_term_id uuid;

CREATE INDEX IF NOT EXISTS idx_attribute_def_semantic_term
    ON public.attribute_def (semantic_term_id)
    WHERE semantic_term_id IS NOT NULL AND is_active;

COMMENT ON COLUMN public.attribute_def.semantic_term_id IS
    'catalog_node id for the semantic_term this custom attribute realizes; BO fields with matching term_node_id bind as JSON_PATH into custom_attributes';

-- Ensure MAPS_TO edge type exists for term → custom_field / column wiring
DO $seed$
DECLARE
    _tenant uuid;
    _term_type_id uuid;
    _cf_type_id uuid;
    _col_type_id uuid;
BEGIN
    SELECT id INTO _tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF _tenant IS NULL THEN
        RETURN;
    END IF;

    SELECT id INTO _term_type_id FROM public.catalog_node_types
      WHERE catalog_type_name IN ('semantic_term', 'SEMANTIC_TERM') LIMIT 1;
    SELECT id INTO _cf_type_id FROM public.catalog_node_types
      WHERE catalog_type_name = 'custom_field' LIMIT 1;
    SELECT id INTO _col_type_id FROM public.catalog_node_types
      WHERE catalog_type_name = 'column' LIMIT 1;

    IF _term_type_id IS NOT NULL AND _cf_type_id IS NOT NULL THEN
        INSERT INTO public.catalog_edge_types (
            id, tenant_id, edge_type_name, description,
            source_node_type_id, target_node_type_id, is_directed, is_active, config
        ) VALUES (
            'c3d4e5f6-a7b8-9012-cdef-123456789012'::uuid,
            _tenant,
            'TERM_MAPS_TO_CUSTOM_FIELD',
            'Semantic term is realized by a custom attribute definition (JSONB key)',
            _term_type_id,
            _cf_type_id,
            true, true,
            '{
                "storage_kind": {"type":"string","enum":["JSONB_KEY"]},
                "json_path": {"type":"string"},
                "jsonb_column": {"type":"string"}
            }'::jsonb
        )
        ON CONFLICT (tenant_id, edge_type_name) DO NOTHING;
    END IF;

    -- MAPS_TO is already used by ResolveSemanticFieldMap; keep available for term→column
    IF _term_type_id IS NOT NULL AND _col_type_id IS NOT NULL THEN
        INSERT INTO public.catalog_edge_types (
            id, tenant_id, edge_type_name, description,
            source_node_type_id, target_node_type_id, is_directed, is_active, config
        ) VALUES (
            'd4e5f6a7-b8c9-0123-def0-234567890123'::uuid,
            _tenant,
            'MAPS_TO',
            'Semantic term maps to a physical column (or JSONB column with json_path in properties)',
            _term_type_id,
            _col_type_id,
            true, true, '{}'::jsonb
        )
        ON CONFLICT (tenant_id, edge_type_name) DO NOTHING;
    END IF;
END
$seed$;


-- Recreate effective view so semantic_term_id appears before origin/precedence columns.
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
