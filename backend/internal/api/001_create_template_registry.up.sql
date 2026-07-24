CREATE TABLE public.template_registry (
  node_id TEXT PRIMARY KEY,
  version TEXT NOT NULL,
  node_type TEXT NOT NULL,
  domain TEXT NOT NULL,
  category TEXT NOT NULL,
  subcategory TEXT NOT NULL,
  owner TEXT NOT NULL,
  tags TEXT[] DEFAULT '{}',
  lineage TEXT[] DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'draft',
  schema_hash TEXT NOT NULL,
  template JSONB NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX ON public.template_registry (domain, category, subcategory);
CREATE INDEX ON public.template_registry USING GIN (tags);
CREATE INDEX ON public.template_registry USING GIN (template jsonb_path_ops);