-- Phase-B Production-ready SQL
-- WARNING: This file performs destructive schema/API changes when executed. Read completely and run in a staging environment first.
-- Replace placeholders and review comment blocks before running in production.

-- How this file works (summary):
-- 1) Creates a default tenant_product_datasource (TPD) to receive unmapped rows.
-- 2) Backfills canonical tenant_datasource_id_uuid from legacy columns or mapping rules.
-- 3) For any remaining unmapped rows, assigns the default TPD id.
-- 4) Adds authoritative FK constraints on the canonical uuid columns (NOT VALID) and attempts to VALIDATE.
-- 5) (Commented) Destructive cleanup steps: drop legacy columns, rename columns. These are intentionally disabled
--    and must be manually enabled in a maintenance window after you verify results and take a DB snapshot.

-- =========================
-- CONFIG
-- Set these values before running in production.
\echo 'Set CONFIG variables at top of file before running in production. Review mapping rules and maintenance plan.'

-- List of target tables (comma-separated in psql style) - edit if you need more/less
-- Default targets: catalog_node, catalog_node_type, tenant_chart
\set TARGET_TABLES 'catalog_node,catalog_node_type,tenant_chart'

-- Default TPD descriptive name (change if you prefer)
\set DEFAULT_TPD_NAME 'phaseb-default-$(date +%Y%m%d%H%M%S)'

-- Batch size for large updates (optional batching logic used below)
\set BATCH_SIZE 20000

-- =========================
-- 0) IMPORTANT: run backup before anything destructive
-- Example (run outside psql): pg_dump -Fc -U <user> -d <db> -f /backups/alpha_pre_phaseb_$(date +%Y%m%d%H%M%S).dump
-- Or take a filesystem/VM snapshot.

-- =========================
-- 1) Create a default tenant_product_datasource to receive unmapped rows
DO $$ DECLARE
  new_tpd uuid;
  helper_tp uuid;
  helper_tenant_instance uuid;
  helper_alpha_product uuid;
  helper_alpha_datasource uuid;
BEGIN
  -- Ensure we have a tenant_product to reference. Try to reuse an existing one, otherwise create a minimal helper.
  SELECT id INTO helper_tp FROM public.tenant_product LIMIT 1;

  IF helper_tp IS NULL THEN
    -- pick candidate tenant_instance, alpha_product and an alpha_datasource to reference
    SELECT id INTO helper_tenant_instance FROM public.tenant_instance LIMIT 1;
    SELECT id INTO helper_alpha_product FROM public.alpha_product LIMIT 1;
    SELECT id INTO helper_alpha_datasource FROM public.alpha_datasource LIMIT 1;

    IF helper_alpha_datasource IS NULL THEN
      RAISE EXCEPTION 'Cannot create helper tenant_product_datasource: no alpha_datasource found to reference. Create one manually or supply values before running this script.';
    END IF;

    IF helper_tenant_instance IS NULL OR helper_alpha_product IS NULL THEN
      RAISE EXCEPTION 'Cannot create helper tenant_product: no tenant_instance or alpha_product found. Create one manually or supply values before running this script.';
    END IF;

    INSERT INTO public.tenant_product (id, tenant_instance_id, alpha_product_id, created_at, updated_at, version, is_active)
    VALUES (gen_random_uuid(), helper_tenant_instance, helper_alpha_product, now(), now(), 1, TRUE)
    RETURNING id INTO helper_tp;
    RAISE NOTICE 'Created helper tenant_product id=%', helper_tp;
  ELSE
    RAISE NOTICE 'Reusing existing tenant_product id=%', helper_tp;
  END IF;
  -- Ensure we have an alpha_datasource to reference for the tenant_product_datasource (reuse any existing if needed)
  IF helper_alpha_datasource IS NULL THEN
    SELECT id INTO helper_alpha_datasource FROM public.alpha_datasource LIMIT 1;
  END IF;

  IF helper_alpha_datasource IS NULL THEN
    RAISE EXCEPTION 'Cannot create default tenant_product_datasource: no alpha_datasource available to reference.';
  END IF;

  -- Try to find an existing tenant_product_datasource for this tenant_product/alpha_datasource pair and reuse it
  SELECT id INTO new_tpd FROM public.tenant_product_datasource WHERE tenant_product_id = helper_tp AND alpha_datasource_id = helper_alpha_datasource LIMIT 1;
  IF new_tpd IS NULL THEN
    INSERT INTO public.tenant_product_datasource (id, tenant_product_id, alpha_datasource_id, is_active, source_name, created_at, updated_at)
    VALUES (gen_random_uuid(), helper_tp, helper_alpha_datasource, TRUE, 'phaseb-default-' || to_char(now(),'YYYYMMDDHH24MISS'), now(), now())
    RETURNING id INTO new_tpd;
    RAISE NOTICE 'Created default tenant_product_datasource id=% (tenant_product_id=%)', new_tpd, helper_tp;
  ELSE
    RAISE NOTICE 'Reusing existing tenant_product_datasource id=% for tenant_product=% and alpha_datasource=%', new_tpd, helper_tp, helper_alpha_datasource;
  END IF;

  -- store in temp table for later reference
  CREATE TEMP TABLE IF NOT EXISTS tmp_phaseb_defaults (tpd_id uuid);
  TRUNCATE tmp_phaseb_defaults;
  INSERT INTO tmp_phaseb_defaults (tpd_id) VALUES (new_tpd);
