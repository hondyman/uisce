-- Migration 20261118_003_seed_mdm_source_scoring_page.down.sql
DELETE FROM public.navigation_menu_nodes
WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key = 'mdm-source-scoring';

DELETE FROM public.page_definitions
WHERE tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999' AND slug = 'mdm-source-scoring';
