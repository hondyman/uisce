-- 20261024_003_bob_repoint_dependents.up.sql
--
-- Step 3 (DDL only). field_bindings and relationship_bindings pointed at business_object_bindings.id.
-- Repoint their FKs at business_object_binding(bo_binding_id). Both are empty today; the ids were
-- preserved by 002 so a row written before this runs stays valid.
--
-- Split into four migrations because the runner executes each file as ONE transaction and
-- business_object_binding has a deferred FK (driving_node_id): rows inserted or updated in a
-- transaction leave pending trigger events, and CREATE INDEX / ALTER TABLE on that table in the same
-- transaction then fail with "pending trigger events". So DDL and data changes are separate files:
--   001 add is_default + one-default index   (DDL only)
--   002 register backends, copy plural rows, backfill defaults   (data only)
--   003 repoint field_bindings / relationship_bindings FKs   (DDL only)
--   004 gold-copy read policy   (DDL only)

DO $$
BEGIN
    IF to_regclass('public.field_bindings') IS NOT NULL THEN
        ALTER TABLE public.field_bindings DROP CONSTRAINT IF EXISTS field_bindings_binding_id_fkey;
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'field_bindings_binding_id_bob_fkey') THEN
            ALTER TABLE public.field_bindings ADD CONSTRAINT field_bindings_binding_id_bob_fkey
                FOREIGN KEY (binding_id) REFERENCES public.business_object_binding (bo_binding_id) ON DELETE CASCADE;
        END IF;
    END IF;
    IF to_regclass('public.relationship_bindings') IS NOT NULL THEN
        ALTER TABLE public.relationship_bindings DROP CONSTRAINT IF EXISTS relationship_bindings_binding_id_fkey;
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'relationship_bindings_binding_id_bob_fkey') THEN
            ALTER TABLE public.relationship_bindings ADD CONSTRAINT relationship_bindings_binding_id_bob_fkey
                FOREIGN KEY (binding_id) REFERENCES public.business_object_binding (bo_binding_id) ON DELETE CASCADE;
        END IF;
    END IF;
END
$$;
