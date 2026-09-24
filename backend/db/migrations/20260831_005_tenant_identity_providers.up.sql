-- Tenant -> IDP registry: which issuer(s) a tenant trusts tokens from.
-- One tenant can have more than one row (post-M&A: a newly-acquired entity
-- may keep its own Keycloak realm for a transition period). One issuer row
-- can also be shared by multiple tenants via tenant_id IS NULL +
-- tenant_identity_provider_grants, for a deliberately cross-tenant realm
-- (super-admin / professional-services accounts) rather than duplicating
-- the same issuer row per tenant.
--
-- Consumed by security.IssuerRegistry (backend/internal/security/idp_registry.go)
-- to resolve an incoming token's "iss" claim to its trusted JWKS endpoint and
-- the tenant(s) that issuer is registered for — see services.ValidateIssuerTenant.
CREATE SCHEMA IF NOT EXISTS semantic;

CREATE TABLE IF NOT EXISTS semantic.tenant_identity_providers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issuer      TEXT NOT NULL UNIQUE, -- must match the token's "iss" claim exactly
    jwks_uri    TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    -- Cross-tenant issuer (super-admin / professional-services realm): true
    -- means every tenant in tenant_identity_provider_grants for this issuer
    -- is deliberately, explicitly trusted — not an accident of config.
    is_cross_tenant BOOLEAN NOT NULL DEFAULT FALSE,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_by  TEXT NOT NULL DEFAULT 'system'
);

-- Which tenant(s) trust a given issuer. A normal tenant-owned Keycloak
-- realm has exactly one row here; a cross-tenant realm has one row per
-- tenant it's explicitly been granted access to.
CREATE TABLE IF NOT EXISTS semantic.tenant_identity_provider_grants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idp_id      UUID NOT NULL REFERENCES semantic.tenant_identity_providers(id) ON DELETE CASCADE,
    tenant_id   UUID NOT NULL,
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    granted_by  TEXT NOT NULL DEFAULT 'system',
    UNIQUE (idp_id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_idp_grants_tenant
    ON semantic.tenant_identity_provider_grants (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_idp_grants_idp
    ON semantic.tenant_identity_provider_grants (idp_id);
