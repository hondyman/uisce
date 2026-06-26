-- Create table for BP instances if it doesn't exist (assuming minimal tables migration might have covered it or not)
-- Using IF NOT EXISTS to be safe, though normally this would be a separate migration
CREATE TABLE IF NOT EXISTS bp_instances (
  id UUID PRIMARY KEY,
  bp_definition_id TEXT NOT NULL,
  tenant_id UUID NOT NULL,
  business_object_type TEXT NOT NULL,
  business_object_id TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create external tasks table
CREATE TABLE external_tasks (
  id UUID PRIMARY KEY,
  bp_instance_id UUID NOT NULL REFERENCES bp_instances(id),
  step_id TEXT NOT NULL,
  system TEXT NOT NULL,              -- 'Salesforce', 'ServiceNow', 'Jira'
  action TEXT NOT NULL,              -- 'create', 'update', 'close', 'comment'
  external_id TEXT,                  -- e.g. SF-CASE-123
  status TEXT NOT NULL,              -- 'created', 'in_progress', 'resolved', 'failed'
  payload JSONB NOT NULL,            -- request payload
  response JSONB,                    -- last response
  llm_decision JSONB,                -- system + reason + metadata
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX external_tasks_instance_idx ON external_tasks (bp_instance_id);
CREATE INDEX external_tasks_system_idx ON external_tasks (system);
CREATE INDEX external_tasks_status_idx ON external_tasks (status);
