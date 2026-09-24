-- 20260920_report_marketplace_and_cloning.sql
-- Template Marketplace, Core Cloning, 3-Way Rebase Tracking, and Smart Folders

-- 1. Folder Hierarchy & ABAC Scoping
ALTER TABLE public.folders
ADD COLUMN IF NOT EXISTS parent_folder_id UUID REFERENCES public.folders(id) ON DELETE CASCADE,
ADD COLUMN IF NOT EXISTS materialized_path TEXT DEFAULT '/',
ADD COLUMN IF NOT EXISTS is_smart_folder BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS smart_filter_criteria JSONB DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_folders_materialized_path 
ON public.folders (tenant_id, materialized_path);

-- 2. Report Definition Extensions for Marketplace & Cloning
ALTER TABLE public.report_definition
ADD COLUMN IF NOT EXISTS cloned_from_id UUID REFERENCES public.report_definition(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS base_version_at_clone INT DEFAULT 1,
ADD COLUMN IF NOT EXISTS current_version INT DEFAULT 1,
ADD COLUMN IF NOT EXISTS is_marketplace_published BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS marketplace_package_id VARCHAR(100),
ADD COLUMN IF NOT EXISTS required_semantic_terms JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS custom_override_delta JSONB DEFAULT '{}'::jsonb;

-- 3. Template Version History & Diff Ledger
CREATE TABLE IF NOT EXISTS public.report_template_versions (
    version_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    report_definition_id UUID NOT NULL REFERENCES public.report_definition(id) ON DELETE CASCADE,
    version_number INT NOT NULL,
    layout_spec JSONB NOT NULL,
    sections_spec JSONB NOT NULL,
    styling_spec JSONB NOT NULL,
    commit_message TEXT NOT NULL,
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_report_version UNIQUE (report_definition_id, version_number)
);

-- 4. Marketplace Packages & Installations
CREATE TABLE IF NOT EXISTS public.report_marketplace_packages (
    package_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    package_code VARCHAR(100) UNIQUE NOT NULL,
    display_name VARCHAR(150) NOT NULL,
    publisher_name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,                      -- REGULATORY, CLIENT_REPORTING, RISK, WEALTH
    description TEXT,
    thumbnail_url TEXT,
    latest_version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    manifest_spec JSONB NOT NULL,
    rating NUMERIC(3, 2) DEFAULT 5.00,
    install_count INT DEFAULT 0,
    is_verified BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.report_marketplace_installs (
    install_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    package_id UUID NOT NULL REFERENCES public.report_marketplace_packages(package_id) ON DELETE CASCADE,
    installed_version VARCHAR(20) NOT NULL,
    installed_report_id UUID REFERENCES public.report_definition(id) ON DELETE SET NULL,
    installed_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_package_install UNIQUE (tenant_id, package_id)
);
