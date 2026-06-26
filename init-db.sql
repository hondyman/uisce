-- Initialize database for SemLayer API
-- This script sets up the catalog tables and initial data

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create tenants table
CREATE TABLE IF NOT EXISTS public.tenants (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    api_key VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create tenant_datasources table
CREATE TABLE IF NOT EXISTS public.tenant_datasources (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    datasource_id uuid NOT NULL,
    connection_string TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, datasource_id)
);

-- Create catalog_node_type table
CREATE TABLE IF NOT EXISTS public.catalog_node_type (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    catalog_type_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    parent_type_id uuid,
    config JSONB,
    properties JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, catalog_type_name)
);

-- Create catalog_edge_type table
-- Create catalog_edge_types table (canonicalized)
CREATE TABLE IF NOT EXISTS public.catalog_edge_types (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    edge_type_name VARCHAR(255) NOT NULL,
    description TEXT,
    source_node_type_id uuid,
    target_node_type_id uuid,
    config JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, edge_type_name)
);

-- Create catalog_node table
CREATE TABLE IF NOT EXISTS public.catalog_node (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_datasource_id uuid NOT NULL,
    node_type_id uuid NOT NULL,
    node_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    properties JSONB,
    qualified_path TEXT,
    parent_id uuid,
    tenant_id uuid NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Table to persist scheduled pre-aggregation jobs
CREATE TABLE IF NOT EXISTS public.scheduled_jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    datasource_id uuid NOT NULL,
    cube_name TEXT NOT NULL,
    pre_name TEXT NOT NULL,
    cron_expr TEXT,
    storage TEXT,
    refresh_key JSONB,
    last_run TIMESTAMP WITH TIME ZONE,
    last_refresh_key_val TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_tenant ON public.scheduled_jobs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_datasource ON public.scheduled_jobs (datasource_id);

-- Table to record execution history of scheduled jobs
CREATE TABLE IF NOT EXISTS public.scheduled_job_runs (
    id SERIAL PRIMARY KEY,
    job_id uuid NOT NULL REFERENCES public.scheduled_jobs(id) ON DELETE CASCADE,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    finished_at TIMESTAMP WITH TIME ZONE,
    success BOOLEAN,
    message TEXT
);

CREATE INDEX IF NOT EXISTS idx_scheduled_job_runs_job ON public.scheduled_job_runs (job_id);

-- Create catalog_edge table
CREATE TABLE IF NOT EXISTS public.catalog_edge (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_datasource_id uuid NOT NULL,
    source_node_id uuid NOT NULL,
    target_node_id uuid NOT NULL,
    relationship_type VARCHAR(255) NOT NULL,
    properties JSONB,
    edge_type_id uuid,
    tenant_id uuid NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_catalog_node_tenant_datasource ON public.catalog_node(tenant_datasource_id);
CREATE INDEX IF NOT EXISTS idx_catalog_node_type ON public.catalog_node(node_type_id);
CREATE INDEX IF NOT EXISTS idx_catalog_node_tenant ON public.catalog_node(tenant_id);
CREATE INDEX IF NOT EXISTS idx_catalog_node_qualified_path ON public.catalog_node(qualified_path);
CREATE INDEX IF NOT EXISTS idx_catalog_node_properties ON public.catalog_node USING GIN(properties);
CREATE INDEX IF NOT EXISTS idx_catalog_node_type_properties ON public.catalog_node_type USING GIN(properties);

CREATE INDEX IF NOT EXISTS idx_catalog_edge_tenant_datasource ON public.catalog_edge(tenant_datasource_id);
CREATE INDEX IF NOT EXISTS idx_catalog_edge_source ON public.catalog_edge(source_node_id);
CREATE INDEX IF NOT EXISTS idx_catalog_edge_target ON public.catalog_edge(target_node_id);
CREATE INDEX IF NOT EXISTS idx_catalog_edge_type ON public.catalog_edge(edge_type_id);
CREATE INDEX IF NOT EXISTS idx_catalog_edge_tenant ON public.catalog_edge(tenant_id);
CREATE INDEX IF NOT EXISTS idx_catalog_edge_properties ON public.catalog_edge USING GIN(properties);
CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_config ON public.catalog_edge_types USING GIN(config);

-- Insert default tenant
INSERT INTO public.tenants (id, name) VALUES ('default', 'Default Tenant')
ON CONFLICT (id) DO NOTHING;

-- Insert default datasource
INSERT INTO public.tenant_datasources (tenant_id, datasource_id, connection_string)
VALUES ('default', 'default', 'postgres://semlayer_user:semlayer_password@localhost:5432/semlayer_db')
ON CONFLICT (tenant_id, datasource_id) DO NOTHING;

-- Insert basic node types
INSERT INTO public.catalog_node_type (id, tenant_id, catalog_type_name, description, config) VALUES
('schema_type', 'default', 'schema', 'Database Schema', '{"description": "Represents a database schema"}'),
('table_type', 'default', 'table', 'Database Table', '{"description": "Represents a database table"}'),
('column_type', 'default', 'column', 'Database Column', '{"description": "Represents a database column"}'),
('semantic_model_type', 'default', 'semantic_model', 'Semantic Model', '{"description": "Represents a semantic model/cube"}'),
('semantic_column_type', 'default', 'semantic_column', 'Semantic Column', '{"description": "Represents a measure or dimension"}'),
('semantic_view_type', 'default', 'semantic_view', 'Semantic View', '{"description": "Represents a semantic view"}'),
('business_term_type', 'default', 'business_term', 'Business Term', '{"description": "Represents a business term or concept"}'),
('semantic_term_type', 'default', 'semantic_term', 'Semantic Term', '{"description": "Represents a semantic term"}')
ON CONFLICT (tenant_id, catalog_type_name) DO NOTHING;

-- Update semantic_model_type with property schema definitions
UPDATE public.catalog_node_type 
SET properties = '[
  {"name": "technical_name", "title": "Technical Name/ID", "data_type": "string", "order": 1, "required": false, "description": "Internal unique identifier"},
  {"name": "model_type", "title": "Model Type", "data_type": "string", "order": 2, "required": true, "input_type": "select", "options": ["core", "custom"], "default": "core", "description": "Core (read-only) or Custom (user-defined)"},
  {"name": "data_source_description", "title": "Data Source Description", "data_type": "string", "order": 3, "required": false, "description": "Description of underlying data source(s)"},
  {"name": "schema_table_reference", "title": "Schema/Table Reference", "data_type": "string", "order": 4, "required": false, "description": "Database schema.table or file paths"},
  {"name": "extends_model_id", "title": "Extends Model", "data_type": "string", "order": 5, "required": false, "input_type": "lookup", "lookup_type": "semantic_model", "description": "Reference to Core or Custom model this extends"},
  {"name": "linked_semantic_terms", "title": "Linked Semantic Terms", "data_type": "array", "order": 6, "required": false, "description": "List of semantic term IDs used in this model"},
  {"name": "overridden_properties", "title": "Overridden Term Properties", "data_type": "jsonb", "order": 7, "required": false, "description": "Override properties for inherited semantic terms"},
  {"name": "model_calculations", "title": "Model-Specific Calculations", "data_type": "jsonb", "order": 8, "required": false, "description": "Complex calculations combining semantic terms"}
]'::jsonb
WHERE catalog_type_name = 'semantic_model' AND tenant_id = 'default';

-- Insert basic edge types
INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, source_node_type_id, target_node_type_id) VALUES
('foreign_key_edge', 'default', 'foreign_key', 'Foreign Key Relationship', 'column_type', 'column_type'),
('has_semantic_edge', 'default', 'has_semantic', 'Has Semantic Mapping', 'business_term_type', 'semantic_model_type'),
('mapped_to_edge', 'default', 'mapped_to', 'Mapped To', 'semantic_column_type', 'column_type'),
('member_of_edge', 'default', 'member_of', 'Member Of', 'semantic_column_type', 'business_term_type'),
('joins_edge', 'default', 'joins', 'Joins With', 'semantic_model_type', 'semantic_model_type'),
('references_edge', 'default', 'references', 'References', 'semantic_model_type', 'semantic_model_type'),
('extends_edge', 'default', 'extends', 'Extends', 'semantic_model_type', 'semantic_model_type'),
('parent_of_edge', 'default', 'parent_of', 'Parent Of', 'business_term_type', 'business_term_type'),
('3be9d6ae-1598-4628-a3dd-b606921a9193', 'default', 'business_term_mapping', 'Business Term to Semantic Term Mapping', 'business_term_type', 'semantic_term_type'),
('semantic_model_extends_edge', 'default', 'semantic_model_extends', 'Semantic Model Extends', 'semantic_model_type', 'semantic_model_type'),
('semantic_model_links_to_edge', 'default', 'semantic_model_links_to', 'Semantic Model Links To Semantic Term', 'semantic_model_type', 'semantic_term_type')
ON CONFLICT (tenant_id, edge_type_name) DO NOTHING;

