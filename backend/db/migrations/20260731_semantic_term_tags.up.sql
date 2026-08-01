-- Migration: Semantic Term Tags
-- Date: 2026-07-31
-- Purpose: Create semantic_term_tags table for UI tag categorization of semantic terms.

CREATE TABLE IF NOT EXISTS public.semantic_term_tags (
    tag_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    semantic_term_id UUID NOT NULL REFERENCES public.semantic_terms(id),
    tag_key VARCHAR(100) NOT NULL,
    tag_label VARCHAR(255) NOT NULL,
    tag_category VARCHAR(100),
    color_code VARCHAR(7),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_semantic_term_tags_tenant_term_key
    ON public.semantic_term_tags(tenant_id, semantic_term_id, tag_key);

CREATE INDEX IF NOT EXISTS idx_semantic_term_tags_tenant
    ON public.semantic_term_tags(tenant_id);

CREATE INDEX IF NOT EXISTS idx_semantic_term_tags_category
    ON public.semantic_term_tags(tag_category);

ALTER TABLE public.semantic_term_tags ENABLE ROW LEVEL SECURITY;

CREATE POLICY "tenant_isolation_semantic_term_tags"
    ON public.semantic_term_tags
    FOR ALL
    USING (((tenant_id)::text = current_setting('uisce.current_tenant'::text, true)));
