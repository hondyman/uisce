-- Create table for validation rules if it doesn't exist
CREATE TABLE IF NOT EXISTS public.catalog_validation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id TEXT NOT NULL,
    datasource_id TEXT NOT NULL,
    rule_name TEXT NOT NULL,
    rule_type TEXT NOT NULL, -- 'sql', 'regex', etc.
    target_entity TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'error', -- 'error', 'warning'
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    is_core BOOLEAN DEFAULT false,
    condition_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add index for fast lookups by tenant/datasource
CREATE INDEX IF NOT EXISTS idx_catalog_validation_rules_tenant_ds ON public.catalog_validation_rules(tenant_id, datasource_id);

-- Seed a sample rule for the Northwinds tenant so the UI isn't empty
INSERT INTO public.catalog_validation_rules (
    tenant_id, datasource_id, rule_name, rule_type, target_entity, severity, description, is_active, condition_json
) VALUES (
    '910638ba-a459-4a3f-bb2d-78391b0595f6', -- Northwinds tenant
    '982aef38-418f-46dc-acd0-35fe8f3b97b0', -- Alpha dataset
    'Customer Name Required',
    'required_field',
    'Customer',
    'error',
    'Customer name must not be empty',
    true,
    '{"field": "name", "operator": "not_empty"}'
);
