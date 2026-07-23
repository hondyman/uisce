-- Create tenants table for first-class tenant management
CREATE TABLE IF NOT EXISTS tenants (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    code        TEXT UNIQUE,
    region      TEXT,
    plan        TEXT NOT NULL DEFAULT 'free',
    max_requests BIGINT,
    window_seconds INT,
    is_suspended BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_tenants_code ON tenants(code);
CREATE INDEX IF NOT EXISTS idx_tenants_region ON tenants(region);
CREATE INDEX IF NOT EXISTS idx_tenants_plan ON tenants(plan);
CREATE INDEX IF NOT EXISTS idx_tenants_is_suspended ON tenants(is_suspended);
