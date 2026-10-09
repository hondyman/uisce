-- Migration: 20261225_009_retire_legacy_workflow_designer.up.sql
-- Retires the legacy Workflow Designer (core-workflow-designer) in favour of Process Designer
-- (client-workflow-studio).
--
-- Nothing is migrated: the legacy designer's Save handler only devDebug-logged
-- (frontend/src/features/workflow/pages/WorkflowDesignerPage.tsx), so it never persisted a flow,
-- and the replacement carries no legacy data. The 41 rows in bp_process_definition are
-- 2026-09-30 E2E/smoke fixtures written by the /api/bp test paths, not designer output, and are
-- deliberately left untouched.
--
-- The legacy page exists only in the gold-copy tenant; every other tenant inherits it through
-- `is_core = true AND tenant_id = <gold>` in page_studio_handler.go and
-- navigation_menu_handler.go. Deleting the single core row therefore retires it everywhere, so
-- the page/alias deletes below are written tenant-wide to clear whatever state exists.
--
-- Idempotent: every statement is keyed on a stable identifier, so a partial re-run is safe.
-- The runner sorts lexically, so this applies after 007 even though 007 is still pending.

-- 1. Menu node first, so no navigation node can be left pointing at a deleted page.
DELETE FROM navigation_menu_nodes
 WHERE node_key = 'operations-workflows-workflow-designer-legacy'
    OR target_page_key = 'core-workflow-designer';

-- 2. Route aliases across every tenant, not just gold-copy.
DELETE FROM route_aliases WHERE page_key = 'core-workflow-designer';

-- 3. Tenant adoption / switch-off records referencing the core page. Defensive: the table is
--    empty today, but a tenant could have adopted the page before this migration ran.
DELETE FROM core_object_adoption
 WHERE object_type = 'page'
   AND core_object_id IN (SELECT id FROM page_definitions WHERE slug = 'core-workflow-designer');

-- 4. The core page itself. Retires it for all tenants via gold-copy inheritance.
DELETE FROM page_definitions WHERE slug = 'core-workflow-designer';

-- 5. Only now repoint the legacy alias at the replacement, so the old bookmarkable URL
--    (/core/workflow-designer) keeps working instead of 404ing. This must stay after the
--    delete in step 2, which otherwise removes the very row this rewrites.
--    Upsert rather than plain UPDATE: 007 no longer seeds this alias, so on a fresh environment
--    the row does not exist yet and needs creating. The same applies to any environment where
--    step 2 removed it.
DO $$
DECLARE
    v_tenant RECORD;
BEGIN
    FOR v_tenant IN SELECT tenant_id FROM tenants LOOP
        INSERT INTO route_aliases (tenant_id, path, page_key, created_at)
        VALUES (v_tenant.tenant_id, '/core/workflow-designer', 'client-workflow-studio', NOW())
        ON CONFLICT (tenant_id, path) DO UPDATE
        SET page_key = EXCLUDED.page_key;
    END LOOP;
END $$;