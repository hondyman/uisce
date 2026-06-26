-- Migration script to bring database up to date with alpha database schema
-- This script adds missing tables, columns, indexes, and constraints without dropping existing data

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Helper to safely add tenant foreign keys when types/columns have been canonicalized
-- This supports a non-destructive two-phase migration: tenants.id_uuid + <table>.tenant_id_uuid
-- If *_uuid columns exist on both sides, the FK will be created against those. Otherwise
-- it will create the FK only if the existing column types match.
CREATE OR REPLACE FUNCTION public.safe_add_tenant_fk(target_table TEXT, target_column TEXT, constraint_name TEXT, on_delete_clause TEXT DEFAULT 'ON DELETE CASCADE') RETURNS VOID AS $$
DECLARE
    target_col_exists BOOLEAN;
    tenant_id_uuid_exists BOOLEAN;
    tenant_id_exists BOOLEAN;
    target_col_uuid_exists BOOLEAN;
    target_col_type TEXT;
    tenant_id_type TEXT;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=target_table) THEN
        RETURN;
    END IF;

    SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=target_table AND column_name=target_column) INTO target_col_exists;
    SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='tenants' AND column_name='id_uuid') INTO tenant_id_uuid_exists;
    SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='tenants' AND column_name='id') INTO tenant_id_exists;
    SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=target_table AND column_name=target_column||'_uuid') INTO target_col_uuid_exists;

    IF target_col_exists AND tenant_id_uuid_exists AND target_col_uuid_exists THEN
        EXECUTE format('ALTER TABLE public.%I ADD CONSTRAINT %I FOREIGN KEY (%I_uuid) REFERENCES public.tenants(id_uuid) %s', target_table, constraint_name, target_column, on_delete_clause);
        RETURN;
    END IF;

    IF target_col_exists AND tenant_id_exists THEN
        SELECT data_type INTO target_col_type FROM information_schema.columns WHERE table_schema='public' AND table_name=target_table AND column_name=target_column LIMIT 1;
        SELECT data_type INTO tenant_id_type FROM information_schema.columns WHERE table_schema='public' AND table_name='tenants' AND column_name='id' LIMIT 1;
        IF target_col_type = tenant_id_type THEN
            EXECUTE format('ALTER TABLE public.%I ADD CONSTRAINT %I FOREIGN KEY (%I) REFERENCES public.tenants(id) %s', target_table, constraint_name, target_column, on_delete_clause);
        END IF;
    END IF;
EXCEPTION WHEN duplicate_object THEN
    NULL;
END;
$$ LANGUAGE plpgsql;
-- Helper: safe_add_column
-- Adds a column to a table only if the target is a real table (not a view) and the column does not exist.
-- Any runtime errors during ALTER are caught and logged as notices so the migration can continue.
DROP FUNCTION IF EXISTS public.safe_add_column(TEXT, TEXT, TEXT);
CREATE OR REPLACE FUNCTION public.safe_add_column(p_schema_name TEXT, p_table_name TEXT, p_column_def TEXT) RETURNS VOID AS $$
DECLARE
    exists_table BOOLEAN;
    exists_column BOOLEAN;
    candidate_col TEXT;
BEGIN
    SELECT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = p_schema_name AND c.relname = p_table_name AND c.relkind = 'r') INTO exists_table;
    IF NOT exists_table THEN
        -- table doesn't exist or is not a base table; skip safely
        RETURN;
    END IF;

    -- column name is first token before space or '(' in the column_def
    candidate_col := split_part(p_column_def, ' ', 1);
    SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = p_schema_name AND table_name = p_table_name AND column_name = candidate_col) INTO exists_column;
    IF exists_column THEN
        RETURN;
    END IF;

    BEGIN
        EXECUTE format('ALTER TABLE %I.%I ADD COLUMN %s', p_schema_name, p_table_name, p_column_def);
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_column skipped for %I.%I column_def="%" due to: %', p_schema_name, p_table_name, p_column_def, SQLERRM;
    END;
END;
$$ LANGUAGE plpgsql;

-- Phase-A canonicalization: add tenants.id_uuid and backfill, plus tenant_id_uuid on tables that have tenant_id
DO $$ DECLARE
    r RECORD;
BEGIN
    -- Ensure tenants.id_uuid exists and is populated
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='tenants' AND column_name='id_uuid') THEN
        ALTER TABLE public.tenants ADD COLUMN id_uuid uuid DEFAULT gen_random_uuid();
        -- backfill from textual id if it's a UUID, else generate
        UPDATE public.tenants SET id_uuid = (CASE WHEN id ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN id::uuid ELSE gen_random_uuid() END);
        ALTER TABLE public.tenants ALTER COLUMN id_uuid SET NOT NULL;
        CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_id_uuid ON public.tenants(id_uuid);
    END IF;

    -- For every public table that has tenant_id, add tenant_id_uuid and attempt a best-effort backfill
    FOR r IN SELECT DISTINCT table_name FROM information_schema.columns WHERE table_schema='public' AND column_name='tenant_id' LOOP
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=r.table_name AND column_name='tenant_id_uuid') THEN
            -- Use safe_add_column to avoid ALTERs on views or missing relations
            PERFORM public.safe_add_column('public', r.table_name::text, 'tenant_id_uuid uuid');
            -- best-effort backfill by joining on textual id -> tenants.id (still guarded)
            BEGIN
                EXECUTE format('UPDATE public.%I SET tenant_id_uuid = t.id_uuid FROM public.tenants t WHERE t.id = public.%I.tenant_id', r.table_name, r.table_name);
            EXCEPTION WHEN others THEN
                -- ignore failures (table may be empty or types incompatible)
                NULL;
            END;
        END IF;
    END LOOP;
END $$;

-- Helper: safe_add_column
-- Adds a column to a table only if the target is a real table (not a view) and the column does not exist.
-- Any runtime errors during ALTER are caught and logged as notices so the migration can continue.
DROP FUNCTION IF EXISTS public.safe_add_column(TEXT, TEXT, TEXT);
CREATE OR REPLACE FUNCTION public.safe_add_column(p_schema_name TEXT, p_table_name TEXT, p_column_def TEXT) RETURNS VOID AS $$
DECLARE
    exists_table BOOLEAN;
    exists_column BOOLEAN;
    candidate_col TEXT;
BEGIN
    SELECT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = p_schema_name AND c.relname = p_table_name AND c.relkind = 'r') INTO exists_table;
    IF NOT exists_table THEN
        -- table doesn't exist or is not a base table; skip safely
        RETURN;
    END IF;

    -- column name is first token before space or '(' in the column_def
    candidate_col := split_part(p_column_def, ' ', 1);
    SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = p_schema_name AND table_name = p_table_name AND column_name = candidate_col) INTO exists_column;
    IF exists_column THEN
        RETURN;
    END IF;

    BEGIN
        EXECUTE format('ALTER TABLE %I.%I ADD COLUMN %s', p_schema_name, p_table_name, p_column_def);
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_column skipped for %I.%I column_def="%" due to: %', p_schema_name, p_table_name, p_column_def, SQLERRM;
    END;
END;
$$ LANGUAGE plpgsql;

-- Create custom types if they don't exist
DO $$ BEGIN
    CREATE TYPE public.fabric_status AS ENUM ('draft', 'published', 'archived');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE public.join_relationship AS ENUM ('one_to_one', 'one_to_many', 'many_to_one', 'many_to_many');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create sequences if they don't exist
CREATE SEQUENCE IF NOT EXISTS public.fabric_defn_audit_audit_id_seq
    INCREMENT BY 1
    MINVALUE 1
    MAXVALUE 9223372036854775807
    START 1
    CACHE 1
    NO CYCLE;

-- Add missing columns to existing tenants table
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'tenants' AND column_name = 'name') THEN
        ALTER TABLE public.tenants ADD COLUMN name VARCHAR(255);
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'tenants' AND column_name = 'is_active') THEN
        ALTER TABLE public.tenants ADD COLUMN is_active BOOLEAN DEFAULT true;
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'tenants' AND column_name = 'tenant_code') THEN
        ALTER TABLE public.tenants ADD COLUMN tenant_code VARCHAR(255);
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'tenants' AND column_name = 'display_name') THEN
        ALTER TABLE public.tenants ADD COLUMN display_name VARCHAR(255) NOT NULL DEFAULT '';
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'tenants' AND column_name = 'description') THEN
        ALTER TABLE public.tenants ADD COLUMN description TEXT;
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'tenants' AND column_name = 'status') THEN
        ALTER TABLE public.tenants ADD COLUMN status VARCHAR(50) DEFAULT 'active';
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'tenants' AND column_name = 'created_at') THEN
        ALTER TABLE public.tenants ADD COLUMN created_at TIMESTAMPTZ DEFAULT now();
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'tenants' AND column_name = 'updated_at') THEN
        ALTER TABLE public.tenants ADD COLUMN updated_at TIMESTAMPTZ DEFAULT now();
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'tenants' AND column_name = 'gold_copy') THEN
        ALTER TABLE public.tenants ADD COLUMN gold_copy BOOLEAN DEFAULT false;
    END IF;
END $$;

-- Update existing tenants to have display_name if empty
UPDATE public.tenants SET display_name = name WHERE display_name = '' AND name IS NOT NULL;

-- Add unique constraints (use conditional DO blocks because "ADD CONSTRAINT IF NOT EXISTS" is not supported)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tenants_unique') THEN
        EXECUTE 'ALTER TABLE public.tenants ADD CONSTRAINT tenants_unique UNIQUE (name)';
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tenants_unique_1') THEN
        EXECUTE 'ALTER TABLE public.tenants ADD CONSTRAINT tenants_unique_1 UNIQUE (tenant_code)';
    END IF;
END $$;



-- Helper: safe_add_constraint
-- Executes an ALTER TABLE ... ADD CONSTRAINT statement only if the constraint name does not already exist.
-- If the target table does not exist, or execution fails (type mismatch, ordering), the error is caught and logged.
CREATE OR REPLACE FUNCTION public.safe_add_constraint(constraint_name TEXT, alter_sql TEXT) RETURNS VOID AS $$
DECLARE
    exists_constraint BOOLEAN;
BEGIN
    SELECT EXISTS(SELECT 1 FROM pg_constraint WHERE conname = constraint_name) INTO exists_constraint;
    IF exists_constraint THEN
        RETURN;
    END IF;

    BEGIN
        EXECUTE alter_sql;
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_constraint skipped constraint % due to: %', constraint_name, SQLERRM;
    END;
END;
$$ LANGUAGE plpgsql;

