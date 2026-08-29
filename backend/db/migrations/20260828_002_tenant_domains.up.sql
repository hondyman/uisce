-- Domain-based tenant discovery for client user auto-provisioning on first
-- login (see design: client users are scoped to exactly one tenant, resolved
-- from their verified email domain, never asserted by the client).
--
-- Additive migration: creates security.tenant_domains and nothing else.
CREATE SCHEMA IF NOT EXISTS security;

CREATE TABLE IF NOT EXISTS security.tenant_domains (
    domain      TEXT PRIMARY KEY,           -- lowercase, e.g. 'acmewealth.com'
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_tenant_domains_tenant_id ON security.tenant_domains(tenant_id);
