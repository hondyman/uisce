-- Staging bindings: which business object a staging table's rows are checked
-- as, field by field. A pipeline rule check in front of a staging load reads
-- the rows through the binding, so the object's rules (which read its field
-- names) evaluate the staging columns bound to them.
--
-- Deliberately not business_object_binding: that table is where a business
-- object's records are read and written (one binding per tenant, object and
-- backend, chosen by the record resolvers). A staging binding is only a
-- mapping for checks and lineage and must never become a record source.
--
--   * staging_bindings: one per (tenant, business object, staging table).
--     The gold copy's bindings are inherited read-only; a tenant's own
--     binding for the same object and table takes precedence.
--   * staging_binding_changes: maker-checker log. Every change is proposed
--     by one person and applied only when another approves it; it is the
--     audit trail either way.

CREATE TABLE IF NOT EXISTS public.staging_bindings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    bo_id uuid NOT NULL REFERENCES public.business_objects(id) ON DELETE CASCADE,
    staging_table text NOT NULL CHECK (staging_table ~ '^staging\.[a-z_][a-z0-9_]*$'),
    -- business object field name -> staging column
    fields jsonb NOT NULL CHECK (jsonb_typeof(fields) = 'object'),
    version integer NOT NULL DEFAULT 1,
    applied_change_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, bo_id, staging_table)
);

CREATE TABLE IF NOT EXISTS public.staging_binding_changes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    bo_id uuid NOT NULL REFERENCES public.business_objects(id) ON DELETE CASCADE,
    staging_table text NOT NULL,
    action text NOT NULL CHECK (action IN ('upsert', 'delete')),
    fields jsonb,
    -- The binding's fields when the change was proposed (NULL: none).
    -- Approval re-checks it, so a change never overwrites a later edit.
    before jsonb,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'applied', 'rejected', 'withdrawn')),
    reason text,
    requested_by text NOT NULL,
    requested_by_name text,
    requested_at timestamptz NOT NULL DEFAULT now(),
    reviewed_by text,
    reviewed_by_name text,
    reviewed_at timestamptz,
    review_comment text,
    applied_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_staging_bindings_lookup
    ON public.staging_bindings (bo_id, staging_table);
CREATE INDEX IF NOT EXISTS idx_staging_binding_changes_status
    ON public.staging_binding_changes (tenant_id, status, requested_at DESC);

ALTER TABLE public.staging_bindings ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.staging_bindings FORCE ROW LEVEL SECURITY;
ALTER TABLE public.staging_binding_changes ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.staging_binding_changes FORCE ROW LEVEL SECURITY;

-- A tenant reads and writes its own rows; it reads the gold copy's bindings
-- and never writes them. Changes are visible only in their own tenant.
DROP POLICY IF EXISTS staging_bindings_tenant ON public.staging_bindings;
CREATE POLICY staging_bindings_tenant ON public.staging_bindings
    USING (tenant_id::text = current_setting('uisce.current_tenant', true))
    WITH CHECK (tenant_id::text = current_setting('uisce.current_tenant', true));
DROP POLICY IF EXISTS staging_bindings_read_gold_copy ON public.staging_bindings;
CREATE POLICY staging_bindings_read_gold_copy ON public.staging_bindings
    FOR SELECT USING (tenant_id = public.uisce_gold_copy_tenant_id());
DROP POLICY IF EXISTS staging_binding_changes_tenant ON public.staging_binding_changes;
CREATE POLICY staging_binding_changes_tenant ON public.staging_binding_changes
    USING (tenant_id::text = current_setting('uisce.current_tenant', true))
    WITH CHECK (tenant_id::text = current_setting('uisce.current_tenant', true));

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON public.staging_bindings, public.staging_binding_changes TO app_user;
    END IF;
END $$;

COMMENT ON TABLE public.staging_bindings IS 'Business object field -> staging column, for pipeline rule checks and lineage; never a record source';
COMMENT ON TABLE public.staging_binding_changes IS 'Maker-checker log for staging bindings: every change is proposed, then applied only on a second person''s approval';
