DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'age') THEN
    BEGIN
      CREATE EXTENSION IF NOT EXISTS age;
    EXCEPTION WHEN OTHERS THEN
      RAISE NOTICE 'CREATE EXTENSION age failed: %', SQLERRM;
    END;

    BEGIN
      EXECUTE 'LOAD ''age''';
    EXCEPTION WHEN OTHERS THEN
      RAISE NOTICE 'LOAD age failed: %', SQLERRM;
    END;

    PERFORM set_config('search_path', 'ag_catalog, "$user", public', true);

    BEGIN
      PERFORM create_graph('semantic_lineage');
    EXCEPTION WHEN OTHERS THEN
      RAISE NOTICE 'create_graph failed or already exists: %', SQLERRM;
    END;

    BEGIN
      -- Labels
      PERFORM create_vlabel('semantic_lineage', 'BO');
      PERFORM create_vlabel('semantic_lineage', 'BOField');
      PERFORM create_vlabel('semantic_lineage', 'Calc');
      PERFORM create_vlabel('semantic_lineage', 'PreAgg');
      PERFORM create_vlabel('semantic_lineage', 'Table');
      PERFORM create_vlabel('semantic_lineage', 'Column');
      PERFORM create_vlabel('semantic_lineage', 'EntitlementPolicy');
      PERFORM create_vlabel('semantic_lineage', 'ASOOptimization');
      PERFORM create_vlabel('semantic_lineage', 'ChangeSet');
      PERFORM create_vlabel('semantic_lineage', 'Tenant');

      -- Edge labels
      PERFORM create_elabel('semantic_lineage', 'DEPENDS_ON');
      PERFORM create_elabel('semantic_lineage', 'DERIVED_FROM');
      PERFORM create_elabel('semantic_lineage', 'GOVERNED_BY');
      PERFORM create_elabel('semantic_lineage', 'OPTIMIZED_BY');
      PERFORM create_elabel('semantic_lineage', 'BELONGS_TO');
      PERFORM create_elabel('semantic_lineage', 'OVERRIDES');
      PERFORM create_elabel('semantic_lineage', 'INCLUDED_IN');
    EXCEPTION WHEN OTHERS THEN
      RAISE NOTICE 'create_vlabel/create_elabel failed: %', SQLERRM;
    END;

  ELSE
    RAISE NOTICE 'Extension age not available; skipping AGE graph creation';
  END IF;
END
$do$;
