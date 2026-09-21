-- RECONSTRUCTION (2026-09-21). The original file was applied to the shared dev DB
-- on 2026-09-04 and its SQL is unrecoverable: it exists in no git ref and no
-- worktree. See backend/docs/migrations/MIGRATION_LOSS_RECORD.md.
--
-- Evidence for this content:
--   * Live DB has  UNIQUE CONSTRAINT catalog_node_unique
--                  (tenant_id, node_type_id, qualified_path)  on public.catalog_node.
--   * 20260904_001_capture_unmanaged_schema_drift.up.sql states that constraint
--     exists "correctly" and was "created by 20260825_001" (no ALTER needed).
--   * The legacy migration 002700_ensure_catalog_node_glossary_schema.sql drops a
--     constraint of this name, so it was re-created afterwards by a later migration.
--   * Nothing on disk creates it, and no Go/SQL code references it by name.
-- Confidence: high that this constraint is the content; it cannot be proven that
-- the original contained nothing else.
--
-- Behaviour: idempotent. On the shared dev DB this migration is already recorded
-- as applied under a different content hash, so the runner logs "content has
-- changed" and skips it (it never re-executes there). On a fresh database it
-- reproduces the live constraint. Companion constraint catalog_node_tenant_path_uniq
-- is created by 20261021_001.

DO $$
BEGIN
    IF to_regclass('public.catalog_node') IS NULL THEN
        RAISE NOTICE 'catalog_sti_unique_constraints: public.catalog_node absent, skipping';
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'public.catalog_node'::regclass
          AND conname = 'catalog_node_unique'
    ) THEN
        ALTER TABLE public.catalog_node
            ADD CONSTRAINT catalog_node_unique UNIQUE (tenant_id, node_type_id, qualified_path);
    END IF;
END
$$;
