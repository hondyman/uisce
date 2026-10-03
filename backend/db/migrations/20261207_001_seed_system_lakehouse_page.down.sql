-- 20261207_001_seed_system_lakehouse_page (down)
DELETE FROM public.navigation_menu_nodes WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key = 'system-lakehouse';
DELETE FROM public.page_definitions WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND slug = 'system-lakehouse';
-- The System section may hold other pages; remove it only if it is now empty.
DELETE FROM public.navigation_menu_nodes p
 WHERE p.tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND p.node_key = 'system'
   AND NOT EXISTS (SELECT 1 FROM public.navigation_menu_nodes c WHERE c.parent_id = p.id);
