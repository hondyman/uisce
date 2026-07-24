-- Create Trade Workflow Tables
-- Extends the existing Business Object model with Workflow capabilities

DROP TABLE IF EXISTS public.compliance_rules CASCADE;
DROP TABLE IF EXISTS public.workflow_stages CASCADE;
DROP TABLE IF EXISTS public.workflow_definitions CASCADE;

CREATE TABLE IF NOT EXISTS public.workflow_definitions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    name varchar(64) NOT NULL,
    description text,
    status varchar(10) CHECK (status IN ('active', 'draft', 'retired')) DEFAULT 'draft',
    stages jsonb DEFAULT '[]'::jsonb, -- Array of stage definitions
    
    created_at timestamptz DEFAULT now(),
    created_by uuid,
    last_modified_at timestamptz DEFAULT now(),
    last_modified_by uuid,

    CONSTRAINT workflow_definitions_pk PRIMARY KEY (id),
    CONSTRAINT workflow_definitions_tenant_fk FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE
);

CREATE INDEX workflow_definitions_tenant_idx ON public.workflow_definitions (tenant_id);

CREATE TABLE IF NOT EXISTS public.workflow_stages (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workflow_id uuid NOT NULL,
    name varchar(32) NOT NULL,
    order_index integer NOT NULL,
    config_json jsonb DEFAULT '{}'::jsonb, -- UI layout, actors, triggers
    
    created_at timestamptz DEFAULT now(),
    
    CONSTRAINT workflow_stages_pk PRIMARY KEY (id),
    CONSTRAINT workflow_stages_workflow_fk FOREIGN KEY (workflow_id) REFERENCES public.workflow_definitions(id) ON DELETE CASCADE
);

CREATE INDEX workflow_stages_workflow_idx ON public.workflow_stages (workflow_id);

CREATE TABLE IF NOT EXISTS public.compliance_rules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    workflow_id uuid NOT NULL,
    rule_code varchar(128) NOT NULL,
    description text,
    config_json jsonb DEFAULT '{}'::jsonb, -- Thresholds, logic
    
    created_at timestamptz DEFAULT now(),
    
    CONSTRAINT compliance_rules_pk PRIMARY KEY (id),
    CONSTRAINT compliance_rules_workflow_fk FOREIGN KEY (workflow_id) REFERENCES public.workflow_definitions(id) ON DELETE CASCADE
);

CREATE INDEX compliance_rules_workflow_idx ON public.compliance_rules (workflow_id);
