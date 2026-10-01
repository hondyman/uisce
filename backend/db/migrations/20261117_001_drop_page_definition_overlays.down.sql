CREATE TABLE IF NOT EXISTS public.page_definition_overlays (
    tenant_id     UUID NOT NULL,
    core_page_id  UUID NOT NULL,
    components    JSONB NOT NULL DEFAULT '{}'::jsonb,
    layout        JSONB,
    tabs          JSONB,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, core_page_id)
);
CREATE INDEX IF NOT EXISTS idx_page_definition_overlays_core ON public.page_definition_overlays (core_page_id);
ALTER TABLE public.page_definition_overlays ENABLE ROW LEVEL SECURITY;
