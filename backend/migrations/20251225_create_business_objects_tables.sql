-- Create Business Objects (BOs) storage
-- Stores the master definitions of all BOs (core and custom)

CREATE TABLE IF NOT EXISTS public.business_objects (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    tenant_datasource_id uuid,
    key varchar(255) NOT NULL,
    name varchar(255) NOT NULL,
    display_name varchar(255) NOT NULL,
    technical_name varchar(255) NOT NULL,
    description text,
    fields jsonb DEFAULT '[]'::jsonb,
    icon varchar(100),
    
    -- Core vs Custom
    is_core boolean DEFAULT false,
    
    -- Clone tracking
    clones_from varchar(255),
    clone_parent_key varchar(255),
    clone_parent_display_name varchar(255),
    
    -- Metadata
    category varchar(100),
    parent_id uuid,
    instance_count integer DEFAULT 0,
    
    -- Configuration and status
    config jsonb DEFAULT '{}'::jsonb,
    is_active boolean DEFAULT true,
    
    created_at timestamptz DEFAULT now(),
    created_by uuid,
    last_modified_at timestamptz DEFAULT now(),
    last_modified_by uuid,
    
    CONSTRAINT business_objects_pk PRIMARY KEY (id),
    CONSTRAINT business_objects_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
    CONSTRAINT business_objects_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    CONSTRAINT business_objects_parent_fk FOREIGN KEY (parent_id) REFERENCES public.business_objects(id) ON DELETE CASCADE,
    CONSTRAINT business_objects_unique UNIQUE (tenant_id, key)
);

CREATE INDEX IF NOT EXISTS business_objects_tenant_idx ON public.business_objects (tenant_id);
CREATE INDEX IF NOT EXISTS business_objects_key_idx ON public.business_objects (key);

-- ============================================================================
-- SUBTYPES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.bo_subtypes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    business_object_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    key varchar(255) NOT NULL,
    name varchar(255) NOT NULL,
    display_name varchar(255) NOT NULL,
    technical_name varchar(255) NOT NULL,
    description text,
    
    -- Clone tracking
    is_core boolean DEFAULT false,
    based_on_entity varchar(255),
    clone_parent_key varchar(255),
    
    sequence integer DEFAULT 0,
    created_at timestamptz DEFAULT now(),
    created_by uuid,
    last_modified_at timestamptz DEFAULT now(),
    last_modified_by uuid,
    
    CONSTRAINT bo_subtypes_pk PRIMARY KEY (id),
    CONSTRAINT bo_subtypes_bo_fk FOREIGN KEY (business_object_id) REFERENCES public.business_objects(id) ON DELETE CASCADE,
    CONSTRAINT bo_subtypes_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
    CONSTRAINT bo_subtypes_unique UNIQUE (tenant_id, business_object_id, key)
);

CREATE INDEX IF NOT EXISTS bo_subtypes_bo_idx ON public.bo_subtypes (business_object_id);
CREATE INDEX IF NOT EXISTS bo_subtypes_tenant_idx ON public.bo_subtypes (tenant_id);

-- ============================================================================
-- FIELDS TABLE (for both entity-level and subtype-level)
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.bo_fields (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    business_object_id uuid,
    subtype_id uuid,
    
    key varchar(255) NOT NULL,
    name varchar(255) NOT NULL,
    display_name varchar(255) NOT NULL,
    technical_name varchar(255) NOT NULL,
    type varchar(50) NOT NULL, -- text, email, number, date, datetime, boolean, currency, json, array, image, reference
    
    is_core boolean DEFAULT false,
    is_required boolean DEFAULT false,
    is_system boolean DEFAULT false, -- cannot be deleted by user
    description text,
    reference_entity varchar(255), -- if type='reference'
    sequence integer DEFAULT 0,
    
    created_at timestamptz DEFAULT now(),
    created_by uuid,
    last_modified_at timestamptz DEFAULT now(),
    last_modified_by uuid,
    
    CONSTRAINT bo_fields_pk PRIMARY KEY (id),
    CONSTRAINT bo_fields_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
    CONSTRAINT bo_fields_bo_fk FOREIGN KEY (business_object_id) REFERENCES public.business_objects(id) ON DELETE CASCADE,
    CONSTRAINT bo_fields_subtype_fk FOREIGN KEY (subtype_id) REFERENCES public.bo_subtypes(id) ON DELETE CASCADE,
    CONSTRAINT bo_fields_bo_or_subtype CHECK ((business_object_id IS NOT NULL AND subtype_id IS NULL) OR (business_object_id IS NULL AND subtype_id IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS bo_fields_bo_idx ON public.bo_fields (business_object_id);
CREATE INDEX IF NOT EXISTS bo_fields_subtype_idx ON public.bo_fields (subtype_id);
CREATE INDEX IF NOT EXISTS bo_fields_tenant_idx ON public.bo_fields (tenant_id);
CREATE INDEX IF NOT EXISTS bo_fields_key_idx ON public.bo_fields (key);

-- ============================================================================
-- BO INSTANCES TABLE (Individual Records)
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.bo_instances (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    datasource_id uuid NOT NULL,
    business_object_id uuid NOT NULL,
    subtype_id uuid,
    
    -- Field values stored as JSONB for flexibility
    core_field_values jsonb DEFAULT '{}'::jsonb,
    custom_field_values jsonb DEFAULT '{}'::jsonb,
    
    -- Metadata
    created_at timestamptz DEFAULT now(),
    created_by uuid,
    last_modified_at timestamptz DEFAULT now(),
    last_modified_by uuid,
    is_deleted boolean DEFAULT false,
    deleted_at timestamptz,
    
    CONSTRAINT bo_instances_pk PRIMARY KEY (id),
    CONSTRAINT bo_instances_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
    CONSTRAINT bo_instances_datasource_fk FOREIGN KEY (datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    CONSTRAINT bo_instances_bo_fk FOREIGN KEY (business_object_id) REFERENCES public.business_objects(id) ON DELETE CASCADE,
    CONSTRAINT bo_instances_subtype_fk FOREIGN KEY (subtype_id) REFERENCES public.bo_subtypes(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS bo_instances_tenant_datasource_idx ON public.bo_instances (tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS bo_instances_bo_idx ON public.bo_instances (business_object_id);
CREATE INDEX IF NOT EXISTS bo_instances_subtype_idx ON public.bo_instances (subtype_id);
CREATE INDEX IF NOT EXISTS bo_instances_deleted_idx ON public.bo_instances (is_deleted);

-- ============================================================================
-- AUDIT LOG FOR BO CHANGES
-- ============================================================================

CREATE TABLE IF NOT EXISTS public.bo_audit_log (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    entity_type varchar(50) NOT NULL, -- business_object, subtype, field, instance
    entity_id uuid NOT NULL,
    action varchar(50) NOT NULL, -- create, update, delete, clone
    changes jsonb,
    
    created_at timestamptz DEFAULT now(),
    created_by uuid,
    
    CONSTRAINT bo_audit_log_pk PRIMARY KEY (id),
    CONSTRAINT bo_audit_log_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS bo_audit_log_tenant_idx ON public.bo_audit_log (tenant_id);
CREATE INDEX IF NOT EXISTS bo_audit_log_entity_idx ON public.bo_audit_log (entity_id);
CREATE INDEX IF NOT EXISTS bo_audit_log_action_idx ON public.bo_audit_log (action);
