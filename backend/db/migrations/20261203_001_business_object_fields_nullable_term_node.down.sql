-- Drops the two partial unique indexes. The column is NOT re-tightened to NOT
-- NULL: rows with a NULL term_node_id may exist by now, and alpha was already
-- nullable before this migration.
DROP INDEX IF EXISTS public.uq_business_object_fields_unbound_name;
DROP INDEX IF EXISTS public.uq_business_object_fields_term;
