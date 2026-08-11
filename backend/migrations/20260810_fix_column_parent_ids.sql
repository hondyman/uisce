-- Fix parent_id for column nodes to point to their parent table's UUID
UPDATE public.catalog_node col
SET parent_id = tbl.id
FROM public.catalog_node tbl
JOIN public.catalog_node_type tbl_type ON tbl.node_type_id = tbl_type.id
WHERE tbl_type.catalog_type_name = 'table'
  AND col.qualified_path LIKE (tbl.qualified_path || '/%')
  AND col.parent_id IS DISTINCT FROM tbl.id;
