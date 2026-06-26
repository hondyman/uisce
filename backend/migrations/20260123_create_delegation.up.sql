-- User delegations (vacation mode)
CREATE TABLE IF NOT EXISTS user_delegation (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  from_user_id TEXT NOT NULL,
  to_user_id TEXT NOT NULL,
  from_date DATE NOT NULL,
  to_date DATE NOT NULL,
  reason TEXT,
  roles TEXT[],                        -- which roles to delegate (empty = all)
  workflows TEXT[],                    -- which workflows to delegate (empty = all)
  status TEXT DEFAULT 'active',        -- active, expired, revoked, paused
  created_at TIMESTAMP DEFAULT NOW(),
  revoked_at TIMESTAMP,
  created_by TEXT,                     -- who created it (usually from_user_id)
  offboarding_id UUID
);
CREATE INDEX IF NOT EXISTS idx_users_delegation_lookup ON user_delegation(from_user_id, to_user_id, from_date, to_date);
CREATE INDEX IF NOT EXISTS idx_users_delegation_status ON user_delegation(status, to_date);
CREATE INDEX IF NOT EXISTS idx_users_delegation_tenant ON user_delegation(tenant_id, from_user_id, status);

-- Employee offboarding
CREATE TABLE IF NOT EXISTS user_offboarding (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  user_id TEXT NOT NULL,             -- who's leaving
  offboarded_by TEXT NOT NULL,       -- admin who initiated
  offboard_date DATE NOT NULL,
  reassign_to_user_id TEXT NOT NULL, -- who takes their approvals
  reason TEXT,                       -- "Left company", "Transferred", etc.
  status TEXT DEFAULT 'active',      -- active, completed, reversed
  pending_count INT DEFAULT 0,       -- how many approvals were reassigned
  created_at TIMESTAMP DEFAULT NOW(),
  completed_at TIMESTAMP,
  UNIQUE (user_id, tenant_id)
);
CREATE INDEX IF NOT EXISTS idx_offboarding_status ON user_offboarding(offboard_date, status);
CREATE INDEX IF NOT EXISTS idx_offboarding_reassign ON user_offboarding(reassign_to_user_id);

-- Delegation usage tracking (which delegation was used for an approval)
CREATE TABLE IF NOT EXISTS delegation_usage (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  delegation_id UUID NOT NULL,
  instance_id UUID NOT NULL,
  approver_role TEXT,
  original_approver_id TEXT,
  delegated_approver_id TEXT,
  action TEXT,                         -- 'approved', 'rejected', 'reassigned'
  action_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_delegation_usage_lookup ON delegation_usage(delegation_id, instance_id);
CREATE INDEX IF NOT EXISTS idx_delegation_usage_original ON delegation_usage(instance_id, original_approver_id);

-- Delegation audit log
CREATE TABLE IF NOT EXISTS delegation_audit (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  delegation_id UUID NOT NULL,
  action TEXT,                         -- 'created', 'revoked', 'paused', 'task_delegated'
  instance_id UUID,                    -- which task was delegated
  actor_user_id TEXT,                  -- who performed the action
  details JSONB,                       -- extra context
  created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_delegation_audit_lookup ON delegation_audit(delegation_id, action);
