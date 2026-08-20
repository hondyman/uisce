-- Phase 3a: Promote inline calculated fields from business_objects.fields JSONB
-- to the public.calc_fields catalog table.
-- Replaces inline calc entries with calculated_ref pointers.
-- Mirrors entries to catalog_node (calculation_term type) for the catalog graph.

BEGIN;

DO $$
DECLARE
    gold_tid UUID;
    calc_count INTEGER := 0;
    bo_count INTEGER := 0;
BEGIN
    SELECT id INTO gold_tid FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF gold_tid IS NULL THEN
        RAISE NOTICE 'No gold_copy tenant found, skipping calc field promotion';
        RETURN;
    END IF;

    RAISE NOTICE 'Promoting inline calc fields for gold_copy tenant: %', gold_tid;

    -- =============================================
    -- Step 1: Create calc_fields rows from inline entries
    -- =============================================

    WITH extracted AS (
        SELECT
            bo.id AS bo_id,
            bo.tenant_id,
            field->>'name' AS name,
            field->>'sql_expr' AS sql_expr,
            field->>'sql' AS sql_expr_alt,
            COALESCE(field->>'data_type', 'number') AS data_type,
            COALESCE((field->>'is_measure')::boolean, false) AS is_measure,
            COALESCE((field->>'is_calculated')::boolean, false) AS is_calculated,
            field->>'type' AS field_type
        FROM public.business_objects bo,
             jsonb_array_elements(bo.fields) AS field
        WHERE bo.tenant_id = gold_tid
          AND (
              (field->>'is_calculated')::boolean = true
              OR field->>'type' = 'calculated'
          )
    ),
    resolved AS (
        SELECT
            bo_id, tenant_id, name,
            COALESCE(sql_expr, sql_expr_alt, '') AS sql_expr,
            data_type, is_measure
        FROM extracted
        WHERE is_calculated = true OR field_type = 'calculated'
    ),
    inserted AS (
        INSERT INTO public.calc_fields (tenant_id, object_id, name, sql_expr, data_type, is_measure)
        SELECT tenant_id, bo_id, name, sql_expr, data_type, is_measure
        FROM resolved
        ON CONFLICT (tenant_id, object_id, name) DO UPDATE
            SET sql_expr = EXCLUDED.sql_expr,
                data_type = EXCLUDED.data_type,
                is_measure = EXCLUDED.is_measure,
                updated_at = clock_timestamp()
        RETURNING id, tenant_id, object_id, name
    )
    SELECT COUNT(*) INTO calc_count FROM inserted;

    RAISE NOTICE '  Inserted/updated % calc_fields rows from inline entries', calc_count;

    -- =============================================
    -- Step 2: Get mapping of (tenant_id, bo_id, name) -> calc_fields.id
    -- =============================================

    -- =============================================
    -- Step 3: Replace inline calc entries with calculated_ref pointers
    -- For each business_objects row, rebuild the fields array:
    --   - If entry is a calc field (is_calculated=true or type='calculated'), replace with
    --     a calculated_ref entry containing the calc_fields.id
    --   - Otherwise keep the entry as-is
    -- =============================================

    -- We do this via a CTE that unnests, transforms, and re-aggregates
    WITH bo_update AS (
        SELECT
            bo.id AS bo_id,
            bo.fields AS old_fields,
            (
                SELECT jsonb_agg(
                    CASE
                        WHEN (elem->>'is_calculated')::boolean = true
                          OR elem->>'type' = 'calculated'
                        THEN (
                            SELECT jsonb_build_object(
                                'type', 'calculated_ref',
                                'name', elem->>'name',
                                'calc_field_id', cf.id::text,
                                'display_name', elem->>'display_name',
                                'data_type', COALESCE(elem->>'data_type', 'number'),
                                'is_measure', COALESCE((elem->>'is_measure')::boolean, false)
                            )
                            FROM public.calc_fields cf
                            WHERE cf.tenant_id = bo.tenant_id
                              AND cf.object_id = bo.id
                              AND cf.name = elem->>'name'
                            LIMIT 1
                        )
                        ELSE elem
                    END
                )
                FROM jsonb_array_elements(bo.fields) AS elem
            ) AS new_fields
        FROM public.business_objects bo
        WHERE bo.tenant_id = gold_tid
          AND bo.fields IS NOT NULL
          AND jsonb_typeof(bo.fields) = 'array'
          AND EXISTS (
              SELECT 1 FROM jsonb_array_elements(bo.fields) f
              WHERE (f->>'is_calculated')::boolean = true OR f->>'type' = 'calculated'
          )
    )
    UPDATE public.business_objects bo
    SET fields = bo_update.new_fields
    FROM bo_update
    WHERE bo.id = bo_update.bo_id;

    -- Count BOs updated
    GET DIAGNOSTICS bo_count = ROW_COUNT;
    RAISE NOTICE '  Updated % business_objects rows — inline calc entries replaced with calculated_ref', bo_count;

    -- =============================================
    -- Step 4: Mirror to catalog_node as calculation_term nodes
    -- =============================================

    INSERT INTO catalog_node (
        id, tenant_id, node_type_id, node_name, description, qualified_path, is_active, properties
    )
    SELECT
        'calc:' || cf.id::text,
        cf.tenant_id,
        (SELECT id FROM catalog_node_type WHERE catalog_type_name = 'calculation_term' LIMIT 1),
        cf.name,
        'Calculated field: ' || cf.name,
        'calc:' || cf.name,
        true,
        jsonb_build_object(
            'calc_field_id', cf.id::text,
            'sql_expr', cf.sql_expr,
            'data_type', cf.data_type,
            'is_measure', cf.is_measure,
            'object_id', cf.object_id::text,
            'source', 'calc_fields_catalog'
        )
    FROM public.calc_fields cf
    WHERE cf.tenant_id = gold_tid
      AND NOT EXISTS (
          SELECT 1 FROM catalog_node cn
          WHERE cn.id = 'calc:' || cf.id::text
      )
    ON CONFLICT (id) DO UPDATE
        SET properties = EXCLUDED.properties;

    RAISE NOTICE '=== Calc Field Promotion Complete ===';
    RAISE NOTICE '  calc_fields rows created/updated: %', calc_count;
    RAISE NOTICE '  business_objects updated: %', bo_count;

END $$;

COMMIT;
