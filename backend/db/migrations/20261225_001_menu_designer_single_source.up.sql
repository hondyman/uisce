-- Wave 1 of the navigation restructure: Menu Designer (navigation_menu_nodes) and
-- Page Designer (page_definitions) become the only source of menus and pages.
-- DB-only; no frontend change. Idempotent.

-- 1. Schema ---------------------------------------------------------------
ALTER TABLE public.navigation_menu_nodes
    ADD COLUMN IF NOT EXISTS required_capability VARCHAR(100),
    ADD COLUMN IF NOT EXISTS hidden BOOLEAN NOT NULL DEFAULT FALSE;

-- Pages reachable only by drill-through / tests: excluded from the menu audit.
ALTER TABLE public.page_definitions
    ADD COLUMN IF NOT EXISTS menu_hidden BOOLEAN NOT NULL DEFAULT FALSE;

-- Legacy coded URLs -> page key, so bookmarks survive removal of coded routes.
CREATE TABLE IF NOT EXISTS public.route_aliases (
    tenant_id  UUID         NOT NULL,
    path       VARCHAR(300) NOT NULL,
    page_key   VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ  DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (tenant_id, path)
);

-- 2. Top-level groups (gold-copy tenant) ---------------------------------
INSERT INTO public.navigation_menu_nodes (id, tenant_id, parent_id, node_key, label, display_order, required_entitlement)
VALUES
 ('e81a3d02-0001-7000-8000-000000000601','99e99e99-99e9-49e9-89e9-99e99e99e999',NULL,'consume','Consume',10,'BASE_USER'),
 ('e81a3d02-0001-7000-8000-000000000602','99e99e99-99e9-49e9-89e9-99e99e99e999',NULL,'catalog','Catalog',30,'BASE_USER'),
 ('e81a3d02-0001-7000-8000-000000000603','99e99e99-99e9-49e9-89e9-99e99e99e999',NULL,'build','Build',40,'BASE_USER'),
 ('e81a3d02-0001-7000-8000-000000000604','99e99e99-99e9-49e9-89e9-99e99e99e999',NULL,'operations','Operations',50,'BASE_USER'),
 ('e81a3d02-0001-7000-8000-000000000605','99e99e99-99e9-49e9-89e9-99e99e99e999',NULL,'intelligence','Intelligence',60,'BASE_USER'),
 ('e81a3d02-0001-7000-8000-000000000606','99e99e99-99e9-49e9-89e9-99e99e99e999',NULL,'platform','Platform',70,'BASE_USER')
ON CONFLICT (tenant_id, node_key) DO NOTHING;

UPDATE public.navigation_menu_nodes SET display_order = 20
 WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key='master-data' AND parent_id IS NULL;

-- Orders live under Consume; System lives under Platform.
UPDATE public.navigation_menu_nodes SET parent_id='e81a3d02-0001-7000-8000-000000000601', display_order=30
 WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key='orders';
UPDATE public.navigation_menu_nodes SET parent_id='e81a3d02-0001-7000-8000-000000000606', display_order=90
 WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key='system';

-- Resolve duplicate display_order inside Master Data (compliance-hub / lakehouse-streaming both 15).
UPDATE public.navigation_menu_nodes SET display_order = 16
 WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999' AND node_key='lakehouse-streaming';

-- 3. Publish the draft pages (decision: published) -----------------------
UPDATE public.page_definitions SET status='published', updated_at=NOW()
 WHERE tenant_id='99e99e99-99e9-49e9-89e9-99e99e99e999'
   AND slug IN ('order-list-tl48','order-detail-tj2e','portfolio-list-9qz0','portfolio-detail-vel6');

-- 4. Place existing designed pages that had no node ----------------------
INSERT INTO public.navigation_menu_nodes (id, tenant_id, parent_id, node_key, label, target_page_key, display_order, required_entitlement)
VALUES
 ('e81a3d02-0001-7000-8000-000000000611','99e99e99-99e9-49e9-89e9-99e99e99e999','e81a3d02-0001-7000-8000-000000000603','cubes-catalog','Cubes','cubes-catalog',10,'BASE_USER'),
 ('e81a3d02-0001-7000-8000-000000000612','99e99e99-99e9-49e9-89e9-99e99e99e999','e81a3d02-0001-7000-8000-000000000604','schedules','Schedules','schedules',10,'BASE_USER'),
 ('e81a3d02-0001-7000-8000-000000000613','99e99e99-99e9-49e9-89e9-99e99e99e999','e81a3d02-0001-7000-8000-000000000601','portfolio-list-9qz0','Portfolios','portfolio-list-9qz0',20,'BASE_USER')
ON CONFLICT (tenant_id, node_key) DO UPDATE SET
    label=EXCLUDED.label, target_page_key=EXCLUDED.target_page_key,
    parent_id=EXCLUDED.parent_id, display_order=EXCLUDED.display_order;

-- 5. Drill-through / test pages: intentionally not on the menu ------------
UPDATE public.page_definitions SET menu_hidden = TRUE
 WHERE slug IN ('cube-designer','data-pipeline-editor','portfolio-detail-vel6','order-detail-tj2e')
    OR slug LIKE 'rls-fixture-%';
