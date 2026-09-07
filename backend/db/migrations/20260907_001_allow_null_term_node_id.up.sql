-- backend/db/migrations/20260907_001_allow_null_term_node_id.up.sql
--
-- Context:
-- business_object_fields.term_node_id was created NOT NULL in
-- 20260902_bo_studio_and_field_overrides.sql. The field-write path in
-- internal/metadata/businessobject_service.go UpdateBusinessObject then
-- minted synthetic term_node_id values via SHA-1(boID + fieldName) for any
-- request field that did not carry an explicit semanticTermId. That created
-- IDs that correspond to no actual catalog_node entry — a second dialect of
-- the same column — exactly the attractor that motivated
-- 20260905_page_builder_facets.sql (stable field-key surrogate IDs, join
-- through the registry rather than re-embed names).
--
-- term_node_id is referentially (not declaratively) an FK to catalog_node.id;
-- the entitlement layer (bp_field_permissions.term_node_id), the masking
-- consolidation, and the catalog-edge emitter all join on it expecting a
-- real term node. Hash-minted IDs break those joins silently when they land.
--
-- Fix:
--   1. Drop the NOT NULL constraint on term_node_id — fields without a
--      semantic binding don't have a term node, full stop.
--   2. Replace the full unique constraint with a partial unique index that
--      treats NULL term_node_id as "binding is field-name keyed" rather than
--      treating it as a value (standard Postgres UNIQUE behavior). Multiple
--      unbound fields per (tenant, bo) is fine; they key by field_name.
--   3. Add a partial unique constraint that enforces name uniqueness for the
--      NULL case so two different fields can't both be unnamed on the same BO.
--
-- This migration is the precondition for removing the SHA-1 fallback in the
-- field-write path. Field-write diff/upsert semantics (COMING SOON) will then
-- use (tenant_id, bo_id, term_node_id) for semantic-bound fields and
-- field_name comparison for unbound fields.

BEGIN;

ALTER TABLE public.business_object_fields
    ALTER COLUMN term_node_id DROP NOT NULL;

ALTER TABLE public.business_object_fields
    DROP CONSTRAINT IF EXISTS uq_tenant_bo_term;

CREATE UNIQUE INDEX IF NOT EXISTS uq_business_object_fields_term
    ON public.business_object_fields (tenant_id, bo_id, term_node_id)
    WHERE term_node_id IS NOT NULL;

-- For fields without a term node binding, enforce uniqueness on field_name.
-- A "name-collision" between bound and unbound fields is intentionally NOT
-- blocked here: the unique-on-term-node already protects the bound side,
-- and we don't want to reject a legitimate refactor that swaps a field's
-- binding from NULL to a real term.
CREATE UNIQUE INDEX IF NOT EXISTS uq_business_object_fields_unbound_name
    ON public.business_object_fields (tenant_id, bo_id, field_name)
    WHERE term_node_id IS NULL;

COMMIT;
