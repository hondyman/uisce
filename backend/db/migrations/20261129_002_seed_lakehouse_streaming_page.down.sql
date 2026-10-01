-- Migration 20261129_002_seed_lakehouse_streaming_page.down.sql
DELETE FROM public.navigation_menu_nodes WHERE id = 'e81a3d02-0001-7000-8000-000000000200';
DELETE FROM public.page_definitions WHERE id = '018f9d02-0001-7000-8000-000000000200';