END $$ LANGUAGE plpgsql;

-- Inspect the created id (client-side): SELECT * FROM tmp_phaseb_defaults;

-- =========================
-- 2) Backfill canonical columns from legacy columns and mapping heuristics
-- For tables where legacy tenant_datasource_id is uuid typed, copy directly.

-- Ensure canonical columns exist on target tables so backfill updates won't fail
DO $$ DECLARE t TEXT; BEGIN
  FOR t IN SELECT unnest(string_to_array('catalog_node,catalog_node_type,tenant_chart', ',')) LOOP
    BEGIN
      EXECUTE format('ALTER TABLE public.%I ADD COLUMN IF NOT EXISTS tenant_datasource_id_uuid uuid', t);
      RAISE NOTICE 'Ensured tenant_datasource_id_uuid exists on %', t;
    EXCEPTION WHEN others THEN
      RAISE NOTICE 'Could not ensure tenant_datasource_id_uuid on %: %', t, SQLERRM;
    END;
  END LOOP;
END $$ LANGUAGE plpgsql;

DO $$ DECLARE
    t TEXT;
BEGIN
  FOR t IN SELECT unnest(string_to_array('catalog_node,catalog_node_type,tenant_chart', ',')) LOOP
        -- if legacy column exists and is uuid typed, copy values
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=t AND column_name='tenant_datasource_id' AND data_type='uuid') THEN
            EXECUTE format('UPDATE public.%I SET tenant_datasource_id_uuid = tenant_datasource_id WHERE tenant_datasource_id_uuid IS NULL AND tenant_datasource_id IS NOT NULL', t);
            RAISE NOTICE 'Copied uuid legacy -> tenant_datasource_id_uuid on %', t;
        END IF;

        -- if legacy column exists and is text/character varying, attempt mapping via tenant_product_datasource.source_name or tpd.id text
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=t AND column_name='tenant_datasource_id' AND data_type IN ('character varying','text')) THEN
            -- Try mapping by matching to tpd.id::text or tpd.source_name
            BEGIN
                EXECUTE format('UPDATE public.%1$I tgt SET tenant_datasource_id_uuid = tpd.id FROM public.tenant_product_datasource tpd WHERE tgt.tenant_datasource_id_uuid IS NULL AND (tpd.id::text = tgt.tenant_datasource_id::text OR (tpd.source_name IS NOT NULL AND tpd.source_name = tgt.tenant_datasource_id::text))', t);
                RAISE NOTICE 'Attempted text-based mapping for %', t;
            EXCEPTION WHEN others THEN
                RAISE NOTICE 'Text-based mapping for % failed: %', t, SQLERRM;
            END;
        END IF;

    END LOOP;
END $$ LANGUAGE plpgsql;

-- 3) For any remaining unmapped rows, assign the default TPD created above
DO $$ DECLARE t TEXT; default_tpd uuid; BEGIN SELECT tpd_id FROM tmp_phaseb_defaults LIMIT 1 INTO default_tpd; FOR t IN SELECT unnest(string_to_array('catalog_node,catalog_node_type,tenant_chart', ',')) LOOP
    EXECUTE format('UPDATE public.%I SET tenant_datasource_id_uuid = %L WHERE tenant_datasource_id_uuid IS NULL', t, default_tpd::text);
    RAISE NOTICE 'Assigned default_tpd to remaining rows in %', t;
