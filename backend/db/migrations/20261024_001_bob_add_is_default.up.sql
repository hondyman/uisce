-- 20261024_001_bob_add_is_default.up.sql
--
-- Step 1 of consolidating business_object_bindings (plural) into business_object_binding.
-- business_object_binding gains is_default. is_core cannot serve: every row is is_core = true, so it
-- does not say which of a BO's bindings is the default. One default per (tenant, bo).
--
-- Split into four migrations because the runner executes each file as ONE transaction and
-- business_object_binding has a deferred FK (driving_node_id): rows inserted or updated in a
-- transaction leave pending trigger events, and CREATE INDEX / ALTER TABLE on that table in the same
-- transaction then fail with "pending trigger events". So DDL and data changes are separate files:
--   001 add is_default + one-default index   (DDL only)
--   002 register backends, copy plural rows, backfill defaults   (data only)
--   003 repoint field_bindings / relationship_bindings FKs   (DDL only)
--   004 gold-copy read policy   (DDL only)

ALTER TABLE public.business_object_binding
    ADD COLUMN IF NOT EXISTS is_default boolean NOT NULL DEFAULT false;

-- Every row starts false, so the index cannot conflict; defaults are set in 002.
CREATE UNIQUE INDEX IF NOT EXISTS uq_bob_one_default
    ON public.business_object_binding (tenant_id, bo_id) WHERE is_default;
