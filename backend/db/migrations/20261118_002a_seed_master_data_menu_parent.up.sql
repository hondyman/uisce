-- 20261118_002a_seed_master_data_menu_parent.up.sql
-- Bridge migration: ensure the gold-copy "Master Data" navigation menu node exists.
--
-- 20261118_003_seed_mdm_source_scoring_page, 20261129_001_seed_validation_rules_page
-- and 20261129_002_seed_lakehouse_streaming_page each insert a menu node whose
-- parent_id is this node's id (4ef1676b-...). No migration or schema snapshot
-- ever created it: it exists on alpha only because it was inserted there directly
-- (2026-09-28). On any freshly built database those three migrations therefore
-- fail with navigation_menu_nodes_parent_id_fkey (23503), which fails `migrate up`
-- and the gated-tests / integration CI jobs that bootstrap a schema.
--
-- Numbering: sorts AFTER 20261118_002 and BEFORE 20261118_003, the first migration
-- that needs the parent. Same bridge pattern as 20261016_002a_seed_gold_copy_tenant.
--
-- Blast radius on alpha: none. The row below is column-for-column the row alpha
-- already has (same id, tenant, node_key, label, display_order, entitlement), so
-- ON CONFLICT DO NOTHING makes this a no-op there; it only creates the row where
-- it is missing. Existing migrations are not edited, so applied checksums are
-- unchanged.

INSERT INTO public.navigation_menu_nodes (
    id, tenant_id, parent_id, node_key, label, target_page_key, display_order, required_entitlement
) VALUES (
    '4ef1676b-e9aa-4012-9e44-bba9091e962d',
    '99e99e99-99e9-49e9-89e9-99e99e99e999',
    NULL,
    'master-data',
    'Master Data',
    NULL,
    0,
    'BASE_USER'
)
ON CONFLICT DO NOTHING;
