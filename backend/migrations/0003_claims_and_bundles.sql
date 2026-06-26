-- +goose Up
CREATE TABLE claim (
  id UUID PRIMARY KEY,
  tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
  user_id TEXT REFERENCES app_user(id) ON DELETE CASCADE,
  asset_id UUID REFERENCES asset(id) ON DELETE CASCADE,
  permission TEXT NOT NULL, -- read, create, update, delete
  scope TEXT[] NOT NULL DEFAULT '{}',
  source TEXT NOT NULL, -- role, bundle, manual, micro_bundle, jit
  granted_by TEXT,
  granted_at TIMESTAMP NOT NULL DEFAULT now(),
  expires_at TIMESTAMP,
  status TEXT NOT NULL DEFAULT 'active' -- active, expired, revoked
);
CREATE INDEX idx_claim_effective ON claim(tenant_id, user_id, asset_id) WHERE status='active';
CREATE INDEX idx_claim_expiry ON claim(expires_at) WHERE status='active';

CREATE TABLE claim_bundle (
  id UUID PRIMARY KEY,
  tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  domain TEXT NOT NULL,
  description TEXT,
  version INT NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'active',
  risk_level TEXT NOT NULL DEFAULT 'medium',
  created_by TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, name, version)
);

CREATE TABLE claim_bundle_item (
  id UUID PRIMARY KEY,
  bundle_id UUID REFERENCES claim_bundle(id) ON DELETE CASCADE,
  asset_id UUID REFERENCES asset(id) ON DELETE CASCADE,
  permission TEXT NOT NULL,
  scope TEXT[] NOT NULL DEFAULT '{}'
);
CREATE INDEX idx_bundle_item_bundle ON claim_bundle_item(bundle_id);

CREATE TABLE user_bundle_assignment (
  id UUID PRIMARY KEY,
  user_id TEXT REFERENCES app_user(id) ON DELETE CASCADE,
  tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
  bundle_id UUID REFERENCES claim_bundle(id) ON DELETE CASCADE,
  assigned_by TEXT,
  assigned_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE (user_id, tenant_id, bundle_id)
);

CREATE TABLE micro_bundle (
  id UUID PRIMARY KEY,
  tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  domain TEXT NOT NULL,
  description TEXT,
  version INT NOT NULL DEFAULT 1,
  created_by TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, name, version)
);

CREATE TABLE jit_addon_grant (
  id UUID PRIMARY KEY,
  user_id TEXT REFERENCES app_user(id) ON DELETE CASCADE,
  tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
  bundle_id UUID REFERENCES micro_bundle(id) ON DELETE CASCADE,
  granted_by TEXT,
  granted_at TIMESTAMP NOT NULL DEFAULT now(),
  expires_at TIMESTAMP NOT NULL,
  reason TEXT,
  status TEXT NOT NULL DEFAULT 'active'
);
CREATE INDEX idx_jit_active ON jit_addon_grant(user_id, tenant_id, expires_at) WHERE status='active';

-- +goose Down
DROP TABLE IF EXISTS jit_addon_grant;
DROP TABLE IF EXISTS micro_bundle;
DROP TABLE IF EXISTS user_bundle_assignment;
DROP TABLE IF EXISTS claim_bundle_item;
DROP TABLE IF EXISTS claim_bundle;
DROP TABLE IF EXISTS claim;
