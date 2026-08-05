-- Rollback: restore NOT NULL constraint on RBAC tenant_id columns

ALTER TABLE bp_roles ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bp_permissions ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bp_field_permissions ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bp_approval_delegations ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bp_delegation_usage_log ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bp_teams ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bp_team_members ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bp_team_permissions ALTER COLUMN tenant_id SET NOT NULL;
