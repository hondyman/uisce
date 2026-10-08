-- Migration 20261225_003_seed_catalog_and_platform_pages.down.sql

DELETE FROM public.route_aliases WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND path = '/catalog/node-types';
UPDATE public.navigation_menu_nodes SET target_page_key = NULL, target_route = '/catalog/node-types' WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND target_page_key = 'catalog-node-types';
DELETE FROM public.page_definitions WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND slug = 'catalog-node-types';

DELETE FROM public.route_aliases WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND path = '/catalog/edge-types';
UPDATE public.navigation_menu_nodes SET target_page_key = NULL, target_route = '/catalog/edge-types' WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND target_page_key = 'catalog-edge-types';
DELETE FROM public.page_definitions WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND slug = 'catalog-edge-types';

DELETE FROM public.route_aliases WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND path = '/core/abbreviations';
UPDATE public.navigation_menu_nodes SET target_page_key = NULL, target_route = '/core/abbreviations' WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND target_page_key = 'core-abbreviations';
DELETE FROM public.page_definitions WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND slug = 'core-abbreviations';

