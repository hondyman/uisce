-- Phase-B destructive apply (test-only)
DO $$ DECLARE
    tbl text;
BEGIN
    FOR tbl IN SELECT unnest(ARRAY['catalog_node','catalog_node_type','tenant_chart']) LOOP
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=tbl AND column_name='tenant_datasource_id_uuid') THEN
            -- rename legacy to backup if present
            IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=tbl AND column_name='tenant_datasource_id') THEN
                BEGIN
                    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=tbl AND column_name='tenant_datasource_id_old') THEN
                        EXECUTE format('ALTER TABLE public.%I RENAME COLUMN tenant_datasource_id TO tenant_datasource_id_old', tbl);
                        RAISE NOTICE 'Renamed legacy tenant_datasource_id -> tenant_datasource_id_old on %', tbl;
                    ELSE
                        RAISE NOTICE 'tenant_datasource_id_old already exists on %, skipping rename', tbl;
                    END IF;
                EXCEPTION WHEN others THEN
                    RAISE NOTICE 'Could not rename legacy column on %: %', tbl, SQLERRM;
                END;
            END IF;

            -- rename canonical into place
            BEGIN
                EXECUTE format('ALTER TABLE public.%I RENAME COLUMN tenant_datasource_id_uuid TO tenant_datasource_id', tbl);
                RAISE NOTICE 'Renamed tenant_datasource_id_uuid -> tenant_datasource_id on %', tbl;
            EXCEPTION WHEN others THEN
                RAISE NOTICE 'Could not rename uuid column on %: %', tbl, SQLERRM;
            END;

            -- add FK on new canonical column (NOT VALID first)
            BEGIN
                EXECUTE format('ALTER TABLE public.%I ADD CONSTRAINT %I FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE NOT VALID', tbl, tbl||'_tenant_product_datasource_fk');
                RAISE NOTICE 'Added NOT VALID FK on %', tbl;
            EXCEPTION WHEN others THEN
                RAISE NOTICE 'Could not add FK on %: %', tbl, SQLERRM;
            END;

            -- try to validate constraint now (may be deferred)
            BEGIN
                EXECUTE format('ALTER TABLE public.%I VALIDATE CONSTRAINT %I', tbl, tbl||'_tenant_product_datasource_fk');
                RAISE NOTICE 'Validated FK on %', tbl;
            EXCEPTION WHEN others THEN
                RAISE NOTICE 'Could not validate FK on % (deferred): %', tbl, SQLERRM;
            END;
        ELSE
            RAISE NOTICE 'No tenant_datasource_id_uuid found on %, skipping', tbl;
        END IF;
    END LOOP;

    -- drop compatibility column on tenant_product_datasource if present
    BEGIN
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='tenant_product_datasource' AND column_name='id_text') THEN
            EXECUTE 'ALTER TABLE public.tenant_product_datasource DROP COLUMN id_text';
            RAISE NOTICE 'Dropped tenant_product_datasource.id_text';
        ELSE
            RAISE NOTICE 'tenant_product_datasource.id_text not present';
        END IF;
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'Could not drop tenant_product_datasource.id_text: %', SQLERRM;
    END;
END $$ LANGUAGE plpgsql;
