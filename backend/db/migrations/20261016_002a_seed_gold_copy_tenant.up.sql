-- 20261016_002a_seed_gold_copy_tenant.up.sql
-- Bridge migration: ensure the gold-copy tenant exists on `public.tenants`
-- so the GSIFI read-inheritance policy on fix_tenant_tag_mapping (and
-- every other GSIFI-tagged table) actually has something to inherit.
--
-- Numbering: lexicographically sorts AFTER 002_gold_copy_sync_role and
-- BEFORE 003_fix_tenant_config — verified by running 003 directly
-- against a fresh DB without this migration; 003's RLS policy fails
-- with "column 'gold_copy' does not exist". So this migration is a
-- hard prerequisite for every FIX migration that uses GSIFI.
--
-- Blast radius (this is NOT a FIX-scoped change despite landing in the
-- same series): every production query that does
-- `WHERE gold_copy = true LIMIT 1` (see `grep -rn 'gold_copy = true'
-- backend/internal/`) silently returns NULL today on environments
-- where this column doesn't exist. This migration makes those queries
-- return the gold-copy row instead — fixing a pre-existing bug, but
-- also changing the effective scope of compliance evaluations,
-- boresolver reads, agentic subsystems, etc. Reviewers: this is the
-- migration to look at hardest. The behavior change is intentional
-- but should be a conscious decision, not a side effect.
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

-- No explicit transaction wrapper: the migration runner (backend/internal/
-- migrations/runner.go) already wraps every file in its own transaction
-- and rejects files that wrap themselves again (see
-- hasTransactionControl/stripTransactionStatements).

ALTER TABLE public.tenants
    ADD COLUMN IF NOT EXISTS gold_copy BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_tenants_gold_copy
    ON public.tenants (gold_copy)
    WHERE gold_copy = true;

-- Insert the well-known gold-copy tenant, but only when no gold-copy
-- tenant exists yet under ANY id - this migration was written assuming
-- a fresh install with none, but an environment can already have a real
-- gold-copy tenant seeded under its own id (e.g. this platform's
-- "northwind" tenant). idx_tenants_gold_copy_true is a partial unique
-- index enforcing at most one gold_copy=true row; inserting a second
-- one under the well-known UUID would violate that constraint (and the
-- product's own single-gold-copy-tenant model), not just fail
-- harmlessly, so the guard must be "does one already exist" rather than
-- ON CONFLICT (id) DO NOTHING (which only protects against re-running
-- against the *same* environment, not against a differently-seeded one).
INSERT INTO public.tenants (id, name, display_name, gold_copy)
SELECT '00000000-0000-0000-0000-000000000001'::uuid, 'Gold Copy', 'Gold Copy', true
WHERE NOT EXISTS (SELECT 1 FROM public.tenants WHERE gold_copy = true);

-- If a row with the gold-copy UUID already existed (from a previous
-- install path) without the gold_copy column having been set, force
-- it true now. Idempotent.
UPDATE public.tenants
SET gold_copy = true
WHERE id = '00000000-0000-0000-0000-000000000001'::uuid
  AND gold_copy = false;