END LOOP; END $$ LANGUAGE plpgsql;

-- 4) Add authoritative FK constraints on the canonical uuid columns (NOT VALID first)
DO $$ DECLARE t TEXT; BEGIN
  FOR t IN SELECT unnest(string_to_array('catalog_node,catalog_node_type,tenant_chart', ',')) LOOP
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=t AND column_name='tenant_datasource_id_uuid') THEN
      BEGIN
        EXECUTE format('ALTER TABLE public.%I ADD CONSTRAINT %I FOREIGN KEY (tenant_datasource_id_uuid) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE NOT VALID', t, t||'_tenant_product_datasource_uuid_fk');
      EXCEPTION WHEN duplicate_object THEN NULL; WHEN others THEN RAISE NOTICE 'Could not add uuid FK on %: %', t, SQLERRM; END;
      BEGIN
        EXECUTE format('ALTER TABLE public.%I VALIDATE CONSTRAINT %I', t, t||'_tenant_product_datasource_uuid_fk');
      EXCEPTION WHEN others THEN RAISE NOTICE 'Validate deferred for %: %', t, SQLERRM; END;
    END IF;
  END LOOP;
END $$ LANGUAGE plpgsql;

-- 5) Verification: count remaining NULL canonical uuids
SELECT 'post_backfill' AS phase,
  COALESCE((SELECT COUNT(*) FROM public.catalog_node WHERE EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node' AND column_name='tenant_datasource_id_uuid') AND tenant_datasource_id_uuid IS NULL), 0) AS catalog_node_nulls,
  COALESCE((SELECT COUNT(*) FROM public.catalog_node_type WHERE EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='catalog_node_type' AND column_name='tenant_datasource_id_uuid') AND tenant_datasource_id_uuid IS NULL), 0) AS catalog_node_type_nulls,
  COALESCE((SELECT COUNT(*) FROM public.tenant_chart WHERE EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='tenant_chart' AND column_name='tenant_datasource_id_uuid') AND tenant_datasource_id_uuid IS NULL), 0) AS tenant_chart_nulls;

-- 6) OPTIONAL DESTRUCTIVE CLEANUP (MANUAL STEP)
-- The following block performs DROP COLUMN and renames. It is COMMENTED OUT intentionally.
-- Uncomment and run only during a maintenance window with a verified backup.

/*
DO $$ DECLARE t TEXT; BEGIN
  -- Drop any constraints still referring to legacy columns (example placeholders)
  -- ALTER TABLE public.catalog_node_type DROP CONSTRAINT IF EXISTS catalog_node_type_tenant_product_datasource_fk;
  -- ALTER TABLE public.tenant_chart DROP CONSTRAINT IF EXISTS tenant_chart_tenant_product_datasource_fk;

  -- Drop legacy columns (may acquire ACCESS EXCLUSIVE lock)
  EXECUTE 'ALTER TABLE public.catalog_node DROP COLUMN IF EXISTS tenant_datasource_id_old';
  EXECUTE 'ALTER TABLE public.catalog_node_type DROP COLUMN IF EXISTS tenant_datasource_id_old';
  EXECUTE 'ALTER TABLE public.tenant_chart DROP COLUMN IF EXISTS tenant_datasource_id_old';

  -- If you previously left tenant_datasource_id_uuid in place and DID NOT rename it, you can rename now:
  -- EXECUTE 'ALTER TABLE public.catalog_node RENAME COLUMN tenant_datasource_id_uuid TO tenant_datasource_id';
  -- EXECUTE 'ALTER TABLE public.catalog_node_type RENAME COLUMN tenant_datasource_id_uuid TO tenant_datasource_id';
  -- EXECUTE 'ALTER TABLE public.tenant_chart RENAME COLUMN tenant_datasource_id_uuid TO tenant_datasource_id';

  -- Drop compatibility column on tenant_product_datasource if desired
  EXECUTE 'ALTER TABLE public.tenant_product_datasource DROP COLUMN IF EXISTS id_text';
END $$ LANGUAGE plpgsql;
*/

-- Final checks: show FKs referencing tenant_product_datasource
SELECT conname, conrelid::regclass AS table, pg_get_constraintdef(oid) AS def FROM pg_constraint WHERE confrelid = 'public.tenant_product_datasource'::regclass ORDER BY conname;

-- End of file
