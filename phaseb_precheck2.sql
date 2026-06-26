-- phaseb_precheck2.sql
-- Safer pre-cleanup PL/pgSQL diagnostics. Non-destructive.
DO $$ DECLARE
  t text;
  default_tpd uuid;
  cnt_null bigint;
  cnt_legacy bigint;
  sample_ids text;
BEGIN
  SELECT id INTO default_tpd FROM public.tenant_product_datasource WHERE source_name LIKE 'phaseb-default-%' ORDER BY created_at DESC LIMIT 1;
  RAISE NOTICE 'default_tpd=%', default_tpd;

  FOR t IN SELECT unnest(ARRAY['catalog_node','catalog_node_type','tenant_chart']) LOOP
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=t AND column_name='tenant_datasource_id_uuid') THEN
      EXECUTE format('SELECT COUNT(*) FROM public.%I WHERE tenant_datasource_id_uuid IS NULL', t) INTO cnt_null;
    ELSE
      cnt_null := -1;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=t AND column_name='tenant_datasource_id') THEN
      EXECUTE format('SELECT COUNT(*) FROM public.%I WHERE tenant_datasource_id IS NOT NULL', t) INTO cnt_legacy;
    ELSE
      cnt_legacy := -1;
    END IF;

    RAISE NOTICE 'table=% null_uuid_count=% legacy_count=%', t, cnt_null, cnt_legacy;

    IF default_tpd IS NOT NULL AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=t AND column_name='tenant_datasource_id_uuid') THEN
      EXECUTE format('SELECT array_to_string(array(SELECT id::text FROM public.%I WHERE tenant_datasource_id_uuid = %L LIMIT 5), '','')', t, default_tpd::text) INTO sample_ids;
      RAISE NOTICE 'sample_ids for % = %', t, sample_ids;
    END IF;
  END LOOP;

  -- print any constraints mentioning legacy name
  RAISE NOTICE 'Constraints referencing tenant_datasource_id (legacy patterns):';
  FOR t IN SELECT conname FROM pg_constraint WHERE pg_get_constraintdef(oid) ILIKE '%tenant_datasource_id%' LOOP
    RAISE NOTICE '  %', t;
  END LOOP;

  -- print views referencing legacy name
  RAISE NOTICE 'Views referencing tenant_datasource_id in definition:';
  FOR t IN SELECT viewname FROM pg_views WHERE definition ILIKE '%tenant_datasource_id%' LOOP
    RAISE NOTICE '  %', t;
  END LOOP;
END $$ LANGUAGE plpgsql;
