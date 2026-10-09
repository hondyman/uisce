-- Rollback migration 009
--
-- Restores the Operations menu node and leaves /core/workflow-designer pointing at the
-- replacement Process Designer. It deliberately does NOT resurrect the retired
-- core-workflow-designer page or its /p/ URL redirect: the frontend redirect
-- (LEGACY_PAGE_SLUGS in frontend/src/AppRoutes.tsx) and the page removal are a single
-- decision, so rolling back the data layer must not half-restore a page whose React component
-- is gone. Rolling this back on its own is safe — the alias still resolves to a live page.

DO $$
DECLARE
    gold_tenant UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    v_parent_id UUID;
BEGIN
    -- Restore the menu node under the same Operations parent the original seed used, pointing
    -- at Process Designer. Guarded so re-running does not create a duplicate.
    SELECT id INTO v_parent_id
      FROM navigation_menu_nodes
     WHERE tenant_id = gold_tenant
       AND node_key = 'operations-workflows';

    IF v_parent_id IS NULL THEN
        RAISE NOTICE 'parent node operations-workflows not found for gold tenant; skipping menu node restore';
    ELSE
        INSERT INTO navigation_menu_nodes
            (tenant_id, parent_id, node_key, label, icon, target_page_key,
             display_order, required_entitlement, created_at)
        VALUES
            (gold_tenant, v_parent_id, 'operations-workflows-workflow-designer-legacy',
             'Workflow Designer (Legacy)', 'GitMerge', 'client-workflow-studio',
             60, 'BASE_USER', NOW())
        ON CONFLICT (tenant_id, node_key) DO UPDATE
        SET label = EXCLUDED.label,
            target_page_key = EXCLUDED.target_page_key;
    END IF;
END $$;