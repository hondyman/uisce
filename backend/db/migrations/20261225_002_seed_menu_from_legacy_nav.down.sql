DELETE FROM public.navigation_menu_nodes WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999' AND target_route IS NOT NULL;
DELETE FROM public.navigation_menu_nodes WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key ~ '^(platform|catalog|build|operations|intelligence|consume)-' AND parent_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM public.navigation_menu_nodes c WHERE c.parent_id=navigation_menu_nodes.id);
DELETE FROM public.route_aliases WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999';
ALTER TABLE public.navigation_menu_nodes DROP COLUMN IF EXISTS target_route;
