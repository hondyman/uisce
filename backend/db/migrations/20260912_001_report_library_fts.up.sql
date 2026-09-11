-- Migration: 20260912_001_report_library_fts.up.sql
-- Description:
-- 1. Ensure pg_trgm extension is active for similarity matching and typo tolerance.
-- 2. Add `search_vector` generated stored tsvector column to public.report_templates with weighted attributes:
--      Weight 'A': template_name (highest priority)
--      Weight 'B': description (secondary context)
--      Weight 'C': category (tertiary taxonomy)
-- 3. Create GIN index on `search_vector` for fast tsquery matching.
-- 4. Create GIN trigram index on `template_name gin_trgm_ops` for fuzzy matching and typo tolerance.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE public.report_templates
    ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(template_name, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(description, '')), 'B') ||
        setweight(to_tsvector('simple', coalesce(category, '')), 'C')
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_report_templates_search_vector
    ON public.report_templates USING gin(search_vector);

CREATE INDEX IF NOT EXISTS idx_report_templates_trgm_name
    ON public.report_templates USING gin(template_name gin_trgm_ops);
