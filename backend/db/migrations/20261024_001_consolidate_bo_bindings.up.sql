-- 20261024_001_consolidate_bo_bindings.up.sql
--
-- Consolidates business_object_bindings (plural) into business_object_binding (singular), which is
-- the binding model the MDM BOs and the rule binding scope use. This migration is additive; the
-- plural table is dropped separately once no code reads it.
--
--   1. business_object_binding gains is_default. is_core cannot serve: every row is is_core = true,
--      so it does not say which of a BO's bindings is the default. One default per (tenant, bo).
--   2. The plural rows are the only binding for the ORM core BOs (account, broker, order, position,
--      security). Their backend is registered in physical_backend if missing (the singular table has
--      an FK to it), then they are copied across keeping their ids.
--   3. field_bindings and relationship_bindings pointed at the plural table's id. Their FKs are
--      repointed at business_object_binding(bo_binding_id). Both are empty today, but the ids are
--      preserved so a row written before this runs stays valid.
--
-- backend_type is not carried over: it is derivable from physical_backend.dialect_name, so storing
-- it again would be a redundant column. Idempotent; skips when the plural table is already gone.

ALTER TABLE public.business_object_binding
    ADD COLUMN IF NOT EXISTS is_default boolean NOT NULL DEFAULT false;

DO $$
BEGIN
    IF to_regclass('public.business_object_bindings') IS NOT NULL THEN
        -- Register the backends the plural rows point at.
        INSERT INTO public.physical_backend
            (backend_id, backend_name, description, storage_tier, dialect_name, driver_class, is_system)
        SELECT DISTINCT p.backend_id,
               COALESCE(tpd.source_name, 'backend ' || p.backend_id::text),
               'Registered by bo binding consolidation',
               'oltp', lower(p.backend_type), '*sql.DB', false
        FROM public.business_object_bindings p
        LEFT JOIN public.tenant_product_datasource tpd ON tpd.id = p.backend_id
        ON CONFLICT (backend_id) DO NOTHING;

        -- Copy the bindings, keeping their ids.
        INSERT INTO public.business_object_binding
            (bo_binding_id, tenant_id, bo_id, backend_id, driving_node_id, base_sql, temporal_override,
             is_core, is_active, is_default, binding_name, created_at, updated_at)
        SELECT p.id, p.tenant_id, p.bo_id, p.backend_id, p.driving_node_id, p.base_sql, p.temporal_override,
               true, true, p.is_default,
               COALESCE(bo.bo_name, 'Business Object') || ' Binding', p.created_at, p.updated_at
        FROM public.business_object_bindings p
        LEFT JOIN public.business_objects bo ON bo.id = p.bo_id
        ON CONFLICT DO NOTHING;
    END IF;
END
$$;

-- A BO with exactly one binding has that binding as its default.
UPDATE public.business_object_binding b
SET is_default = true
WHERE NOT b.is_default
  AND (SELECT count(*) FROM public.business_object_binding x
       WHERE x.tenant_id = b.tenant_id AND x.bo_id = b.bo_id) = 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_bob_one_default
    ON public.business_object_binding (tenant_id, bo_id) WHERE is_default;

-- Repoint the dependents at the surviving table.
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
