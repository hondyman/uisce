DELETE FROM public.navigation_menu_nodes WHERE id IN (
 'e81a3d02-0001-7000-8000-000000000611','e81a3d02-0001-7000-8000-000000000612','e81a3d02-0001-7000-8000-000000000613');

UPDATE public.navigation_menu_nodes SET parent_id=NULL, display_order=0
 WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key='orders';
UPDATE public.navigation_menu_nodes SET parent_id=NULL, display_order=900
 WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key='system';
UPDATE public.navigation_menu_nodes SET display_order=0
 WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key='master-data';
UPDATE public.navigation_menu_nodes SET display_order=15
 WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key='lakehouse-streaming';

DELETE FROM public.navigation_menu_nodes WHERE id IN (
 'e81a3d02-0001-7000-8000-000000000601','e81a3d02-0001-7000-8000-000000000602','e81a3d02-0001-7000-8000-000000000603',
 'e81a3d02-0001-7000-8000-000000000604','e81a3d02-0001-7000-8000-000000000605','e81a3d02-0001-7000-8000-000000000606');

DROP TABLE IF EXISTS public.route_aliases;
ALTER TABLE public.page_definitions DROP COLUMN IF EXISTS menu_hidden;
ALTER TABLE public.navigation_menu_nodes DROP COLUMN IF EXISTS hidden, DROP COLUMN IF EXISTS required_capability;
-- Published status of the order-*/portfolio-* pages is intentionally not reverted.
