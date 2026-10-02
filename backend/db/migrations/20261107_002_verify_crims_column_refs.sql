-- Verification queries for CRIMS column reference catalog
-- Run these after POST /api/catalog/admin/sync-crims to validate the migration

-- ============================================================
-- V1: FK constraints without COLUMN_REFERENCES_COLUMN edges
-- Expected: 0 rows (all FKs should have edges)
-- ============================================================
-- V1: Orphan FKs — FK constraints in pg_constraint that have no corresponding edge
SELECT
    con.conname AS constraint_name,
    sn.nspname AS source_schema,
    sc.relname AS source_table,
    sa.attname AS source_column,
    tn.nspname AS target_schema,
    tc.relname AS target_table,
    ta.attname AS target_column
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_namespace sn ON sn.oid = con.connamespace
JOIN pg_catalog.pg_class sc ON sc.oid = con.conrelid
JOIN pg_catalog.pg_namespace tn ON tn.oid = tc.relnamespace
JOIN pg_catalog.pg_class tc ON tc.oid = con.confrelid
CROSS JOIN LATERAL unnest(con.conkey, con.confkey) WITH ORDINALITY AS k(src_attnum, tgt_attnum, ord)
JOIN pg_catalog.pg_attribute sa ON sa.attrelid = con.conrelid AND sa.attnum = k.src_attnum
JOIN pg_catalog.pg_attribute ta ON ta.attrelid = con.confrelid AND ta.attnum = k.tgt_attnum
WHERE con.contype = 'f'
  AND sn.nspname IN ('mdm', 'ref', 'orm')
  AND NOT EXISTS (
      SELECT 1 FROM catalog_edge ce
      JOIN catalog_edge_types cet ON cet.id = ce.edge_type_id
      WHERE ce.edge_type_id = 'c0c1c2c3-d4e5-f6a7-b8c9-d0e1f2a3b4c5'::uuid
        AND ce.properties->>'constraint_name' = con.conname
  )
ORDER BY sn.nspname, sc.relname, sa.attnum;

-- ============================================================
-- V2: Edges pointing at non-existent column nodes
-- Expected: 0 rows (no broken edges)
-- ============================================================
SELECT
    ce.id AS edge_id,
    cn_src.qualified_path AS source_path,
    ce.properties->>'constraint_name' AS constraint_name,
    CASE
        WHEN cn_src.id IS NULL THEN 'source_not_found'
        WHEN cn_tgt.id IS NULL THEN 'target_not_found'
    END AS issue
FROM catalog_edge ce
JOIN catalog_edge_types cet ON cet.id = ce.edge_type_id
LEFT JOIN catalog_node cn_src ON cn_src.id = ce.source_node_id
LEFT JOIN catalog_node cn_tgt ON cn_tgt.id = ce.target_node_id
WHERE cet.edge_type_name = 'COLUMN_REFERENCES_COLUMN'
  AND ce.properties->>'reference_kind' = 'FK_UUID'
  AND (cn_src.id IS NULL OR cn_tgt.id IS NULL);

-- ============================================================
-- V3: Count reconciliation — DB FKs vs catalog edges
-- Expected: reconciled = true (counts match)
-- ============================================================
WITH db_fks AS (
    SELECT count(DISTINCT con.conname) AS n
    FROM pg_catalog.pg_constraint con
    JOIN pg_catalog.pg_namespace sn ON sn.oid = con.connamespace
    WHERE con.contype = 'f'
      AND sn.nspname IN ('mdm', 'ref', 'orm')
),
catalog_edges AS (
    SELECT count(*) AS n
    FROM catalog_edge ce
    JOIN catalog_edge_types cet ON cet.id = ce.edge_type_id
    WHERE cet.edge_type_name = 'COLUMN_REFERENCES_COLUMN'
      AND ce.properties->>'reference_kind' = 'FK_UUID'
),
catalog_nodes AS (
    SELECT
        count(CASE WHEN node_type = 'TABLE' THEN 1 END) AS table_nodes,
        count(CASE WHEN node_type = 'ATTRIBUTE' THEN 1 END) AS column_nodes
    FROM catalog_node
    WHERE qualified_path LIKE 'crims.%'
)
SELECT
    db_fks.n AS db_unique_fk_count,
    catalog_edges.n AS catalog_edge_count,
    catalog_nodes.table_nodes AS table_nodes_created,
    catalog_nodes.column_nodes AS column_nodes_created,
    db_fks.n = catalog_edges.n AS fk_edges_reconciled
FROM db_fks, catalog_edges, catalog_nodes;

-- ============================================================
-- V4: Orphan columns — columns named *_id that have no outgoing FK edge
-- These are candidates for FK_CODE or ENUM reference kinds in v2
-- ============================================================
SELECT
    cn.qualified_path,
    cn.properties
FROM catalog_node cn
JOIN catalog_node_type cnt ON cnt.id = cn.node_type_id
WHERE cn.qualified_path LIKE 'crims.mdm.%'
  AND cn.node_type = 'ATTRIBUTE'
  AND cn.qualified_path ~ '/\w+_id$'
  AND NOT EXISTS (
      SELECT 1 FROM catalog_edge ce
      JOIN catalog_edge_types cet ON cet.id = ce.edge_type_id
      WHERE ce.source_node_id = cn.id
        AND cet.edge_type_name = 'COLUMN_REFERENCES_COLUMN'
  )
ORDER BY cn.qualified_path;

-- ============================================================
-- Summary: Quick count check
-- ============================================================
SELECT
    (SELECT count(*) FROM pg_catalog.pg_constraint con
     JOIN pg_catalog.pg_namespace sn ON sn.oid = con.connamespace
     WHERE con.contype = 'f' AND sn.nspname IN ('mdm', 'ref', 'orm')) AS total_db_fk_constraints,
    (SELECT count(*) FROM catalog_edge ce
     JOIN catalog_edge_types cet ON cet.id = ce.edge_type_id
     WHERE cet.edge_type_name = 'COLUMN_REFERENCES_COLUMN') AS total_column_ref_edges,
    (SELECT count(*) FROM catalog_node WHERE qualified_path LIKE 'crims.%' AND node_type = 'TABLE') AS total_tables,
    (SELECT count(*) FROM catalog_node WHERE qualified_path LIKE 'crims.%' AND node_type = 'ATTRIBUTE') AS total_columns;
