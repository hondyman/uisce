-- Migration: 20260126_create_missing_relationship_tables.sql
-- Create table for Business Object Relationships (Definitions)
CREATE TABLE IF NOT EXISTS public.business_object_relationships (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    source_object_id uuid NOT NULL,
    target_object_id uuid NOT NULL,
    
    cardinality varchar(50), -- 'One-to-Many', 'Many-to-One', 'One-to-One'
    relationship_type varchar(50), -- 'association', 'composition', 'inheritance'
    description text,
    is_user_applied boolean DEFAULT true,
    
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    created_by uuid,
    
    CONSTRAINT business_object_relationships_pk PRIMARY KEY (id),
    CONSTRAINT bo_relationships_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
    CONSTRAINT bo_relationships_source_fk FOREIGN KEY (source_object_id) REFERENCES public.business_objects(id) ON DELETE CASCADE,
    CONSTRAINT bo_relationships_target_fk FOREIGN KEY (target_object_id) REFERENCES public.business_objects(id) ON DELETE CASCADE,
    CONSTRAINT bo_relationships_unique UNIQUE (tenant_id, source_object_id, target_object_id)
);

CREATE INDEX IF NOT EXISTS idx_bo_relationships_tenant ON public.business_object_relationships(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bo_relationships_source ON public.business_object_relationships(source_object_id);
CREATE INDEX IF NOT EXISTS idx_bo_relationships_target ON public.business_object_relationships(target_object_id);

-- Create table for Relationship Suggestions (AI generated)
CREATE TABLE IF NOT EXISTS public.relationship_suggestions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    datasource_id uuid, -- Optional in schema but used in queries
    
    source_entity_id uuid NOT NULL,
    target_entity_id uuid NOT NULL,
    
    confidence float,
    rationale text,
    scoring_breakdown jsonb DEFAULT '{}'::jsonb,
    
    accepted boolean DEFAULT false,
    accepted_at timestamptz,
    
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    
    CONSTRAINT relationship_suggestions_pk PRIMARY KEY (id),
    CONSTRAINT rel_suggestions_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE,
    CONSTRAINT rel_suggestions_source_fk FOREIGN KEY (source_entity_id) REFERENCES public.business_objects(id) ON DELETE CASCADE,
    CONSTRAINT rel_suggestions_target_fk FOREIGN KEY (target_entity_id) REFERENCES public.business_objects(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rel_suggestions_tenant ON public.relationship_suggestions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rel_suggestions_tenant_ds ON public.relationship_suggestions(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_rel_suggestions_source ON public.relationship_suggestions(source_entity_id);
