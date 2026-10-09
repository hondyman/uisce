-- Migration: 20261225_010_retire_bundle_explorer.up.sql
-- Retires the Bundle Explorer page (bundle-explorer) and the semantic bundle authoring routes
-- (/fabric/bundles, /fabric/bundles/create, /fabric/bundles/:bundleId/edit).
--
-- Scope note: this removes the bundle EXPLORER and AUTHORING surface only. Views Catalog
-- (features/views/pages/BundleEditor.tsx), private-markets bundles, claim bundles and the
-- artifact bundle tables are separate features and are deliberately untouched.
--
-- The bundle-explorer page lived only in the gold-copy tenant; other tenants inherit it through
-- `is_core = true AND tenant_id = <gold>` (page_studio_handler.go, navigation_menu_handler.go),
-- so deleting the core row retires it everywhere. Deletes below are written tenant-wide to clear
-- whatever state exists.
--
-- Idempotent: every statement is keyed on a stable identifier, so a partial re-run is safe.
-- Sorts after 008, which no longer seeds these rows.

-- 1. Menu nodes first, so no navigation node is left pointing at a deleted page. Two nodes
--    targeted the page: the Catalog "Bundle Explorer" entry and the Build "Bundles" entry,
--    which opened the same page via /fabric/bundles.
-- navigation_menu_nodes has no target_route column (none of the migrations or the schema snapshot add
-- one), so the match is on the node keys and the page slug only.
DELETE FROM navigation_menu_nodes
 WHERE node_key IN ('catalog-config-bundle-explorer', 'build-models-bundles')
    OR target_page_key = 'bundle-explorer';

-- 2. Route aliases across every tenant, covering both the /bundle-explorer and /fabric/bundles
--    paths that resolved to this one page.
DELETE FROM route_aliases
 WHERE page_key = 'bundle-explorer'
    OR path IN ('/bundle-explorer', '/fabric/bundles');

-- 3. Tenant adoption / switch-off records referencing the core page. Defensive: the table is
--    empty today, but a tenant could have adopted the page before this migration ran.
DELETE FROM core_object_adoption
 WHERE object_type = 'page'
   AND core_object_id IN (SELECT id FROM page_definitions WHERE slug = 'bundle-explorer');

-- 4. The core page itself. Retires it for all tenants via gold-copy inheritance.
DELETE FROM page_definitions WHERE slug = 'bundle-explorer';