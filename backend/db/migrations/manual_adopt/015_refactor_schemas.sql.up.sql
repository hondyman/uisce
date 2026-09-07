-- Migration 015: Refactor Schemas
-- Move tables from public/dma to logical domain schemas

-- 1. Create new schemas
CREATE SCHEMA IF NOT EXISTS platform;
CREATE SCHEMA IF NOT EXISTS wealth;
CREATE SCHEMA IF NOT EXISTS metadata;
CREATE SCHEMA IF NOT EXISTS analytics;

-- 2. Move Platform Tables
ALTER TABLE IF EXISTS public.tenants SET SCHEMA platform;
ALTER TABLE IF EXISTS public.users SET SCHEMA platform;
ALTER TABLE IF EXISTS public.roles SET SCHEMA platform;
ALTER TABLE IF EXISTS public.permissions SET SCHEMA platform;
ALTER TABLE IF EXISTS public.user_roles SET SCHEMA platform;
ALTER TABLE IF EXISTS public.role_permissions SET SCHEMA platform;
ALTER TABLE IF EXISTS public.api_endpoints_catalog SET SCHEMA platform;
ALTER TABLE IF EXISTS public.api_endpoint_entity_mappings SET SCHEMA platform;
ALTER TABLE IF EXISTS public.api_endpoint_datasource_mappings SET SCHEMA platform;
ALTER TABLE IF EXISTS public.datasources SET SCHEMA platform;
ALTER TABLE IF EXISTS public.access_control_policies SET SCHEMA platform;

-- 3. Move Wealth Tables
ALTER TABLE IF EXISTS public.clients SET SCHEMA wealth;
ALTER TABLE IF EXISTS public.portfolios SET SCHEMA wealth;
ALTER TABLE IF EXISTS public.assets SET SCHEMA wealth;
ALTER TABLE IF EXISTS public.transactions SET SCHEMA wealth;
ALTER TABLE IF EXISTS public.orders SET SCHEMA wealth;
ALTER TABLE IF EXISTS public.market_prices SET SCHEMA wealth;
ALTER TABLE IF EXISTS public.compliance_records SET SCHEMA wealth;
-- Note: audit_trail might be shared, but if it was wealth-specific, move it. 
-- If it's the central audit, maybe platform? Let's assume platform for central audit if it exists there.
-- Checking previous migrations, audit_trail was in wealth_app_schema.sql.
ALTER TABLE IF EXISTS public.audit_trail SET SCHEMA wealth; 

-- 4. Move Metadata Tables
ALTER TABLE IF EXISTS public.meta_objects SET SCHEMA metadata;
ALTER TABLE IF EXISTS public.meta_processes SET SCHEMA metadata;
ALTER TABLE IF EXISTS public.meta_views SET SCHEMA metadata;
ALTER TABLE IF EXISTS public.catalog_node SET SCHEMA metadata;
ALTER TABLE IF EXISTS public.catalog_edge_type SET SCHEMA metadata;
ALTER TABLE IF EXISTS public.entity_registry SET SCHEMA metadata;
ALTER TABLE IF EXISTS public.catalog_validation_rules SET SCHEMA metadata;
ALTER TABLE IF EXISTS public.catalog_validation_rules_audit SET SCHEMA metadata;
ALTER TABLE IF EXISTS public.data_domains SET SCHEMA metadata;

-- 5. Move Analytics Tables
-- Some might already be in semantic_layer schema, let's consolidate to 'analytics' or keep 'semantic_layer' as alias?
-- Plan said 'analytics'. Let's move semantic_layer tables to analytics if we want to enforce the new name,
-- OR we can just rename the schema.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'semantic_layer') THEN
        ALTER SCHEMA semantic_layer RENAME TO analytics;
    END IF;
END
$$;

-- Move any other analytics tables that were in public
ALTER TABLE IF EXISTS public.fabric_defn SET SCHEMA analytics;
ALTER TABLE IF EXISTS public.views SET SCHEMA analytics;
ALTER TABLE IF EXISTS public.semantic_assets SET SCHEMA analytics;
ALTER TABLE IF EXISTS public.relationship_suggestions SET SCHEMA analytics;
ALTER TABLE IF EXISTS public.relationship_suggestion_audit SET SCHEMA analytics;

-- 6. Update Search Path for convenience (optional, but good for default user)
-- ALTER ROLE postgres SET search_path TO platform, wealth, metadata, analytics, public;
