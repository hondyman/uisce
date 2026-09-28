-- page_definition_overlays was superseded by core_object_adoption
-- (20261116_001, which copied its rows over as extensions). Nothing reads
-- it since the Page Studio core-page lifecycle shipped.
DROP TABLE IF EXISTS public.page_definition_overlays;
