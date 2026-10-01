-- Page templates: starting points for new pages, shown in a gallery. A
-- template is a page bundle (the page plus the fragments it depends on, each
-- pinned by content hash) with a name, description and category. Like
-- fragments, a version is immutable and core templates live in the gold copy
-- (is_core). A distinct object type from fragments: a fragment is a piece
-- pages reference live; a template is copied once into a new page.
CREATE TABLE IF NOT EXISTS public.page_templates (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    slug          TEXT        NOT NULL,
    version       INTEGER     NOT NULL CHECK (version >= 1),
    name          TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    category      TEXT        NOT NULL DEFAULT 'general',
    bundle        JSONB       NOT NULL,
    bundle_hash   TEXT        NOT NULL,
    is_core       BOOLEAN     NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by    TEXT,
    UNIQUE (tenant_id, slug, version)
);

CREATE INDEX IF NOT EXISTS idx_page_templates_category ON public.page_templates (category);

ALTER TABLE public.page_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.page_templates FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS page_templates_tenant_gold_policy ON public.page_templates;
CREATE POLICY page_templates_tenant_gold_policy ON public.page_templates
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
        OR (is_core = true AND tenant_id = uisce_get_gold_tenant())
    )
    WITH CHECK (tenant_id = uisce_get_current_tenant());
