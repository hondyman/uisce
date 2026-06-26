-- Migration: Create bo_fields table (Flattened from redesign)

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='business_entity')
     AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='business_objects') THEN
    EXECUTE 'ALTER TABLE business_entity RENAME TO business_objects';
  ELSE
    RAISE NOTICE 'Skipping rename business_entity -> business_objects (either not present or already renamed)';
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS bo_fields (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    business_object_id uuid NOT NULL,  -- Always set: parent BO
    subtype_id uuid,                   -- NULL for parent fields, UUID for subtype fields (references business_objects)
    
    key varchar(255) NOT NULL,
    name varchar(255) NOT NULL,
    display_name varchar(255),
    technical_name varchar(255),
    type varchar(50) NOT NULL,
    
    is_core boolean DEFAULT false,
    is_subtype_only boolean DEFAULT false,  -- NEW: True = custom to subtype, False = inherited/parent
    is_required boolean DEFAULT false,
    is_system boolean DEFAULT false,
    description text,
    reference_entity varchar(255),
    sequence integer DEFAULT 0,
    
    created_at timestamptz DEFAULT now(),
    created_by uuid,
    last_modified_at timestamptz DEFAULT now(),
    last_modified_by uuid,
    
    CONSTRAINT bo_fields_pk PRIMARY KEY (id),
    CONSTRAINT bo_fields_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
    CONSTRAINT bo_fields_bo_fk FOREIGN KEY (business_object_id) REFERENCES public.business_objects(id) ON DELETE CASCADE,
    CONSTRAINT bo_fields_subtype_fk FOREIGN KEY (subtype_id) REFERENCES public.business_objects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS bo_fields_bo_idx ON public.bo_fields (business_object_id);
CREATE INDEX IF NOT EXISTS bo_fields_subtype_idx ON public.bo_fields (subtype_id);
CREATE INDEX IF NOT EXISTS bo_fields_tenant_idx ON public.bo_fields (tenant_id);
CREATE INDEX IF NOT EXISTS bo_fields_key_idx ON public.bo_fields (key);
CREATE INDEX IF NOT EXISTS bo_fields_bo_subtype_idx ON public.bo_fields (business_object_id, subtype_id);
