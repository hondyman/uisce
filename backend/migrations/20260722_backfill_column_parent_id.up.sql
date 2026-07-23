BEGIN;

-- Backfill parent_id on column nodes from PARENT_OF edges
-- The frontend matches columns to tables via parent_id, but seed data uses edges instead
UPDATE catalog_node c
SET parent_id = e.target_node_id
FROM catalog_node c2
JOIN catalog_edge e ON e.source_node_id = c2.id
JOIN catalog_node_type nt ON c2.node_type_id = nt.id
WHERE c.id = c2.id
  AND c.id IN (
      SELECT source_node_id FROM catalog_edge
      WHERE edge_type_id IN (
          SELECT id FROM catalog_edge_types
          WHERE edge_type_name = 'PARENT_OF'
          AND (tenant_id = '*' OR tenant_id = '00000000-0000-0000-0000-000000000000')
      )
  )
  AND nt.catalog_type_name = 'column'
  AND c.parent_id IS NULL;

COMMIT;
