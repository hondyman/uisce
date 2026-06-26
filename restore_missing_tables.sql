-- Restore Missing Tables: Abbreviations, Lookups, Config

-- 1. Restore Abbreviations (sml.abbreviation_lookup)
CREATE SCHEMA IF NOT EXISTS sml;

CREATE TABLE IF NOT EXISTS sml.abbreviation_lookup (
    id SERIAL PRIMARY KEY,
    abbreviation TEXT NOT NULL,
    full_word TEXT NOT NULL,
    notes TEXT,
    tenant_id VARCHAR(255) NOT NULL DEFAULT 'uisce',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_abbreviations_tenant_abbr 
ON sml.abbreviation_lookup(tenant_id, UPPER(abbreviation));

CREATE INDEX IF NOT EXISTS idx_abbreviations_tenant 
ON sml.abbreviation_lookup(tenant_id);


-- 2. Restore Lookups and Lookup Values (public)
CREATE TABLE IF NOT EXISTS public.lookups (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  name text NOT NULL,
  description text,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_lookups_tenant ON public.lookups (tenant_id);

CREATE TABLE IF NOT EXISTS public.lookup_values (
  id uuid NOT NULL DEFAULT gen_random_uuid(),
  lookup_id uuid NOT NULL,
  tenant_id uuid NOT NULL,
  value text NOT NULL,
  label text NOT NULL,
  parent_id uuid,
  metadata jsonb DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_lookup_values_lookup_id ON public.lookup_values (lookup_id);


-- 3. Restore Tenant Configs (public)
CREATE TABLE IF NOT EXISTS public.tenant_configs (
    tenant_id VARCHAR(255) PRIMARY KEY,
    tier INTEGER NOT NULL DEFAULT 0,
    concurrency_limit INTEGER NOT NULL DEFAULT 10,
    token_rate INTEGER NOT NULL DEFAULT 100,
    burst_tokens INTEGER NOT NULL DEFAULT 200,
    cpu_limit DECIMAL(5,2) NOT NULL DEFAULT 10.0,
    memory_limit BIGINT NOT NULL DEFAULT 104857600,
    cache_ttl INTERVAL NOT NULL DEFAULT '5 minutes',
    priority INTEGER NOT NULL DEFAULT 1,
    features JSONB NOT NULL DEFAULT '{
        "automation_auto_apply": false,
        "conversational_features": true,
        "advanced_analytics": false,
        "custom_integrations": false
    }',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenant_configs_tenant_id ON tenant_configs(tenant_id);

-- Insert Default Configs
INSERT INTO tenant_configs (tenant_id, tier, concurrency_limit, token_rate, burst_tokens, cpu_limit, memory_limit, cache_ttl, priority, features)
VALUES
    ('default_bronze', 0, 10, 100, 200, 10.0, 104857600, '5 minutes', 1,
     '{"automation_auto_apply": false, "conversational_features": true, "advanced_analytics": false, "custom_integrations": false}'),
    ('default_silver', 1, 50, 500, 1000, 25.0, 524288000, '10 minutes', 5,
     '{"automation_auto_apply": true, "conversational_features": true, "advanced_analytics": false, "custom_integrations": false}'),
    ('default_gold', 2, 100, 1000, 2000, 50.0, 1073741824, '15 minutes', 10,
     '{"automation_auto_apply": true, "conversational_features": true, "advanced_analytics": true, "custom_integrations": true}')
ON CONFLICT (tenant_id) DO NOTHING;
