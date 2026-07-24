-- Minimal dev tables to suppress warnings and allow basic queries
-- Keep Postgres outside Docker; run via psql against local DB

BEGIN;

CREATE TABLE IF NOT EXISTS business_object_def (
  bo_def_id uuid PRIMARY KEY,
  tenant_id uuid NOT NULL,
  bo_key text NOT NULL,
  name text NOT NULL,
  display_name text NOT NULL,
  description text,
  driver_table_name text,
  created_at timestamptz NOT NULL DEFAULT now(),
  created_by text,
  updated_at timestamptz,
  updated_by text,
  config jsonb DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_business_object_def_tenant_key
  ON business_object_def (tenant_id, bo_key);

CREATE TABLE IF NOT EXISTS workflow_audit_log (
  id uuid PRIMARY KEY,
  tenant_id uuid,
  entity_type text,
  entity_id text,
  action text,
  user_id text,
  created_at timestamptz NOT NULL DEFAULT now(),
  metadata jsonb DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_workflow_audit_log_tenant
  ON workflow_audit_log (tenant_id, created_at);

CREATE TABLE IF NOT EXISTS tenant_quotas (
  tenant_id uuid PRIMARY KEY,
  max_requests int DEFAULT 0,
  window_seconds int DEFAULT 60,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS my_approvals_queue (
  id uuid PRIMARY KEY,
  tenant_id uuid,
  payload jsonb DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

COMMIT;
