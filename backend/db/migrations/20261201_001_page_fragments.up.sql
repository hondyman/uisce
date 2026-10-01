-- Page fragments: reusable pieces of a page (widgets, layout nodes, variables,
-- queries) that pages and other fragments reference by slug and exact version.
-- A version is immutable - publishing changes creates the next version - so a
-- page never changes under its author. Core fragments live in the gold copy
-- (is_core) and are adopted per tenant through core_object_adoption with
-- object_type = 'page_fragment' (that table is already generic; no change).
-- content_hash is the sha256 of the canonical content, so a bundle can pin the
-- exact fragment it was exported with.
CREATE TABLE IF NOT EXISTS public.page_fragments (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    slug          TEXT        NOT NULL,
    version       INTEGER     NOT NULL CHECK (version >= 1),
    name          TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    content       JSONB       NOT NULL,
    content_hash  TEXT        NOT NULL,
    is_core       BOOLEAN     NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by    TEXT,
    UNIQUE (tenant_id, slug, version)
);

CREATE INDEX IF NOT EXISTS idx_page_fragments_core ON public.page_fragments (is_core);

ALTER TABLE public.page_fragments ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.page_fragments FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS page_fragments_tenant_gold_policy ON public.page_fragments;
CREATE POLICY page_fragments_tenant_gold_policy ON public.page_fragments
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
        OR (is_core = true AND tenant_id = uisce_get_gold_tenant())
    )
    WITH CHECK (tenant_id = uisce_get_current_tenant());
