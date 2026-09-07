-- Page builder facets: field-key registry, CRUD capability, and widget policy.
--
-- Design context: existing entitlement (bp_field_permissions.field_name) and
-- validation (catalog_validation_rules.target_entity) facets key on plain
-- field-name strings, not stable IDs — an earlier attempt to migrate to
-- term_node_id (see add_term_node_id_to_field_permissions.sql) was never
-- finished. Rather than repeat that mistake in the new facets, they
-- reference a field-key registry surrogate ID instead of embedding names
-- directly. When/if the entitlement and validation tables are migrated to
-- stable IDs, these tables become a backfill-and-repoint instead of a third
-- and fourth table needing restructuring.
--
-- Note: table names are schema-qualified (public.*) deliberately. The
-- `postgres` role's search_path on alpha is `vend, public` — an unqualified
-- CREATE TABLE here lands in vend, not alongside every other BO/entitlement/
-- validation table, which all live in public. Discovered by running this
-- migration unqualified first and finding the tables in the wrong schema.
--
-- RECOVERY NOTE (2026-09-06): this file was deleted by a concurrent
-- session's git-clean-class operation on a shared working tree before it
-- was ever committed. Reconstructed by dumping the actual live DDL from
-- alpha (`pg_dump --schema-only`) rather than retyping from memory, per the
-- standing rule that conversation carry-forward is not a trustworthy
-- reconstruction source on its own. Every column, type, default, and
-- constraint below is verified to match what is currently live, not what
-- this effort intended to create.

CREATE TABLE IF NOT EXISTS public.bo_field_key_registry (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bo_name         VARCHAR(255) NOT NULL,
    field_name      VARCHAR(255) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bo_name, field_name)
);

COMMENT ON TABLE public.bo_field_key_registry IS
    'Stable surrogate ID per (bo_name, field_name). Name-keying elsewhere is a silent-breakage-on-rename risk; join through this table rather than re-embedding names in new facet tables.';

-- CRUD capability: what an object/field supports, independent of who may do it
-- (that remains bp_field_permissions' job — effective writability is
-- capability AND entitlement, evaluated together at render/save time).
CREATE TABLE IF NOT EXISTS public.bo_crud_capabilities (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bo_name             VARCHAR(255) NOT NULL,
    field_key_id        UUID REFERENCES public.bo_field_key_registry(id), -- NULL = object-level default row
    can_create          BOOLEAN NOT NULL DEFAULT true,
    can_read            BOOLEAN NOT NULL DEFAULT true,
    can_update          BOOLEAN NOT NULL DEFAULT true,
    can_delete          BOOLEAN NOT NULL DEFAULT true,
    -- Stage conditions (e.g. "immutable_after_create") rather than booleans,
    -- so the reason a field is read-only is recorded, not just the fact.
    conditions          JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (bo_name, field_key_id)
);

-- Postgres treats NULLs as distinct under a plain UNIQUE constraint, so the
-- constraint above does not stop duplicate object-level default rows
-- (field_key_id IS NULL). A partial unique index closes that gap.
CREATE UNIQUE INDEX IF NOT EXISTS idx_bo_crud_capabilities_bo_default
    ON public.bo_crud_capabilities(bo_name) WHERE field_key_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_bo_crud_capabilities_bo_name ON public.bo_crud_capabilities(bo_name);

-- Widget policy: (field_type, cardinality) -> default + allowed widget keys.
-- Consolidates the frontend FieldDataType->FieldWidget enum and the backend
-- component_extensibility/forms/generator.go 3-type stub into one
-- server-governed source of truth. allowed_widget_keys is what a page's
-- widget override is validated against server-side.
CREATE TABLE IF NOT EXISTS public.bo_widget_policy (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_type          VARCHAR(100) NOT NULL,
    cardinality         VARCHAR(50) NOT NULL, -- e.g. 'one', 'many'
    default_widget_key  VARCHAR(100) NOT NULL,
    allowed_widget_keys TEXT[] NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (field_type, cardinality)
);

-- Sparse per-breakpoint fallback, not a full breakpoint axis on every widget
-- row: most widgets need no mobile-specific variant, so only exceptions get
-- a row here. A widget with no matching row is assumed to render as-is at
-- every breakpoint. fallback_widget_key = NULL means the widget is not
-- supported at that breakpoint at all — pages referencing it must declare
-- an alternative or fail to save (build-time check, not a runtime squeeze).
CREATE TABLE IF NOT EXISTS public.bo_widget_breakpoint_fallback (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    widget_key          VARCHAR(100) NOT NULL,
    breakpoint          VARCHAR(20) NOT NULL CHECK (breakpoint IN ('mobile', 'tablet')),
    fallback_widget_key VARCHAR(100), -- NULL = not_supported at this breakpoint
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (widget_key, breakpoint)
);
