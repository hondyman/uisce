-- 20261016_011_page_definitions_core_status.up.sql
-- Adds the metadata the Page Designer's list view needs to distinguish
-- gold-copy-authored pages from tenant-authored ones and to show a
-- publish state, neither of which page_definitions carried before this.
--
-- is_core mirrors the same convention business_objects.is_core already
-- uses on this platform: true means the row was authored under the
-- gold-copy tenant and every other tenant sees it read-only via
-- inheritance; false is an ordinary tenant-authored page. Nothing
-- currently writes true here - this migration only adds the column so
-- the write path can exist later without another schema change.
--
-- status is a simple draft/published lifecycle flag for the list view's
-- status chip. No workflow/approval semantics are attached to
-- "published" here - it's a display and (future) visibility flag, not a
-- governance gate.

ALTER TABLE public.page_definitions
    ADD COLUMN IF NOT EXISTS is_core BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'draft';

ALTER TABLE public.page_definitions
    DROP CONSTRAINT IF EXISTS page_definitions_status_check;

ALTER TABLE public.page_definitions
    ADD CONSTRAINT page_definitions_status_check CHECK (status IN ('draft', 'published'));

CREATE INDEX IF NOT EXISTS idx_page_definitions_is_core
    ON public.page_definitions (is_core);
