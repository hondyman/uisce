DROP VIEW IF EXISTS public.attribute_def_effective;

DROP POLICY IF EXISTS attr_def_write ON public.attribute_def;
DROP POLICY IF EXISTS attr_def_read ON public.attribute_def;

DROP TABLE IF EXISTS public.attribute_def;

-- Leave catalog_edge_types / custom_field node type in place (may be referenced).
-- Safe cast functions are left in place (idempotent helpers).
