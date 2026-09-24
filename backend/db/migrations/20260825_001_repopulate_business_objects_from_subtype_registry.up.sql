-- backend/db/migrations/20260825_001_repopulate_business_objects_from_subtype_registry.up.sql
-- Repopulates business_objects from oms.subtype_registry (STI) using the new schema
-- (bo_key, bo_name, bo_type, model_id, classification_node_id, business_key_node_id,
--  semantic_id_node_id, grain_node_id, sti_discriminator_column, active_subtype_filter).
--
-- Also creates the required catalog_node rows for classification/business_key/semantic_id/grain
-- per BO, and seeds business_object_fields from field_allowlist JSONB.
--
-- Idempotent: uses ON CONFLICT (tenant_id, bo_key) DO UPDATE for parent BOs.

BEGIN;

DO $$
DECLARE
    bo_type_id UUID := '06bb774c-8666-4ab1-84eb-4f4d439ac84c'; -- business_object node type
    model_node_id UUID;
    cls_node_id UUID;
    bk_node_id UUID;
    sem_node_id UUID;
    grain_node_id UUID;
    parent_bo_id UUID;
    v_bo_id UUID;
    field_term_id UUID;
    tenant_uuid UUID;
    root_obj TEXT;
    subtype_code TEXT;
    sc TEXT;
    rec RECORD;
    f TEXT;
    bo_key_name TEXT;
