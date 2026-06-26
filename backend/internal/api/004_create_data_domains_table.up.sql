-- Migration: create a three-level hierarchical data_domains table
CREATE TABLE IF NOT EXISTS public.data_domain (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  parent_id UUID NULL,
  level SMALLINT NOT NULL DEFAULT 1,
  description TEXT NULL,
  created_by TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT fk_parent FOREIGN KEY(parent_id) REFERENCES public.data_domain(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_data_domain_parent ON public.data_domain(parent_id);
CREATE INDEX IF NOT EXISTS idx_data_domain_slug ON public.data_domain(lower(slug));

-- Optional seed: create a top-level 'finance' domain and a couple of children
INSERT INTO public.data_domain (id, name, slug, parent_id, level, description, created_by)
VALUES
  (gen_random_uuid(), 'Finance', 'finance', NULL, 1, 'Finance domain', 'system')
ON CONFLICT (slug) DO NOTHING;
