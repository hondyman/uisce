-- business_object_fields.term_node_id: a field without a semantic binding has
-- no term node. 20261002_business_object_studio_engine creates the column NOT
-- NULL with no partial unique indexes, but the field-write path stores NULL for
-- unbound fields (the old SHA-1 minted-id fallback is gone), and the shared
-- alpha database already has the nullable column and both indexes below (they
-- came from 20260907_001_allow_null_term_node_id on the unmerged
-- metadata-gen3-crud branch, whose file never reached main). A database built
-- from main's migrations therefore rejected unbound fields and lacked the
-- uniqueness rules. Idempotent, so it is a no-op on alpha and corrects fresh
-- databases.
ALTER TABLE public.business_object_fields
    ALTER COLUMN term_node_id DROP NOT NULL;

-- Bound fields: unique per term node. NULL is not a value here.
CREATE UNIQUE INDEX IF NOT EXISTS uq_business_object_fields_term
    ON public.business_object_fields (tenant_id, bo_id, term_node_id)
    WHERE term_node_id IS NOT NULL;

-- Unbound fields: keyed by name instead.
CREATE UNIQUE INDEX IF NOT EXISTS uq_business_object_fields_unbound_name
    ON public.business_object_fields (tenant_id, bo_id, field_name)
    WHERE term_node_id IS NULL;
