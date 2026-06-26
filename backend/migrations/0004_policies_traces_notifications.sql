-- +goose Up
CREATE TABLE claim_conflict (
  id UUID PRIMARY KEY,
  user_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  asset_id UUID NOT NULL,
  conflict_type TEXT NOT NULL,
  details JSONB NOT NULL,
  detected_at TIMESTAMP NOT NULL DEFAULT now(),
  resolved_at TIMESTAMP,
  resolution_action TEXT
);

CREATE TABLE claim_usage_log (
  id UUID PRIMARY KEY,
  claim_id UUID REFERENCES role_claim(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  asset_id UUID NOT NULL,
  used_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX idx_usage_claim_time ON claim_usage_log(claim_id, used_at DESC);

CREATE TABLE claim_drift (
  id UUID PRIMARY KEY,
  claim_id UUID REFERENCES role_claim(id) ON DELETE CASCADE,
  drift_type TEXT NOT NULL,
  last_used_at TIMESTAMP,
  detected_at TIMESTAMP NOT NULL DEFAULT now(),
  suggested_action TEXT
);

CREATE TABLE policy (
  id UUID PRIMARY KEY,
  tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  scope TEXT NOT NULL,
  definition JSONB NOT NULL,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  version INT NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE policy_simulation_result (
  id UUID PRIMARY KEY,
  policy_id UUID REFERENCES policy(id) ON DELETE CASCADE,
  simulated_by TEXT NOT NULL,
  simulated_at TIMESTAMP NOT NULL DEFAULT now(),
  affected_claims JSONB,
  affected_users JSONB,
  affected_assets JSONB,
  risk_flags TEXT[],
  notes TEXT
);

CREATE TABLE access_decision_log (
  id UUID PRIMARY KEY,
  user_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  asset_id UUID NOT NULL,
  action TEXT NOT NULL,
  decision TEXT NOT NULL,
  reason TEXT,
  evaluated_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX idx_decision_recent ON access_decision_log(tenant_id, evaluated_at DESC);

CREATE TABLE access_decision_trace (
  id UUID PRIMARY KEY,
  decision_log_id UUID REFERENCES access_decision_log(id) ON DELETE CASCADE,
  user_id TEXT NOT NULL,
  asset_id UUID NOT NULL,
  action TEXT NOT NULL,
  decision TEXT NOT NULL,
  evaluated_claims JSONB NOT NULL,
  matched_policies JSONB NOT NULL,
  tenant_scope TEXT NOT NULL,
  reason TEXT NOT NULL,
  evaluated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE notification_subscription (
  id UUID PRIMARY KEY,
  user_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  asset_id UUID,
  asset_type TEXT,
  event_types TEXT[] NOT NULL,
  delivery_method TEXT NOT NULL
);

CREATE TABLE semantic_notification (
  id UUID PRIMARY KEY,
  event_type TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  asset_id UUID,
  asset_type TEXT,
  affected_users TEXT[],
  message TEXT NOT NULL,
  triggered_by TEXT,
  timestamp TIMESTAMP NOT NULL DEFAULT now(),
  delivery JSONB
);

-- +goose Down
DROP TABLE IF EXISTS semantic_notification;
DROP TABLE IF EXISTS notification_subscription;
DROP TABLE IF EXISTS access_decision_trace;
DROP TABLE IF EXISTS access_decision_log;
DROP TABLE IF EXISTS policy_simulation_result;
DROP TABLE IF EXISTS policy;
DROP TABLE IF EXISTS claim_drift;
DROP TABLE IF EXISTS claim_usage_log;
DROP TABLE IF EXISTS claim_conflict;