BEGIN
    -- For each unique (tenant_id, root_object) in subtype_registry,
    -- iterate subtypes and upsert parent + child BOs + fields.
    -- Only process tenant IDs that exist in the tenants table.
    FOR tenant_uuid, root_obj IN
        SELECT DISTINCT sr.tenant_id, sr.root_object
        FROM oms.subtype_registry sr
        WHERE sr.is_active = true
          AND EXISTS (SELECT 1 FROM tenants t WHERE t.id = sr.tenant_id)
    LOOP
        bo_key_name := 'oms.' || root_obj;

        -- ── 1. Create the 4 catalog_node rows for this BO (one-time per BO key per tenant) ──
        -- We use a deterministic placeholder model_id (the classification node id itself)
        -- since the model concept is not yet seeded. These 4 nodes are the semantic anchors.

        INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, is_active)
        VALUES (gen_random_uuid(), tenant_uuid, bo_type_id, bo_key_name,
                'business_object/' || bo_key_name || '/classification',
                jsonb_build_object('role', 'classification', 'bo_key', bo_key_name),
                true)
        ON CONFLICT (tenant_id, node_type_id, qualified_path) DO NOTHING
        RETURNING id INTO cls_node_id;

        INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, is_active)
        VALUES (gen_random_uuid(), tenant_uuid, bo_type_id, bo_key_name || ' (business key)',
                'business_object/' || bo_key_name || '/business_key',
                jsonb_build_object('role', 'business_key', 'bo_key', bo_key_name),
                true)
        ON CONFLICT (tenant_id, node_type_id, qualified_path) DO NOTHING
        RETURNING id INTO bk_node_id;

        INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, is_active)
        VALUES (gen_random_uuid(), tenant_uuid, bo_type_id, bo_key_name || ' (semantic id)',
                'business_object/' || bo_key_name || '/semantic_id',
                jsonb_build_object('role', 'semantic_id', 'bo_key', bo_key_name),
                true)
        ON CONFLICT (tenant_id, node_type_id, qualified_path) DO NOTHING
        RETURNING id INTO sem_node_id;

        INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, is_active)
        VALUES (gen_random_uuid(), tenant_uuid, bo_type_id, bo_key_name || ' (grain)',
                'business_object/' || bo_key_name || '/grain',
                jsonb_build_object('role', 'grain', 'bo_key', bo_key_name),
                true)
        ON CONFLICT (tenant_id, node_type_id, qualified_path) DO NOTHING
        RETURNING id INTO grain_node_id;

        -- Use classification node as the model_id placeholder (no real model exists yet)
        model_node_id := COALESCE(cls_node_id, gen_random_uuid());

        -- Fetch cls_node_id again if it was just created (ON CONFLICT didn't set the variable)
        SELECT id INTO cls_node_id FROM catalog_node
        WHERE tenant_id = tenant_uuid
          AND node_type_id = bo_type_id
          AND qualified_path = 'business_object/' || bo_key_name || '/classification'
        LIMIT 1;

        SELECT id INTO bk_node_id FROM catalog_node
        WHERE tenant_id = tenant_uuid
          AND node_type_id = bo_type_id
          AND qualified_path = 'business_object/' || bo_key_name || '/business_key'
        LIMIT 1;

        SELECT id INTO sem_node_id FROM catalog_node
        WHERE tenant_id = tenant_uuid
          AND node_type_id = bo_type_id
          AND qualified_path = 'business_object/' || bo_key_name || '/semantic_id'
        LIMIT 1;

        SELECT id INTO grain_node_id FROM catalog_node
        WHERE tenant_id = tenant_uuid
          AND node_type_id = bo_type_id
          AND qualified_path = 'business_object/' || bo_key_name || '/grain'
        LIMIT 1;

        -- ── 2. Upsert parent BO (the root STI object) ──
        INSERT INTO business_objects
            (id, tenant_id, model_id, bo_key, bo_name, bo_type, description,
             classification_node_id, business_key_node_id, semantic_id_node_id, grain_node_id,
             sti_discriminator_column, active_subtype_filter, is_active, is_core)
        VALUES
            (gen_random_uuid(), tenant_uuid, model_node_id, bo_key_name, root_obj,
             'ENTITY',
             'STI root object for ' || root_obj || ' — subtypes managed via oms.subtype_registry',
             cls_node_id, bk_node_id, sem_node_id, grain_node_id,
             'subtype_code', bo_key_name, true, true)
        ON CONFLICT (tenant_id, bo_key) DO UPDATE SET
            bo_name        = EXCLUDED.bo_name,
            description    = EXCLUDED.description,
            model_id       = EXCLUDED.model_id,
            classification_node_id = EXCLUDED.classification_node_id,
            business_key_node_id     = EXCLUDED.business_key_node_id,
            semantic_id_node_id      = EXCLUDED.semantic_id_node_id,
            grain_node_id            = EXCLUDED.grain_node_id,
            sti_discriminator_column  = EXCLUDED.sti_discriminator_column,
            active_subtype_filter     = EXCLUDED.active_subtype_filter,
            is_active  = EXCLUDED.is_active,
            is_core    = EXCLUDED.is_core,
            updated_at = NOW()
        RETURNING id INTO parent_bo_id;

        -- ── 3. Upsert child BOs (subtypes) ──
        FOR rec IN
            SELECT sr.subtype_code FROM oms.subtype_registry sr
            WHERE sr.tenant_id = tenant_uuid
              AND sr.root_object = root_obj
              AND sr.is_active = true
        LOOP
            sc := rec.subtype_code;
            INSERT INTO business_objects
                (id, tenant_id, model_id, bo_key, bo_name, bo_type, description,
                 classification_node_id, business_key_node_id, semantic_id_node_id, grain_node_id,
                 sti_discriminator_column, active_subtype_filter, is_active, is_core)
            VALUES
                (gen_random_uuid(), tenant_uuid, model_node_id,
                 bo_key_name || '/' || sc, sc,
                 'ENTITY',
                 'STI subtype ' || sc || ' of ' || root_obj,
                 cls_node_id, bk_node_id, sem_node_id, grain_node_id,
                 'subtype_code', bo_key_name, true, false)
            ON CONFLICT (tenant_id, bo_key) DO UPDATE SET
                bo_name       = EXCLUDED.bo_name,
                description   = EXCLUDED.description,
                is_core       = EXCLUDED.is_core,
                updated_at    = NOW();

            -- ── 4. Seed business_object_fields from field_allowlist ──
            FOR f IN
                SELECT jsonb_array_elements_text(sr.field_allowlist)
                FROM oms.subtype_registry sr
                WHERE sr.tenant_id = tenant_uuid
                  AND sr.root_object = root_obj
                  AND sr.subtype_code = sc
                  AND sr.is_active = true
            LOOP
                -- Find matching semantic_term in catalog_node by name match
                SELECT cn.id INTO field_term_id
                FROM catalog_node cn
                WHERE cn.tenant_id = tenant_uuid
                  AND cn.node_type = 'semantic_term'
                  AND (cn.node_name = f OR cn.properties->>'term_key' = f)
                LIMIT 1;

                -- Fall back to a generated placeholder if no semantic term exists
                IF field_term_id IS NULL THEN
                    INSERT INTO catalog_node (id, tenant_id, node_type_id, node_name, qualified_path, properties, is_active)
                    VALUES (
                        gen_random_uuid(), tenant_uuid, bo_type_id, f,
                        'semantic_term/' || f,
                        jsonb_build_object('term_key', f, 'source', 'oms.subtype_registry.seed', 'bo_key', bo_key_name || '/' || sc),
                        true
                    )
                    ON CONFLICT (tenant_id, node_type_id, qualified_path) DO UPDATE
                        SET properties = EXCLUDED.properties
                    RETURNING id INTO field_term_id;
                END IF;

                INSERT INTO business_object_fields
                    (id, tenant_id, bo_id, term_node_id, field_name, field_role,
                     aggregation_type, binding_requirement, eligibility_source,
                     subtype_scope, is_exposed, inherits_defaults)
                VALUES
                    (gen_random_uuid(), tenant_uuid, parent_bo_id, field_term_id, f,
                     'DIMENSION', 'NONE', 'REQUIRED', 'DIRECT',
                     'ALL', true, true)
                ON CONFLICT (tenant_id, bo_id, field_name) DO UPDATE SET
                    term_node_id    = EXCLUDED.term_node_id,
                    field_role      = EXCLUDED.field_role,
                    updated_at      = NOW();
            END LOOP;
        END LOOP;
    END LOOP;
END;
$$;

COMMIT;
