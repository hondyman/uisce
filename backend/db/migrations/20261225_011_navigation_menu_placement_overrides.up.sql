-- Migration: 20261225_011_navigation_menu_placement_overrides.up.sql
-- Per-tenant hide for a gold-copy menu entry (navigation_menu_nodes).
--
-- Gold-copy entries are read-only for every other tenant, so a tenant that does not want one on its
-- menu records a row here instead of editing or deleting the core row. The row is the whole
-- override: its presence means "hidden for this tenant". The gold row is never written.
--
-- Hiding a folder hides everything under it (applied in navigation_menu_handler.go, applyHidden).
--
-- Idempotent: safe to re-run.

CREATE TABLE IF NOT EXISTS navigation_menu_placement_overrides (
    tenant_id  uuid        NOT NULL,
    node_id    uuid        NOT NULL REFERENCES navigation_menu_nodes(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, node_id)
);

ALTER TABLE navigation_menu_placement_overrides ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
         WHERE schemaname = 'public'
           AND tablename = 'navigation_menu_placement_overrides'
           AND policyname = 'tenant_isolation_nav_menu_overrides'
    ) THEN
        CREATE POLICY tenant_isolation_nav_menu_overrides ON public.navigation_menu_placement_overrides
            USING (tenant_id = public.uisce_get_current_tenant())
            WITH CHECK (tenant_id = public.uisce_get_current_tenant());
    END IF;
END $$;
