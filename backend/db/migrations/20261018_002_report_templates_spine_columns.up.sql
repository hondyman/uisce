-- 20261018_002_report_templates_spine_columns.up.sql
-- Report Builder spine plan Phase 2, ticket 2.1
-- (HANDOFF_REPORT_BUILDER_SPINE_PLAN.md). Adds typed columns to
-- report_templates for the CoreReportDefinition shape, replacing what
-- layout_config's opaque JSONB blob currently carries with no
-- server-side reader (0.1's gap table: confirmed zero code anywhere
-- reads layout_config['elements'] by key).
--
-- Additive only, gated-transition pattern - matches the incremental-
-- column precedent already established for page_definitions (see
-- 20261016_016_page_definitions_presentation_events.up.sql,
-- 20261016_017_page_definitions_filter_bar.up.sql): layout_config and
-- parameter_schema are kept as read-fallbacks, not dropped or renamed,
-- until Phase 3+ proves parity against the new columns.

-- bands: deliberately NO backfill from layout_config's elements array.
-- 0.1 decided this explicitly - inventing the band shape under migration
-- pressure, before Phase 3.1/3.2 have designed it, means guessing at a
-- structure those tickets would then be locked into. Starts empty; a
-- transformation tool from elements (if existing reports warrant one) is
-- optional future tooling, not part of this migration.
ALTER TABLE public.report_templates
    ADD COLUMN IF NOT EXISTS bands JSONB NOT NULL DEFAULT '[]'::jsonb;

-- parameters: the one column with both a reader today
-- (ParametersDialog.tsx / ReportParametersToolbar.tsx already round-trip
-- ParamSpec-shaped data, ticket 1.2) and a defined target shape -
-- backfilled below from parameter_schema's existing content.
ALTER TABLE public.report_templates
    ADD COLUMN IF NOT EXISTS parameters JSONB NOT NULL DEFAULT '[]'::jsonb;

-- presentation_events: format-only rules (suppress-repeat, ticket 1.4;
-- more vocabulary as Phase 4's grouped band lands). New capability, no
-- existing data to backfill.
ALTER TABLE public.report_templates
    ADD COLUMN IF NOT EXISTS presentation_events JSONB NOT NULL DEFAULT '[]'::jsonb;

-- grouping: subtotal/band-break support for the Grouped Summary report
-- kind (Phase 4). Nullable, no default row shape yet - Phase 4 defines
-- it when the grouped band designer is built, same discipline as bands.
ALTER TABLE public.report_templates
    ADD COLUMN IF NOT EXISTS grouping JSONB;

-- primary_business_object_id: the 0.4 cube-vs-BO grounding decision,
-- additive alongside semantic_view_ids - never replacing it. Nullable so
-- cube/semantic-view-grounded reports (report_activities.go's
-- QuerySemanticViewsActivity path) keep working unmigrated; every
-- BO-grounded report the Phase 3+ builder produces sets this and leaves
-- semantic_view_ids null. No FK constraint to a business_objects table
-- here deliberately - business object identity in this codebase is
-- resolved through the catalog/BO service, not a local FK (matches how
-- semantic_view_ids is also just a bare uuid[] with no FK).
ALTER TABLE public.report_templates
    ADD COLUMN IF NOT EXISTS primary_business_object_id UUID;

-- is_core: fixes the confirmed silent-drop bug (0.1's gap table -
-- handleCloneReport's is_core=false has no column to land in today, so
-- SSRSReportBuilder.tsx's isCoreTemplate check can currently only ever
-- be satisfied by its tenant_id === goldCopyId fallback). Mirrors
-- page_definitions.is_core's existing, working gold-copy pattern rather
-- than forking a second mechanism for reports.
ALTER TABLE public.report_templates
    ADD COLUMN IF NOT EXISTS is_core BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_report_templates_is_core
    ON public.report_templates (is_core);

-- Backfill parameters from parameter_schema's existing wrapper shape.
-- buildSavePayload (frontend/src/components/reporting/builderSerialization.ts)
-- has always written parameter_schema as {"parameters": [...]}, not a
-- bare array - confirmed by reading that file, not assumed.
UPDATE public.report_templates
SET parameters = COALESCE(parameter_schema -> 'parameters', '[]'::jsonb)
WHERE parameter_schema IS NOT NULL
  AND jsonb_typeof(parameter_schema -> 'parameters') = 'array';

-- is_core is deliberately NOT backfilled from tenant_id = gold_copy here.
-- A gold-copy tenant can hold non-core rows too (personal/draft/test
-- reports - confirmed live in this session's own DB read: report names
-- like "TwoSidedReport_*"/"DeleteTestReport_*" under real tenant ids,
-- the same pattern would apply to the gold-copy tenant's own test
-- fixtures). Guessing is_core from tenant_id would misclassify those.
-- Existing rows start is_core = false; marking real core templates is a
-- data-correction task for whoever owns that content, not this
-- migration's job to infer.
