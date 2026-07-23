-- Idempotent migration for Parent-Child BO Inheritance
DO $$
BEGIN
    -- 1. Add self-referencing inheritance column to business_objects.
    --    We intentionally use a dedicated parent_bo_id so the inheritance seam
    --    is explicit and distinct from any other parent_id semantics.
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'business_objects' AND column_name = 'parent_bo_id'
    ) THEN
        ALTER TABLE public.business_objects
        ADD COLUMN parent_bo_id uuid REFERENCES public.business_objects(id) ON DELETE SET NULL;
    END IF;

    -- Backfill parent_bo_id from the legacy parent_id column so existing
    -- clone/subtype relationships continue to resolve.
    UPDATE public.business_objects
    SET parent_bo_id = parent_id
    WHERE parent_bo_id IS NULL AND parent_id IS NOT NULL;

    -- 2. Add semantic binding columns to bo_fields if they do not already exist.
    --    These columns store the canonical mapping metadata produced by the
    --    binding-first Business Object wizard.
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'bo_fields' AND column_name = 'semantic_term_id'
    ) THEN
        ALTER TABLE public.bo_fields
        ADD COLUMN semantic_term_id uuid;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'bo_fields' AND column_name = 'field_role'
    ) THEN
        ALTER TABLE public.bo_fields
        ADD COLUMN field_role varchar(50);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'bo_fields' AND column_name = 'binding_requirement'
    ) THEN
        ALTER TABLE public.bo_fields
        ADD COLUMN binding_requirement varchar(50);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'bo_fields' AND column_name = 'binding_status'
    ) THEN
        ALTER TABLE public.bo_fields
        ADD COLUMN binding_status varchar(50);
    END IF;

    -- 3. Add structural inheritance auditing fields to bo_fields.
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'bo_fields' AND column_name = 'parent_field_id'
    ) THEN
        ALTER TABLE public.bo_fields
        ADD COLUMN parent_field_id uuid REFERENCES public.bo_fields(id) ON DELETE SET NULL,
        ADD COLUMN eligibility_source varchar(50) DEFAULT 'DIRECT',
        ADD COLUMN is_inherited_override boolean DEFAULT false;
    END IF;

    -- 4. Optimization indexes for cascading tenant compilations.
    CREATE INDEX IF NOT EXISTS idx_bo_parent_hierarchy
    ON public.business_objects (tenant_id, parent_bo_id)
    WHERE parent_bo_id IS NOT NULL;

    CREATE INDEX IF NOT EXISTS idx_bo_field_parent_link
    ON public.bo_fields (parent_field_id)
    WHERE parent_field_id IS NOT NULL;

    CREATE INDEX IF NOT EXISTS idx_bo_field_semantic_term
    ON public.bo_fields (business_object_id, semantic_term_id)
    WHERE semantic_term_id IS NOT NULL;
END $$;
