-- 20261016_012_page_definitions_tabs.up.sql
-- Adds the column multi-tab pages actually need. The Page Designer editor
-- (frontend/src/pages/page-studio/PageEditor.tsx) already builds and
-- edits a `tabs: PageTab[]` array in memory, but page_studio_handler.go's
-- upsert request never decoded it and page_definitions had nowhere to put
-- it - every save silently dropped a page's extra tabs back down to
-- whatever `layout` held (the first/implicit tab only). This column is
-- additive and optional: a page with no tabs (the common case) stores an
-- empty array here and keeps using `layout` exactly as before.
ALTER TABLE public.page_definitions
    ADD COLUMN IF NOT EXISTS tabs JSONB NOT NULL DEFAULT '[]'::jsonb;
