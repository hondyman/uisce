-- Rollback migration 010
--
-- This does NOT resurrect the Bundle Explorer page. Doing so would leave a published core page
-- pointing at domain component `fabric.BundleExplorer` and the /fabric/bundles routes at
-- `BundleEditor` — React modules that this change deletes. A page whose component is gone renders
-- blank, which is worse than an absent page.
--
-- To fully restore the feature, restore the frontend modules first:
--   frontend/src/components/BundleExplorer.tsx
--   frontend/src/pages/bundles/*            (authoring screens; bp-designer is unrelated and kept)
--   the fabric.BundleExplorer registration in frontend/src/features/fabric/studio.tsx
--   the bundleExplorerBlueprint entry + registration in blueprints/fabricPages.ts and blueprints/index.ts
--   the routes in frontend/src/AppRoutes.tsx (root "", fabric/bundles/create, fabric/bundles/:bundleId/edit)
-- then re-insert the page_definitions row and route_aliases rows by hand.
--
-- What this rollback does do is clear any rows that a re-run of 008 may have re-seeded, so the
-- database matches the removed code rather than dangling on it.
-- No target_route column exists on navigation_menu_nodes (see the up migration).
DELETE FROM navigation_menu_nodes
 WHERE node_key IN ('catalog-config-bundle-explorer', 'build-models-bundles')
    OR target_page_key = 'bundle-explorer';

DELETE FROM route_aliases
 WHERE page_key = 'bundle-explorer'
    OR path IN ('/bundle-explorer', '/fabric/bundles');

DELETE FROM page_definitions WHERE slug = 'bundle-explorer';