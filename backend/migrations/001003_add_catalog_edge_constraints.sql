DO $$
BEGIN
  IF (SELECT COUNT(*) FROM information_schema.columns WHERE table_name='catalog_edge' AND column_name IN ('edge_type_id','source_node_id','target_node_id')) = 3 THEN

    -- First, clean up any orphaned references
    -- Delete edges with invalid edge_type_id
    DELETE FROM catalog_edge 
    WHERE edge_type_id IS NOT NULL 
      AND edge_type_id NOT IN (SELECT id FROM catalog_edge_types);

    -- Delete edges with invalid source_node_id
    DELETE FROM catalog_edge 
    WHERE source_node_id NOT IN (SELECT id FROM catalog_node);

    -- Delete edges with invalid target_node_id
    DELETE FROM catalog_edge 
    WHERE target_node_id NOT IN (SELECT id FROM catalog_node);

    -- Now add the foreign key constraints
    -- Constraint for edge_type_id
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_catalog_edge_edge_type') THEN
      ALTER TABLE catalog_edge
      ADD CONSTRAINT fk_catalog_edge_edge_type
      FOREIGN KEY (edge_type_id)
      REFERENCES catalog_edge_types(id)
      ON DELETE CASCADE;
    END IF;

    -- Constraint for source_node_id
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_catalog_edge_source_node') THEN
      ALTER TABLE catalog_edge
      ADD CONSTRAINT fk_catalog_edge_source_node
      FOREIGN KEY (source_node_id)
      REFERENCES catalog_node(id)
      ON DELETE CASCADE;
    END IF;

    -- Constraint for target_node_id
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_catalog_edge_target_node') THEN
      ALTER TABLE catalog_edge
      ADD CONSTRAINT fk_catalog_edge_target_node
      FOREIGN KEY (target_node_id)
      REFERENCES catalog_node(id)
      ON DELETE CASCADE;
    END IF;

    -- Add indexes for better query performance
    CREATE INDEX IF NOT EXISTS idx_catalog_edge_edge_type_id ON catalog_edge(edge_type_id);
    CREATE INDEX IF NOT EXISTS idx_catalog_edge_source_node_id ON catalog_edge(source_node_id);
    CREATE INDEX IF NOT EXISTS idx_catalog_edge_target_node_id ON catalog_edge(target_node_id);

  ELSE
    RAISE NOTICE 'Skipping 001003: required catalog_edge *_id columns not present yet';
  END IF;
END;
$$;
