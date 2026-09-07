-- One-time backfill of bo_field_key_registry from the live structure facet
-- (public.business_object_fields joined to public.business_objects).
-- Idempotent: safe to re-run, ON CONFLICT DO NOTHING against the
-- (bo_name, field_name) unique constraint.
--
-- Run once against a freshly migrated environment after
-- 20260905_page_builder_facets.sql, before seeding bo_widget_policy or
-- writing bo_crud_capabilities rows that reference field_key_id.
--
-- Ongoing ownership: ideally the live BO service (internal/metadata,
-- BusinessObjectService.CreateBusinessObject / field-add path) upserts a
-- registry row itself as part of its own write path once that wiring
-- exists, so this backfill only has to run once per environment rather
-- than being re-run to catch drift. Until that wiring lands, re-run this
-- script after any out-of-band field addition. See
-- backend/internal/pagebuilder/BO_SERVICES.md for why that wiring is
-- currently blocked (internal/metadata's CreateBusinessObject/UpdateBusinessObject
-- don't function against the live schema as of this writing).
--
-- RECOVERY NOTE (2026-09-06): this file was deleted, pre-commit, by a
-- concurrent session's git-clean-class operation on a shared working tree.
-- Reconstructed and re-verified against live alpha: re-running it against
-- the current 81-row registry is a confirmed no-op (0 rows inserted, all
-- already present).

INSERT INTO public.bo_field_key_registry (bo_name, field_name)
SELECT DISTINCT bo.bo_name, f.field_name
FROM public.business_object_fields f
JOIN public.business_objects bo ON bo.id = f.bo_id
ON CONFLICT (bo_name, field_name) DO NOTHING;
