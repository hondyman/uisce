-- Scanner MERGE copies edge_type_name onto catalog_edge.relationship_type, but
-- edges inserted before that mapping (or via paths that left the column at
-- its default 'related_to') are invisible to GetBusinessObjectRelationships,
-- which excludes relationship_type = 'related_to'.
-- Only rows whose edge_type is FOREIGN_KEY are updated; descriptive
-- related_to validation edges use a different edge_type_id.

UPDATE public.catalog_edge e
SET relationship_type = 'foreign_key'
FROM public.catalog_edge_types t
WHERE e.edge_type_id = t.id
  AND LOWER(t.edge_type_name) IN ('foreign_key', 'fk')
  AND e.relationship_type = 'related_to';
