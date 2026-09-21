-- Reverses 20261024_001_consolidate_bo_bindings.up.sql as far as it safely can. The plural table is
-- not recreated here (it is dropped separately), so the dependents' FKs are dropped rather than
-- repointed back. The copied bindings are left in place: removing them would orphan any rule scope
-- or field binding that has since referenced them.
ALTER TABLE IF EXISTS public.field_bindings DROP CONSTRAINT IF EXISTS field_bindings_binding_id_bob_fkey;
ALTER TABLE IF EXISTS public.relationship_bindings DROP CONSTRAINT IF EXISTS relationship_bindings_binding_id_bob_fkey;
DROP INDEX IF EXISTS public.uq_bob_one_default;
ALTER TABLE public.business_object_binding DROP COLUMN IF EXISTS is_default;
