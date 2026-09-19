-- Presentation-only event rules for Page Studio (FieldChange analog).
-- page_studio_handler.go must round-trip this column or saves silently drop rules.
ALTER TABLE public.page_definitions
    ADD COLUMN IF NOT EXISTS presentation_events JSONB NOT NULL DEFAULT '[]'::jsonb;