-- Create views for Hasura
CREATE OR REPLACE VIEW public.business_terms AS
SELECT
    cn.id,
    cn.node_name as name,
    cn.display_name,
    cn.description,
    (cn.properties->>'business_term_id') as business_term_id,
    (cn.properties->>'category') as category,
    (cn.properties->>'sub_category') as sub_category,
    (cn.properties->>'owner') as owner,
    (cn.properties->>'steward') as steward,
    (cn.properties->>'status') as status,
    (cn.properties->>'version') as version,
    (cn.properties->>'parent_id') as parent_id,
    (cn.properties->>'tags') as tags,
    cn.properties,
    cn.created_at,
    cn.updated_at,
    cn.tenant_id
FROM public.catalog_node cn
JOIN public.catalog_node_type cnt ON cn.node_type_id = cnt.id
WHERE cnt.catalog_type_name = 'business_term';

CREATE OR REPLACE VIEW public.semantic_models AS
SELECT
    cn.id,
    cn.node_name as name,
    cn.display_name,
    cn.description,
    (cn.properties->>'sql_table') as sql_table,
    (cn.properties->>'data_source') as data_source,
    (cn.properties->>'is_view')::boolean as is_view,
    (cn.properties->>'business_term_ids') as business_term_ids,
    (cn.properties->>'business_terms') as business_terms,
    cn.properties,
    cn.created_at,
    cn.updated_at,
    cn.tenant_id
