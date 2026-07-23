-- +goose Up
CREATE TABLE IF NOT EXISTS asset (
  id UUID PRIMARY KEY,
  tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  asset_type TEXT NOT NULL, -- view, metric, dimension, dashboard
  domain TEXT NOT NULL,
  certified BOOLEAN NOT NULL DEFAULT FALSE,
  sensitivity TEXT NOT NULL DEFAULT 'medium',
  created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_asset_tenant_domain ON asset(tenant_id, domain);
CREATE INDEX IF NOT EXISTS idx_asset_cert ON asset(tenant_id, certified);

CREATE TABLE IF NOT EXISTS role (
  id UUID PRIMARY KEY,
  tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  UNIQUE (tenant_id, name)
);

CREATE TABLE IF NOT EXISTS role_member (
  role_id UUID REFERENCES role(id) ON DELETE CASCADE,
  user_id TEXT REFERENCES app_user(id) ON DELETE CASCADE,
  tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
  PRIMARY KEY (role_id, user_id, tenant_id)
);

CREATE TABLE IF NOT EXISTS role_claim (
  id UUID PRIMARY KEY,
  role_id UUID REFERENCES role(id) ON DELETE CASCADE,
  asset_id UUID REFERENCES asset(id) ON DELETE CASCADE,
  permission TEXT NOT NULL,
  scope TEXT[] NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_role_claim_role ON role_claim(role_id);
