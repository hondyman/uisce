-- User roles for RBAC
CREATE TABLE IF NOT EXISTS user_roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  role TEXT NOT NULL,
  tenant_id UUID NOT NULL,
  assigned_by TEXT,
  assigned_at TIMESTAMP DEFAULT NOW(),
  revoked_at TIMESTAMP,
  UNIQUE(user_id, role, tenant_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_tenant ON user_roles(user_id, tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_tenant ON user_roles(role, tenant_id);

-- Workflow Access Control
CREATE TABLE IF NOT EXISTS workflow_access (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  bp_def_id UUID NOT NULL,
  tenant_id UUID NOT NULL,
  initiator_role TEXT NOT NULL,
  can_initiate_on_behalf_of BOOLEAN DEFAULT false,
  max_concurrent_per_user INT DEFAULT 0,
  rate_limit_per_hour INT DEFAULT 0,
  required_fields TEXT[],
  readonly_fields_after_start TEXT[],
  created_at TIMESTAMP DEFAULT NOW(),
  -- FOREIGN KEY (bp_def_id) REFERENCES business_process_definition(id), -- Assuming table exists, usually safer to be loose in dev
  UNIQUE (bp_def_id, initiator_role)
);
CREATE INDEX IF NOT EXISTS idx_workflow_access_tenant_role ON workflow_access(tenant_id, initiator_role);

-- Instance Creation Log for Rate Limiting
CREATE TABLE IF NOT EXISTS instance_creation_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  tenant_id UUID NOT NULL,
  bp_def_id UUID NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_instance_creation_log_limit ON instance_creation_log(user_id, tenant_id, created_at);

-- User Queue Assignment
CREATE TABLE IF NOT EXISTS user_queue_assignment (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL,
  tenant_id UUID NOT NULL,
  role TEXT NOT NULL,
  queue_name TEXT,
  is_primary BOOLEAN DEFAULT true,
  enabled BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  UNIQUE (user_id, role, tenant_id)
);
CREATE INDEX IF NOT EXISTS idx_user_queue_assignment_lookup ON user_queue_assignment(user_id, role, tenant_id);

-- Materialized View for Approvals Queue
-- Create only if dependent tables exist, otherwise create a no-rows fallback so downstream code doesn't break
DO $do$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class WHERE relname = 'workflow_instance')
     AND EXISTS (SELECT 1 FROM pg_class WHERE relname = 'business_process_step')
     AND EXISTS (SELECT 1 FROM pg_class WHERE relname = 'business_process_definition') THEN
    EXECUTE $$
      DROP MATERIALIZED VIEW IF EXISTS my_approvals_queue;
      CREATE MATERIALIZED VIEW my_approvals_queue AS
      SELECT 
          i.id as instance_id,
          i.bp_key,
          COALESCE(s.step_key, i.current_step_key) as step_key,
          i.current_approver_role,
          i.current_step_key,
          i.created_at as instance_created_at,
          i.sla_expires_at,
          CASE 
              WHEN i.sla_expires_at < NOW() THEN 'BREACHED'
              WHEN i.sla_expires_at < NOW() + INTERVAL '1 hour' THEN 'CRITICAL'
              WHEN i.sla_expires_at < NOW() + INTERVAL '24 hours' THEN 'AT_RISK'
              ELSE 'OK'
          END as sla_status,
          EXTRACT(EPOCH FROM (i.sla_expires_at - NOW())) / 3600 as hours_remaining,
          i.data->>'applicant_name' as applicant_name,
          i.data->>'amount' as amount,
          i.data->>'entity' as entity,
          i.tenant_id,
          i.status,
          i.assigned_to_user,
          json_build_object(
              'step_name', COALESCE(s.label, i.current_step_key),
              'bp_name', bp.name
          ) as metadata
      FROM workflow_instance i
      LEFT JOIN business_process_step s ON i.current_step_key = s.step_key AND i.bp_def_id = s.bp_def_id
      LEFT JOIN business_process_definition bp ON i.bp_def_id = bp.id
      WHERE i.status = 'running'
          AND i.sla_expires_at > NOW() - INTERVAL '7 days'
      ORDER BY i.sla_expires_at ASC;

      CREATE INDEX IF NOT EXISTS idx_my_approvals_role ON my_approvals_queue(current_approver_role);
      CREATE INDEX IF NOT EXISTS idx_my_approvals_sla ON my_approvals_queue(sla_status);
    $$;
  ELSE
    -- fallback: empty materialized view
    EXECUTE $$
      DROP MATERIALIZED VIEW IF EXISTS my_approvals_queue;
      CREATE MATERIALIZED VIEW my_approvals_queue AS SELECT NULL::UUID AS instance_id WHERE false;
    $$;
  END IF;
END
$do$;
