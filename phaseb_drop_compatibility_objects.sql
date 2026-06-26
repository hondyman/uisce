-- phaseb_drop_compatibility_objects.sql
-- Scans all schemas for remaining compatibility objects (tenant_datasource_id_old, id_text, constraints/views referencing legacy names).
-- By default this script only prints the DDL it *would* run. To actually perform the drops/updates, set run := true below.

DO $$
DECLARE
  run boolean := true; -- set to true to execute the DDL (enabled by user request)
  r RECORD;
  stmt text;
  vdef text;
BEGIN
  RAISE NOTICE 'Scanning for tenant_datasource_id_old columns...';
  FOR r IN
    SELECT table_schema, table_name
    FROM information_schema.columns
    WHERE column_name = 'tenant_datasource_id_old'
  LOOP
    stmt := format('ALTER TABLE %I.%I DROP COLUMN IF EXISTS tenant_datasource_id_old', r.table_schema, r.table_name);
    RAISE NOTICE '%', stmt;
    IF run THEN EXECUTE stmt; END IF;
  END LOOP;

  RAISE NOTICE 'Scanning for tenant_product_datasource.id_text columns...';
  FOR r IN
    SELECT table_schema, table_name
    FROM information_schema.columns
    WHERE table_name = 'tenant_product_datasource' AND column_name = 'id_text'
  LOOP
    stmt := format('ALTER TABLE %I.%I DROP COLUMN IF EXISTS id_text', r.table_schema, r.table_name);
    RAISE NOTICE '%', stmt;
    IF run THEN EXECUTE stmt; END IF;
  END LOOP;

  RAISE NOTICE 'Scanning for constraints referencing legacy names...';
  FOR r IN
    SELECT n.nspname AS schema_name, c.relname AS table_name, con.conname
    FROM pg_constraint con
    JOIN pg_class c ON con.conrelid = c.oid
    JOIN pg_namespace n ON c.relnamespace = n.oid
    WHERE pg_get_constraintdef(con.oid) ILIKE '%tenant_datasource_id_old%' OR pg_get_constraintdef(con.oid) ILIKE '%tenant_datasource_id_uuid%'
  LOOP
    stmt := format('ALTER TABLE %I.%I DROP CONSTRAINT IF EXISTS %I', r.schema_name, r.table_name, r.conname);
    RAISE NOTICE '%', stmt;
    IF run THEN EXECUTE stmt; END IF;
  END LOOP;

  RAISE NOTICE 'Scanning for views referencing legacy column names...';
  FOR r IN
    SELECT schemaname, viewname, definition
    FROM pg_views
    WHERE definition ILIKE '%tenant_datasource_id_old%' OR definition ILIKE '%tenant_datasource_id_uuid%'
  LOOP
    vdef := replace(r.definition, 'tenant_datasource_id_old', 'tenant_datasource_id');
    vdef := replace(vdef, 'tenant_datasource_id_uuid', 'tenant_datasource_id');
    stmt := format('CREATE OR REPLACE VIEW %I.%I AS %s', r.schemaname, r.viewname, vdef);
    RAISE NOTICE '%', stmt;
    IF run THEN EXECUTE stmt; END IF;
  END LOOP;

  RAISE NOTICE 'Done. Review the above statements. To apply them, re-run this script with run := true.';
END $$ LANGUAGE plpgsql;

-- End of file
