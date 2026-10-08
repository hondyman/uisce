-- 20261006_001_seed_lakehouse_status_page (down)
DELETE FROM public.navigation_menu_nodes WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key = 'lakehouse-status';
DELETE FROM public.page_definitions WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND slug = 'lakehouse-status';
