-- 20261226_002_seed_core_domains_page (down)
-- Removes the gold-copy core-domains page. The hand-built route it replaced is not restored.
DELETE FROM public.page_definitions WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND slug = 'core-domains';
