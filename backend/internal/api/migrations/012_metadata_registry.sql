-- Metadata Registry Schema

-- 1. Meta Objects (Business Objects)
CREATE TABLE IF NOT EXISTS meta_objects (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NULL,
  name TEXT NOT NULL,
  version_major INT NOT NULL,
  version_minor INT NOT NULL,
  version_patch INT NOT NULL,
  status TEXT CHECK (status IN ('draft','active','deprecated')),
  valid_from TIMESTAMP NOT NULL DEFAULT now(),
  valid_to TIMESTAMP NULL,
  payload JSONB NOT NULL
);

-- 2. Meta Views (UI Definitions)
CREATE TABLE IF NOT EXISTS meta_views (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NULL,
  name TEXT NOT NULL,
  version_major INT NOT NULL,
  version_minor INT NOT NULL,
  version_patch INT NOT NULL,
  status TEXT CHECK (status IN ('draft','active','deprecated')),
  valid_from TIMESTAMP NOT NULL DEFAULT now(),
  valid_to TIMESTAMP NULL,
  payload JSONB NOT NULL
);

-- 3. Meta Processes (Workflow Definitions)
CREATE TABLE IF NOT EXISTS meta_processes (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NULL,
  name TEXT NOT NULL,
  version_major INT NOT NULL,
  version_minor INT NOT NULL,
  version_patch INT NOT NULL,
  status TEXT CHECK (status IN ('draft','active','deprecated')),
  valid_from TIMESTAMP NOT NULL DEFAULT now(),
  valid_to TIMESTAMP NULL,
  payload JSONB NOT NULL
);

-- 4. Meta Metrics (KPI Definitions)
CREATE TABLE IF NOT EXISTS meta_metrics (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NULL,
  name TEXT NOT NULL,
  version_major INT NOT NULL,
  version_minor INT NOT NULL,
  version_patch INT NOT NULL,
  status TEXT CHECK (status IN ('draft','active','deprecated')),
  valid_from TIMESTAMP NOT NULL DEFAULT now(),
  valid_to TIMESTAMP NULL,
  payload JSONB NOT NULL
);

-- Indexes for fast lookup of active versions
CREATE INDEX IF NOT EXISTS idx_meta_objects_active ON meta_objects (tenant_id, status, version_major, version_minor, version_patch);
CREATE INDEX IF NOT EXISTS idx_meta_views_active ON meta_views (tenant_id, status, version_major, version_minor, version_patch);
CREATE INDEX IF NOT EXISTS idx_meta_processes_active ON meta_processes (tenant_id, status, version_major, version_minor, version_patch);
CREATE INDEX IF NOT EXISTS idx_meta_metrics_active ON meta_metrics (tenant_id, status, version_major, version_minor, version_patch);

-- Row-Level Security (RLS) Policies
-- Ensure tenants can only see Core (tenant_id IS NULL) or their own extensions

ALTER TABLE meta_objects ENABLE ROW LEVEL SECURITY;
ALTER TABLE meta_views ENABLE ROW LEVEL SECURITY;
ALTER TABLE meta_processes ENABLE ROW LEVEL SECURITY;
ALTER TABLE meta_metrics ENABLE ROW LEVEL SECURITY;

-- Policy: Read Core + Own Tenant
CREATE POLICY tenant_meta_read_objects ON meta_objects
  FOR SELECT USING (
    tenant_id IS NULL OR tenant_id = current_setting('app.tenant_id', true)
  );

CREATE POLICY tenant_meta_read_views ON meta_views
  FOR SELECT USING (
    tenant_id IS NULL OR tenant_id = current_setting('app.tenant_id', true)
  );

CREATE POLICY tenant_meta_read_processes ON meta_processes
  FOR SELECT USING (
    tenant_id IS NULL OR tenant_id = current_setting('app.tenant_id', true)
  );

CREATE POLICY tenant_meta_read_metrics ON meta_metrics
  FOR SELECT USING (
    tenant_id IS NULL OR tenant_id = current_setting('app.tenant_id', true)
  );

-- Policy: Write Own Tenant Only (No writing to Core by tenants)
CREATE POLICY tenant_meta_write_objects ON meta_objects
  FOR ALL USING (
    tenant_id = current_setting('app.tenant_id', true)
  );

CREATE POLICY tenant_meta_write_views ON meta_views
  FOR ALL USING (
    tenant_id = current_setting('app.tenant_id', true)
  );

CREATE POLICY tenant_meta_write_processes ON meta_processes
  FOR ALL USING (
    tenant_id = current_setting('app.tenant_id', true)
  );

CREATE POLICY tenant_meta_write_metrics ON meta_metrics
  FOR ALL USING (
    tenant_id = current_setting('app.tenant_id', true)
  );
