-- catalog_edge_rejection_store backs the AI-suggested-relationship "reject"
-- action in the glossary UI's Relationships tab (AISuggestionCard):
-- POST/GET /api/semantic-mapper/rejections, used by
-- internal/analytics/term_relationship_service.go's RecordRejection /
-- ListRejections / DeleteRejection. The handler and frontend component were
-- both fully built, but this table was never created, so every call 500'd
-- and the frontend showed "AI suggestions unavailable: API Error: 404" (the
-- underlying handler wasn't even mounted - see the /api Route() fix
-- alongside this migration) or, once mounted, 500'd on the missing table.
--
-- Table name is schema-qualified (public.*) deliberately - the shared
-- `alpha` database's `postgres` role has search_path 'vend, public', so an
-- unqualified CREATE TABLE lands in the wrong schema (see
-- 20260905_page_builder_facets.sql for the same note).
CREATE TABLE IF NOT EXISTS public.catalog_edge_rejection_store (
    rejection_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    source_node_id UUID NOT NULL,
    rejected_target_id UUID NOT NULL,
    edge_type_id UUID NOT NULL,
    rejected_by TEXT NOT NULL DEFAULT 'user',
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, source_node_id, rejected_target_id, edge_type_id)
);

CREATE INDEX IF NOT EXISTS idx_catalog_edge_rejection_store_tenant
    ON public.catalog_edge_rejection_store (tenant_id);
