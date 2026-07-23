CREATE TABLE public.template_versions (
  node_id TEXT NOT NULL,
  version TEXT NOT NULL,
  schema_hash TEXT NOT NULL,
  template JSONB NOT NULL,
  steward_notes TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  PRIMARY KEY (node_id, version)
);