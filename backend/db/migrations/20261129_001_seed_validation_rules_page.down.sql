-- Migration 20261129_001_seed_validation_rules_page.down.sql
-- Removes the core Page Studio page definition and navigation menu node for Validation Rules

DELETE FROM public.navigation_menu_nodes
WHERE node_key = 'validation-rules'
  AND tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999';

DELETE FROM public.page_definitions
WHERE slug = 'validation-rules'
  AND tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999';
