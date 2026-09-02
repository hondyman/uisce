-- Corrected replacement for the never-run
-- backend/migrations/20260812000002_sync_existing_bos_to_catalog.up.sql,
-- which references business_objects.key/name/display_name/driver_table_id/
-- driver_table_name/datasource_id — none of which exist on the live table
-- (it has bo_key/bo_name, and driving-table info comes from
-- driver_table_name directly, not business_object_bindings).
--
-- Populates one catalog_node per business_objects row, of type
-- 'business_object' (the id already in live use — see
-- SemanticGraphService.Initialize's tie-break), named by bo_key so
-- GetNodeByName(BOName) can find it, with tenant_datasource_id left NULL
-- to match SemanticGraphService.GetNodeByName's "unscoped" fallback (no
-- caller currently supplies a real datasource id).

DO $$
DECLARE
    v_bo_type_id UUID;
    v_bo RECORD;
    v_created INT := 0;
    v_updated INT := 0;
BEGIN
    SELECT cnt.id INTO v_bo_type_id
    FROM catalog_node_type cnt
    WHERE cnt.catalog_type_name = 'business_object'
    ORDER BY (SELECT count(*) FROM catalog_node cn WHERE cn.node_type_id = cnt.id) DESC
    LIMIT 1;

    IF v_bo_type_id IS NULL THEN
        RAISE EXCEPTION 'no business_object catalog_node_type found';
    END IF;

    FOR v_bo IN
        SELECT id, tenant_id, bo_key, bo_name, driver_table_name
        FROM business_objects
        WHERE is_active = true
    LOOP
        IF EXISTS (
            SELECT 1 FROM catalog_node
            WHERE node_type_id = v_bo_type_id
              AND node_name = v_bo.bo_key
              AND tenant_datasource_id IS NULL
        ) THEN
            UPDATE catalog_node SET
                properties = jsonb_build_object(
                    'bo_key', v_bo.bo_key,
                    'driving_table', replace(trim(leading '/' from v_bo.driver_table_name), '/', '.')
                ),
                description = v_bo.bo_name,
                updated_at = now()
            WHERE node_type_id = v_bo_type_id
              AND node_name = v_bo.bo_key
              AND tenant_datasource_id IS NULL;
            v_updated := v_updated + 1;
        ELSE
            INSERT INTO catalog_node (
                id, node_type_id, node_name, tenant_id, tenant_datasource_id,
                properties, description, qualified_path, created_at, updated_at
            ) VALUES (
                v_bo.id, v_bo_type_id, v_bo.bo_key, v_bo.tenant_id, NULL,
                jsonb_build_object(
                    'bo_key', v_bo.bo_key,
                    'driving_table', replace(trim(leading '/' from v_bo.driver_table_name), '/', '.')
                ),
                v_bo.bo_name, v_bo.bo_key, now(), now()
            )
            ON CONFLICT (id) DO NOTHING;
            v_created := v_created + 1;
        END IF;
    END LOOP;

    RAISE NOTICE 'BO catalog sync: % created, % updated', v_created, v_updated;
END $$;