-- Create core tables that the backend expects
CREATE TABLE IF NOT EXISTS public.tenant (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.app_user (
  id TEXT PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  display_name TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS public.user_tenant (
  user_id TEXT REFERENCES app_user(id) ON DELETE CASCADE,
  tenant_id TEXT REFERENCES tenant(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, tenant_id)
);

-- Semantic assets registry
CREATE TABLE IF NOT EXISTS public.asset (
  id UUID PRIMARY KEY,
  tenant_id TEXT REFERENCES tenant(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  asset_type TEXT NOT NULL, -- 'view','metric','dimension','dashboard'
  domain TEXT NOT NULL,
  certified BOOLEAN NOT NULL DEFAULT FALSE,
  sensitivity TEXT NOT NULL DEFAULT 'medium', -- 'low','medium','high'
  created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_asset_tenant_domain ON asset(tenant_id, domain);
CREATE INDEX IF NOT EXISTS idx_asset_cert ON asset(tenant_id, certified);

-- Roles and role membership
CREATE TABLE IF NOT EXISTS public.role (
  id UUID PRIMARY KEY,
  tenant_id TEXT REFERENCES tenant(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  UNIQUE (tenant_id, name)
);

CREATE TABLE IF NOT EXISTS public.role_member (
  role_id UUID REFERENCES role(id) ON DELETE CASCADE,
  user_id TEXT REFERENCES app_user(id) ON DELETE CASCADE,
  tenant_id TEXT REFERENCES tenant(id) ON DELETE CASCADE,
  PRIMARY KEY (role_id, user_id, tenant_id)
);

CREATE TABLE IF NOT EXISTS public.role_claim (
  id UUID PRIMARY KEY,
  role_id UUID REFERENCES role(id) ON DELETE CASCADE,
  asset_id UUID REFERENCES asset(id) ON DELETE CASCADE,
  permission TEXT NOT NULL,
  scope TEXT[] NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_role_claim_role ON role_claim(role_id);

-- Alpha datasource table
CREATE TABLE IF NOT EXISTS public.alpha_datasource (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    datasource_name varchar NOT NULL,
    datasource_code varchar NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    config jsonb NULL,
    created_at timestamptz NULL,
    updated_at timestamptz NULL,
    datasource_type varchar NULL,
    CONSTRAINT alpha_datasources_pk PRIMARY KEY (id),
    CONSTRAINT alpha_datasources_unique UNIQUE (datasource_code)
);

-- Alpha product table
CREATE TABLE IF NOT EXISTS public.alpha_product (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    product_name varchar(255) NULL,
    is_active bool DEFAULT true NOT NULL,
    product_code varchar(255) NULL,
    status varchar(50) DEFAULT 'active'::character varying NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT product_pkey PRIMARY KEY (id),
    CONSTRAINT product_unique UNIQUE (product_name)
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS public.audit_logs (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    datasource_id uuid NOT NULL,
    policy_id uuid NOT NULL,
    access_time timestamptz DEFAULT now() NOT NULL,
    "action" varchar(50) NOT NULL,
    details jsonb NULL,
    instance_datasource_id uuid NULL,
    tenant_id uuid NOT NULL,
    "type" varchar(50) NOT NULL,
    resource_type varchar(50) NOT NULL,
    resource_id uuid NOT NULL,
    request_id varchar(100) NULL,
    "timestamp" timestamptz DEFAULT now() NOT NULL,
    duration int8 NULL,
    status varchar(50) NOT NULL,
    error_message text NULL,
    request_data jsonb NULL,
    response_data jsonb NULL,
    ip_address inet NULL,
    user_agent text NULL,
    CONSTRAINT audit_logs_pkey PRIMARY KEY (id),
    CONSTRAINT action_check CHECK ((action)::text = ANY (ARRAY[('allow'::character varying)::text, ('deny'::character varying)::text, ('mask'::character varying)::text]))
);

-- Bundle change proposal table
CREATE TABLE IF NOT EXISTS public.bundle_change_proposal (
    id uuid NOT NULL,
    bundle_id uuid NULL,
    proposed_version int4 NULL,
    change_type text NULL,
    details jsonb NULL,
    fitness_score float8 NULL,
    risk_score float8 NULL,
    impact jsonb NULL,
    status text NULL,
    created_at timestamptz NULL,
    decided_at timestamptz NULL,
    decided_by text NULL,
    CONSTRAINT bundle_change_proposal_pkey PRIMARY KEY (id)
);

-- Candidate bundles table
CREATE TABLE IF NOT EXISTS public.candidate_bundles (
    id text NOT NULL,
    tenant_id text NULL,
    "name" text NULL,
    description text NULL,
    claims jsonb NULL,
    "scope" text NULL,
    score float8 NULL,
    risk float8 NULL,
    explanations jsonb NULL,
    status text NULL,
    created_at timestamptz NULL,
    CONSTRAINT candidate_bundles_pkey PRIMARY KEY (id)
);

-- Claim bundle table
CREATE TABLE IF NOT EXISTS public.claim_bundle (
    id uuid NOT NULL,
    "name" text NULL,
    "version" int4 NULL,
    "domain" text NULL,
    description text NULL,
    created_by text NULL,
    created_at timestamptz NULL,
    status text NULL,
    risk_level text NULL,
    CONSTRAINT claim_bundle_pkey PRIMARY KEY (id)
);

-- Claim bundle item table
CREATE TABLE IF NOT EXISTS public.claim_bundle_item (
    id uuid NOT NULL,
    bundle_id uuid NULL,
    model_id uuid NULL,
    "permission" text NULL,
    "scope" jsonb NULL,
    CONSTRAINT claim_bundle_item_pkey PRIMARY KEY (id)
);

-- Connection pool metrics table
CREATE TABLE IF NOT EXISTS public.connection_pool_metrics (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    pool_name varchar(100) NOT NULL,
    total_connections int4 NOT NULL,
    active_connections int4 NOT NULL,
    idle_connections int4 NOT NULL,
    waiting_requests int4 NOT NULL,
    created_at timestamptz DEFAULT now() NULL,
    CONSTRAINT connection_pool_metrics_pkey PRIMARY KEY (id)
);

-- Drift reports table
CREATE TABLE IF NOT EXISTS public.drift_reports (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    generated_at timestamptz NOT NULL,
    schema_hash text NOT NULL,
    severity_summary jsonb NOT NULL,
    changelog_md text NULL,
    changelog_html text NULL,
    raw_report jsonb NOT NULL,
    CONSTRAINT drift_reports_pkey PRIMARY KEY (id)
);

-- Explorer saved query table
CREATE TABLE IF NOT EXISTS public.explorer_saved_query (
    id uuid NOT NULL,
    created_at timestamp DEFAULT now() NOT NULL,
    updated_at timestamp DEFAULT now() NOT NULL,
    owner_user_id text NOT NULL,
    owner_tenant_id text NOT NULL,
    "name" text NOT NULL,
    description text NULL,
    tags _text NULL,
    view_name text NOT NULL,
    query jsonb NOT NULL,
    viz_config jsonb NULL,
    last_run_at timestamp NULL,
    last_duration_ms int4 NULL,
    CONSTRAINT explorer_saved_query_pkey PRIMARY KEY (id)
);

-- Fabric definition table
CREATE TABLE IF NOT EXISTS public.fabric_defn (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    tenant_datasource_id uuid NOT NULL,
    model_key text NOT NULL,
    "version" int4 NOT NULL,
    status public.fabric_status DEFAULT 'draft'::fabric_status NOT NULL,
    is_current bool DEFAULT false NOT NULL,
    title text NULL,
    description text NULL,
    source_config jsonb NOT NULL,
    resolved_config jsonb NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    published_at timestamptz NULL,
    checksum_sha256 bytea NULL,
    updated_at timestamptz NULL,
    CONSTRAINT fabric_defn_pkey PRIMARY KEY (id),
    CONSTRAINT fabric_defn_tenant_id_model_key_version_key UNIQUE (tenant_id, model_key, version)
);

-- Fabric definition audit table
CREATE TABLE IF NOT EXISTS public.fabric_defn_audit (
    audit_id bigserial NOT NULL,
    defn_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    model_key text NOT NULL,
    "version" int4 NOT NULL,
    "action" text NOT NULL,
    "at" timestamptz DEFAULT now() NOT NULL,
    actor_id uuid NOT NULL,
    before_doc jsonb NULL,
    after_doc jsonb NULL,
    CONSTRAINT fabric_defn_audit_action_check CHECK ((action = ANY (ARRAY['create'::text, 'update'::text, 'publish'::text, 'archive'::text]))),
    CONSTRAINT fabric_defn_audit_pkey PRIMARY KEY (audit_id)
);

-- Fabric definition index table
CREATE TABLE IF NOT EXISTS public.fabric_defn_index (
    defn_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    model_key text NOT NULL,
    "version" int4 NOT NULL,
    kind text NOT NULL,
    "name" text NOT NULL,
    "type" text NULL,
    relationship public.join_relationship NULL,
    title text NULL,
    description text NULL,
    "sql" text NULL,
    CONSTRAINT fabric_defn_index_kind_check CHECK ((kind = ANY (ARRAY['dimension'::text, 'measure'::text, 'join'::text]))),
    CONSTRAINT fabric_defn_index_tenant_id_model_key_version_kind_name_key UNIQUE (tenant_id, model_key, version, kind, name)
);

-- Integration logs table
CREATE TABLE IF NOT EXISTS public.integration_logs (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    integration_id uuid NOT NULL,
    event_id uuid NOT NULL,
    request_id uuid NOT NULL,
    execution_start timestamptz NOT NULL,
    execution_end timestamptz NOT NULL,
    duration_ms int4 NOT NULL,
    status varchar(20) NOT NULL,
    status_code int4 NULL,
    error_message text NULL,
    request_body text NULL,
    response_body text NULL,
    attempt_count int4 DEFAULT 1 NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT integration_logs_pkey PRIMARY KEY (id)
);

-- Message templates table
CREATE TABLE IF NOT EXISTS public.message_templates (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    "name" varchar(255) NOT NULL,
    description text NULL,
    teams_json text NOT NULL,
    slack_json text NOT NULL,
    tenant_id uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT message_templates_pkey PRIMARY KEY (id)
);

-- Model upgrade audit table
CREATE TABLE IF NOT EXISTS public.model_upgrade_audit (
    id uuid NOT NULL,
    diff_id uuid NULL,
    model_name text NOT NULL,
    field_path text NULL,
    rule_id text NULL,
    provenance text NULL,
    decision text NULL,
    reviewer text NULL,
    reason text NULL,
    event_type text NOT NULL,
    decided_at timestamp DEFAULT now() NOT NULL,
    CONSTRAINT model_upgrade_audit_event_type_check CHECK ((event_type = ANY (ARRAY['upgrade'::text, 'tuning'::text, 'preagg'::text]))),
    CONSTRAINT model_upgrade_audit_pkey PRIMARY KEY (id)
);

-- Performance metrics table
CREATE TABLE IF NOT EXISTS public.performance_metrics (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id varchar(255) NOT NULL,
    metric_name varchar(100) NOT NULL,
    metric_value numeric(15, 6) NOT NULL,
    labels jsonb NULL,
    collected_at timestamptz DEFAULT now() NULL,
    CONSTRAINT performance_metrics_pkey PRIMARY KEY (id)
);

-- Permissions table
CREATE TABLE IF NOT EXISTS public.permissions (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    "name" varchar(255) NOT NULL,
    description varchar(255) NULL,
    resource_type varchar(50) NOT NULL,
    "action" varchar(50) NOT NULL,
    is_system bool DEFAULT false NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT permissions_pkey PRIMARY KEY (id),
    CONSTRAINT uq_name UNIQUE (name)
);

-- Policies table
CREATE TABLE IF NOT EXISTS public.policies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    "name" text NOT NULL,
    rules jsonb NULL,
    start_date timestamp NULL,
    end_date timestamp NULL,
    schedule jsonb NULL,
    location_rules jsonb NULL,
    priority int4 DEFAULT 0 NOT NULL,
    active bool DEFAULT true NOT NULL,
    created_at timestamp DEFAULT now() NOT NULL,
    updated_at timestamp DEFAULT now() NOT NULL,
    CONSTRAINT policies_pkey PRIMARY KEY (id)
);

-- Policy evaluation table
CREATE TABLE IF NOT EXISTS public.policy_evaluation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    run_id uuid NOT NULL,
    policy_id text NOT NULL,
    policy_version int4 NOT NULL,
    evaluated_at timestamptz DEFAULT now() NOT NULL,
    "result" jsonb NOT NULL,
    decision text NOT NULL,
    exit_code int4 NOT NULL,
    actor text NULL,
    context jsonb NULL,
    CONSTRAINT policy_evaluation_pkey PRIMARY KEY (id)
);

-- Policy set table
CREATE TABLE IF NOT EXISTS public.policy_set (
    id text NOT NULL,
    "version" int4 NOT NULL,
    title text NULL,
    "source" text NULL,
    "owner" text NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    spec jsonb NOT NULL,
    active bool DEFAULT true NOT NULL,
    CONSTRAINT policy_set_pkey PRIMARY KEY (id)
);

-- Policy version history table
CREATE TABLE IF NOT EXISTS public.policy_version_history (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    policy_id text NOT NULL,
    "version" int4 NOT NULL,
    spec jsonb NOT NULL,
    author text NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    change_summary text NULL,
    CONSTRAINT policy_version_history_pkey PRIMARY KEY (id)
);

-- Prepared statement metrics table
CREATE TABLE IF NOT EXISTS public.prepared_statement_metrics (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    query_hash varchar(64) NOT NULL,
    query_text text NOT NULL,
    execution_count int8 DEFAULT 0 NULL,
    total_time_ms int8 DEFAULT 0 NULL,
    avg_time_ms numeric(10, 2) DEFAULT 0 NULL,
    last_executed timestamptz NULL,
    created_at timestamptz DEFAULT now() NULL,
    CONSTRAINT prepared_statement_metrics_pkey PRIMARY KEY (id),
    CONSTRAINT prepared_statement_metrics_query_hash_key UNIQUE (query_hash)
);

-- Rule config changelog table
CREATE TABLE IF NOT EXISTS public.rule_config_changelog (
    id uuid NOT NULL,
    rule_id text NOT NULL,
    old_aggressiveness numeric NULL,
    new_aggressiveness numeric NULL,
    old_auto_accept bool NULL,
    new_auto_accept bool NULL,
    "scope" text NOT NULL,
    reason text NOT NULL,
    triggered_by text NOT NULL,
    triggered_at timestamp DEFAULT now() NOT NULL,
    CONSTRAINT rule_config_changelog_pkey PRIMARY KEY (id)
);

-- Users table (extend existing if needed)
CREATE TABLE IF NOT EXISTS public.users (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    username varchar(255) NOT NULL,
    password_hash varchar(255) NULL,
    email varchar(255) NULL,
    firstname varchar(255) NULL,
    lastname varchar(255) NULL,
    tenant_id uuid NULL,
    first_name varchar(255) NULL,
    last_name varchar(255) NULL,
    status varchar(50) DEFAULT 'active'::character varying NOT NULL,
    last_login timestamptz NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_username_key UNIQUE (username)
);

-- Active requests table
CREATE TABLE IF NOT EXISTS public.active_requests (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    request_id varchar(255) NOT NULL,
    resource_type varchar(50) NOT NULL,
    resource_id uuid NOT NULL,
    user_id uuid NOT NULL,
    start_time timestamptz DEFAULT now() NOT NULL,
    status varchar(50) DEFAULT 'running'::character varying NOT NULL,
    progress int4 NULL,
    CONSTRAINT active_requests_pkey PRIMARY KEY (id),
    CONSTRAINT active_requests_status_check CHECK (((status)::text = ANY (ARRAY[('running'::character varying)::text, ('completed'::character varying)::text, ('failed'::character varying)::text, ('cancelled'::character varying)::text]))),
    CONSTRAINT active_requests_tenant_request_key UNIQUE (tenant_id, request_id)
);

-- API groups table
CREATE TABLE IF NOT EXISTS public.api_groups (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    "name" varchar(100) NOT NULL,
    display_name varchar(200) NULL,
    description text NULL,
    parent_group_id uuid NULL,
    is_active bool DEFAULT true NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NULL,
    updated_at timestamptz DEFAULT now() NULL,
    CONSTRAINT api_groups_pkey PRIMARY KEY (id),
    CONSTRAINT api_groups_tenant_id_name_key UNIQUE (tenant_id, name)
);

-- API keys table
CREATE TABLE IF NOT EXISTS public.api_keys (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    key_hash varchar(255) NOT NULL,
    description text NULL,
    permissions jsonb NOT NULL,
    expires_at timestamptz NULL,
    is_active bool DEFAULT true NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    last_used_at timestamptz NULL,
    CONSTRAINT api_keys_key_hash_key UNIQUE (key_hash),
    CONSTRAINT api_keys_pkey PRIMARY KEY (id),
    CONSTRAINT api_keys_tenant_name_key UNIQUE (tenant_id, name)
);

-- API workflow approvals table
CREATE TABLE IF NOT EXISTS public.api_workflow_approvals (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    request_id varchar(50) NOT NULL,
    request_type varchar(50) NOT NULL,
    requested_by uuid NOT NULL,
    status varchar(20) NOT NULL,
    resource_id uuid NULL,
    resource_type varchar(50) NULL,
    details jsonb NULL,
    approver_id uuid NULL,
    approval_step int4 DEFAULT 1 NULL,
    "comments" text NULL,
    created_at timestamptz DEFAULT now() NULL,
    updated_at timestamptz DEFAULT now() NULL,
    CONSTRAINT api_workflow_approvals_pkey PRIMARY KEY (id),
    CONSTRAINT api_workflow_approvals_status_check CHECK (((status)::text = ANY (ARRAY[('PENDING'::character varying)::text, ('APPROVED'::character varying)::text, ('REJECTED'::character varying)::text])))
);

-- APIs table
CREATE TABLE IF NOT EXISTS public.apis (
    id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    group_id uuid NULL,
    cloned_from_id uuid NULL,
    "name" varchar(255) NOT NULL,
    display_name varchar(255) NOT NULL,
    description text NULL,
    base_path varchar(255) NULL,
    api_type varchar(50) NOT NULL,
    visibility varchar(50) DEFAULT 'private'::character varying NOT NULL,
    "version" varchar(50) DEFAULT '1.0.0'::character varying NOT NULL,
    status varchar(50) DEFAULT 'draft'::character varying NOT NULL,
    specification jsonb NULL,
    is_core bool DEFAULT false NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    CONSTRAINT apis_api_type_check CHECK (((api_type)::text = ANY ((ARRAY['REST'::character varying, 'GraphQL'::character varying, 'OpenAPI'::character varying])::text[]))),
    CONSTRAINT apis_pkey PRIMARY KEY (id),
    CONSTRAINT apis_status_check CHECK (((status)::text = ANY ((ARRAY['draft'::character varying, 'active'::character varying, 'deprecated'::character varying, 'retired'::character varying])::text[]))),
    CONSTRAINT apis_visibility_check CHECK (((visibility)::text = ANY ((ARRAY['public'::character varying, 'private'::character varying, 'tenant-specific'::character varying])::text[])))
);

-- Broker APIs table
CREATE TABLE IF NOT EXISTS public.broker_apis (
    id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    api_id uuid NOT NULL,
    api_name varchar(255) NOT NULL,
    broker_id uuid NOT NULL,
    broker_name varchar(255) NOT NULL,
    status varchar(50) NOT NULL,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL,
    CONSTRAINT broker_apis_pkey PRIMARY KEY (id)
);

-- Broker events table
CREATE TABLE IF NOT EXISTS public.broker_events (
    id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    broker_id uuid NOT NULL,
    broker_name varchar(255) NOT NULL,
    event_name varchar(255) NOT NULL,
    event_description text NULL,
    "schema" jsonb NULL,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL,
    CONSTRAINT broker_events_pkey PRIMARY KEY (id),
    CONSTRAINT broker_events_tenant_id_event_name_key UNIQUE (tenant_id, event_name)
);

-- Customers table (extend existing)
DO $$ BEGIN
    PERFORM public.safe_add_column('public','customers','tenant_id uuid');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','customers','created_by uuid');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','customers','updated_by uuid');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','customers','created_at timestamp DEFAULT now()');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','customers','updated_at timestamp DEFAULT now()');
END $$;

-- Drift log entries table
CREATE TABLE IF NOT EXISTS public.drift_log_entries (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    report_id uuid NULL,
    severity text NULL,
    qualified_path text NOT NULL,
    explanation text NOT NULL,
    CONSTRAINT drift_log_entries_pkey PRIMARY KEY (id),
    CONSTRAINT drift_log_entries_severity_check CHECK ((severity = ANY (ARRAY['breaking'::text, 'medium'::text, 'low'::text])))
);

-- Event subscriptions table
CREATE TABLE IF NOT EXISTS public.event_subscriptions (
    id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    event_id uuid NOT NULL,
    event_name varchar(255) NOT NULL,
    callback_url text NOT NULL,
    status varchar(50) NOT NULL,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL,
    CONSTRAINT event_subscriptions_pkey PRIMARY KEY (id)
);

-- Exposed APIs table
CREATE TABLE IF NOT EXISTS public.exposed_apis (
    id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    api_id uuid NOT NULL,
    broker_api_id varchar(255) NOT NULL,
    "name" varchar(255) NOT NULL,
    description text NULL,
    is_active bool DEFAULT true NOT NULL,
    exposed_at timestamptz NOT NULL,
    last_synced_at timestamptz NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT exposed_apis_pkey PRIMARY KEY (id),
    CONSTRAINT exposed_apis_tenant_id_api_id_key UNIQUE (tenant_id, api_id)
);

-- Integration audit logs table
CREATE TABLE IF NOT EXISTS public.integration_audit_logs (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    "type" varchar(50) NOT NULL,
    "action" varchar(50) NOT NULL,
    resource_type varchar(50) NOT NULL,
    resource_id uuid NOT NULL,
    user_id uuid NOT NULL,
    "timestamp" timestamptz DEFAULT now() NOT NULL,
    duration int8 NULL,
    status varchar(50) NOT NULL,
    error_message text NULL,
    request_data jsonb NULL,
    response_data jsonb NULL,
    ip_address varchar(50) NULL,
    user_agent text NULL,
    CONSTRAINT integration_audit_logs_pkey PRIMARY KEY (id),
    CONSTRAINT integration_audit_logs_status_check CHECK (((status)::text = ANY (ARRAY[('success'::character varying)::text, ('error'::character varying)::text, ('warning'::character varying)::text, ('info'::character varying)::text]))),
    CONSTRAINT integration_audit_logs_type_check CHECK (((type)::text = ANY (ARRAY[('integration_execution'::character varying)::text, ('business_rule_execution'::character varying)::text, ('api_request'::character varying)::text, ('security_event'::character varying)::text])))
);

-- Integration configs table
CREATE TABLE IF NOT EXISTS public.integration_configs (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    description text NULL,
    event_id uuid NOT NULL,
    broker_endpoint varchar(1024) NOT NULL,
    headers text NULL,
    is_active bool DEFAULT true NOT NULL,
    retry_count int4 DEFAULT 3 NOT NULL,
    retry_delay int4 DEFAULT 60 NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT integration_configs_pkey PRIMARY KEY (id)
);

-- Integrations table
CREATE TABLE IF NOT EXISTS public.integrations (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    display_name varchar(255) NOT NULL,
    description text NULL,
    "type" varchar(50) NOT NULL,
    endpoint varchar(1024) NOT NULL,
    methods jsonb NOT NULL,
    required_permissions jsonb NOT NULL,
    rate_limits jsonb NOT NULL,
    timeout int4 NOT NULL,
    retry_policy jsonb NOT NULL,
    config jsonb NULL,
    status varchar(50) DEFAULT 'active'::character varying NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT integrations_pkey PRIMARY KEY (id),
    CONSTRAINT integrations_status_check CHECK (((status)::text = ANY (ARRAY[('active'::character varying)::text, ('inactive'::character varying)::text, ('draft'::character varying)::text, ('deprecated'::character varying)::text]))),
    CONSTRAINT integrations_tenant_name_key UNIQUE (tenant_id, name),
    CONSTRAINT integrations_type_check CHECK (((type)::text = ANY (ARRAY[('rest'::character varying)::text, ('kafka'::character varying)::text, ('azure'::character varying)::text])))
);

-- IP whitelist table
CREATE TABLE IF NOT EXISTS public.ip_whitelist (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    ip_address varchar(50) NOT NULL,
    description text NULL,
    created_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT ip_whitelist_pkey PRIMARY KEY (id),
    CONSTRAINT ip_whitelist_tenant_ip_key UNIQUE (tenant_id, ip_address)
);

-- Message targets table
CREATE TABLE IF NOT EXISTS public.message_targets (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    "name" varchar(255) NOT NULL,
    description text NULL,
    "type" varchar(50) NOT NULL,
    webhook_url text NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    template_id uuid NULL,
    tenant_id uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT message_targets_pkey PRIMARY KEY (id)
);

-- Metadata tables table (extend existing)
DO $$ BEGIN
    PERFORM public.safe_add_column('public','metadata_tables','tenant_id uuid');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','metadata_tables','datasource_id uuid');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','metadata_tables','display_name varchar(255)');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','metadata_tables','schema_name varchar(255) DEFAULT ''public''');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','metadata_tables','is_view bool DEFAULT false');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','metadata_tables','is_system bool DEFAULT false');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','metadata_tables','created_by uuid');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','metadata_tables','updated_by uuid');
END $$;

-- Orders table (extend existing)
DO $$ BEGIN
    PERFORM public.safe_add_column('public','orders','tenant_id uuid');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','orders','created_by uuid');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','orders','updated_by uuid');
END $$;

-- Policy violation table
CREATE TABLE IF NOT EXISTS public.policy_violation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    evaluation_id uuid NOT NULL,
    rule_id text NOT NULL,
    code text NOT NULL,
    severity text NOT NULL,
    on_violation text NOT NULL,
    message text NOT NULL,
    change_id text NOT NULL,
    object_fqn text NULL,
    details jsonb NULL,
    "explain" jsonb NULL,
    CONSTRAINT policy_violation_pkey PRIMARY KEY (id)
);

-- Roles table
CREATE TABLE IF NOT EXISTS public.roles (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    rolename varchar(255) NOT NULL,
    description varchar(255) NULL,
    is_active bool DEFAULT true NOT NULL,
    is_alpha bool DEFAULT false NOT NULL,
    tenant_id uuid NULL,
    is_system bool DEFAULT false NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT roles_pkey PRIMARY KEY (id),
    CONSTRAINT roles_unique UNIQUE (tenant_id, rolename)
);

-- Tenant connections table
CREATE TABLE IF NOT EXISTS public.tenant_connections (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    connection_name varchar(255) NOT NULL,
    database_type varchar(50) NOT NULL,
    dsn text NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT tenant_connections_pkey PRIMARY KEY (id)
);

-- Tenant instance table
CREATE TABLE IF NOT EXISTS public.tenant_instance (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    instance_name varchar(255) NOT NULL,
    config jsonb NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    url varchar(255) NULL,
    display_name varchar(255) NOT NULL,
    description text NULL,
    status varchar(50) DEFAULT 'active'::character varying NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT tenant_instance_pkey PRIMARY KEY (id),
    CONSTRAINT tenant_instance_unique UNIQUE (tenant_id, instance_name),
    CONSTRAINT tenant_instance_unique_1 UNIQUE (url)
);

-- Tenant product table
CREATE TABLE IF NOT EXISTS public.tenant_product (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_instance_id uuid NOT NULL,
    alpha_product_id uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    "version" float4 NOT NULL,
    is_active bool DEFAULT false NOT NULL,
    CONSTRAINT tenant_product_pkey PRIMARY KEY (id),
    CONSTRAINT tenant_product_uniq UNIQUE (tenant_instance_id, alpha_product_id)
);

-- Tenant product datasource table (extend existing)
-- Tenant product datasource shim (Phase-A): create minimal table if missing so constraints can be added safely
CREATE TABLE IF NOT EXISTS public.tenant_product_datasource (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_product_id uuid NULL,
    alpha_datasource_id uuid NULL,
    is_active bool DEFAULT true,
    config jsonb NULL,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    CONSTRAINT tenant_product_datasource_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_tenant_product_datasource_tenant_product ON public.tenant_product_datasource USING btree (tenant_product_id);

DO $$ BEGIN
    PERFORM public.safe_add_column('public','tenant_product_datasource','is_active bool DEFAULT true');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','tenant_product_datasource','config jsonb');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','tenant_product_datasource','created_at timestamptz DEFAULT now()');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','tenant_product_datasource','updated_at timestamptz DEFAULT now()');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','tenant_product_datasource','source_name varchar');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','tenant_product_datasource','id_text varchar');
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_column('public','tenant_product_datasource','chart bytea');
END $$;

-- Tenant user table
CREATE TABLE IF NOT EXISTS public.tenant_user (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    user_id uuid NOT NULL,
    CONSTRAINT tenant_user_pkey PRIMARY KEY (id),
    CONSTRAINT tenant_user_unique UNIQUE (tenant_id, user_id)
);

-- User role table
CREATE TABLE IF NOT EXISTS public.user_role (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    role_id uuid NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    CONSTRAINT user_role_pkey PRIMARY KEY (id),
    CONSTRAINT user_role_unique UNIQUE (user_id, role_id)
);

-- API definitions table
CREATE TABLE IF NOT EXISTS public.api_definitions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    "name" varchar(100) NOT NULL,
    display_name varchar(200) NULL,
    description text NULL,
    api_type varchar(20) NOT NULL,
    "version" varchar(20) NOT NULL,
    status varchar(20) NOT NULL,
    visibility varchar(20) NOT NULL,
    base_path varchar(200) NULL,
    group_id uuid NULL,
    specification jsonb NULL,
    is_core bool DEFAULT false NULL,
    cloned_from_id uuid NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NULL,
    updated_at timestamptz DEFAULT now() NULL,
    deprecated_at timestamptz NULL,
    sunset_at timestamptz NULL,
    successor_api_id uuid NULL,
    CONSTRAINT api_definitions_api_type_check CHECK (((api_type)::text = ANY (ARRAY[('REST'::character varying)::text, ('GRAPHQL'::character varying)::text, ('OPENAPI'::character varying)::text]))),
    CONSTRAINT api_definitions_pkey PRIMARY KEY (id),
    CONSTRAINT api_definitions_status_check CHECK (((status)::text = ANY (ARRAY[('DRAFT'::character varying)::text, ('ACTIVE'::character varying)::text, ('DEPRECATED'::character varying)::text, ('RETIRED'::character varying)::text]))),
    CONSTRAINT api_definitions_tenant_id_name_version_key UNIQUE (tenant_id, name, version),
    CONSTRAINT api_definitions_visibility_check CHECK (((visibility)::text = ANY (ARRAY[('PUBLIC'::character varying)::text, ('PRIVATE'::character varying)::text, ('TENANT_SPECIFIC'::character varying)::text])))
);

-- API documentation table
CREATE TABLE IF NOT EXISTS public.api_documentation (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    api_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    content_type varchar(50) DEFAULT 'MARKDOWN'::character varying NULL,
    "content" text NULL,
    "section" varchar(100) DEFAULT 'OVERVIEW'::character varying NULL,
    order_index int4 DEFAULT 0 NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NULL,
    updated_at timestamptz DEFAULT now() NULL,
    CONSTRAINT api_documentation_content_type_check CHECK (((content_type)::text = ANY (ARRAY[('MARKDOWN'::character varying)::text, ('HTML'::character varying)::text, ('ASCIIDOC'::character varying)::text]))),
    CONSTRAINT api_documentation_pkey PRIMARY KEY (id)
);

-- API endpoints table
CREATE TABLE IF NOT EXISTS public.api_endpoints (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    api_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    "path" varchar(200) NOT NULL,
    http_method varchar(10) NULL,
    operation_id varchar(100) NULL,
    summary varchar(200) NULL,
    description text NULL,
    request_schema jsonb NULL,
    response_schema jsonb NULL,
    deprecated bool DEFAULT false NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NULL,
    updated_at timestamptz DEFAULT now() NULL,
    CONSTRAINT api_endpoints_api_id_path_http_method_key UNIQUE (api_id, path, http_method),
    CONSTRAINT api_endpoints_http_method_check CHECK (((http_method)::text = ANY (ARRAY[('GET'::character varying)::text, ('POST'::character varying)::text, ('PUT'::character varying)::text, ('DELETE'::character varying)::text, ('PATCH'::character varying)::text, ('OPTIONS'::character varying)::text, ('HEAD'::character varying)::text]))),
    CONSTRAINT api_endpoints_pkey PRIMARY KEY (id)
);

-- API hooks table
CREATE TABLE IF NOT EXISTS public.api_hooks (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    "name" varchar(100) NOT NULL,
    display_name varchar(200) NULL,
    description text NULL,
    event_type varchar(50) NOT NULL,
    api_id uuid NULL,
    endpoint_id uuid NULL,
    script text NULL,
    execution_order int4 DEFAULT 100 NULL,
    is_active bool DEFAULT true NULL,
    execution_location varchar(20) DEFAULT 'SERVER'::character varying NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NULL,
    updated_at timestamptz DEFAULT now() NULL,
    CONSTRAINT api_hooks_event_type_check CHECK (((event_type)::text = ANY ((ARRAY['pre-request'::character varying, 'post-request'::character varying, 'user.created'::character varying, 'user.updated'::character varying, 'user.deleted'::character varying])::text[]))),
    CONSTRAINT api_hooks_execution_location_check CHECK (((execution_location)::text = 'gateway'::text)),
    CONSTRAINT api_hooks_pkey PRIMARY KEY (id)
);

-- API metrics table
CREATE TABLE IF NOT EXISTS public.api_metrics (
    id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    "timestamp" timestamptz DEFAULT now() NOT NULL,
    api_id uuid NULL,
    endpoint_id uuid NULL,
    response_time int4 NULL,
    status_code int4 NULL,
    request_size int4 NULL,
    response_size int4 NULL,
    client_ip varchar(45) NULL,
    user_id uuid NULL,
    error_type varchar(100) NULL,
    error_message text NULL,
    CONSTRAINT api_metrics_pkey PRIMARY KEY (id, "timestamp")
) PARTITION BY RANGE ("timestamp");

-- API metrics partitions
CREATE TABLE IF NOT EXISTS public.api_metrics_current_month PARTITION OF public.api_metrics
    FOR VALUES FROM ('2025-07-01 00:00:00-04') TO ('2025-08-01 00:00:00-04');

CREATE TABLE IF NOT EXISTS public.api_metrics_next_month PARTITION OF public.api_metrics
    FOR VALUES FROM ('2025-08-01 00:00:00-04') TO ('2025-09-01 00:00:00-04');

-- API security configs table
CREATE TABLE IF NOT EXISTS public.api_security_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    api_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    auth_methods jsonb DEFAULT '["API_KEY"]'::jsonb NULL,
    rate_limit_requests int4 DEFAULT 60 NULL,
    rate_limit_period varchar(10) DEFAULT 'MINUTE'::character varying NULL,
    ip_whitelist _text NULL,
    require_approval bool DEFAULT false NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NULL,
    updated_at timestamptz DEFAULT now() NULL,
    CONSTRAINT api_security_configs_api_id_tenant_id_key UNIQUE (api_id, tenant_id),
    CONSTRAINT api_security_configs_pkey PRIMARY KEY (id),
    CONSTRAINT api_security_configs_rate_limit_period_check CHECK (((rate_limit_period)::text = ANY (ARRAY[('SECOND'::character varying)::text, ('MINUTE'::character varying)::text, ('HOUR'::character varying)::text, ('DAY'::character varying)::text])))
);

-- Business rules table
CREATE TABLE IF NOT EXISTS public.business_rules (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    description text NULL,
    table_id uuid NOT NULL,
    event_type varchar(50) NOT NULL,
    script text NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    execution_order int4 DEFAULT 100 NOT NULL,
    execution_location varchar(50) DEFAULT 'server'::character varying NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    "version" int4 DEFAULT 1 NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT business_rules_event_type_check CHECK (((event_type)::text = ANY (ARRAY[('BeforeInsert'::character varying)::text, ('AfterInsert'::character varying)::text, ('BeforeUpdate'::character varying)::text, ('AfterUpdate'::character varying)::text, ('BeforeDelete'::character varying)::text, ('AfterDelete'::character varying)::text, ('RowInit'::character varying)::text, ('RowSelect'::character varying)::text, ('RowEdit'::character varying)::text]))),
    CONSTRAINT business_rules_execution_location_check CHECK (((execution_location)::text = ANY (ARRAY[('server'::character varying)::text, ('client'::character varying)::text, ('both'::character varying)::text]))),
    CONSTRAINT business_rules_pkey PRIMARY KEY (id),
    CONSTRAINT business_rules_tenant_name_key UNIQUE (tenant_id, name)
);

-- Integration credentials table
CREATE TABLE IF NOT EXISTS public.integration_credentials (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    integration_id uuid NOT NULL,
    credential_type varchar(50) NOT NULL,
    credential_name varchar(255) NOT NULL,
    credentials jsonb NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT integration_credentials_integration_name_key UNIQUE (integration_id, credential_name),
    CONSTRAINT integration_credentials_pkey PRIMARY KEY (id),
    CONSTRAINT integration_credentials_type_check CHECK (((credential_type)::text = ANY (ARRAY[('api_key'::character varying)::text, ('oauth2'::character varying)::text, ('basic_auth'::character varying)::text, ('certificate'::character varying)::text, ('aws'::character varying)::text, ('azure'::character varying)::text, ('gcp'::character varying)::text, ('kafka'::character varying)::text])))
);

-- Integration metrics table
CREATE TABLE IF NOT EXISTS public.integration_metrics (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    integration_id uuid NOT NULL,
    "timestamp" timestamptz DEFAULT now() NOT NULL,
    requests_count int4 DEFAULT 0 NOT NULL,
    success_count int4 DEFAULT 0 NOT NULL,
    error_count int4 DEFAULT 0 NOT NULL,
    total_duration int8 DEFAULT 0 NOT NULL,
    avg_duration float8 DEFAULT 0 NOT NULL,
    min_duration int8 DEFAULT 0 NOT NULL,
    max_duration int8 DEFAULT 0 NOT NULL,
    CONSTRAINT integration_metrics_pkey PRIMARY KEY (id)
);

-- Integration versions table
CREATE TABLE IF NOT EXISTS public.integration_versions (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    integration_id uuid NOT NULL,
    "version" int4 NOT NULL,
    config jsonb NOT NULL,
    "comment" text NULL,
    created_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT integration_versions_integration_id_version_key UNIQUE (integration_id, version),
    CONSTRAINT integration_versions_pkey PRIMARY KEY (id)
);

-- Message delivery logs table
CREATE TABLE IF NOT EXISTS public.message_delivery_logs (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    message_id uuid NOT NULL,
    target_id uuid NOT NULL,
    status varchar(50) NOT NULL,
    status_code int4 NULL,
    error_message text NULL,
    attempt_count int4 DEFAULT 1 NOT NULL,
    request_body text NULL,
    response_body text NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT message_delivery_logs_pkey PRIMARY KEY (id)
);

-- Metadata columns table
-- Metadata tables shim (Phase-A): create a minimal metadata_tables table if it doesn't exist
-- This is intentionally minimal and non-destructive so subsequent constraints/views can be created.
CREATE TABLE IF NOT EXISTS public.metadata_tables (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NULL,
    name varchar(255) NOT NULL,
    display_name varchar(255) DEFAULT '' NOT NULL,
    description text NULL,
    datasource_id uuid NULL,
    schema_name varchar(255) DEFAULT 'public',
    is_view bool DEFAULT false,
    is_system bool DEFAULT false,
    is_active bool DEFAULT true,
    created_by uuid NULL,
    updated_by uuid NULL,
    created_at timestamptz DEFAULT now(),
    updated_at timestamptz DEFAULT now(),
    CONSTRAINT metadata_tables_pkey PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_metadata_tables_datasource ON public.metadata_tables USING btree (datasource_id);
CREATE INDEX IF NOT EXISTS idx_metadata_tables_name ON public.metadata_tables USING btree (tenant_id, name);

CREATE TABLE IF NOT EXISTS public.metadata_columns (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    table_id uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    display_name varchar(255) NOT NULL,
    description text NULL,
    data_type varchar(50) NOT NULL,
    is_required bool DEFAULT false NOT NULL,
    is_primary_key bool DEFAULT false NOT NULL,
    is_foreign_key bool DEFAULT false NOT NULL,
    reference_table_id uuid NULL,
    reference_column_id uuid NULL,
    default_value text NULL,
    max_length int4 NULL,
    "precision" int4 NULL,
    "scale" int4 NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT metadata_columns_pkey PRIMARY KEY (id),
    CONSTRAINT metadata_columns_table_name_key UNIQUE (table_id, name)
);

-- Metadata fields table
CREATE TABLE IF NOT EXISTS public.metadata_fields (
    id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    table_id uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    display_name varchar(255) NOT NULL,
    description text NULL,
    data_type varchar(50) NOT NULL,
    is_required bool DEFAULT false NULL,
    is_key bool DEFAULT false NULL,
    default_value text NULL,
    min_value text NULL,
    max_value text NULL,
    regex_pattern text NULL,
    list_of_values jsonb NULL,
    ui_properties jsonb NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL,
    CONSTRAINT metadata_fields_pkey PRIMARY KEY (id),
    CONSTRAINT metadata_fields_table_id_name_key UNIQUE (table_id, name)
);

-- Metadata relationships table
CREATE TABLE IF NOT EXISTS public.metadata_relationships (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_id uuid NOT NULL,
    "name" varchar(255) NOT NULL,
    description text NULL,
    source_table uuid NOT NULL,
    source_column uuid NOT NULL,
    target_table uuid NOT NULL,
    target_column uuid NOT NULL,
    relation_type varchar(50) NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT metadata_relationships_pkey PRIMARY KEY (id),
    CONSTRAINT metadata_relationships_tenant_id_name_key UNIQUE (tenant_id, name)
);

-- Role integration permissions table
CREATE TABLE IF NOT EXISTS public.role_integration_permissions (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    role_id uuid NOT NULL,
    integration_id uuid NOT NULL,
    can_execute bool DEFAULT false NOT NULL,
    can_update bool DEFAULT false NOT NULL,
    can_delete bool DEFAULT false NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT role_integration_permissions_pkey PRIMARY KEY (id),
    CONSTRAINT role_integration_permissions_role_integration_key UNIQUE (role_id, integration_id)
);

-- Role permissions table
CREATE TABLE IF NOT EXISTS public.role_permissions (
    role_id uuid NOT NULL,
    permission_id uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT role_permissions_pkey PRIMARY KEY (role_id, permission_id)
);

-- Tenant chart table
CREATE TABLE IF NOT EXISTS public.tenant_chart (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    tenant_datasource_id uuid NOT NULL,
    chart_name varchar NOT NULL,
    is_active bool DEFAULT true NOT NULL,
    config jsonb NULL,
    chart bytea NULL,
    cloned_from uuid NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT tenant_chart_pkey PRIMARY KEY (id),
    CONSTRAINT tenant_chart_unique UNIQUE (tenant_datasource_id, chart_name)
);

-- API access rules table
CREATE TABLE IF NOT EXISTS public.api_access_rules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    "name" varchar(100) NOT NULL,
    description text NULL,
    api_id uuid NULL,
    endpoint_id uuid NULL,
    role_id uuid NULL,
    "permission" varchar(20) NOT NULL,
    resources _text NULL,
    is_active bool DEFAULT true NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NULL,
    updated_at timestamptz DEFAULT now() NULL,
    CONSTRAINT api_access_rules_permission_check CHECK (((permission)::text = ANY (ARRAY[('READ'::character varying)::text, ('WRITE'::character varying)::text, ('ADMIN'::character varying)::text, ('DENY'::character varying)::text]))),
    CONSTRAINT api_access_rules_pkey PRIMARY KEY (id)
);

-- API audit logs table
CREATE TABLE IF NOT EXISTS public.api_audit_logs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    "timestamp" timestamptz DEFAULT now() NULL,
    event_type varchar(50) NOT NULL,
    user_id uuid NULL,
    api_id uuid NULL,
    endpoint_id uuid NULL,
    resource varchar(255) NULL,
    "action" varchar(50) NULL,
    status varchar(20) NULL,
    details jsonb NULL,
    ip_address varchar(45) NULL,
    user_agent text NULL,
    CONSTRAINT api_audit_logs_pkey PRIMARY KEY (id)
);

-- Business rule versions table
CREATE TABLE IF NOT EXISTS public.business_rule_versions (
    id uuid DEFAULT uuid_generate_v4() NOT NULL,
    business_rule_id uuid NOT NULL,
    script text NOT NULL,
    "version" int4 NOT NULL,
    "comment" text NULL,
    created_by uuid NOT NULL,
    created_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT business_rule_versions_pkey PRIMARY KEY (id),
    CONSTRAINT business_rule_versions_rule_version_key UNIQUE (business_rule_id, version)
);

-- Catalog edge type table (extend existing)
ALTER TABLE public.catalog_edge_types ADD COLUMN IF NOT EXISTS config jsonb;
ALTER TABLE public.catalog_edge_types ADD COLUMN IF NOT EXISTS is_active bool DEFAULT true;
ALTER TABLE public.catalog_edge_types ADD COLUMN IF NOT EXISTS created_at timestamptz DEFAULT now();
ALTER TABLE public.catalog_edge_types ADD COLUMN IF NOT EXISTS updated_at timestamptz DEFAULT now();
ALTER TABLE public.catalog_edge_types ADD COLUMN IF NOT EXISTS tenant_id uuid;
ALTER TABLE public.catalog_edge_types ADD COLUMN IF NOT EXISTS core_id uuid;

-- Catalog node table (extend existing)
ALTER TABLE public.catalog_node ADD COLUMN IF NOT EXISTS is_alpha bool DEFAULT false;
ALTER TABLE public.catalog_node ADD COLUMN IF NOT EXISTS core_id uuid;
DO $$ BEGIN
    PERFORM public.safe_add_column('public','catalog_node','tenant_datasource_id_uuid uuid');
END $$;

-- Catalog node type table (extend existing)
ALTER TABLE public.catalog_node_type ADD COLUMN IF NOT EXISTS tenant_id uuid;
ALTER TABLE public.catalog_node_type ADD COLUMN IF NOT EXISTS core_id uuid;
DO $$ BEGIN
    PERFORM public.safe_add_column('public','catalog_node_type','tenant_datasource_id_uuid uuid');
END $$;

-- Catalog edge table (extend existing)
ALTER TABLE public.catalog_edge ADD COLUMN IF NOT EXISTS tenant_id uuid;
ALTER TABLE public.catalog_edge ADD COLUMN IF NOT EXISTS core_id uuid;

-- Metadata events table
CREATE TABLE IF NOT EXISTS public.metadata_events (
    id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    table_id uuid NOT NULL,
    field_id uuid NULL,
    event_type varchar(50) NOT NULL,
    "name" varchar(255) NOT NULL,
    description text NULL,
    script text NOT NULL,
    is_active bool DEFAULT true NULL,
    execution_order int4 DEFAULT 0 NULL,
    execution_location varchar(50) DEFAULT 'server'::character varying NULL,
    created_by uuid NOT NULL,
    updated_by uuid NOT NULL,
    "version" int4 DEFAULT 1 NOT NULL,
    created_at timestamp NOT NULL,
    updated_at timestamp NOT NULL,
    CONSTRAINT metadata_events_pkey PRIMARY KEY (id),
    CONSTRAINT metadata_events_tenant_id_table_id_field_id_event_type_name_key UNIQUE (tenant_id, table_id, field_id, event_type, name)
);

-- Metadata event logs table
CREATE TABLE IF NOT EXISTS public.metadata_event_logs (
    id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    event_id uuid NOT NULL,
    request_id uuid NOT NULL,
    user_id uuid NULL,
    execution_start timestamp NOT NULL,
    execution_end timestamp NOT NULL,
    duration_ms int4 NOT NULL,
    status varchar(50) NOT NULL,
    error_message text NULL,
    input_data jsonb NULL,
    output_data jsonb NULL,
    created_at timestamp NOT NULL,
    CONSTRAINT metadata_event_logs_pkey PRIMARY KEY (id)
);

-- Metadata event versions table
CREATE TABLE IF NOT EXISTS public.metadata_event_versions (
    id uuid NOT NULL,
    event_id uuid NOT NULL,
    script text NOT NULL,
    "version" int4 NOT NULL,
    "comment" text NULL,
    created_by uuid NOT NULL,
    created_at timestamp NOT NULL,
    CONSTRAINT metadata_event_versions_pkey PRIMARY KEY (id),
    CONSTRAINT metadata_event_versions_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.metadata_events(id) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON public.audit_logs USING btree (resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_timestamp ON public.audit_logs USING btree (tenant_id, "timestamp");
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON public.audit_logs USING btree (user_id);

CREATE INDEX IF NOT EXISTS idx_fabric_defn_is_current ON public.fabric_defn USING btree (is_current);
CREATE INDEX IF NOT EXISTS idx_fabric_defn_model_key ON public.fabric_defn USING btree (model_key);
CREATE INDEX IF NOT EXISTS idx_fabric_defn_tenant_datasource_id ON public.fabric_defn USING btree (tenant_datasource_id);
CREATE INDEX IF NOT EXISTS idx_fabric_defn_tenant_id ON public.fabric_defn USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_integration_logs_event ON public.integration_logs USING btree (event_id);
CREATE INDEX IF NOT EXISTS idx_integration_logs_execution_start ON public.integration_logs USING btree (execution_start);
CREATE INDEX IF NOT EXISTS idx_integration_logs_integration ON public.integration_logs USING btree (integration_id);
CREATE INDEX IF NOT EXISTS idx_integration_logs_status ON public.integration_logs USING btree (status);
CREATE INDEX IF NOT EXISTS idx_integration_logs_tenant ON public.integration_logs USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_message_templates_tenant_id ON public.message_templates USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_performance_metrics_name_collected ON public.performance_metrics USING btree (metric_name, collected_at DESC);
CREATE INDEX IF NOT EXISTS idx_performance_metrics_tenant_collected ON public.performance_metrics USING btree (tenant_id, collected_at DESC);

CREATE INDEX IF NOT EXISTS idx_policies_active_priority ON public.policies USING btree (active, priority DESC) WHERE (active = true);

CREATE INDEX IF NOT EXISTS idx_policy_evaluation_run_id ON public.policy_evaluation USING btree (run_id);

CREATE INDEX IF NOT EXISTS idx_policy_version_history_policy_id_version ON public.policy_version_history USING btree (policy_id, version);

CREATE INDEX IF NOT EXISTS idx_active_requests_status ON public.active_requests USING btree (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_active_requests_tenant ON public.active_requests USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_api_groups_tenant_id ON public.api_groups USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_api_keys_expires ON public.api_keys USING btree (tenant_id, expires_at, is_active);
CREATE INDEX IF NOT EXISTS idx_api_keys_permissions_gin ON public.api_keys USING gin (permissions);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_active ON public.api_keys USING btree (tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON public.api_keys USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_api_workflow_approvals_tenant_id_status ON public.api_workflow_approvals USING btree (tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_apis_group_id ON public.apis USING btree (group_id);
CREATE INDEX IF NOT EXISTS idx_apis_name ON public.apis USING btree (name);
CREATE INDEX IF NOT EXISTS idx_apis_tenant_id ON public.apis USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_broker_apis_api_id ON public.broker_apis USING btree (api_id);
CREATE INDEX IF NOT EXISTS idx_broker_apis_tenant_id ON public.broker_apis USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_broker_events_tenant_id ON public.broker_events USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_event_subscriptions_event_id ON public.event_subscriptions USING btree (event_id);
CREATE INDEX IF NOT EXISTS idx_event_subscriptions_tenant_id ON public.event_subscriptions USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_exposed_apis_tenant_id ON public.exposed_apis USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_integration_audit_logs_request_gin ON public.integration_audit_logs USING gin (request_data);
CREATE INDEX IF NOT EXISTS idx_integration_audit_logs_resource ON public.integration_audit_logs USING btree (tenant_id, resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_integration_audit_logs_response_gin ON public.integration_audit_logs USING gin (response_data);
CREATE INDEX IF NOT EXISTS idx_integration_audit_logs_status ON public.integration_audit_logs USING btree (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_integration_audit_logs_tenant ON public.integration_audit_logs USING btree (tenant_id);
CREATE INDEX IF NOT EXISTS idx_integration_audit_logs_timestamp ON public.integration_audit_logs USING btree (tenant_id, "timestamp");
CREATE INDEX IF NOT EXISTS idx_integration_audit_logs_type_action ON public.integration_audit_logs USING btree (tenant_id, type, action);
CREATE INDEX IF NOT EXISTS idx_integration_audit_logs_user ON public.integration_audit_logs USING btree (tenant_id, user_id);

CREATE INDEX IF NOT EXISTS idx_integration_configs_tenant ON public.integration_configs USING btree (tenant_id);
CREATE INDEX IF NOT EXISTS idx_integration_configs_tenant_event ON public.integration_configs USING btree (tenant_id, event_id);

CREATE INDEX IF NOT EXISTS idx_integrations_config_gin ON public.integrations USING gin (config);
CREATE INDEX IF NOT EXISTS idx_integrations_description_trgm ON public.integrations USING gin (description gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_integrations_methods_gin ON public.integrations USING gin (methods);
CREATE INDEX IF NOT EXISTS idx_integrations_name ON public.integrations USING btree (tenant_id, name);
CREATE INDEX IF NOT EXISTS idx_integrations_required_permissions_gin ON public.integrations USING gin (required_permissions);
CREATE INDEX IF NOT EXISTS idx_integrations_status ON public.integrations USING btree (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_integrations_tenant_type ON public.integrations USING btree (tenant_id, type);

CREATE INDEX IF NOT EXISTS idx_ip_whitelist_tenant ON public.ip_whitelist USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_message_targets_tenant_id ON public.message_targets USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_metadata_tables_datasource ON public.metadata_tables USING btree (datasource_id);
CREATE INDEX IF NOT EXISTS idx_metadata_tables_description_trgm ON public.metadata_tables USING gin (description gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_metadata_tables_name ON public.metadata_tables USING btree (tenant_id, name);
CREATE INDEX IF NOT EXISTS idx_metadata_tables_tenant_active ON public.metadata_tables USING btree (tenant_id, is_active);

CREATE INDEX IF NOT EXISTS idx_roles_tenant_active ON public.roles USING btree (tenant_id, updated_at DESC) WHERE (is_active = true);

CREATE INDEX IF NOT EXISTS idx_business_rules_description_trgm ON public.business_rules USING gin (description gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_business_rules_name ON public.business_rules USING btree (tenant_id, name);
CREATE INDEX IF NOT EXISTS idx_business_rules_table_event ON public.business_rules USING btree (table_id, event_type, is_active);
CREATE INDEX IF NOT EXISTS idx_business_rules_tenant_active ON public.business_rules USING btree (tenant_id, is_active);

CREATE INDEX IF NOT EXISTS idx_integration_credentials_credentials_gin ON public.integration_credentials USING gin (credentials);
CREATE INDEX IF NOT EXISTS idx_integration_credentials_integration_id ON public.integration_credentials USING btree (integration_id);

CREATE INDEX IF NOT EXISTS idx_integration_metrics_tenant_integration ON public.integration_metrics USING btree (tenant_id, integration_id);
CREATE INDEX IF NOT EXISTS idx_integration_metrics_timestamp ON public.integration_metrics USING btree (tenant_id, "timestamp");

CREATE INDEX IF NOT EXISTS idx_integration_versions_integration_id ON public.integration_versions USING btree (integration_id);

CREATE INDEX IF NOT EXISTS idx_message_delivery_logs_message_id ON public.message_delivery_logs USING btree (message_id);
CREATE INDEX IF NOT EXISTS idx_message_delivery_logs_target_id ON public.message_delivery_logs USING btree (target_id);

CREATE INDEX IF NOT EXISTS idx_metadata_columns_reference_table ON public.metadata_columns USING btree (reference_table_id);
CREATE INDEX IF NOT EXISTS idx_metadata_columns_table_id ON public.metadata_columns USING btree (table_id);

CREATE INDEX IF NOT EXISTS idx_role_integration_permissions_integration ON public.role_integration_permissions USING btree (integration_id);
CREATE INDEX IF NOT EXISTS idx_role_integration_permissions_role ON public.role_integration_permissions USING btree (role_id);

CREATE INDEX IF NOT EXISTS idx_tenant_chart_alpha_datasource_id ON public.tenant_chart USING btree (tenant_datasource_id, chart_name);
CREATE INDEX IF NOT EXISTS idx_tenant_chart_cloned_from ON public.tenant_chart USING btree (cloned_from);

CREATE INDEX IF NOT EXISTS idx_api_access_rules_api_id ON public.api_access_rules USING btree (api_id);
CREATE INDEX IF NOT EXISTS idx_api_access_rules_role_id ON public.api_access_rules USING btree (role_id);

CREATE INDEX IF NOT EXISTS idx_api_audit_logs_tenant_id_timestamp ON public.api_audit_logs USING btree (tenant_id, "timestamp");

CREATE INDEX IF NOT EXISTS idx_business_rule_versions_rule_id ON public.business_rule_versions USING btree (business_rule_id);

CREATE INDEX IF NOT EXISTS idx_api_definitions_group_id ON public.api_definitions USING btree (group_id);
CREATE INDEX IF NOT EXISTS idx_api_definitions_tenant_id ON public.api_definitions USING btree (tenant_id);

CREATE INDEX IF NOT EXISTS idx_api_documentation_api_id ON public.api_documentation USING btree (api_id);

CREATE INDEX IF NOT EXISTS idx_api_endpoints_api_id ON public.api_endpoints USING btree (api_id);

CREATE INDEX IF NOT EXISTS idx_api_hooks_api_id ON public.api_hooks USING btree (api_id);
CREATE INDEX IF NOT EXISTS idx_api_hooks_endpoint_id ON public.api_hooks USING btree (endpoint_id);

CREATE INDEX IF NOT EXISTS idx_api_metrics_api_id_timestamp ON ONLY public.api_metrics USING btree (api_id, "timestamp");
CREATE INDEX IF NOT EXISTS idx_api_metrics_tenant_id_timestamp ON ONLY public.api_metrics USING btree (tenant_id, "timestamp");

CREATE INDEX IF NOT EXISTS idx_api_security_configs_api_id ON public.api_security_configs USING btree (api_id);

CREATE INDEX IF NOT EXISTS idx_policy_violation_evaluation_id ON public.policy_violation USING btree (evaluation_id);

-- Add foreign key constraints (only if they don't exist)
DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('active_requests','tenant_id','active_requests_tenant_fk','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('active_requests_user_fk', 'ALTER TABLE public.active_requests ADD CONSTRAINT active_requests_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_groups_created_by_fkey', 'ALTER TABLE public.api_groups ADD CONSTRAINT api_groups_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_groups_parent_group_id_fkey', 'ALTER TABLE public.api_groups ADD CONSTRAINT api_groups_parent_group_id_fkey FOREIGN KEY (parent_group_id) REFERENCES public.api_groups(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('api_groups','tenant_id','api_groups_tenant_id_fkey');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_groups_updated_by_fkey', 'ALTER TABLE public.api_groups ADD CONSTRAINT api_groups_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_keys_created_by_fk', 'ALTER TABLE public.api_keys ADD CONSTRAINT api_keys_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('api_keys','tenant_id','api_keys_tenant_fk','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_workflow_approvals_approver_id_fkey', 'ALTER TABLE public.api_workflow_approvals ADD CONSTRAINT api_workflow_approvals_approver_id_fkey FOREIGN KEY (approver_id) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_workflow_approvals_requested_by_fkey', 'ALTER TABLE public.api_workflow_approvals ADD CONSTRAINT api_workflow_approvals_requested_by_fkey FOREIGN KEY (requested_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('api_workflow_approvals','tenant_id','api_workflow_approvals_tenant_id_fkey');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('apis_cloned_from_id_fkey', 'ALTER TABLE public.apis ADD CONSTRAINT apis_cloned_from_id_fkey FOREIGN KEY (cloned_from_id) REFERENCES public.apis(id) ON DELETE SET NULL');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('apis_group_id_fkey', 'ALTER TABLE public.apis ADD CONSTRAINT apis_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.api_groups(id) ON DELETE SET NULL');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('apis','tenant_id','apis_tenant_id_fkey','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('broker_apis','tenant_id','broker_apis_tenant_id_fkey','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('broker_events','tenant_id','broker_events_tenant_id_fkey','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('customers','tenant_id','customers_tenant_id_fkey','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('drift_log_entries_report_id_fkey', 'ALTER TABLE public.drift_log_entries ADD CONSTRAINT drift_log_entries_report_id_fkey FOREIGN KEY (report_id) REFERENCES public.drift_reports(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('event_subscriptions_event_id_fkey', 'ALTER TABLE public.event_subscriptions ADD CONSTRAINT event_subscriptions_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.broker_events(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('event_subscriptions','tenant_id','event_subscriptions_tenant_id_fkey','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('exposed_apis_api_id_fkey', 'ALTER TABLE public.exposed_apis ADD CONSTRAINT exposed_apis_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.apis(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('exposed_apis','tenant_id','exposed_apis_tenant_id_fkey','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('integration_audit_logs','tenant_id','integration_audit_logs_tenant_fk','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('integration_audit_logs_user_fk', 'ALTER TABLE public.integration_audit_logs ADD CONSTRAINT integration_audit_logs_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('integration_configs_created_by_fk', 'ALTER TABLE public.integration_configs ADD CONSTRAINT integration_configs_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('integration_configs_updated_by_fk', 'ALTER TABLE public.integration_configs ADD CONSTRAINT integration_configs_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('integrations_created_by_fk', 'ALTER TABLE public.integrations ADD CONSTRAINT integrations_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('integrations','tenant_id','integrations_tenant_fk','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('integrations_updated_by_fk', 'ALTER TABLE public.integrations ADD CONSTRAINT integrations_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('ip_whitelist_created_by_fk', 'ALTER TABLE public.ip_whitelist ADD CONSTRAINT ip_whitelist_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('ip_whitelist','tenant_id','ip_whitelist_tenant_fk','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('fk_template', 'ALTER TABLE public.message_targets ADD CONSTRAINT fk_template FOREIGN KEY (template_id) REFERENCES public.message_templates(id) ON DELETE SET NULL');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('metadata_tables_created_by_fk', 'ALTER TABLE public.metadata_tables ADD CONSTRAINT metadata_tables_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('metadata_tables','tenant_id','metadata_tables_tenant_fk','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('metadata_tables_updated_by_fk', 'ALTER TABLE public.metadata_tables ADD CONSTRAINT metadata_tables_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('orders','tenant_id','orders_tenant_id_fkey','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('policy_violation_evaluation_id_fkey', 'ALTER TABLE public.policy_violation ADD CONSTRAINT policy_violation_evaluation_id_fkey FOREIGN KEY (evaluation_id) REFERENCES public.policy_evaluation(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('roles','tenant_id','roles_tenant_fk','ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('tenant_connections','tenant_id','fk_tenant','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('tenant_instance','tenant_id','tenant_instance_tenant_fk','ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('tenant_product_product_fk', 'ALTER TABLE public.tenant_product ADD CONSTRAINT tenant_product_product_fk FOREIGN KEY (alpha_product_id) REFERENCES public.alpha_product(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('tenant_product_tenant_instance_fk', 'ALTER TABLE public.tenant_product ADD CONSTRAINT tenant_product_tenant_instance_fk FOREIGN KEY (tenant_instance_id) REFERENCES public.tenant_instance(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('tenant_product_datasource_alpha_datasource_fk', 'ALTER TABLE public.tenant_product_datasource ADD CONSTRAINT tenant_product_datasource_alpha_datasource_fk FOREIGN KEY (alpha_datasource_id) REFERENCES public.alpha_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('tenant_product_datasource_tenant_product_fk', 'ALTER TABLE public.tenant_product_datasource ADD CONSTRAINT tenant_product_datasource_tenant_product_fk FOREIGN KEY (tenant_product_id) REFERENCES public.tenant_product(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('tenant_user','tenant_id','tenant_user_tenant_fk','ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('tenant_user_users_fk', 'ALTER TABLE public.tenant_user ADD CONSTRAINT tenant_user_users_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('user_role_roles_fk', 'ALTER TABLE public.user_role ADD CONSTRAINT user_role_roles_fk FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('user_role_users_fk', 'ALTER TABLE public.user_role ADD CONSTRAINT user_role_users_fk FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_definitions_cloned_from_id_fkey', 'ALTER TABLE public.api_definitions ADD CONSTRAINT api_definitions_cloned_from_id_fkey FOREIGN KEY (cloned_from_id) REFERENCES public.api_definitions(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_definitions_created_by_fkey', 'ALTER TABLE public.api_definitions ADD CONSTRAINT api_definitions_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_definitions_group_id_fkey', 'ALTER TABLE public.api_definitions ADD CONSTRAINT api_definitions_group_id_fkey FOREIGN KEY (group_id) REFERENCES public.api_groups(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('api_definitions','tenant_id','api_definitions_tenant_id_fkey','');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_definitions_updated_by_fkey', 'ALTER TABLE public.api_definitions ADD CONSTRAINT api_definitions_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_documentation_api_id_fkey', 'ALTER TABLE public.api_documentation ADD CONSTRAINT api_documentation_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_documentation_created_by_fkey', 'ALTER TABLE public.api_documentation ADD CONSTRAINT api_documentation_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('api_documentation','tenant_id','api_documentation_tenant_id_fkey','');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_documentation_updated_by_fkey', 'ALTER TABLE public.api_documentation ADD CONSTRAINT api_documentation_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_endpoints_api_id_fkey', 'ALTER TABLE public.api_endpoints ADD CONSTRAINT api_endpoints_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_endpoints_created_by_fkey', 'ALTER TABLE public.api_endpoints ADD CONSTRAINT api_endpoints_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('api_endpoints','tenant_id','api_endpoints_tenant_id_fkey','');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_endpoints_updated_by_fkey', 'ALTER TABLE public.api_endpoints ADD CONSTRAINT api_endpoints_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_hooks_api_id_fkey', 'ALTER TABLE public.api_hooks ADD CONSTRAINT api_hooks_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_hooks_created_by_fkey', 'ALTER TABLE public.api_hooks ADD CONSTRAINT api_hooks_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_hooks_endpoint_id_fkey', 'ALTER TABLE public.api_hooks ADD CONSTRAINT api_hooks_endpoint_id_fkey FOREIGN KEY (endpoint_id) REFERENCES public.api_endpoints(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('api_hooks','tenant_id','api_hooks_tenant_id_fkey','');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_hooks_updated_by_fkey', 'ALTER TABLE public.api_hooks ADD CONSTRAINT api_hooks_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_metrics_api_id_fkey', 'ALTER TABLE public.api_metrics ADD CONSTRAINT api_metrics_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_metrics_endpoint_id_fkey', 'ALTER TABLE public.api_metrics ADD CONSTRAINT api_metrics_endpoint_id_fkey FOREIGN KEY (endpoint_id) REFERENCES public.api_endpoints(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('api_metrics','tenant_id','api_metrics_tenant_id_fkey','');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_metrics_user_id_fkey', 'ALTER TABLE public.api_metrics ADD CONSTRAINT api_metrics_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_security_configs_api_id_fkey', 'ALTER TABLE public.api_security_configs ADD CONSTRAINT api_security_configs_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_security_configs_created_by_fkey', 'ALTER TABLE public.api_security_configs ADD CONSTRAINT api_security_configs_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('api_security_configs','tenant_id','api_security_configs_tenant_id_fkey','');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_security_configs_updated_by_fkey', 'ALTER TABLE public.api_security_configs ADD CONSTRAINT api_security_configs_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('business_rules_created_by_fk', 'ALTER TABLE public.business_rules ADD CONSTRAINT business_rules_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_constraint('business_rules_table_fk', 'ALTER TABLE public.business_rules ADD CONSTRAINT business_rules_table_fk FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_constraint skipped for % due to: %','business_rules_table_fk', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('business_rules','tenant_id','business_rules_tenant_fk','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('business_rules_updated_by_fk', 'ALTER TABLE public.business_rules ADD CONSTRAINT business_rules_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('catalog_edge_types_catalog_node_type_fk', 'ALTER TABLE public.catalog_edge_types ADD CONSTRAINT catalog_edge_types_catalog_node_type_fk FOREIGN KEY (source_node_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('catalog_edge_types_catalog_node_type_fk_1', 'ALTER TABLE public.catalog_edge_types ADD CONSTRAINT catalog_edge_types_catalog_node_type_fk_1 FOREIGN KEY (target_node_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('catalog_edge_types','tenant_id','catalog_edge_types_tenants_fk','ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('catalog_node_catalog_node_type_fk', 'ALTER TABLE public.catalog_node ADD CONSTRAINT catalog_node_catalog_node_type_fk FOREIGN KEY (node_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('catalog_node_tenant_product_datasource_fk', 'ALTER TABLE public.catalog_node ADD CONSTRAINT catalog_node_tenant_product_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('catalog_node','tenant_id','catalog_node_tenants_fk','ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('catalog_node_type_catalog_node_type_fk', 'ALTER TABLE public.catalog_node_type ADD CONSTRAINT catalog_node_type_catalog_node_type_fk FOREIGN KEY (parent_type_id) REFERENCES public.catalog_node_type(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('catalog_node_type_tenant_product_datasource_fk', 'ALTER TABLE public.catalog_node_type ADD CONSTRAINT catalog_node_type_tenant_product_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('catalog_node_type','tenant_id','catalog_node_type_tenants_fk','ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('catalog_edge_catalog_edge_types_fk', 'ALTER TABLE public.catalog_edge ADD CONSTRAINT catalog_edge_catalog_edge_types_fk FOREIGN KEY (edge_type_id) REFERENCES public.catalog_edge_types(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('catalog_edge_catalog_sourcee_node_fk', 'ALTER TABLE public.catalog_edge ADD CONSTRAINT catalog_edge_catalog_sourcee_node_fk FOREIGN KEY (source_node_id) REFERENCES public.catalog_node(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('catalog_edge_catalog_target_node_fk', 'ALTER TABLE public.catalog_edge ADD CONSTRAINT catalog_edge_catalog_target_node_fk FOREIGN KEY (target_node_id) REFERENCES public.catalog_node(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('integration_credentials_created_by_fk', 'ALTER TABLE public.integration_credentials ADD CONSTRAINT integration_credentials_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('integration_credentials_integration_fk', 'ALTER TABLE public.integration_credentials ADD CONSTRAINT integration_credentials_integration_fk FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('integration_credentials_updated_by_fk', 'ALTER TABLE public.integration_credentials ADD CONSTRAINT integration_credentials_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('integration_metrics_integration_fk', 'ALTER TABLE public.integration_metrics ADD CONSTRAINT integration_metrics_integration_fk FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_tenant_fk('integration_metrics','tenant_id','integration_metrics_tenant_fk','ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('integration_versions_created_by_fk', 'ALTER TABLE public.integration_versions ADD CONSTRAINT integration_versions_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('integration_versions_integration_fk', 'ALTER TABLE public.integration_versions ADD CONSTRAINT integration_versions_integration_fk FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('fk_target', 'ALTER TABLE public.message_delivery_logs ADD CONSTRAINT fk_target FOREIGN KEY (target_id) REFERENCES public.message_targets(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('metadata_columns_created_by_fk', 'ALTER TABLE public.metadata_columns ADD CONSTRAINT metadata_columns_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('metadata_columns_reference_column_fk', 'ALTER TABLE public.metadata_columns ADD CONSTRAINT metadata_columns_reference_column_fk FOREIGN KEY (reference_column_id) REFERENCES public.metadata_columns(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_constraint('metadata_columns_reference_table_fk', 'ALTER TABLE public.metadata_columns ADD CONSTRAINT metadata_columns_reference_table_fk FOREIGN KEY (reference_table_id) REFERENCES public.metadata_tables(id)');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_constraint skipped for % due to: %','metadata_columns_reference_table_fk', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_constraint('metadata_columns_table_fk', 'ALTER TABLE public.metadata_columns ADD CONSTRAINT metadata_columns_table_fk FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_constraint skipped for % due to: %','metadata_columns_table_fk', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('metadata_columns_updated_by_fk', 'ALTER TABLE public.metadata_columns ADD CONSTRAINT metadata_columns_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_constraint('metadata_fields_table_id_fkey', 'ALTER TABLE public.metadata_fields ADD CONSTRAINT metadata_fields_table_id_fkey FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_constraint skipped for % due to: %','metadata_fields_table_id_fkey', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_tenant_fk('metadata_fields','tenant_id','metadata_fields_tenant_id_fkey','ON DELETE CASCADE');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_tenant_fk skipped for % due to: %','metadata_fields_tenant_id_fkey', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('metadata_relationships_source_column_fk', 'ALTER TABLE public.metadata_relationships ADD CONSTRAINT metadata_relationships_source_column_fk FOREIGN KEY (source_column) REFERENCES public.metadata_columns(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_constraint('metadata_relationships_source_table_fk', 'ALTER TABLE public.metadata_relationships ADD CONSTRAINT metadata_relationships_source_table_fk FOREIGN KEY (source_table) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_constraint skipped for % due to: %','metadata_relationships_source_table_fk', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('metadata_relationships_target_column_fk', 'ALTER TABLE public.metadata_relationships ADD CONSTRAINT metadata_relationships_target_column_fk FOREIGN KEY (target_column) REFERENCES public.metadata_columns(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_constraint('metadata_relationships_target_table_fk', 'ALTER TABLE public.metadata_relationships ADD CONSTRAINT metadata_relationships_target_table_fk FOREIGN KEY (target_table) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_constraint skipped for % due to: %','metadata_relationships_target_table_fk', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_tenant_fk('metadata_relationships','tenant_id','metadata_relationships_tenant_fk','ON DELETE CASCADE');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_tenant_fk skipped for % due to: %','metadata_relationships_tenant_fk', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('role_integration_permissions_integration_fk', 'ALTER TABLE public.role_integration_permissions ADD CONSTRAINT role_integration_permissions_integration_fk FOREIGN KEY (integration_id) REFERENCES public.integrations(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('role_integration_permissions_role_fk', 'ALTER TABLE public.role_integration_permissions ADD CONSTRAINT role_integration_permissions_role_fk FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('role_permissions_permission_fk', 'ALTER TABLE public.role_permissions ADD CONSTRAINT role_permissions_permission_fk FOREIGN KEY (permission_id) REFERENCES public.permissions(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('role_permissions_role_fk', 'ALTER TABLE public.role_permissions ADD CONSTRAINT role_permissions_role_fk FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('tenant_chart_tenant_product_datasource_fk', 'ALTER TABLE public.tenant_chart ADD CONSTRAINT tenant_chart_tenant_product_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_access_rules_api_id_fkey', 'ALTER TABLE public.api_access_rules ADD CONSTRAINT api_access_rules_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_access_rules_created_by_fkey', 'ALTER TABLE public.api_access_rules ADD CONSTRAINT api_access_rules_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_access_rules_endpoint_id_fkey', 'ALTER TABLE public.api_access_rules ADD CONSTRAINT api_access_rules_endpoint_id_fkey FOREIGN KEY (endpoint_id) REFERENCES public.api_endpoints(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_access_rules_role_id_fkey', 'ALTER TABLE public.api_access_rules ADD CONSTRAINT api_access_rules_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_tenant_fk('api_access_rules','tenant_id','api_access_rules_tenant_id_fkey');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_tenant_fk skipped for % due to: %','api_access_rules_tenant_id_fkey', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_access_rules_updated_by_fkey', 'ALTER TABLE public.api_access_rules ADD CONSTRAINT api_access_rules_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_audit_logs_api_id_fkey', 'ALTER TABLE public.api_audit_logs ADD CONSTRAINT api_audit_logs_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.api_definitions(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_audit_logs_endpoint_id_fkey', 'ALTER TABLE public.api_audit_logs ADD CONSTRAINT api_audit_logs_endpoint_id_fkey FOREIGN KEY (endpoint_id) REFERENCES public.api_endpoints(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_tenant_fk('api_audit_logs','tenant_id','api_audit_logs_tenant_id_fkey');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_tenant_fk skipped for % due to: %','api_audit_logs_tenant_id_fkey', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('api_audit_logs_user_id_fkey', 'ALTER TABLE public.api_audit_logs ADD CONSTRAINT api_audit_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('business_rule_versions_created_by_fk', 'ALTER TABLE public.business_rule_versions ADD CONSTRAINT business_rule_versions_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('business_rule_versions_rule_fk', 'ALTER TABLE public.business_rule_versions ADD CONSTRAINT business_rule_versions_rule_fk FOREIGN KEY (business_rule_id) REFERENCES public.business_rules(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('metadata_events_field_id_fkey', 'ALTER TABLE public.metadata_events ADD CONSTRAINT metadata_events_field_id_fkey FOREIGN KEY (field_id) REFERENCES public.metadata_fields(id) ON DELETE SET NULL');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_constraint('metadata_events_table_id_fkey', 'ALTER TABLE public.metadata_events ADD CONSTRAINT metadata_events_table_id_fkey FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_constraint skipped for % due to: %','metadata_events_table_id_fkey', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_tenant_fk('metadata_events','tenant_id','metadata_events_tenant_id_fkey','ON DELETE CASCADE');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_tenant_fk skipped for % due to: %','metadata_events_tenant_id_fkey', SQLERRM;
    END;
END $$;

DO $$ BEGIN
    PERFORM public.safe_add_constraint('metadata_event_logs_event_id_fkey', 'ALTER TABLE public.metadata_event_logs ADD CONSTRAINT metadata_event_logs_event_id_fkey FOREIGN KEY (event_id) REFERENCES public.metadata_events(id) ON DELETE CASCADE');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    BEGIN
        PERFORM public.safe_add_tenant_fk('metadata_event_logs','tenant_id','metadata_event_logs_tenant_id_fkey','ON DELETE CASCADE');
    EXCEPTION WHEN others THEN
        RAISE NOTICE 'safe_add_tenant_fk skipped for % due to: %','metadata_event_logs_tenant_id_fkey', SQLERRM;
    END;
END $$;

-- Create missing functions
CREATE OR REPLACE FUNCTION public._fabric_defn_index_put(p_defn_id uuid, p_tenant_id uuid, p_model_key text, p_version integer, p_kind text, p_name text, p_type text, p_relationship join_relationship, p_title text, p_description text, p_sql text)
 RETURNS void
 LANGUAGE sql
AS $function$
  insert into fabric_defn_index
    (defn_id, tenant_id, model_key, version, kind, name, type, relationship, title, description, sql)
  values
    (p_defn_id, p_tenant_id, p_model_key, p_version, p_kind, p_name, p_type, p_relationship, p_title, p_description, p_sql)
  on conflict (tenant_id, model_key, version, kind, name) do update
    set type = excluded.type,
        relationship = excluded.relationship,
        title = excluded.title,
        description = excluded.description,
        sql = excluded.sql;
$function$;

CREATE OR REPLACE FUNCTION public.clone_api_for_tenant(p_source_api_id uuid, p_target_tenant_id uuid, p_new_name character varying, p_user_id uuid)
 RETURNS uuid
 LANGUAGE plpgsql
AS $function$
DECLARE
    v_source_api RECORD;
    v_new_api_id UUID;
    v_endpoint RECORD;
    v_new_endpoint_id UUID;
    v_hook RECORD;
    v_security RECORD;
    v_doc RECORD;
BEGIN
    -- Get source API details
    SELECT * INTO v_source_api FROM public.api_definitions WHERE id = p_source_api_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Source API not found';
    END IF;
    
    -- Create new API definition
    INSERT INTO public.api_definitions (
        tenant_id, name, display_name, description, api_type, version, status,
        visibility, base_path, group_id, specification, is_core, cloned_from_id,
        created_by, updated_by
    ) VALUES (
        p_target_tenant_id, p_new_name, v_source_api.display_name, v_source_api.description,
        v_source_api.api_type, v_source_api.version, 'DRAFT', 'TENANT_SPECIFIC',
        v_source_api.base_path, v_source_api.group_id, v_source_api.specification,
        false, p_source_api_id, p_user_id, p_user_id
    ) RETURNING id INTO v_new_api_id;
    
    -- Clone endpoints
    FOR v_endpoint IN SELECT * FROM public.api_endpoints WHERE api_id = p_source_api_id LOOP
        INSERT INTO public.api_endpoints (
            api_id, tenant_id, path, http_method, operation_id, summary,
            description, request_schema, response_schema, deprecated,
            created_by, updated_by
        ) VALUES (
            v_new_api_id, p_target_tenant_id, v_endpoint.path, v_endpoint.http_method,
            v_endpoint.operation_id, v_endpoint.summary, v_endpoint.description,
            v_endpoint.request_schema, v_endpoint.response_schema, v_endpoint.deprecated,
            p_user_id, p_user_id
        ) RETURNING id INTO v_new_endpoint_id;
        
        -- Clone hooks for this endpoint
        FOR v_hook IN SELECT * FROM public.api_hooks WHERE endpoint_id = v_endpoint.id LOOP
            INSERT INTO public.api_hooks (
                tenant_id, name, display_name, description, event_type,
                api_id, endpoint_id, script, execution_order, is_active,
                execution_location, created_by, updated_by
            ) VALUES (
                p_target_tenant_id, v_hook.name, v_hook.display_name, v_hook.description,
                v_hook.event_type, v_new_api_id, v_new_endpoint_id, v_hook.script,
                v_hook.execution_order, v_hook.is_active, v_hook.execution_location,
                p_user_id, p_user_id
            );
        END LOOP;
    END LOOP;
    
    -- Clone API-level hooks
    FOR v_hook IN SELECT * FROM public.api_hooks WHERE api_id = p_source_api_id AND endpoint_id IS NULL LOOP
        INSERT INTO public.api_hooks (
            tenant_id, name, display_name, description, event_type,
            api_id, endpoint_id, script, execution_order, is_active,
            execution_location, created_by, updated_by
        ) VALUES (
            p_target_tenant_id, v_hook.name, v_hook.display_name, v_hook.description,
            v_hook.event_type, v_new_api_id, v_hook.endpoint_id, v_hook.script,
            v_hook.execution_order, v_hook.is_active, v_hook.execution_location,
            p_user_id, p_user_id
        );
    END LOOP;

    -- Clone security configurations
    FOR v_security IN SELECT * FROM public.api_security_configs WHERE api_id = p_source_api_id LOOP
        INSERT INTO public.api_security_configs (
            api_id, tenant_id, auth_methods, rate_limit_requests, rate_limit_period,
            ip_whitelist, require_approval, created_by, updated_by
        ) VALUES (
            v_new_api_id, p_target_tenant_id, v_security.auth_methods, v_security.rate_limit_requests,
            v_security.rate_limit_period, v_security.ip_whitelist, v_security.require_approval,
            p_user_id, p_user_id
        );
    END LOOP;

    -- Clone documentation
    FOR v_doc IN SELECT * FROM public.api_documentation WHERE api_id = p_source_api_id LOOP
        INSERT INTO public.api_documentation (
            api_id, tenant_id, content_type, content, section, order_index,
            created_by, updated_by
        ) VALUES (
            v_new_api_id, p_target_tenant_id, v_doc.content_type, v_doc.content,
            v_doc.section, v_doc.order_index, p_user_id, p_user_id
        );
    END LOOP;

    RETURN v_new_api_id;
END;
$function$;

CREATE OR REPLACE FUNCTION public.execute_business_rules(p_tenant_id uuid, p_table_name text, p_event_type text, p_data jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
AS $function$
DECLARE
    v_rule RECORD;
    v_result JSONB;
    v_modified_data JSONB := p_data;
BEGIN
    FOR v_rule IN 
        SELECT br.id, br.script
        FROM business_rules br
        JOIN metadata_tables mt ON br.table_id = mt.id
        WHERE br.tenant_id = p_tenant_id
          AND mt.name = p_table_name
          AND br.event_type = p_event_type
          AND br.is_active = TRUE
        ORDER BY br.execution_order
    LOOP
        INSERT INTO audit_logs (
            id, tenant_id, type, action, resource_type, resource_id, 
            status, request_data
        )
        VALUES (
            uuid_generate_v4(), p_tenant_id, 'business_rule_execution', 'execute',
            'business_rule', v_rule.id, 'success', p_data
        );
    END LOOP;
    RETURN v_modified_data;
END;
$function$;

CREATE OR REPLACE FUNCTION public.fabric_defn_refresh_index()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
declare
  v jsonb;
  -- generic fields we expect in each item
  it_name text; it_type text; it_title text; it_desc text; it_sql text; it_rel join_relationship;
begin
  -- Clear existing index rows for this definition
  delete from fabric_defn_index where defn_id = new.id;

  -- Helper to extract array items from a JSON path and write to index
  -- We index both core.* and custom.* arrays; resolved_config should already have overrides applied.
  -- Dimensions
  for v in
    select jsonb_array_elements(coalesce(new.resolved_config #> '{core,dimensions}', '[]'::jsonb))
    union all
    select jsonb_array_elements(coalesce(new.resolved_config #> '{custom,dimensions}', '[]'::jsonb))
  loop
    it_name := (v->>'name');
    it_type := (v->>'type');
    it_title := (v->>'title');
    it_desc := (v->>'description');
    it_sql := (v->>'sql');
    perform _fabric_defn_index_put(new.id, new.tenant_id, new.model_key, new.version,
                                   'dimension', it_name, it_type, null, it_title, it_desc, it_sql);
  end loop;

  -- Measures
  for v in
    select jsonb_array_elements(coalesce(new.resolved_config #> '{core,measures}', '[]'::jsonb))
    union all
    select jsonb_array_elements(coalesce(new.resolved_config #> '{custom,measures}', '[]'::jsonb))
  loop
    it_name := (v->>'name');
    it_type := (v->>'type');
    it_title := (v->>'title');
    it_desc := (v->>'description');
    it_sql := (v->>'sql');
    perform _fabric_defn_index_put(new.id, new.tenant_id, new.model_key, new.version,
                                   'measure', it_name, it_type, null, it_title, it_desc, it_sql);
  end loop;

  -- Joins
  for v in
    select jsonb_array_elements(coalesce(new.resolved_config #> '{core,joins}', '[]'::jsonb))
    union all
    select jsonb_array_elements(coalesce(new.resolved_config #> '{custom,joins}', '[]'::jsonb))
  loop
    it_name := (v->>'name');
    it_rel := (v->>'relationship')::join_relationship;
    it_title := (v->>'title');
    it_desc := (v->>'description');
    it_sql := (v->>'sql');
    perform _fabric_defn_index_put(new.id, new.tenant_id, new.model_key, new.version,
                                   'join', it_name, null, it_rel, it_title, it_desc, it_sql);
  end loop;

  return new;
end;
$function$;

CREATE OR REPLACE FUNCTION public.set_tenant_context(p_tenant_id uuid)
 RETURNS void
 LANGUAGE plpgsql
AS $function$
BEGIN
    PERFORM set_config('app.current_tenant_id', p_tenant_id::TEXT, FALSE);
END;
$function$;

CREATE OR REPLACE FUNCTION public.update_timestamp()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$function$;

-- Create missing views
CREATE OR REPLACE VIEW public.business_rule_summary_view
AS SELECT br.id,
    br.tenant_id,
    br.name,
    br.description,
    br.table_id,
    mt.name AS table_name,
    br.event_type,
    br.is_active,
    br.execution_order,
    br.execution_location,
    br.version,
    br.created_at,
    br.updated_at,
    t.name AS tenant_name,
    u1.username AS created_by_username,
    u2.username AS updated_by_username
   FROM business_rules br
     JOIN metadata_tables mt ON br.table_id = mt.id
    JOIN tenants t ON br.tenant_id::text = t.id::text
     JOIN users u1 ON br.created_by = u1.id
     JOIN users u2 ON br.updated_by = u2.id;

CREATE OR REPLACE VIEW public.catalog_edge_ve
AS SELECT ns.id AS subject_node_id,
    ns.node_name AS subject_node_name,
    ce.relationship_type,
    ce.id AS edge_id,
    cet.config AS edge_defn,
    no.node_name AS object_node_name,
    no.id AS object_node_id
   FROM catalog_edge ce
    JOIN catalog_node ns ON ns.id = ce.source_node_id
    JOIN catalog_node no ON no.id = ce.target_node_id
    LEFT JOIN catalog_edge_types cet ON cet.id = ce.edge_type_id;

CREATE OR REPLACE VIEW public.catalog_node_vw
AS SELECT cn.tenant_datasource_id,
    tpd.source_name,
    cn.id AS node_id,
    cn.node_name,
    cnt.catalog_type_name,
    cnt.config AS catalog_defn,
    cn.node_type_id,
    cn.description,
    cn.qualified_path,
    cn.properties,
    cn.parent_id
   FROM catalog_node cn
     JOIN catalog_node_type cnt ON cnt.id = cn.node_type_id
    JOIN tenant_product_datasource tpd ON tpd.id::text = cn.tenant_datasource_id::text;

CREATE OR REPLACE VIEW public.integration_summary_view
AS SELECT i.id,
    i.tenant_id,
    i.name,
    i.display_name,
    i.type,
    i.status,
    i.created_at,
    i.updated_at,
    COALESCE(m.requests_count, 0::bigint) AS total_requests,
    COALESCE(m.success_count, 0::bigint) AS successful_requests,
    COALESCE(m.error_count, 0::bigint) AS failed_requests,
    COALESCE(m.avg_duration, 0::double precision) AS avg_duration_ms,
    t.name AS tenant_name,
    u.username AS created_by_username
   FROM integrations i
     LEFT JOIN ( SELECT integration_metrics.integration_id,
            sum(integration_metrics.requests_count) AS requests_count,
            sum(integration_metrics.success_count) AS success_count,
            sum(integration_metrics.error_count) AS error_count,
            avg(integration_metrics.avg_duration) AS avg_duration
           FROM integration_metrics
          WHERE integration_metrics."timestamp" > (now() - '30 days'::interval)
          GROUP BY integration_metrics.integration_id) m ON i.id = m.integration_id
    JOIN tenants t ON i.tenant_id::text = t.id::text
     JOIN users u ON i.created_by = u.id;

-- Create missing triggers
DO $$ BEGIN
    CREATE TRIGGER update_product_updated_at BEFORE UPDATE ON public.alpha_product FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON public.tenants FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON public.api_keys FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_tenant_connections_updated_at BEFORE UPDATE ON public.tenant_connections FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_tenant_instance_updated_at BEFORE UPDATE ON public.tenant_instance FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_integration_timestamp BEFORE UPDATE ON public.integrations FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_integrations_updated_at BEFORE UPDATE ON public.integrations FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_business_rule_timestamp BEFORE UPDATE ON public.business_rules FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_business_rules_updated_at BEFORE UPDATE ON public.business_rules FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_metadata_table_timestamp BEFORE UPDATE ON public.metadata_tables FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_metadata_tables_updated_at BEFORE UPDATE ON public.metadata_tables FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_metadata_column_timestamp BEFORE UPDATE ON public.metadata_columns FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_metadata_columns_updated_at BEFORE UPDATE ON public.metadata_columns FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_metadata_relationships_updated_at BEFORE UPDATE ON public.metadata_relationships FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_roles_updated_at BEFORE UPDATE ON public.roles FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER update_integration_credentials_updated_at BEFORE UPDATE ON public.integration_credentials FOR EACH ROW EXECUTE FUNCTION update_timestamp();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TRIGGER fabric_defn_refresh_index_trigger AFTER INSERT OR UPDATE ON public.fabric_defn FOR EACH ROW EXECUTE FUNCTION fabric_defn_refresh_index();
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

-- Create missing partitions for api_metrics (if they don't exist)
DO $$ BEGIN
    CREATE TABLE IF NOT EXISTS public.api_metrics_current_month PARTITION OF public.api_metrics
        FOR VALUES FROM ('2025-07-01 00:00:00-04') TO ('2025-08-01 00:00:00-04');
EXCEPTION
    WHEN duplicate_table THEN null;
END $$;

DO $$ BEGIN
    CREATE TABLE IF NOT EXISTS public.api_metrics_next_month PARTITION OF public.api_metrics
        FOR VALUES FROM ('2025-08-01 00:00:00-04') TO ('2025-09-01 00:00:00-04');
EXCEPTION
    WHEN duplicate_table THEN null;
END $$;

-- Add missing indexes to partitions
CREATE INDEX IF NOT EXISTS api_metrics_current_month_api_id_timestamp_idx ON public.api_metrics_current_month USING btree (api_id, "timestamp");
CREATE INDEX IF NOT EXISTS api_metrics_current_month_tenant_id_timestamp_idx ON public.api_metrics_current_month USING btree (tenant_id, "timestamp");
CREATE INDEX IF NOT EXISTS api_metrics_next_month_api_id_timestamp_idx ON public.api_metrics_next_month USING btree (api_id, "timestamp");
CREATE INDEX IF NOT EXISTS api_metrics_next_month_tenant_id_timestamp_idx ON public.api_metrics_next_month USING btree (tenant_id, "timestamp");

-- Grant permissions (adjust as needed for your user)
-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO semlayer_user;
-- GRANT USAGE ON SCHEMA public TO semlayer_user;

COMMIT;

-- Deferred constraints pass (Phase-A): attempt any constraints that were skipped earlier
-- This uses safe_add_constraint/safe_add_tenant_fk so failures are logged but non-fatal.
DO $$ DECLARE
BEGIN
    -- metadata tables related
    PERFORM public.safe_add_constraint('metadata_tables_created_by_fk', 'ALTER TABLE public.metadata_tables ADD CONSTRAINT metadata_tables_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
    PERFORM public.safe_add_constraint('metadata_tables_updated_by_fk', 'ALTER TABLE public.metadata_tables ADD CONSTRAINT metadata_tables_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)');
    PERFORM public.safe_add_constraint('metadata_columns_reference_table_fk', 'ALTER TABLE public.metadata_columns ADD CONSTRAINT metadata_columns_reference_table_fk FOREIGN KEY (reference_table_id) REFERENCES public.metadata_tables(id)');
    PERFORM public.safe_add_constraint('metadata_columns_table_fk', 'ALTER TABLE public.metadata_columns ADD CONSTRAINT metadata_columns_table_fk FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');
    PERFORM public.safe_add_constraint('metadata_fields_table_id_fkey', 'ALTER TABLE public.metadata_fields ADD CONSTRAINT metadata_fields_table_id_fkey FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');
    PERFORM public.safe_add_constraint('metadata_relationships_source_table_fk', 'ALTER TABLE public.metadata_relationships ADD CONSTRAINT metadata_relationships_source_table_fk FOREIGN KEY (source_table) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');
    PERFORM public.safe_add_constraint('metadata_relationships_target_table_fk', 'ALTER TABLE public.metadata_relationships ADD CONSTRAINT metadata_relationships_target_table_fk FOREIGN KEY (target_table) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');
    PERFORM public.safe_add_constraint('metadata_events_table_id_fkey', 'ALTER TABLE public.metadata_events ADD CONSTRAINT metadata_events_table_id_fkey FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');

    -- tenant_product_datasource related
    PERFORM public.safe_add_constraint('tenant_product_datasource_alpha_datasource_fk', 'ALTER TABLE public.tenant_product_datasource ADD CONSTRAINT tenant_product_datasource_alpha_datasource_fk FOREIGN KEY (alpha_datasource_id) REFERENCES public.alpha_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
    PERFORM public.safe_add_constraint('tenant_product_datasource_tenant_product_fk', 'ALTER TABLE public.tenant_product_datasource ADD CONSTRAINT tenant_product_datasource_tenant_product_fk FOREIGN KEY (tenant_product_id) REFERENCES public.tenant_product(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
    PERFORM public.safe_add_constraint('tenant_product_product_fk', 'ALTER TABLE public.tenant_product ADD CONSTRAINT tenant_product_product_fk FOREIGN KEY (alpha_product_id) REFERENCES public.alpha_product(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
    PERFORM public.safe_add_constraint('tenant_product_tenant_instance_fk', 'ALTER TABLE public.tenant_product ADD CONSTRAINT tenant_product_tenant_instance_fk FOREIGN KEY (tenant_instance_id) REFERENCES public.tenant_instance(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');

    -- catalog/node related constraints depending on tenant_product_datasource
    PERFORM public.safe_add_constraint('catalog_node_tenant_product_datasource_fk', 'ALTER TABLE public.catalog_node ADD CONSTRAINT catalog_node_tenant_product_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
    PERFORM public.safe_add_constraint('catalog_node_type_tenant_product_datasource_fk', 'ALTER TABLE public.catalog_node_type ADD CONSTRAINT catalog_node_type_tenant_product_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');
    PERFORM public.safe_add_constraint('tenant_chart_tenant_product_datasource_fk', 'ALTER TABLE public.tenant_chart ADD CONSTRAINT tenant_chart_tenant_product_datasource_fk FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED');

    -- business rules table fk
    PERFORM public.safe_add_constraint('business_rules_table_fk', 'ALTER TABLE public.business_rules ADD CONSTRAINT business_rules_table_fk FOREIGN KEY (table_id) REFERENCES public.metadata_tables(id) ON DELETE CASCADE');

    -- other miscellaneous constraints that sometimes fail due to ordering
    PERFORM public.safe_add_constraint('active_requests_user_fk', 'ALTER TABLE public.active_requests ADD CONSTRAINT active_requests_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id)');
    PERFORM public.safe_add_constraint('exposed_apis_api_id_fkey', 'ALTER TABLE public.exposed_apis ADD CONSTRAINT exposed_apis_api_id_fkey FOREIGN KEY (api_id) REFERENCES public.apis(id) ON DELETE CASCADE');
    PERFORM public.safe_add_constraint('integration_audit_logs_user_fk', 'ALTER TABLE public.integration_audit_logs ADD CONSTRAINT integration_audit_logs_user_fk FOREIGN KEY (user_id) REFERENCES public.users(id)');
    PERFORM public.safe_add_constraint('integration_configs_created_by_fk', 'ALTER TABLE public.integration_configs ADD CONSTRAINT integration_configs_created_by_fk FOREIGN KEY (created_by) REFERENCES public.users(id)');
    PERFORM public.safe_add_constraint('integration_configs_updated_by_fk', 'ALTER TABLE public.integration_configs ADD CONSTRAINT integration_configs_updated_by_fk FOREIGN KEY (updated_by) REFERENCES public.users(id)');

EXCEPTION WHEN others THEN
    RAISE NOTICE 'Deferred constraints pass encountered an error: %', SQLERRM;
END $$;

    -- Phase-B canonicalization (destructive): convert legacy textual tenant/datasource id columns to uuid
    -- This is destructive: it drops legacy columns and renames canonical *_uuid columns into place.
    -- Rollback strategy: prior to running this in production, take a DB snapshot/export and review the mapping
    -- between legacy textual ids and generated/assigned uuids (see comments below for manual rollback SQL).
    DO $$ DECLARE
        tbl TEXT;
        uuid_regex TEXT := '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$';
        cnt BIGINT;
        legacy_exists BOOLEAN;
        legacy_type TEXT;
    BEGIN
        -- Target tables that historically stored tenant_product_datasource ids as text
        FOR tbl IN SELECT unnest(ARRAY['catalog_node','catalog_node_type','tenant_chart']) LOOP

            -- Ensure a canonical tenant_datasource_id_uuid column exists for safe backfill
            PERFORM public.safe_add_column('public', tbl, 'tenant_datasource_id_uuid uuid');

            -- Inspect legacy column existence and type
            SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=tbl AND column_name='tenant_datasource_id') INTO legacy_exists;
            SELECT data_type FROM information_schema.columns WHERE table_schema='public' AND table_name=tbl AND column_name='tenant_datasource_id' LIMIT 1 INTO legacy_type;

            -- 1) Backfill from existing tenant_datasource_id values via tenant_product_datasource mapping (only if legacy column exists)
            IF legacy_exists THEN
                BEGIN
                    IF legacy_type IN ('character varying','text') THEN
                        EXECUTE format('UPDATE public.%I cn SET tenant_datasource_id_uuid = tpd.id '
                                       || 'FROM public.tenant_product_datasource tpd '
                                       || 'WHERE (tpd.id::text = cn.tenant_datasource_id::text OR (tpd.id_text IS NOT NULL AND tpd.id_text = cn.tenant_datasource_id::text)) '
                                       || 'AND cn.tenant_datasource_id_uuid IS NULL', tbl);
                    ELSIF legacy_type = 'uuid' THEN
                        EXECUTE format('UPDATE public.%I SET tenant_datasource_id_uuid = tenant_datasource_id WHERE tenant_datasource_id_uuid IS NULL', tbl);
                    ELSE
                        RAISE NOTICE 'Phase-B: legacy column % on % has unsupported type %; skipping mapping', 'tenant_datasource_id', tbl, legacy_type;
                    END IF;
                EXCEPTION WHEN others THEN
                    RAISE NOTICE 'Phase-B backfill mapping (text->uuid) for % failed: %', tbl, SQLERRM;
                END;

                -- 2) If the existing tenant_datasource_id contains UUID strings (text), cast them directly
                BEGIN
                    IF legacy_type IN ('character varying','text') THEN
                        EXECUTE format('UPDATE public.%I SET tenant_datasource_id_uuid = tenant_datasource_id::uuid '
                                       || 'WHERE tenant_datasource_id ~ %L AND tenant_datasource_id_uuid IS NULL', tbl, uuid_regex);
                    END IF;
                EXCEPTION WHEN others THEN
                    RAISE NOTICE 'Phase-B cast backfill for % failed: %', tbl, SQLERRM;
                END;
            END IF;

            -- 3) If there are still NULLs, leave them NULL but log how many remain
            BEGIN
                IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=tbl AND column_name='tenant_datasource_id_uuid') THEN
                    EXECUTE format('SELECT COUNT(1) FROM public.%I WHERE tenant_datasource_id_uuid IS NULL', tbl) INTO cnt;
                ELSE
                    cnt := 0;
                END IF;
                IF cnt > 0 THEN
                    RAISE NOTICE 'Phase-B: % has % rows with NULL tenant_datasource_id_uuid after backfill; manual review advised', tbl, cnt;
                END IF;
            EXCEPTION WHEN others THEN
                NULL;
            END;

            -- 4) If legacy textual column does not exist, rename the uuid column into place; otherwise keep both and create FK on the uuid column
            BEGIN
                IF NOT legacy_exists THEN
                    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=tbl AND column_name='tenant_datasource_id_uuid') THEN
                        EXECUTE format('ALTER TABLE public.%I RENAME COLUMN tenant_datasource_id_uuid TO tenant_datasource_id', tbl);
                        PERFORM public.safe_add_constraint(format('%s_tenant_product_datasource_fk', tbl),
                            format('ALTER TABLE public.%I ADD CONSTRAINT %I FOREIGN KEY (tenant_datasource_id) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED', tbl, tbl||'_tenant_product_datasource_fk'));
                    END IF;
                ELSE
                    -- legacy column exists; create FK on the uuid column if present and leave legacy column in place
                    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=tbl AND column_name='tenant_datasource_id_uuid') THEN
                        PERFORM public.safe_add_constraint(format('%s_tenant_product_datasource_uuid_fk', tbl),
                            format('ALTER TABLE public.%I ADD CONSTRAINT %I FOREIGN KEY (tenant_datasource_id_uuid) REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE ON UPDATE CASCADE DEFERRABLE INITIALLY DEFERRED', tbl, tbl||'_tenant_product_datasource_uuid_fk'));
                        RAISE NOTICE 'Phase-B: left legacy tenant_datasource_id on % in place and created FK on tenant_datasource_id_uuid; manual cleanup may be required', tbl;
                    ELSE
                        RAISE NOTICE 'Phase-B: no tenant_datasource_id or tenant_datasource_id_uuid found on %; skipped FK creation', tbl;
                    END IF;
                END IF;
            EXCEPTION WHEN others THEN
                RAISE NOTICE 'Phase-B: could not finalize column rename/constraint on %: %', tbl, SQLERRM;
            END;

        END LOOP;

        -- Clean up compatibility column on tenant_product_datasource if present (destructive)
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='tenant_product_datasource' AND column_name='id_text') THEN
            BEGIN
                EXECUTE 'ALTER TABLE public.tenant_product_datasource DROP COLUMN id_text';
            EXCEPTION WHEN others THEN
                RAISE NOTICE 'Phase-B: could not drop tenant_product_datasource.id_text: %', SQLERRM;
            END;
        END IF;

    EXCEPTION WHEN others THEN
        RAISE NOTICE 'Phase-B canonicalization encountered an error: %', SQLERRM;
    END $$;

    -- Phase-B rollback notes (manual):
    -- To rollback a table conversion, you can recreate the old textual column and backfill it from tenant_product_datasource.id via
    --   ALTER TABLE public.<table> ADD COLUMN tenant_datasource_id_text varchar;
    --   UPDATE public.<table> t SET tenant_datasource_id_text = tpd.id::text FROM public.tenant_product_datasource tpd WHERE t.tenant_datasource_id = tpd.id;
    -- Then drop the uuid FK and rename columns back. Always snapshot the DB before running destructive changes.
