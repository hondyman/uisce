-- How each tenant adopts each core (gold-copy) object. No row = the core
-- object is used as-is ("vanilla") and active. A row records a choice:
--
--   active = false  the tenant has switched the core object off in their
--                   environment (any extension is kept for when they
--                   switch it back on).
--   mode 'extended' the tenant customized the core object. extension is
--                   their full copy; base_snapshot is the core as it was
--                   when they last took it (base_version). Diffing the two
--                   gives their customizations; diffing base_snapshot with
--                   the current core gives what changed on upgrade
--                   (backend/internal/corecustom).
--   mode 'cloned'   the tenant took an independent copy (clone_object_id,
--                   a tenant-owned object). No upgrade path.
--
-- The core object itself is never written by a tenant. Generic over
-- object_type so every core object type can adopt the same lifecycle;
-- Page Studio ('page') is the first.
CREATE TABLE IF NOT EXISTS public.core_object_adoption (
    tenant_id        UUID        NOT NULL,
    object_type      TEXT        NOT NULL,
    core_object_id   UUID        NOT NULL,
    active           BOOLEAN     NOT NULL DEFAULT true,
    mode             TEXT        NOT NULL DEFAULT 'vanilla',
    base_version     INTEGER,
    base_snapshot    JSONB,
    extension        JSONB,
    clone_object_id  UUID,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by       TEXT,
    PRIMARY KEY (tenant_id, object_type, core_object_id),
    CONSTRAINT core_object_adoption_mode_check CHECK (mode IN ('vanilla', 'extended', 'cloned')),
    CONSTRAINT core_object_adoption_extended_check CHECK (
        mode <> 'extended' OR (extension IS NOT NULL AND base_snapshot IS NOT NULL AND base_version IS NOT NULL)
    ),
    CONSTRAINT core_object_adoption_cloned_check CHECK (mode <> 'cloned' OR clone_object_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_core_object_adoption_core
    ON public.core_object_adoption (object_type, core_object_id);

ALTER TABLE public.core_object_adoption ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.core_object_adoption FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS core_object_adoption_tenant_policy ON public.core_object_adoption;
CREATE POLICY core_object_adoption_tenant_policy ON public.core_object_adoption
    FOR ALL
    USING (tenant_id = uisce_get_current_tenant())
    WITH CHECK (tenant_id = uisce_get_current_tenant());

-- page_definition_overlays (20261018_001) was the first cut at this: it
-- could only add components and replace tabs wholesale and had no base to
-- diff an upgrade against. Carry any rows over as extensions (base = the
-- core as it is now) so none are lost. The table is left in place for
-- builds that still read it and dropped by a later migration.
INSERT INTO public.core_object_adoption (tenant_id, object_type, core_object_id, mode, base_version, base_snapshot, extension, updated_at)
SELECT o.tenant_id, 'page', o.core_page_id, 'extended', p.version,
       jsonb_build_object('name', p.name, 'description', p.description, 'layout', p.layout, 'tabs', p.tabs,
                          'components', p.components, 'dataSources', p.data_sources,
                          'presentationEvents', p.presentation_events, 'filterBar', p.filter_bar, 'app', p.app_model),
       jsonb_build_object('name', p.name, 'description', p.description, 'layout', COALESCE(o.layout, p.layout),
                          'tabs', COALESCE(o.tabs, p.tabs),
                          'components', p.components || COALESCE(o.components, '{}'::jsonb), 'dataSources', p.data_sources,
                          'presentationEvents', p.presentation_events, 'filterBar', p.filter_bar, 'app', p.app_model),
       o.updated_at
FROM public.page_definition_overlays o
JOIN public.page_definitions p ON p.id = o.core_page_id AND p.is_core
ON CONFLICT DO NOTHING;
