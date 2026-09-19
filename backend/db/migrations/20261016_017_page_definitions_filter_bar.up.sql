-- Page-wide filter bar (slicers above tabs). PageEditor already edits
-- draft.filterBar in memory; without this column the handler drops it on save
-- the same way tabs used to drop.
ALTER TABLE public.page_definitions
    ADD COLUMN IF NOT EXISTS filter_bar JSONB NOT NULL DEFAULT '{}'::jsonb;
