-- Business Process Designer Schema
-- Enables low-code, configuration-driven workflow and validation step creation
-- All step types, operators, and events are stored in JSONB for easy administration

-- Step Types (the palette of available steps)
CREATE TABLE IF NOT EXISTS process_step_types (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key           TEXT UNIQUE NOT NULL,
  label         TEXT NOT NULL,
  description   TEXT,
  icon_svg      TEXT,
  default_data  JSONB DEFAULT '{}'::jsonb,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now(),
  tenant_id     UUID,
  is_system     BOOLEAN DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_process_step_types_tenant ON process_step_types(tenant_id);
CREATE INDEX IF NOT EXISTS idx_process_step_types_key ON process_step_types(key);

-- Validation Operators (equals, greaterThan, inList, etc.)
CREATE TABLE IF NOT EXISTS validation_operators (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key           TEXT UNIQUE NOT NULL,
  label         TEXT NOT NULL,
  description   TEXT,
  value_type    TEXT NOT NULL CHECK (value_type IN ('string','number','boolean','list','date','currency')),
  config        JSONB DEFAULT '{}'::jsonb,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now(),
  tenant_id     UUID,
  is_system     BOOLEAN DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_validation_operators_tenant ON validation_operators(tenant_id);
CREATE INDEX IF NOT EXISTS idx_validation_operators_key ON validation_operators(key);

-- Workflow Events (Client Application Submitted, Client Data Updated, etc.)
CREATE TABLE IF NOT EXISTS workflow_events (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key           TEXT UNIQUE NOT NULL,
  label         TEXT NOT NULL,
  description   TEXT,
  event_type    TEXT CHECK (event_type IN ('on_start', 'on_update', 'on_submit', 'on_approval', 'on_completion', 'custom')),
  config        JSONB DEFAULT '{}'::jsonb,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now(),
  tenant_id     UUID,
  is_system     BOOLEAN DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_workflow_events_tenant ON workflow_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_workflow_events_key ON workflow_events(key);

-- Business Object Metadata (client, account, transaction, etc.)
-- Fields are stored as JSONB: [{name:"net_worth", type:"number", label:"Net Worth"}]
CREATE TABLE IF NOT EXISTS business_objects (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name          TEXT NOT NULL,
  display_name  TEXT NOT NULL,
  description   TEXT,
  fields        JSONB NOT NULL,
  icon          TEXT,
  config        JSONB DEFAULT '{}'::jsonb,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now(),
  tenant_id     UUID,
  is_system     BOOLEAN DEFAULT false,
  UNIQUE(name, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_business_objects_tenant ON business_objects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_business_objects_name ON business_objects(name);

-- Process Definitions (the canvas with nodes and edges)
CREATE TABLE IF NOT EXISTS processes (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name          TEXT NOT NULL,
  description   TEXT,
  version       INTEGER DEFAULT 1,
  nodes         JSONB NOT NULL,
  edges         JSONB NOT NULL,
  config        JSONB DEFAULT '{}'::jsonb,
  status        TEXT CHECK (status IN ('draft', 'published', 'archived')) DEFAULT 'draft',
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now(),
  created_by    UUID,
  updated_by    UUID,
  tenant_id     UUID,
  datasource_id UUID
);

CREATE INDEX IF NOT EXISTS idx_processes_tenant_datasource ON processes(tenant_id, datasource_id);
CREATE INDEX IF NOT EXISTS idx_processes_name ON processes(name);
CREATE INDEX IF NOT EXISTS idx_processes_status ON processes(status);

-- Process Versions (full history and rollback support)
CREATE TABLE IF NOT EXISTS process_versions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  process_id    UUID NOT NULL REFERENCES processes(id) ON DELETE CASCADE,
  version_num   INTEGER NOT NULL,
  nodes         JSONB NOT NULL,
  edges         JSONB NOT NULL,
  change_log    TEXT,
  created_at    TIMESTAMPTZ DEFAULT now(),
  created_by    UUID,
  tenant_id     UUID,
  UNIQUE(process_id, version_num)
);

CREATE INDEX IF NOT EXISTS idx_process_versions_process ON process_versions(process_id);
CREATE INDEX IF NOT EXISTS idx_process_versions_tenant ON process_versions(tenant_id);

-- Validation Rules (rules bound to validation steps)
CREATE TABLE IF NOT EXISTS validation_rules (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  process_id    UUID NOT NULL REFERENCES processes(id) ON DELETE CASCADE,
  node_id       TEXT NOT NULL,
  field         TEXT NOT NULL,
  operator_key  TEXT NOT NULL,
  operator_id   UUID REFERENCES validation_operators(id),
  value         TEXT,
  message       TEXT,
  severity      TEXT CHECK (severity IN ('block', 'warning', 'info')) DEFAULT 'warning',
  order_index   INTEGER DEFAULT 0,
  enabled       BOOLEAN DEFAULT true,
  config        JSONB DEFAULT '{}'::jsonb,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now(),
  tenant_id     UUID
);

CREATE INDEX IF NOT EXISTS idx_validation_rules_process_node ON validation_rules(process_id, node_id);
CREATE INDEX IF NOT EXISTS idx_validation_rules_tenant ON validation_rules(tenant_id);

-- Event Handlers (links steps to events)
CREATE TABLE IF NOT EXISTS event_handlers (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  process_id    UUID NOT NULL REFERENCES processes(id) ON DELETE CASCADE,
  node_id       TEXT NOT NULL,
  event_id      UUID NOT NULL REFERENCES workflow_events(id),
  on_failure    TEXT CHECK (on_failure IN ('reject', 'route', 'escalate')) DEFAULT 'reject',
  escalation_role TEXT,
  config        JSONB DEFAULT '{}'::jsonb,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now(),
  tenant_id     UUID
);

CREATE INDEX IF NOT EXISTS idx_event_handlers_process_node ON event_handlers(process_id, node_id);
CREATE INDEX IF NOT EXISTS idx_event_handlers_event ON event_handlers(event_id);
CREATE INDEX IF NOT EXISTS idx_event_handlers_tenant ON event_handlers(tenant_id);

-- Step Templates (reusable configurations)
CREATE TABLE IF NOT EXISTS step_templates (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name          TEXT NOT NULL,
  description   TEXT,
  step_type_key TEXT NOT NULL,
  config        JSONB NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now(),
  created_by    UUID,
  tenant_id     UUID,
  is_public     BOOLEAN DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_step_templates_tenant ON step_templates(tenant_id);
CREATE INDEX IF NOT EXISTS idx_step_templates_step_type ON step_templates(step_type_key);

-- Rule Templates (reusable validation rule patterns)
CREATE TABLE IF NOT EXISTS rule_templates (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name          TEXT NOT NULL,
  description   TEXT,
  field_pattern TEXT,
  operator_key  TEXT NOT NULL,
  config        JSONB NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now(),
  created_by    UUID,
  tenant_id     UUID,
  is_public     BOOLEAN DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_rule_templates_tenant ON rule_templates(tenant_id);

-- Audit log for process executions
CREATE TABLE IF NOT EXISTS process_execution_log (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  process_id    UUID NOT NULL REFERENCES processes(id),
  execution_id  TEXT NOT NULL,
  node_id       TEXT,
  event_fired   TEXT,
  status        TEXT CHECK (status IN ('started', 'in_progress', 'completed', 'failed')) DEFAULT 'in_progress',
  result        JSONB,
  error_message TEXT,
  created_at    TIMESTAMPTZ DEFAULT now(),
  tenant_id     UUID,
  UNIQUE(process_id, execution_id)
);

CREATE INDEX IF NOT EXISTS idx_process_execution_log_process ON process_execution_log(process_id);
CREATE INDEX IF NOT EXISTS idx_process_execution_log_tenant ON process_execution_log(tenant_id);

-- Grants (ABAC-style permissions for process designer role)
CREATE TABLE IF NOT EXISTS process_designer_permissions (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  role          TEXT NOT NULL,
  process_id    UUID REFERENCES processes(id) ON DELETE CASCADE,
  permission    TEXT CHECK (permission IN ('view', 'edit', 'publish', 'execute')) NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT now(),
  tenant_id     UUID,
  UNIQUE(role, process_id, permission, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_process_designer_permissions_tenant_role ON process_designer_permissions(tenant_id, role);
