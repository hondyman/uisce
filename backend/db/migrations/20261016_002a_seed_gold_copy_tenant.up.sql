-- 20261016_009_seed_gold_copy_tenant.up.sql
-- Bridge migration: ensure the gold-copy tenant exists on `public.tenants`
-- so the GSIFI read-inheritance policy on fix_tenant_tag_mapping (and
-- every other GSIFI-tagged table) actually has something to inherit.
--
-- Why this is here: pre-existing runner-applied migrations (003, 004,
-- 005, 007) all reference `(SELECT id FROM public.tenants WHERE
-- gold_copy = true LIMIT 1)` for the gold-copy inheritance half of the
-- RLS policy. On environments where the gold-copy tenant row was never
-- inserted (e.g. alpha as of 2026-09-13), that subquery returns NULL
-- and the read-inheritance clause silently degenerates to "current
-- tenant only" — gold-copy defaults are invisible to non-gold tenants.
--
-- The fix is two parts:
--   1. ADD COLUMN IF NOT EXISTS gold_copy on public.tenants. Defensive —
--      idempotent if already present. The "real" tenant-columns
--      migration lives at backend/migrations/20260214_add_region_to_
--      tenants.up.sql but is in the OTHER migrations directory
--      (manual_adopt, invisible to the runner — see
--      backend/db/MIGRATION_DIRECTORY_DRIFT_AUDIT.md).
--   2. INSERT a gold-copy tenant row with the well-known UUID
--      '00000000-0000-0000-0000-000000000001' (per AGENTS.md
--      "Gold Copy tenant id"). Idempotent via ON CONFLICT (id) DO
--      NOTHING.
--
-- This migration is independent of the FIX subsystem — it's needed by
-- every GSIFI-tagged table in the repo, not just the FIX ones. Land
-- it once and the inheritance pattern lights up everywhere.

BEGIN;

ALTER TABLE public.tenants
    ADD COLUMN IF NOT EXISTS gold_copy BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_tenants_gold_copy
    ON public.tenants (gold_copy)
    WHERE gold_copy = true;

-- Insert the well-known gold-copy tenant. ON CONFLICT (id) DO NOTHING
-- makes this idempotent across repeated apply runs.
INSERT INTO public.tenants (id, name, gold_copy)
VALUES ('00000000-0000-0000-0000-000000000001'::uuid, 'Gold Copy', true)
ON CONFLICT (id) DO NOTHING;

-- If a row with the gold-copy UUID already existed (from a previous
-- install path) without the gold_copy column having been set, force
-- it true now. Idempotent.
UPDATE public.tenants
SET gold_copy = true
WHERE id = '00000000-0000-0000-0000-000000000001'::uuid
  AND gold_copy = false;

COMMIT;
