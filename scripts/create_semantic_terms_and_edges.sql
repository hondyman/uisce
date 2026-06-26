-- Usage:
-- psql -d postgres://postgres:postgres@localhost:5432/alpha -v tenant_id='00000000-0000-0000-0000-000000000000' -v tenant_datasource_id='11111111-1111-1111-1111-111111111111' -f scripts/create_semantic_terms_and_edges.sql
--
-- This script: for each column node in catalog_node (node_type_id = a64c1011...) it
-- 1) finds or creates a semantic term node (node_type_id = 820b942a-...)
-- 2) inserts a mapping edge between the semantic term and the column using
--    edge_type_id = 97d82101-2b84-47a6-9ec0-f930fe389c3c
--
-- Note: This script is intentionally idempotent (uses ON CONFLICT DO NOTHING).
-- It uses UPPER(node_name) to find existing terms (same as backend service).

\set semantic_term_node_type '820b942a-9c9e-4abc-acdc-84616db33098'
\set column_node_type 'a64c1011-16e8-4ddf-b447-363bf8e15c9a'
\set edge_type '97d82101-2b84-47a6-9ec0-f930fe389c3c'

\set tenant_datasource_id ''
\set tenant_id ''

-- Build a list of candidate terms from columns (case-insensitive node_name fallback to qualified_path last component)
CREATE TEMP TABLE IF NOT EXISTS tmp_columns AS
SELECT
  id AS column_node_id,
  tenant_datasource_id,
  tenant_id,
  UPPER(COALESCE(NULLIF(node_name,''),
    split_part(qualified_path, '/', array_length(string_to_array(qualified_path,'/'),1)))) AS term_name,
  properties->>'data_type' AS data_type
FROM catalog_node
WHERE node_type_id = :'column_node_type'
  AND (:'tenant_datasource_id' = '' OR tenant_datasource_id::text = :'tenant_datasource_id')
  AND (:'tenant_id' = '' OR tenant_id::text = :'tenant_id');

CREATE TEMP TABLE IF NOT EXISTS tmp_unique_terms AS
SELECT DISTINCT tenant_datasource_id, tenant_id, term_name, data_type
FROM tmp_columns;

-- unique_terms equivalent is tmp_unique_terms

INSERT INTO catalog_node (id, tenant_datasource_id, node_type_id, node_name, qualified_path, tenant_id, created_at, updated_at, properties)
SELECT gen_random_uuid(), ut.tenant_datasource_id, :'semantic_term_node_type', ut.term_name, CONCAT('/semantic/', ut.term_name), ut.tenant_id, now(), now(), jsonb_build_object('data_type', COALESCE(ut.data_type, ''))
FROM tmp_unique_terms ut
ON CONFLICT (tenant_datasource_id, node_type_id, qualified_path) DO NOTHING;
 

INSERT INTO catalog_edge (id, tenant_datasource_id, source_node_id, target_node_id, edge_type_id, relationship_type, tenant_id, created_at, updated_at)
SELECT gen_random_uuid(), c.tenant_datasource_id, cn.id, c.column_node_id, :'edge_type', 'mapped_to', c.tenant_id, now(), now()
FROM tmp_columns c
JOIN catalog_node cn ON cn.tenant_datasource_id = c.tenant_datasource_id
  AND cn.node_type_id = :'semantic_term_node_type'
  AND UPPER(cn.node_name) = c.term_name
WHERE NOT EXISTS (
  SELECT 1 FROM catalog_edge ce
  WHERE ce.tenant_datasource_id = c.tenant_datasource_id
    AND ce.source_node_id = cn.id
    AND ce.target_node_id = c.column_node_id
    AND ce.edge_type_id = :'edge_type'
);

\echo 'Complete.'
