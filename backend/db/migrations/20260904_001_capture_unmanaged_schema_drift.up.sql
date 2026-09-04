-- Migration: 20260904_001_capture_unmanaged_schema_drift.up.sql
-- Description: Capture schema elements confirmed to exist in live DB (100.84.50.65:5432 alpha)
--             but NOT created by any migration in db/migrations/ (the oms.migration_log runner).
--
-- PROVENANCE (from incident-step1-evidence.md commit 68b8b3e12):
--   Live DB: pg_dump --schema-only against alpha (100.84.50.65:5432)
--   Baseline: pg_dump --schema-only against uisce_drift_baseline (db/migrations/ replay)
--   Diff: backend/schema-drift.diff (~105K lines; 98K additions vs baseline)
--
-- KEY FINDINGS:
--   1. tenants.is_gold_copy — column EXISTS in live, NOT created by db/migrations/.
--      The public.tenants table itself is NOT created by db/migrations/ — it is created
--      by backend/migrations/ (legacy directory, not tracked by oms.migration_log runner).
--      This migration adds the column if the table exists (idempotent).
--   2. catalog_node_unique — EXISTS correctly as (tenant_id, node_type_id, qualified_path).
--      Created by 20260825_001 — no drift, no ALTER needed.
--   3. Full schema drift: 1300+ tables not in db/migrations/ baseline. Core tables
--      (tenants, business_objects, etc.) created by backend/migrations/.

-- =====================================================================
-- tenants.is_gold_copy — idempotent column add
-- The tenants table itself is created by backend/migrations/ (legacy, not tracked).
-- This migration adds the column if the table exists and the column is absent.
-- =====================================================================
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = 'tenants'
  ) THEN
    ALTER TABLE public.tenants
      ADD COLUMN IF NOT EXISTS is_gold_copy boolean DEFAULT false;
    RAISE NOTICE 'Added is_gold_copy to public.tenants';
  ELSE
    RAISE NOTICE 'public.tenants does not exist; skipping is_gold_copy column add';
  END IF;
END $$;

COMMENT ON COLUMN public.tenants.is_gold_copy IS
  'Flag indicating this tenant is the gold copy / reference tenant. '
  'Used by catalog_driver, slo_report_generator, reconciler, catalog_scan_service. '
  'Added by 20260904_001 — column existed in live DB via untracked mechanism.';
