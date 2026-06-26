-- +goose Up
-- Persistent storage for governance access control policies
CREATE TABLE IF NOT EXISTS access_control_policies (
    id uuid PRIMARY KEY,
    policy_id text NOT NULL UNIQUE,
    scope text NOT NULL,
    role text NOT NULL,
    permissions text[] NOT NULL DEFAULT '{}',
    duration_days integer NOT NULL DEFAULT 0,
    requires_certification boolean NOT NULL DEFAULT false,
    max_claims_per_user integer NOT NULL DEFAULT 0,
    approval_threshold integer NOT NULL DEFAULT 0,
    renewal_conditions jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_access_policies_scope ON access_control_policies(scope);
CREATE INDEX IF NOT EXISTS idx_access_policies_role ON access_control_policies(role);
CREATE INDEX IF NOT EXISTS idx_access_policies_created ON access_control_policies(created_at);

-- +goose Down
DROP TABLE IF EXISTS access_control_policies;
