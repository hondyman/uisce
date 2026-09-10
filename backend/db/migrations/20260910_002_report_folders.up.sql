-- 20260910_002_report_folders.up.sql
-- Up migration for private per-user report folders and folder item mappings

-- 1. Create report_folders table
CREATE TABLE IF NOT EXISTS public.report_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    parent_id UUID REFERENCES public.report_folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_report_folders_user FOREIGN KEY (user_id) REFERENCES public.app_user(id) ON DELETE CASCADE
);

-- 2. Indexes for tenant & user filtering, parent traversal
CREATE INDEX IF NOT EXISTS idx_report_folders_tenant_user 
    ON public.report_folders (tenant_id, user_id);

CREATE INDEX IF NOT EXISTS idx_report_folders_parent 
    ON public.report_folders (parent_id);

-- 3. Case-insensitive sibling name uniqueness per user & parent
CREATE UNIQUE INDEX IF NOT EXISTS uq_report_folders_root_name
    ON public.report_folders (tenant_id, user_id, LOWER(name))
    WHERE parent_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_report_folders_child_name
    ON public.report_folders (tenant_id, user_id, parent_id, LOWER(name))
    WHERE parent_id IS NOT NULL;

-- 4. Create report_folder_items junction table (normalized, no redundant tenant/user)
CREATE TABLE IF NOT EXISTS public.report_folder_items (
    folder_id UUID NOT NULL REFERENCES public.report_folders(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES public.report_templates(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (folder_id, template_id)
);

-- 5. Index for reverse lookup (finding which folders contain a report)
CREATE INDEX IF NOT EXISTS idx_report_folder_items_template 
    ON public.report_folder_items (template_id);
