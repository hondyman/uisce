-- Phase 1: ORM Suite product + crims_orm datasource for gold_copy tenant
-- Registers the ORM Suite product and CRIMS ORM datasource on the gold_copy tenant
-- so that ORM-based business objects (Trade, Order, Execution, etc.) are properly bound.

BEGIN;

DO $$
DECLARE
    gold_tid UUID;
    orm_product_id UUID;
    orm_ds_id UUID;
    orm_instance_id UUID;
    orm_tenant_product_id UUID;
    orm_conn_id UUID;
    orm_tpd_id UUID;
    _ds_type_exists boolean;
BEGIN
    -- Resolve gold_copy tenant
    SELECT id INTO gold_tid FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF gold_tid IS NULL THEN
        RAISE NOTICE 'No gold_copy tenant found, skipping ORM Suite setup';
        RETURN;
    END IF;

    RAISE NOTICE 'Setting up ORM Suite for gold_copy tenant: %', gold_tid;

    -- 1. Ensure alpha_product for ORM Suite
    SELECT id INTO orm_product_id FROM alpha_product WHERE product_code = 'orm_suite' LIMIT 1;
    IF orm_product_id IS NULL THEN
        INSERT INTO alpha_product (product_name, product_code, description, is_active)
        VALUES ('ORM Suite', 'orm_suite', 'Order Management & Trading Suite', true)
        RETURNING id INTO orm_product_id;
        RAISE NOTICE 'Created alpha_product orm_suite: %', orm_product_id;
    ELSE
        RAISE NOTICE 'alpha_product orm_suite already exists: %', orm_product_id;
    END IF;

    -- 2. Ensure alpha_datasource for CRIMS ORM (datasource_type = 'ORM')
    -- First check if datasource_type column exists
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'alpha_datasource'
          AND column_name = 'datasource_type'
    ) INTO _ds_type_exists;

    IF _ds_type_exists THEN
        UPDATE alpha_datasource SET
            datasource_name = 'CRIMS Front Office ORM',
            description = 'CRIMS Front Office order management and trading system relational model (ORM schema)',
            is_active = true,
            datasource_type = 'ORM'
        WHERE datasource_code = 'crims_orm'
        RETURNING id INTO orm_ds_id;

        IF orm_ds_id IS NULL THEN
            INSERT INTO alpha_datasource (datasource_code, datasource_name, description, is_active, datasource_type)
            VALUES ('crims_orm', 'CRIMS Front Office ORM', 'CRIMS Front Office order management and trading system relational model (ORM schema)', true, 'ORM')
            RETURNING id INTO orm_ds_id;
            RAISE NOTICE 'Created alpha_datasource crims_orm: %', orm_ds_id;
        ELSE
            RAISE NOTICE 'Updated alpha_datasource crims_orm: %', orm_ds_id;
        END IF;
    ELSE
        -- No datasource_type column - use simpler insert
        UPDATE alpha_datasource SET
            datasource_name = 'CRIMS Front Office ORM',
            description = 'CRIMS Front Office order management and trading system relational model (ORM schema)',
            is_active = true
        WHERE datasource_code = 'crims_orm'
        RETURNING id INTO orm_ds_id;

        IF orm_ds_id IS NULL THEN
            INSERT INTO alpha_datasource (datasource_code, datasource_name, description, is_active)
            VALUES ('crims_orm', 'CRIMS Front Office ORM', 'CRIMS Front Office order management and trading system relational model (ORM schema)', true)
            RETURNING id INTO orm_ds_id;
            RAISE NOTICE 'Created alpha_datasource crims_orm (no datasource_type col): %', orm_ds_id;
        ELSE
            RAISE NOTICE 'Updated alpha_datasource crims_orm: %', orm_ds_id;
        END IF;
    END IF;

    -- 3. Ensure tenant_instance for gold_copy ORM Suite
    SELECT id INTO orm_instance_id FROM tenant_instance
    WHERE tenant_id = gold_tid AND instance_name = 'orm_suite_primary'
    LIMIT 1;

    IF orm_instance_id IS NULL THEN
        INSERT INTO tenant_instance (tenant_id, instance_name)
        VALUES (gold_tid, 'orm_suite_primary')
        RETURNING id INTO orm_instance_id;
        RAISE NOTICE 'Created tenant_instance orm_suite_primary: %', orm_instance_id;
    ELSE
        RAISE NOTICE 'tenant_instance orm_suite_primary already exists: %', orm_instance_id;
    END IF;

    -- 4. Ensure tenant_product (ORM Suite product linked to tenant_instance)
    SELECT id INTO orm_tenant_product_id FROM tenant_product
    WHERE tenant_id = gold_tid AND product_name = 'ORM Suite'
    LIMIT 1;

    IF orm_tenant_product_id IS NULL THEN
        INSERT INTO tenant_product (tenant_id, datasource_id, product_name)
        VALUES (gold_tid, orm_instance_id, 'ORM Suite')
        RETURNING id INTO orm_tenant_product_id;
        RAISE NOTICE 'Created tenant_product ORM Suite: %', orm_tenant_product_id;
    ELSE
        RAISE NOTICE 'tenant_product ORM Suite already exists: %', orm_tenant_product_id;
    END IF;

    -- 5. Ensure connections entry for CRIMS ORM (host=100.84.50.65, port=5432, database=crims, schema=orm)
    SELECT id INTO orm_conn_id FROM connections
    WHERE name = 'CRIMS ORM' AND host = '100.84.50.65'
    LIMIT 1;

    IF orm_conn_id IS NULL THEN
        INSERT INTO connections (name, type, host, port, database, schema)
        VALUES ('CRIMS ORM', 'postgresql', '100.84.50.65', 5432, 'crims', 'orm')
        RETURNING id INTO orm_conn_id;
        RAISE NOTICE 'Created connections CRIMS ORM: %', orm_conn_id;
    ELSE
        RAISE NOTICE 'connections CRIMS ORM already exists: %', orm_conn_id;
    END IF;

    -- 6. Ensure tenant_product_datasource linking ORM Suite product to CRIMS ORM datasource
    -- First check if it already exists
    SELECT tpd.id INTO orm_tpd_id
    FROM tenant_product_datasource tpd
    JOIN tenant_product tp ON tp.id = tpd.tenant_product_id
    WHERE tp.tenant_id = gold_tid
      AND tp.product_name = 'ORM Suite'
      AND tpd.source_name = 'orm_suite_primary'
    LIMIT 1;

    IF orm_tpd_id IS NOT NULL THEN
        RAISE NOTICE 'tenant_product_datasource ORM Suite already exists: %', orm_tpd_id;
    ELSE
        INSERT INTO tenant_product_datasource (
            tenant_product_id,
            alpha_datasource_id,
            connection_id,
            source_name,
            is_active
        ) VALUES (
            orm_tenant_product_id,
            orm_ds_id,
            orm_conn_id,
            'orm_suite_primary',
            true
        )
        RETURNING id INTO orm_tpd_id;
        RAISE NOTICE 'Created tenant_product_datasource ORM Suite: %', orm_tpd_id;
    END IF;

    -- Verify summary
    RAISE NOTICE '=== ORM Suite Setup Complete for gold_copy tenant % ===', gold_tid;
    RAISE NOTICE '  alpha_product id: % (product_code=orm_suite)', orm_product_id;
    RAISE NOTICE '  alpha_datasource id: % (datasource_code=crims_orm, datasource_type=ORM)', orm_ds_id;
    RAISE NOTICE '  tenant_instance id: % (instance_name=orm_suite_primary)', orm_instance_id;
    RAISE NOTICE '  tenant_product id: % (product_name=ORM Suite)', orm_tenant_product_id;
    RAISE NOTICE '  connections id: % (name=CRIMS ORM)', orm_conn_id;
    RAISE NOTICE '  tenant_product_datasource id: % (source_name=orm_suite_primary)', orm_tpd_id;

END $$;

COMMIT;