FROM public.catalog_node cn
JOIN public.catalog_node_type cnt ON cn.node_type_id = cnt.id
WHERE cnt.catalog_type_name IN ('semantic_model', 'semantic_view');

CREATE OR REPLACE VIEW public.semantic_columns AS
SELECT
    cn.id,
    cn.node_name as name,
    cn.display_name,
    cn.description,
    (cn.properties->>'type') as column_type,
    (cn.properties->>'sql') as sql,
    (cn.properties->>'primary_key')::boolean as primary_key,
    (cn.properties->>'business_term_ids') as business_term_ids,
    (cn.properties->>'business_terms') as business_terms,
    cn.parent_id as model_id,
    cn.properties,
    cn.created_at,
    cn.updated_at,
    cn.tenant_id
FROM public.catalog_node cn
JOIN public.catalog_node_type cnt ON cn.node_type_id = cnt.id
WHERE cnt.catalog_type_name = 'semantic_column';

-- Grant permissions for Hasura
GRANT SELECT ON ALL TABLES IN SCHEMA public TO semlayer_user;
GRANT USAGE ON SCHEMA public TO semlayer_user;

-- Validation Rules Engine Tables
CREATE TABLE IF NOT EXISTS public.validation_rules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    datasource_id uuid NOT NULL,
    rule_name varchar(255) NOT NULL,
    rule_type varchar(50) NOT NULL,
    description text NULL,
    account_types text[] DEFAULT '{}'::text[] NOT NULL,
    parameters jsonb NOT NULL,
    severity varchar(20) NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    evaluation_order int4 DEFAULT 100 NOT NULL,
    allow_override bool DEFAULT false NOT NULL,
    required_authority varchar(50) NULL,
    created_by uuid NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT validation_rules_pkey PRIMARY KEY (id),
    CONSTRAINT validation_rules_tenant_datasource_name_key UNIQUE (tenant_id, datasource_id, rule_name),
    CONSTRAINT validation_rules_severity_check CHECK (severity = ANY (ARRAY['BLOCK'::character varying, 'WARNING'::character varying, 'INFO'::character varying]))
);

CREATE INDEX idx_validation_rules_tenant ON public.validation_rules USING btree (tenant_id, datasource_id);
CREATE INDEX idx_validation_rules_type ON public.validation_rules USING btree (rule_type);
CREATE INDEX idx_validation_rules_active ON public.validation_rules USING btree (is_active) WHERE (is_active = true);

CREATE TABLE IF NOT EXISTS public.validation_results (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    datasource_id uuid NOT NULL,
    account_id varchar(255) NOT NULL,
    account_type varchar(50) NOT NULL,
    rule_id uuid NOT NULL,
    rule_type varchar(50) NOT NULL,
    passed bool NOT NULL,
    severity varchar(20) NOT NULL,
    message text NULL,
    failed_value jsonb NULL,
    threshold_value jsonb NULL,
    details jsonb NULL,
    executed_at timestamptz DEFAULT now() NOT NULL,
    expires_at timestamptz NULL,
    CONSTRAINT validation_results_pkey PRIMARY KEY (id),
    CONSTRAINT validation_results_rule_fk FOREIGN KEY (rule_id) REFERENCES public.validation_rules(id) ON DELETE CASCADE
);

CREATE INDEX idx_validation_results_tenant ON public.validation_results USING btree (tenant_id, datasource_id);
CREATE INDEX idx_validation_results_account ON public.validation_results USING btree (account_id);
CREATE INDEX idx_validation_results_executed ON public.validation_results USING btree (executed_at DESC);
