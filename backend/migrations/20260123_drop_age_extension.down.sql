-- Rollback: Re-enable Apache AGE extension
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'age') THEN
    BEGIN
      CREATE EXTENSION IF NOT EXISTS age;
      PERFORM pg_catalog.loada 'age' FROM pg_catalog.pg_namespace; -- best-effort LOAD
      SET search_path = ag_catalog, "$user", public;
      -- Recreate the lineage graph if function available
      IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'create_graph') THEN
        PERFORM create_graph('semantic_lineage');
      END IF;
    EXCEPTION WHEN OTHERS THEN
      RAISE NOTICE 'Could not enable AGE extension: %', SQLERRM;
    END;
  ELSE
    RAISE NOTICE 'AGE extension not available on this server - skipping';
  END IF;
END
$do$;
